package detect

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/gxespino/cmux/internal/model"
	"github.com/gxespino/cmux/internal/tmux"
)

// processInfo holds parsed ps output for one process.
type processInfo struct {
	pid  int
	ppid int
	comm string
}

// buildProcessTable runs `ps` once and returns a map of PID→processInfo
// and a map of PPID→[]child PIDs.
func buildProcessTable() (map[int]processInfo, map[int][]int) {
	out, err := exec.Command("ps", "-eo", "pid,ppid,comm").Output()
	if err != nil {
		return nil, nil
	}

	procs := make(map[int]processInfo)
	children := make(map[int][]int)

	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pid, err1 := strconv.Atoi(fields[0])
		ppid, err2 := strconv.Atoi(fields[1])
		if err1 != nil || err2 != nil {
			continue
		}
		// comm may contain path, take the basename
		comm := fields[2]
		if idx := strings.LastIndex(comm, "/"); idx >= 0 {
			comm = comm[idx+1:]
		}
		procs[pid] = processInfo{pid: pid, ppid: ppid, comm: comm}
		children[ppid] = append(children[ppid], pid)
	}

	return procs, children
}

// findClaudeDescendant walks the process tree from parentPID looking for
// any descendant named "claude". Returns the PID or 0 if not found.
func findClaudeDescendant(parentPID int, procs map[int]processInfo, children map[int][]int) int {
	for _, childPID := range children[parentPID] {
		info := procs[childPID]
		if info.comm == "claude" {
			return childPID
		}
		// Recurse one more level (shell → node → claude)
		found := findClaudeDescendant(childPID, procs, children)
		if found > 0 {
			return found
		}
	}
	return 0
}

// EnrichAll detects Claude status for all windows in a single pass.
// This is much more efficient than calling pgrep per-window.
func EnrichAll(windows []model.Window) {
	procs, children := buildProcessTable()
	if procs == nil {
		return
	}

	for i := range windows {
		w := &windows[i]
		claudePID := findClaudeDescendant(w.PanePID, procs, children)
		if claudePID == 0 {
			w.IsClaudePane = false
			w.Status = model.StatusExited
			continue
		}

		w.IsClaudePane = true
		w.ClaudePID = claudePID
		w.Status = detectStatusFromPane(w.PaneID, claudePID, procs)
	}
}

// detectStatusFromPane reads captured pane output to determine Claude's status.
//
// Claude Code's status bar sits at the very bottom of the visible pane.
// We capture the full visible pane, strip trailing empty lines, then check
// the last few non-empty lines for status indicators:
//
//	Working:  "... (running) · esc to interrupt"
//	Idle:     "... esc to interrupt"  (no "(running)")
//
// IMPORTANT: Only check the status bar region (bottom lines) for "(running)"
// because the content area above may contain "(running)" from previous tool output.
func detectStatusFromPane(paneID string, claudePID int, procs map[int]processInfo) model.Status {
	if _, alive := procs[claudePID]; !alive {
		return model.StatusExited
	}

	captured, err := tmux.CapturePaneVisible(paneID)
	if err != nil {
		return model.StatusUnknown
	}

	lines := strings.Split(captured, "\n")

	// Strip trailing empty/whitespace-only lines to find the true bottom
	lastNonEmpty := len(lines) - 1
	for lastNonEmpty >= 0 && strings.TrimSpace(lines[lastNonEmpty]) == "" {
		lastNonEmpty--
	}
	if lastNonEmpty < 0 {
		return model.StatusUnknown
	}

	// Scan the last 5 non-empty lines from the bottom for the status bar.
	// The status bar always contains "esc to interrupt" — that's the definitive marker.
	// We avoid matching on "tokens" alone because Claude's output content can also
	// contain that word, leading to false Working detection.
	statusBarText := ""
	scanned := 0
	for i := lastNonEmpty; i >= 0 && scanned < 5; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		scanned++
		if strings.Contains(line, "esc to interrupt") {
			statusBarText += " " + line
		}
	}

	// Priority 1: Status bar contains "(running)" = actively working
	if strings.Contains(statusBarText, "(running)") {
		return model.StatusWorking
	}

	// Priority 2: Status bar has "esc to interrupt" without "(running)" = idle
	if strings.Contains(statusBarText, "esc to interrupt") {
		return model.StatusIdle
	}

	// Priority 3: Check last 8 non-empty lines for prompt or error
	scanned = 0
	for i := lastNonEmpty; i >= 0 && scanned < 8; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		scanned++
		if strings.Contains(line, "APIError") {
			return model.StatusError
		}
		if strings.HasPrefix(line, "❯") {
			return model.StatusIdle
		}
	}

	// Default: alive but unclear - assume idle
	return model.StatusIdle
}
