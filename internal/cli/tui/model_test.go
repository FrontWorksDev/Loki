package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/FrontWorksDev/Loki/pkg/processor"
	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModel(t *testing.T) {
	m := NewModel()
	if m.State() != StateWaiting {
		t.Errorf("NewModel().State() = %v, want StateWaiting", m.State())
	}
	if m.TotalFiles() != 0 {
		t.Errorf("NewModel().TotalFiles() = %d, want 0", m.TotalFiles())
	}
	if m.Completed() != 0 {
		t.Errorf("NewModel().Completed() = %d, want 0", m.Completed())
	}
	if m.Failed() != 0 {
		t.Errorf("NewModel().Failed() = %d, want 0", m.Failed())
	}
}

func TestModel_BatchStartMsg(t *testing.T) {
	m := NewModel()
	updated, _ := m.Update(BatchStartMsg{TotalFiles: 5})
	um := updated.(Model)

	if um.State() != StateProcessing {
		t.Errorf("State() = %v, want StateProcessing", um.State())
	}
	if um.TotalFiles() != 5 {
		t.Errorf("TotalFiles() = %d, want 5", um.TotalFiles())
	}
}

func TestModel_ProgressMsg(t *testing.T) {
	tests := []struct {
		name        string
		progress    processor.Progress
		wantComp    int
		wantFailed  int
		wantCurrent string
	}{
		{
			name: "1件完了",
			progress: processor.Progress{
				Total: 3, Completed: 1, Failed: 0, Current: "photo.jpg",
			},
			wantComp: 1, wantFailed: 0, wantCurrent: "photo.jpg",
		},
		{
			name: "1件成功1件失敗",
			progress: processor.Progress{
				Total: 3, Completed: 1, Failed: 1, Current: "icon.png",
			},
			wantComp: 1, wantFailed: 1, wantCurrent: "icon.png",
		},
		{
			name: "全件完了",
			progress: processor.Progress{
				Total: 3, Completed: 3, Failed: 0, Current: "last.jpg",
			},
			wantComp: 3, wantFailed: 0, wantCurrent: "last.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel()
			// Start batch first
			started, _ := m.Update(BatchStartMsg{TotalFiles: tt.progress.Total})
			m = started.(Model)

			updated, _ := m.Update(ProgressMsg{Progress: tt.progress})
			um := updated.(Model)

			if um.Completed() != tt.wantComp {
				t.Errorf("Completed() = %d, want %d", um.Completed(), tt.wantComp)
			}
			if um.Failed() != tt.wantFailed {
				t.Errorf("Failed() = %d, want %d", um.Failed(), tt.wantFailed)
			}
			if um.CurrentFile() != tt.wantCurrent {
				t.Errorf("CurrentFile() = %q, want %q", um.CurrentFile(), tt.wantCurrent)
			}
		})
	}
}

func TestModel_BatchCompleteMsg(t *testing.T) {
	m := NewModel()
	started, _ := m.Update(BatchStartMsg{TotalFiles: 3})
	m = started.(Model)

	results := []processor.BatchResult{
		{Item: processor.BatchItem{InputPath: "a.jpg"}, Result: &processor.Result{}},
		{Item: processor.BatchItem{InputPath: "b.jpg"}, Result: &processor.Result{}},
		{Item: processor.BatchItem{InputPath: "c.jpg"}, Error: errors.New("decode error")},
	}
	updated, _ := m.Update(BatchCompleteMsg{
		Results:      results,
		SuccessCount: 2,
		FailCount:    1,
	})
	um := updated.(Model)

	if um.State() != StateCompleted {
		t.Errorf("State() = %v, want StateCompleted", um.State())
	}
	if um.SuccessCount() != 2 {
		t.Errorf("SuccessCount() = %d, want 2", um.SuccessCount())
	}
	if um.FailCount() != 1 {
		t.Errorf("FailCount() = %d, want 1", um.FailCount())
	}
}

func TestModel_BatchErrorMsg(t *testing.T) {
	m := NewModel()
	testErr := errors.New("scan failed")

	updated, _ := m.Update(BatchErrorMsg{Err: testErr})
	um := updated.(Model)

	if um.State() != StateError {
		t.Errorf("State() = %v, want StateError", um.State())
	}
	if um.Err() != testErr {
		t.Errorf("Err() = %v, want %v", um.Err(), testErr)
	}
}

func TestModel_QuitKeys(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{name: "qキー", key: "q"},
		{name: "ctrl+c", key: "ctrl+c"},
		{name: "escキー", key: "esc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel()
			_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})

			// For ctrl+c and esc, use specific key types
			if tt.key == "ctrl+c" {
				_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
			} else if tt.key == "esc" {
				_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
			}

			if cmd == nil {
				t.Error("expected a quit command, got nil")
			}
		})
	}
}

func TestModel_View(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() Model
		contains []string
	}{
		{
			name: "Waiting状態",
			setup: func() Model {
				return NewModel()
			},
			contains: []string{"処理を開始しています..."},
		},
		{
			name: "Processing状態",
			setup: func() Model {
				m := NewModel()
				updated, _ := m.Update(BatchStartMsg{TotalFiles: 10})
				m = updated.(Model)
				updated, _ = m.Update(ProgressMsg{
					Progress: processor.Progress{
						Total: 10, Completed: 3, Failed: 0, Current: "photo.jpg",
					},
				})
				return updated.(Model)
			},
			contains: []string{"[3/10]", "photo.jpg"},
		},
		{
			name: "Completed状態",
			setup: func() Model {
				m := NewModel()
				updated, _ := m.Update(BatchStartMsg{TotalFiles: 2})
				m = updated.(Model)
				updated, _ = m.Update(BatchCompleteMsg{
					Results:      []processor.BatchResult{},
					SuccessCount: 2,
					FailCount:    0,
				})
				return updated.(Model)
			},
			contains: []string{"完了", "成功 2", "失敗 0"},
		},
		{
			name: "Completed状態_失敗あり",
			setup: func() Model {
				m := NewModel()
				updated, _ := m.Update(BatchStartMsg{TotalFiles: 2})
				m = updated.(Model)
				updated, _ = m.Update(BatchCompleteMsg{
					Results: []processor.BatchResult{
						{Item: processor.BatchItem{InputPath: "bad.jpg"}, Error: errors.New("decode error")},
					},
					SuccessCount: 1,
					FailCount:    1,
				})
				return updated.(Model)
			},
			contains: []string{"完了", "成功 1", "失敗 1", "失敗ファイル", "bad.jpg", "decode error"},
		},
		{
			name: "Error状態",
			setup: func() Model {
				m := NewModel()
				updated, _ := m.Update(BatchErrorMsg{Err: errors.New("scan failed")})
				return updated.(Model)
			},
			contains: []string{"エラー", "scan failed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setup()
			view := m.View()
			for _, s := range tt.contains {
				if !strings.Contains(view, s) {
					t.Errorf("View() に %q が含まれていません。\n出力:\n%s", s, view)
				}
			}
		})
	}
}
