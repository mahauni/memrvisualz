package memory

import (
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/prometheus/procfs"
)

var lastID int64

func nextID() int {
	return int(atomic.AddInt64(&lastID, 1))
}

const (
	padding  = 2
	maxWidth = 80
)

type TickMsg struct {
	tag int
	ID  int
}

type Model struct {
	procfs   procfs.FS
	percent  float64
	progress progress.Model

	Interval time.Duration

	tag int
	id  int
}

func New() Model {
	var m Model

	p, err := procfs.NewFS("/proc")
	if err != nil {
		log.Fatalf("could not get process: %s", err)
	}

	m = Model{
		procfs:   p,
		Interval: time.Second,
		id:       nextID(),
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, nil

	case tea.WindowSizeMsg:
		m.progress.Width = min(msg.Width-padding*2-4, maxWidth)
		return m, nil

	case TickMsg:
		if msg.ID > 0 && msg.ID != m.id {
			return m, nil
		}

		if msg.tag > 0 && msg.tag != m.tag {
			return m, nil
		}

		m.percent += 0.25
		if m.percent > 1.0 {
			m.percent = 1.0
			return m, nil
		}

		m.tag++
		return m, m.tick(m.id, m.tag)

	default:
		return m, nil
	}
}

func (m Model) View() string {
	pad := strings.Repeat(" ", padding)
	return "\n" +
		pad + m.progress.ViewAs(m.percent)
}

func (m Model) ID() int {
	return m.id
}

func (m Model) Tick() tea.Msg {
	return TickMsg{
		ID:  m.id,
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
