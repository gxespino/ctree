package model

import (
	"fmt"
	"strings"
	"time"
)

// Status represents the current state of a Claude session window.
type Status int

const (
	StatusUnknown Status = iota
	StatusWorking        // Claude is actively processing
	StatusPaused         // Claude is waiting for user input (permission, question)
	StatusIdle           // At prompt, user has already seen output, nothing to do
	StatusUnread         // New output since user last visited, needs attention
	StatusDone           // Finished working, no input required from user
	StatusError
	StatusExited
)

func (s Status) String() string {
	switch s {
	case StatusWorking:
		return "Working…"
	case StatusPaused:
		return "Paused"
	case StatusIdle:
		return "Idle"
	case StatusUnread:
		return "Unread"
	case StatusDone:
		return "Done"
	case StatusError:
		return "Error"
	case StatusExited:
		return "Exited"
	default:
		return "?"
	}
}

// Window represents a tmux window that is (or was) running Claude.
type Window struct {
	SessionName string
	WindowIndex int
	WindowID    string
	WindowName  string
	PaneID      string
	PanePID     int

	WorkingDir   string
	Status       Status
	LastActivity time.Time

	GitBranch  string
	GitAdded   int
	GitRemoved int
	GitDirty   bool

	ClaudePID      int
	IsClaudePane   bool
	IsActiveWindow bool
}

// FilterValue implements bubbles/list.Item for search/filter.
func (w Window) FilterValue() string {
	return w.WindowName + " " + w.WorkingDir + " " + w.GitBranch
}

// Title returns the display name for this window.
func (w Window) Title() string {
	if w.WorkingDir != "" {
		parts := strings.Split(w.WorkingDir, "/")
		if last := parts[len(parts)-1]; last != "" {
			return last
		}
	}
	return w.WindowName
}

// Description returns a secondary line for this window.
func (w Window) Description() string {
	parts := []string{w.Status.String()}
	if !w.LastActivity.IsZero() {
		parts = append(parts, RelativeTime(w.LastActivity))
	}
	return strings.Join(parts, " · ")
}

// Target returns the tmux target string for this window.
func (w Window) Target() string {
	return fmt.Sprintf("%s:%d", w.SessionName, w.WindowIndex)
}

// RelativeTime formats a time as a human-readable relative duration.
func RelativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
