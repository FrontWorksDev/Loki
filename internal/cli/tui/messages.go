// Package tui provides Bubble Tea based TUI components for image compression progress display.
package tui

import "github.com/FrontWorksDev/Loki/pkg/processor"

// ProgressMsg wraps processor.Progress for Bubble Tea message passing.
type ProgressMsg struct {
	Progress processor.Progress
}

// BatchStartMsg notifies the TUI that batch processing has started.
type BatchStartMsg struct {
	TotalFiles int
}

// BatchCompleteMsg notifies the TUI that batch processing has completed.
type BatchCompleteMsg struct {
	Results []processor.BatchResult
}

// BatchErrorMsg notifies the TUI that an overall batch processing error occurred.
type BatchErrorMsg struct {
	Err error
}
