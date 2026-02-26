package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gxespino/ctree/internal/detect"
	"github.com/gxespino/ctree/internal/git"
	"github.com/gxespino/ctree/internal/hookdata"
	"github.com/gxespino/ctree/internal/model"
	"github.com/gxespino/ctree/internal/tmux"
)

var pollCount int

const pollInterval = 250 * time.Millisecond

// capturePreviewCmd fetches visible pane content for the preview panel.
func capturePreviewCmd(paneID string, maxLines, maxWidth int) tea.Cmd {
	return func() tea.Msg {
		raw, err := tmux.CapturePaneVisible(paneID)
		if err != nil {
			return previewResultMsg{paneID: paneID, err: err}
		}

		lines := strings.Split(raw, "\n")

		// Take the last maxLines lines (most recent output)
		if len(lines) > maxLines {
			lines = lines[len(lines)-maxLines:]
		}

		// Truncate each line to fit the preview width
		for i, line := range lines {
			if len(line) > maxWidth {
				lines[i] = line[:maxWidth]
			}
		}

		return previewResultMsg{
			paneID:  paneID,
			content: strings.Join(lines, "\n"),
		}
	}
}

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
// tea.Printf("\a") does not work because bubbletea silently drops
// printLineMessages while the alternate screen is active.
func bellCmd() tea.Cmd {
	return func() tea.Msg {
		os.Stderr.Write([]byte("\a"))
		return nil
	}
}
