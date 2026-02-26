package tmux

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gxespino/ctree/internal/model"
)

// Format string for list-panes. Fields separated by tab character.
const paneFormat = "#{session_name}\t#{window_index}\t#{window_id}\t#{window_name}\t#{pane_id}\t#{pane_pid}\t#{pane_current_path}\t#{window_activity}\t#{window_active}"

// ListAllPanes returns all tmux panes across all sessions.
func ListAllPanes() ([]model.Window, error) {
	out, err := exec.Command("tmux", "list-panes", "-a", "-F", paneFormat).Output()
	if err != nil {
		return nil, fmt.Errorf("tmux list-panes: %w", err)
	}

	var windows []model.Window
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		w, err := parsePaneLine(line)
		if err != nil {
			continue
		}
		windows = append(windows, w)
	}
	return windows, nil
}

func parsePaneLine(line string) (model.Window, error) {
	fields := strings.Split(line, "\t")
	if len(fields) < 9 {
		return model.Window{}, fmt.Errorf("expected 9 fields, got %d", len(fields))
	}

	windowIndex, _ := strconv.Atoi(fields[1])
	panePID, _ := strconv.Atoi(fields[5])
	activityEpoch, _ := strconv.ParseInt(fields[7], 10, 64)

	return model.Window{
		SessionName:    fields[0],
		WindowIndex:    windowIndex,
		WindowID:       fields[2],
		WindowName:     fields[3],
		PaneID:         fields[4],
		PanePID:        panePID,
		WorkingDir:     fields[6],
		LastActivity:   time.Unix(activityEpoch, 0),
		IsActiveWindow: fields[8] == "1",
	}, nil
}

// SelectWindow switches tmux focus to the specified window,
// then focuses the main (non-sidebar) pane so the user lands on Claude.
func SelectWindow(sessionName string, windowIndex int) error {
	target := fmt.Sprintf("%s:%d", sessionName, windowIndex)
	if err := exec.Command("tmux", "select-window", "-t", target).Run(); err != nil {
		return err
	}

	// Find the non-ctree pane and focus it
	out, err := exec.Command("tmux", "list-panes", "-t", target,
		"-F", "#{pane_id} #{pane_title}").Output()
	if err != nil {
		return nil // window switched, pane focus is best-effort
	}

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		paneID := parts[0]
		paneTitle := parts[1]
		if paneTitle != "ctree-sidebar" {
			_ = exec.Command("tmux", "select-pane", "-t", paneID).Run()
			return nil
		}
	}

	return nil
}

// NewClaudeWindow creates a new tmux window and starts claude in it.
func NewClaudeWindow(name, dir string) error {
	args := []string{"new-window"}
	if name != "" {
		args = append(args, "-n", name)
	}
	if dir != "" {
		args = append(args, "-c", dir)
	}
	args = append(args, "claude")
	return exec.Command("tmux", args...).Run()
}

// CapturePaneVisible captures the visible pane content, trimming trailing empty lines.
func CapturePaneVisible(paneID string) (string, error) {
	out, err := exec.Command("tmux", "capture-pane", "-t", paneID, "-p").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\n "), nil
}
