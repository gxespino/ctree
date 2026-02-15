package ui

import (
	"time"

	"github.com/gxespino/cmux/internal/model"
)

// tickMsg triggers the next poll cycle.
type tickMsg time.Time

// pollResultMsg carries fresh tmux + process detection data.
type pollResultMsg struct {
	windows []model.Window
	err     error
}

// gitResultMsg carries git metadata for a specific window.
type gitResultMsg struct {
	windowID string
	branch   string
	added    int
	removed  int
	dirty    bool
	err      error
}

// errMsg wraps any error.
type errMsg struct{ err error }

// jumpedMsg indicates user jumped to a window.
type jumpedMsg struct {
	windowID string
}

// newWorkspaceResultMsg indicates whether new workspace creation succeeded.
type newWorkspaceResultMsg struct {
	err error
}
