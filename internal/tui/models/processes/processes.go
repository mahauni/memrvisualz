package processes

import (
	"fmt"
	"log"
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

func New() Model {
	var m Model

	p, err := procfs.NewFS("/proc")
	if err != nil {
		log.Fatalf("could not get process: %s", err)
	}

	// stat, err := p.Stat()
	// if err != nil {
	// 	log.Fatalf("could not get process stat: %s", err)
	// }

	columns := []table.Column{
		{Title: "Command", Width: 20},
		{Title: "RAM", Width: 20},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(7),
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

		meminfo, _ := m.procfs.Meminfo()

		// Meminfo fields are in KB, convert to bytes
		totalBytes := *meminfo.MemTotal * uint64(1024)
		freeBytes := *meminfo.MemFree * uint64(1024)
		buffersBytes := *meminfo.Buffers * uint64(1024)
		cachedBytes := *meminfo.Cached * uint64(1024)

		usedBytes := totalBytes - freeBytes - buffersBytes - cachedBytes
		usedGB := float64(usedBytes) / (1024 * 1024 * 1024)
		totalGB := float64(totalBytes) / (1024 * 1024 * 1024)

		m.table.SetRows([]table.Row{
			{
				"Testing",
				fmt.Sprintf("%.2fGiB/%.2fGiB", usedGB, totalGB),
			},
		})

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
