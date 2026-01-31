// Package main provides a demo application for the Bubble Tea TUI.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/FrontWorksDev/Loki/internal/cli"
)

func main() {
	p := tea.NewProgram(cli.NewModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
