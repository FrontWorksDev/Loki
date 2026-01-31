// Package cli provides TUI functionality using Bubble Tea.
package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the state of the TUI application.
type Model struct {
	message  string
	quitting bool
}

// NewModel creates a new Model with default values.
func NewModel() Model {
	return Model{
		message:  "Hello, World! Press 'q' to quit.",
		quitting: false,
	}
}

// Init initializes the model. It returns an optional initial command.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model accordingly.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the current state of the model as a string.
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}
	return fmt.Sprintf("\n  %s\n\n", m.message)
}

// IsQuitting returns whether the application is in quitting state.
func (m Model) IsQuitting() bool {
	return m.quitting
}

// Message returns the current message.
func (m Model) Message() string {
	return m.message
}
