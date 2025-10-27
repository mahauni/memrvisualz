package ram

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

	tslc "github.com/NimbleMarkets/ntcharts/linechart/timeserieslinechart"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/prometheus/procfs"
)

var lastID int64

func nextID() int {
	return int(atomic.AddInt64(&lastID, 1))
}

type Model struct {
	procfs procfs.FS
	zone   *zone.Manager
	chart  tslc.Model
	points []tslc.TimePoint

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

	// create new time series chart
	width := 36
	height := 10
	minYValue := 0.0
	maxYValue := 100.0

	zoneManager := zone.New()

	chart := tslc.New(width, height)
	chart.XLabelFormatter = tslc.HourTimeLabelFormatter()
	chart.UpdateHandler = tslc.SecondUpdateHandler(1)
	chart.SetYRange(minYValue, maxYValue)
	chart.SetViewYRange(minYValue, maxYValue)
	chart.SetZoneManager(zoneManager)

	// chart.SetYStep(0)
	// chart.SetXStep(0)

	// set default data set line color to red
	chart.SetStyle(
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")), // red
	)

	m = Model{
		procfs:   p,
		zone:     zoneManager,
		chart:    chart,
		Interval: time.Second,
		id:       nextID(),
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case TickMsg:
		// this is strange, because its printing twice
		if msg.ID > 0 && msg.ID != m.id {
			return m, nil
		}

		if msg.tag > 0 && msg.tag != m.tag {
			return m, nil
		}

		// make here the progress shit

		meminfo, err := m.procfs.Meminfo()
		if err == nil {
			total := float64(*meminfo.MemTotal)
			free := float64(*meminfo.MemFree)
			buffers := float64(*meminfo.Buffers)
			cached := float64(*meminfo.Cached)
			used := total - free - buffers - cached
			usedPercent := (used / total) * 100.0

			fmt.Println(usedPercent)

			// Add point
			m.points = append(m.points, tslc.TimePoint{
				Time:  time.Now(),
				Value: usedPercent,
			})

			// Keep only last N points (optional)
			if len(m.points) > 100 {
				m.points = m.points[len(m.points)-100:]
			}

			// Update chart
			for _, v := range m.points {
				m.chart.Push(v)
			}
		}

		m.tag++
		return m, m.tick(m.id, m.tag)
	}

	m.chart, cmd = m.chart.Update(msg)
	m.chart.DrawBrailleAll()
	return m, cmd
}

func (m Model) View() string {
	s := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Render("Used RAM"),
		lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("63")). // purple
			Render(m.chart.View()),
	)

	return m.zone.Scan(s)
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
