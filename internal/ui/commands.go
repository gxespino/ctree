package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gxespino/cmux/internal/detect"
	"github.com/gxespino/cmux/internal/git"
	"github.com/gxespino/cmux/internal/hookdata"
	"github.com/gxespino/cmux/internal/model"
	"github.com/gxespino/cmux/internal/tmux"
)

var pollCount int

const pollInterval = 1500 * time.Millisecond

// tickCmd schedules the next poll after a delay.
func tickCmd() tea.Cmd {
	return tea.Tick(pollInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// pollTmuxCmd discovers tmux panes and enriches them with Claude status.
func pollTmuxCmd() tea.Cmd {
	return func() tea.Msg {
		// Periodically clean up stale hook status files (~every 15s)
		pollCount++
		if pollCount%10 == 0 {
			hookdata.Cleanup(5 * time.Minute)
		}

		allPanes, err := tmux.ListAllPanes()
		if err != nil {
			return errMsg{err}
		}

		// Single-pass detection: prefers hook status, falls back to pane scraping
		detect.EnrichAll(allPanes)

		// Filter to only Claude panes
		var result []model.Window
		for _, w := range allPanes {
			if w.IsClaudePane {
				result = append(result, w)
			}
		}

		return pollResultMsg{windows: result}
	}
}

// pollGitCmd fetches git metadata for a single window's working directory.
func pollGitCmd(windowID, workingDir string) tea.Cmd {
	return func() tea.Msg {
		branch, added, removed, dirty, err := git.GetStats(workingDir)
		return gitResultMsg{
			windowID: windowID,
			branch:   branch,
			added:    added,
			removed:  removed,
			dirty:    dirty,
			err:      err,
		}
	}
}

// jumpToWindowCmd switches tmux focus to the given window.
func jumpToWindowCmd(sessionName string, windowIndex int) tea.Cmd {
	return func() tea.Msg {
		err := tmux.SelectWindow(sessionName, windowIndex)
		if err != nil {
			return errMsg{err}
		}
		return jumpedMsg{windowID: fmt.Sprintf("%s:%d", sessionName, windowIndex)}
	}
}

// newWorkspaceCmd creates a new tmux window running claude.
func newWorkspaceCmd() tea.Cmd {
	return func() tea.Msg {
		err := tmux.NewClaudeWindow("", "")
		return newWorkspaceResultMsg{err: err}
	}
}

// bellCmd plays a terminal bell (BEL character).
func bellCmd() tea.Cmd {
	return tea.Printf("\a")
}
