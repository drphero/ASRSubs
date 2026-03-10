package main

import (
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type DiagnosticsEntry struct {
	ID        string `json:"id"`
	Level     string `json:"level"`
	Source    string `json:"source"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

type DiagnosticsSummary struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	Level   string `json:"level"`
}

type DiagnosticsSnapshot struct {
	Summary DiagnosticsSummary `json:"summary"`
	Entries []DiagnosticsEntry `json:"entries"`
}

type diagnosticsState struct {
	mu      sync.RWMutex
	entries []DiagnosticsEntry
}

func (a *App) initDiagnostics() {
	a.diagnostics = diagnosticsState{}
}

func (a *App) GetDiagnosticsSnapshot() DiagnosticsSnapshot {
	a.diagnostics.mu.RLock()
	defer a.diagnostics.mu.RUnlock()

	return diagnosticsSnapshot(a.diagnostics.entries)
}

func (a *App) recordDiagnostic(level string, source string, message string) {
	entry := DiagnosticsEntry{
		ID:        time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level,
		Source:    source,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	a.diagnostics.mu.Lock()
	a.diagnostics.entries = append(a.diagnostics.entries, entry)
	if len(a.diagnostics.entries) > 100 {
		a.diagnostics.entries = append([]DiagnosticsEntry(nil), a.diagnostics.entries[len(a.diagnostics.entries)-100:]...)
	}
	snapshot := diagnosticsSnapshot(a.diagnostics.entries)
	a.diagnostics.mu.Unlock()

	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "diagnostics:entry", snapshot)
	}
}

func diagnosticsSnapshot(entries []DiagnosticsEntry) DiagnosticsSnapshot {
	copied := append([]DiagnosticsEntry(nil), entries...)
	summary := DiagnosticsSummary{
		Title:   "Ready",
		Message: "Logs will appear here as the app does work.",
		Level:   "info",
	}

	if len(copied) > 0 {
		last := copied[len(copied)-1]
		summary = DiagnosticsSummary{
			Title:   diagnosticsTitle(last.Level),
			Message: last.Message,
			Level:   last.Level,
		}
	}

	return DiagnosticsSnapshot{
		Summary: summary,
		Entries: copied,
	}
}

func diagnosticsTitle(level string) string {
	switch level {
	case "error":
		return "Needs attention"
	case "warning":
		return "Check this state"
	default:
		return "App activity"
	}
}
