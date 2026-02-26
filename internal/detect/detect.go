package detect

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/gxespino/ctree/internal/hookdata"
	"github.com/gxespino/ctree/internal/model"
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
// Uses hook-based status files written by Claude Code lifecycle events.
// Process liveness is always verified via the process table.
func EnrichAll(windows []model.Window) {
	procs, children := buildProcessTable()
	if procs == nil {
		return
	}

	hookStatuses := hookdata.ReadAll()

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

		if _, alive := procs[claudePID]; !alive {
			w.Status = model.StatusExited
			continue
		}

		if hs, ok := hookStatuses[w.PaneID]; ok {
			w.Status = mapHookStatus(hs.Status)
		} else {
			// No hook file yet — session predates hook setup or
			// hasn't had any events. Default to Idle until a hook fires.
			w.Status = model.StatusIdle
		}
	}
}

// mapHookStatus converts a hook status string to a model.Status.
func mapHookStatus(hookStatus string) model.Status {
	switch hookStatus {
	case "working":
		return model.StatusWorking
	case "paused":
		return model.StatusPaused
	case "idle":
		return model.StatusIdle
	case "stopped":
		return model.StatusExited
	default:
		return model.StatusUnknown
	}
}
