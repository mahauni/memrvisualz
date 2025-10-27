package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mahauni/memrvisualz/internal/tui"
)

func main() {
	if _, err := tea.NewProgram(tui.NewTuiModel(), tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
