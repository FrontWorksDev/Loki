package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

// State represents the TUI state.
type State int

const (
	// StateWaiting indicates the TUI is waiting for processing to start.
	StateWaiting State = iota
	// StateProcessing indicates batch processing is in progress.
	StateProcessing
	// StateCompleted indicates batch processing has completed.
	StateCompleted
	// StateError indicates an error occurred during batch processing.
	StateError
)

// Model is the Bubble Tea model for the progress bar TUI.
type Model struct {
	state       State
	progress    progress.Model
	totalFiles  int
	completed   int
	failed      int
	currentFile string
	results     []BatchResultInfo
	err         error
}

// BatchResultInfo holds summary info about a failed batch result for display.
type BatchResultInfo struct {
	InputPath string
	Error     string
}

// NewModel creates a new TUI model with default settings.
func NewModel() Model {
	p := progress.New(
		progress.WithDefaultGradient(),
	)
	return Model{
		state:    StateWaiting,
		progress: p,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - 8
		if m.progress.Width < 20 {
			m.progress.Width = 20
		}
		return m, nil

	case BatchStartMsg:
		m.state = StateProcessing
		m.totalFiles = msg.TotalFiles
		return m, nil

	case ProgressMsg:
		p := msg.Progress
		m.completed = p.Completed
		m.failed = p.Failed
		m.currentFile = p.Current
		var percent float64
		if m.totalFiles > 0 {
			percent = float64(m.completed+m.failed) / float64(m.totalFiles)
		}
		cmd := m.progress.SetPercent(percent)
		return m, cmd

	case BatchCompleteMsg:
		m.state = StateCompleted
		m.completed = msg.SuccessCount
		m.failed = msg.FailCount
		m.results = make([]BatchResultInfo, 0)
		for _, r := range msg.Results {
			if r.Error != nil {
				m.results = append(m.results, BatchResultInfo{
					InputPath: r.Item.InputPath,
					Error:     r.Error.Error(),
				})
			}
		}
		cmd := m.progress.SetPercent(1.0)
		return m, cmd

	case BatchErrorMsg:
		m.state = StateError
		m.err = msg.Err
		return m, nil

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	var b strings.Builder

	switch m.state {
	case StateWaiting:
		b.WriteString("\n  処理を開始しています...\n\n")

	case StateProcessing:
		processed := m.completed + m.failed
		b.WriteString("\n")
		b.WriteString("  " + m.progress.View() + "\n\n")
		b.WriteString(fmt.Sprintf("  [%d/%d] %s\n\n", processed, m.totalFiles, m.currentFile))

	case StateCompleted:
		successCount := m.completed
		failCount := m.failed
		b.WriteString("\n")
		b.WriteString("  " + m.progress.View() + "\n\n")
		b.WriteString(fmt.Sprintf("  完了: 成功 %d, 失敗 %d\n", successCount, failCount))
		if len(m.results) > 0 {
			b.WriteString("\n  失敗ファイル:\n")
			for _, r := range m.results {
				b.WriteString(fmt.Sprintf("    - %s: %s\n", r.InputPath, r.Error))
			}
		}
		b.WriteString("\n  qキーで終了\n\n")

	case StateError:
		b.WriteString(fmt.Sprintf("\n  エラー: %v\n\n  qキーで終了\n\n", m.err))
	}

	return b.String()
}

// State returns the current state of the model.
func (m Model) State() State {
	return m.state
}

// TotalFiles returns the total number of files to process.
func (m Model) TotalFiles() int {
	return m.totalFiles
}

// Completed returns the number of successfully completed files.
func (m Model) Completed() int {
	return m.completed
}

// Failed returns the number of failed files.
func (m Model) Failed() int {
	return m.failed
}

// CurrentFile returns the currently processing file name.
func (m Model) CurrentFile() string {
	return m.currentFile
}

// SuccessCount returns the number of successful results after completion.
func (m Model) SuccessCount() int {
	return m.completed
}

// FailCount returns the number of failed results after completion.
func (m Model) FailCount() int {
	return m.failed
}

// Err returns the error if the model is in error state.
func (m Model) Err() error {
	return m.err
}
