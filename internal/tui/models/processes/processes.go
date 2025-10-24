package processes

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/prometheus/procfs"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

var lastID int64

func nextID() int {
	return int(atomic.AddInt64(&lastID, 1))
}

type Model struct {
	table  table.Model
	procfs procfs.FS

	Interval time.Duration

	tag int
	id  int
}

type TickMsg struct {
	tag int
	ID  int
}

type procAggregate struct {
	name  string
	user  string
	cpu   float64
	mem   float64
	count int
}

func New() Model {
	var m Model

	p, err := procfs.NewFS("/proc")
	if err != nil {
		log.Fatalf("could not get process: %s", err)
	}

	columns := []table.Column{
		{Title: "USER", Width: 20},
		{Title: "NAME", Width: 10},
		{Title: "COUNT", Width: 10},
		{Title: "CPU (%)", Width: 10},
		{Title: "MEM (MiB)", Width: 20},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(18),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	m = Model{
		table:    t,
		procfs:   p,
		Interval: time.Second,
		id:       nextID(),
	}

	return m
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			return m, tea.Batch(
				tea.Printf("Let's go to %s!", m.table.SelectedRow()[1]),
			)
		}

	case TickMsg:
		if msg.ID > 0 && msg.ID != m.id {
			return m, nil
		}

		if msg.tag > 0 && msg.tag != m.tag {
			return m, nil
		}

		fs := m.procfs

		total1, _ := fs.Stat()
		procs, _ := fs.AllProcs()
		cpu1 := make(map[int]float64)
		for _, p := range procs {
			stat, err := p.Stat()
			if err != nil {
				continue
			}
			cpu1[p.PID] = stat.CPUTime()
		}

		procs, _ = fs.AllProcs()
		total2, _ := fs.Stat()
		totalDelta := float64(
			(total2.CPUTotal.User - total1.CPUTotal.User) +
				(total2.CPUTotal.System - total1.CPUTotal.System) +
				(total2.CPUTotal.Idle - total1.CPUTotal.Idle) +
				(total2.CPUTotal.Iowait - total1.CPUTotal.Iowait) +
				(total2.CPUTotal.Nice - total1.CPUTotal.Nice) +
				(total2.CPUTotal.Steal - total1.CPUTotal.Steal),
		)

		aggregates := make(map[string]*procAggregate)

		for _, p := range procs {
			stat, err := p.Stat()
			if err != nil {
				continue
			}

			status, err := p.NewStatus()
			if err != nil {
				continue
			}

			usr, err := user.LookupId(fmt.Sprintf("%d", status.UIDs[1]))
			if err != nil {
				continue
			}

			memMB := getPrivateMemory(p.PID)
			cpuPct := 0.0
			if prev, ok := cpu1[p.PID]; ok && totalDelta > 0 {
				delta := float64(stat.CPUTime() - prev)
				cpuPct = (delta / totalDelta) * 100.0
			}

			key := stat.Comm
			if agg, ok := aggregates[key]; ok {
				agg.cpu += cpuPct
				agg.mem += memMB
				agg.count++
			} else {
				aggregates[key] = &procAggregate{
					name:  key,
					user:  usr.Username,
					cpu:   cpuPct,
					mem:   memMB,
					count: 1,
				}
			}
		}

		var aggs []*procAggregate
		for _, agg := range aggregates {
			aggs = append(aggs, agg)
		}

		sort.Slice(aggs, func(i, j int) bool { return aggs[i].mem > aggs[j].mem })

		var rows []table.Row
		for _, agg := range aggs {
			rows = append(rows, table.Row{
				agg.name,
				agg.user,
				fmt.Sprintf("%dx", agg.count),
				fmt.Sprintf("%.1f%%", agg.cpu),
				fmt.Sprintf("%.2f MiB", agg.mem),
			})
		}

		m.table.SetRows(rows)

		m.tag++
		return m, m.tick(m.id, m.tag)
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return baseStyle.Render(m.table.View()) + "\n"
}

func (m Model) ID() int {
	return m.id
}

func (m Model) Tick() tea.Msg {
	return TickMsg{
		ID: m.id,

		tag: m.tag,
	}
}

func (m Model) tick(id, tag int) tea.Cmd {
	return tea.Tick(m.Interval, func(t time.Time) tea.Msg {
		return TickMsg{
			ID:  id,
			tag: tag,
		}
	})
}

func getPrivateMemory(pid int) float64 {
	filePath := fmt.Sprintf("/proc/%d/statm", pid)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0
	}

	// statm fields: size resident shared text lib data dt
	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return 0
	}

	resident, err1 := strconv.Atoi(fields[1])
	shared, err2 := strconv.Atoi(fields[2])
	if err1 != nil || err2 != nil {
		return 0
	}

	privatePages := max(resident-shared, 0)

	// convert pages to MiB (page size usually 4096 bytes)
	return float64(privatePages*4096) / (1024 * 1024)
}
