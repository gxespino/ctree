package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gxespino/cmux/internal/hookdata"
)

// hookInput is the subset of Claude Code's hook JSON payload we parse.
type hookInput struct {
	SessionID string `json:"session_id"`
}

// Run handles the "cmux hook <event>" subcommand.
// Reads $TMUX_PANE for pane identification, reads Claude Code's JSON
// payload from stdin, and writes the status to a hook data file.
func Run(event string) error {
	paneID := os.Getenv("TMUX_PANE")
	if paneID == "" {
		// Not in a tmux pane — nothing to do
		return nil
	}

	// Read JSON from stdin (Claude Code sends hook payload)
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	var input hookInput
	if len(data) > 0 {
		_ = json.Unmarshal(data, &input) // best-effort
	}

	status := mapEventToStatus(event)
	if status == "" {
		return nil // unknown event, silently ignore
	}

	return hookdata.Write(hookdata.HookStatus{
		PaneID:    paneID,
		SessionID: input.SessionID,
		Status:    status,
		Timestamp: time.Now(),
	})
}

// mapEventToStatus converts a hook event name to a status string.
func mapEventToStatus(event string) string {
	switch event {
	case "prompt-submit":
		return "working"
	case "stop":
		return "idle"
	case "notification":
		return "idle"
	case "session-end":
		return "stopped"
	default:
		return ""
	}
}
