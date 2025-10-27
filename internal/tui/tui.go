package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mahauni/memrvisualz/internal/tui/models/processes"
	"github.com/mahauni/memrvisualz/internal/tui/models/ram"
)

type sessionState uint

type panel struct {
	name  string
	view  string
	state sessionState
	width float64
}

type TuiModel struct {
	// State
	state      sessionState
	quitting   bool
	suspending bool
	width      int
	height     int

	// Views
	ram       ram.Model
	processes processes.Model
}

const (
	processesView sessionState = iota
	ramView
)

var (
	baseModelStyle = lipgloss.NewStyle().
			Height(5).
			Align(lipgloss.Center, lipgloss.Center).
			BorderStyle(lipgloss.HiddenBorder())
	baseFocusedModelStyle = lipgloss.NewStyle().
				Height(5).
				Align(lipgloss.Center, lipgloss.Center).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("69"))

	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

func NewTuiModel() TuiModel {
	m := TuiModel{
		state:      processesView,
		quitting:   false,
		suspending: false,
	}

	m.ram = ram.New()
	m.processes = processes.New()

	return m
}

func (m TuiModel) Init() tea.Cmd {
	var cmd tea.Cmd

	cmd = tea.Batch(
		tea.EnterAltScreen,
		m.ram.Tick,
		m.processes.Tick,
	)

	return cmd
}

func (m TuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.ResumeMsg:
		m.suspending = false
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "ctrl+z":
			m.suspending = true
			return m, tea.Suspend
		// make better way to visualize the tabs
		case "tab":
			if m.state == processesView {
				m.state = ramView
			} else {
				m.state = processesView
			}
		}

		switch m.state {
		// update whichever model is focused
		case ramView:
			m.ram, cmd = m.ram.Update(msg)
			cmds = append(cmds, cmd)
		case processesView:
			m.processes, cmd = m.processes.Update(msg)
			cmds = append(cmds, cmd)
		}

	case ram.TickMsg:
		m.ram, cmd = m.ram.Update(msg)
		cmds = append(cmds, cmd)
		_, _ = m.processes.Update(msg)
	case processes.TickMsg:
		m.processes, cmd = m.processes.Update(msg)
		cmds = append(cmds, cmd)
		_, _ = m.ram.Update(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// send the resize to the Views
		m.ram, cmd = m.ram.Update(msg)
		cmds = append(cmds, cmd)
		m.processes, cmd = m.processes.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m TuiModel) View() string {
	var s string
	if m.suspending {
		return ""
	}

	if m.quitting {
		return "Bye!\n"
	}

	panels := []panel{
		{"ram", m.ram.View(), ramView, 0.5},
		{"processes", m.processes.View(), processesView, 0.5},
		// later you can just add new ones here
		// {"logs", m.logs.View(), logsView},
	}

	var rendered []string
	for _, p := range panels {
		width := int(float64(m.width) * p.width)
		style := baseModelStyle.Width(width)
		if m.state == p.state {
			style = baseFocusedModelStyle.Width(width)
		}
		rendered = append(rendered, style.Render(fmt.Sprintf("%4s", p.view)))
	}

	s += lipgloss.JoinHorizontal(lipgloss.Top, rendered...)

	model := m.currentFocusedModel()
	s += helpStyle.Render(fmt.Sprintf("\ntab: focus next • n: new %s • q: exit\n", model))
	return s
}

func (m *TuiModel) currentFocusedModel() string {
	if m.state == processesView {
		return "timer"
	}
	return "spinner"
}
