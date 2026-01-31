package cli

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModel(t *testing.T) {
	m := NewModel()

	if m.Message() != "Hello, World! Press 'q' to quit." {
		t.Errorf("expected default message, got %q", m.Message())
	}

	if m.IsQuitting() {
		t.Error("expected IsQuitting() to be false initially")
	}
}

func TestModelInit(t *testing.T) {
	m := NewModel()
	cmd := m.Init()

	if cmd != nil {
		t.Error("expected Init() to return nil")
	}
}

func TestModelUpdateQuit(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"q key", "q"},
		{"ctrl+c", "ctrl+c"},
		{"esc key", "esc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel()
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			if tt.key == "ctrl+c" {
				keyMsg = tea.KeyMsg{Type: tea.KeyCtrlC}
			} else if tt.key == "esc" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
			}

			updatedModel, cmd := m.Update(keyMsg)
			updated := updatedModel.(Model)

			if !updated.IsQuitting() {
				t.Errorf("expected IsQuitting() to be true after pressing %s", tt.key)
			}

			if cmd == nil {
				t.Error("expected a quit command")
			}
		})
	}
}

func TestModelUpdateOtherKey(t *testing.T) {
	m := NewModel()
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}

	updatedModel, cmd := m.Update(keyMsg)
	updated := updatedModel.(Model)

	if updated.IsQuitting() {
		t.Error("expected IsQuitting() to be false after pressing 'a'")
	}

	if cmd != nil {
		t.Error("expected no command after pressing 'a'")
	}
}

func TestModelView(t *testing.T) {
	t.Run("normal view", func(t *testing.T) {
		m := NewModel()
		view := m.View()

		if view != "\n  Hello, World! Press 'q' to quit.\n\n" {
			t.Errorf("unexpected view: %q", view)
		}
	})

	t.Run("quitting view", func(t *testing.T) {
		m := NewModel()
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
		updatedModel, _ := m.Update(keyMsg)
		updated := updatedModel.(Model)

		view := updated.View()
		if view != "Goodbye!\n" {
			t.Errorf("expected 'Goodbye!' view, got %q", view)
		}
	})
}
