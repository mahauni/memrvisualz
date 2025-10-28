package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mahauni/memrvisualz/internal/tui/models/memory"
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
	memory    memory.Model
}

const (
	processesView sessionState = iota
	ramView
	memoryView

	// Put the views behind this one to make the tab work
	totalViews
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
	m.memory = memory.New()

	return m
}

func (m TuiModel) Init() tea.Cmd {
	var cmd tea.Cmd

	cmd = tea.Batch(
		tea.EnterAltScreen,
		m.ram.Tick,
		m.processes.Tick,
		m.memory.Tick,
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
			m.state = sessionState((int(m.state) + 1) % int(totalViews))
		}

		switch m.state {
		// update whichever model is focused
		case ramView:
			m.ram, cmd = m.ram.Update(msg)
			cmds = append(cmds, cmd)
		case processesView:
			m.processes, cmd = m.processes.Update(msg)
			cmds = append(cmds, cmd)
		case memoryView:
			m.memory, cmd = m.memory.Update(msg)
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
	case memory.TickMsg:
		m.memory, cmd = m.memory.Update(msg)
		cmds = append(cmds, cmd)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// send the resize to the Views
		m.ram, cmd = m.ram.Update(msg)
		cmds = append(cmds, cmd)
		m.processes, cmd = m.processes.Update(msg)
		cmds = append(cmds, cmd)
		m.memory, cmd = m.memory.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m TuiModel) View() string {
	if m.suspending {
		return ""
	}
	if m.quitting {
		return "Bye!\n"
	}

	// Define your layout
	panelRows := [][]panel{
		// Row 1: 2 panels
		{
			{"ram", m.ram.View(), ramView, 0.5},
			{"processes", m.processes.View(), processesView, 0.5},
		},
		// Row 2: 3 panels
		{
			{"memory", m.memory.View(), memoryView, 0.33},
			// {"cpu", m.cpu.View(), cpuView, 0.33},
			// {"disk", m.disk.View(), diskView, 0.34},
		},
	}

	var rows []string

	for _, rowPanels := range panelRows {
		var renderedPanels []string

		for _, p := range rowPanels {
			width := int(float64(m.width) * p.width)
			style := baseModelStyle.Width(width)
			if m.state == p.state {
				style = baseFocusedModelStyle.Width(width)
			}
			renderedPanels = append(renderedPanels, style.Render(p.view))
		}

		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, renderedPanels...))
	}

	// Find current panel name for help text
	var currentPanelName string
	for _, rowPanels := range panelRows {
		for _, p := range rowPanels {
			if m.state == p.state {
				currentPanelName = p.name
				break
			}
		}
	}

	s := lipgloss.JoinVertical(lipgloss.Left, rows...)
	s += helpStyle.Render(fmt.Sprintf("\ntab: focus next • viewing: %s • q: exit\n", currentPanelName))

	return s
}
