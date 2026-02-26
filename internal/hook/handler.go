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
	SessionID        string `json:"session_id"`
	NotificationType string `json:"notification_type"`
}

// Run handles the "cmux hook <event>" subcommand.
// Reads $TMUX_PANE for pane identification, reads Claude Code's JSON
// payload from stdin, and writes the status to a hook data file.
func Run(event string) error {
	paneID := os.Getenv("TMUX_PANE")
	if paneID == "" {
		return nil
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	var input hookInput
	if len(data) > 0 {
		_ = json.Unmarshal(data, &input) // best-effort
	}

	status := mapEventToStatus(event, input.NotificationType)
	if status == "" {
		return nil
	}

	// Don't let "idle" (from Stop) overwrite a recent "paused" (from
	// PermissionRequest). Both fire nearly simultaneously when Claude
	// shows a permission dialog — Stop fires because Claude "stopped
	// responding", but the session is actually waiting for user input.
	if status == "idle" {
		if existing, _ := hookdata.Read(paneID); existing != nil {
			if existing.Status == "paused" && time.Since(existing.Timestamp) < 10*time.Second {
				return nil
			}
		}
	}

	return hookdata.Write(hookdata.HookStatus{
		PaneID:    paneID,
		SessionID: input.SessionID,
		Status:    status,
		Timestamp: time.Now(),
	})
}

// mapEventToStatus converts a hook event name to a status string.
// For notification events, notificationType distinguishes between
// idle notifications and input-needed notifications (elicitation, permission).
func mapEventToStatus(event, notificationType string) string {
	switch event {
	case "prompt-submit":
		return "working"
	case "stop":
		return "idle"
	case "notification":
		// Only map elicitation_dialog to "paused" (AskUserQuestion).
		// Don't map permission_prompt here — PermissionRequest already
		// handles that, and the delayed Notification can race with Stop
		// and overwrite "idle" back to "paused".
		if notificationType == "elicitation_dialog" {
			return "paused"
		}
		return "idle"
	case "permission-request":
		return "paused"
	case "post-tool-use":
		return "working"
	case "session-end":
		return "stopped"
	default:
		return ""
	}
}
