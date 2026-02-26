package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gxespino/ctree/internal/hookdata"
	"github.com/gxespino/ctree/internal/slack"
	"github.com/gxespino/ctree/internal/state"
)

// hookInput is the subset of Claude Code's hook JSON payload we parse.
type hookInput struct {
	SessionID        string         `json:"session_id"`
	NotificationType string         `json:"notification_type"`
	ToolName         string         `json:"tool_name"`
	ToolInput        map[string]any `json:"tool_input"`
	CWD              string         `json:"cwd"`
}

// Run handles the "ctree hook <event>" subcommand.
// Reads $TMUX_PANE for pane identification, reads Claude Code's JSON
// payload from stdin, and writes the status to a hook data file.
// For permission-request events, optionally forwards to Slack for remote approval.
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

	// Slack: handle permission requests (bidirectional) and notifications (one-way).
	// Only fires when the user has toggled Slack on via the TUI (s key).
	if state.GetSlack() {
		if event == "permission-request" {
			if decision := handlePermissionRequest(input); decision != "" {
				writeDecision(decision)
			}
		}
		if event == "notification" && input.NotificationType == "elicitation_dialog" {
			handleNotification(input)
		}
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

// handlePermissionRequest sends a Slack message and waits for a threaded reply.
// Returns "allow", "deny", "ask", or "" (fall through to terminal).
func handlePermissionRequest(input hookInput) string {
	cfg, err := slack.LoadConfig()
	if cfg == nil || err != nil {
		return ""
	}

	text := formatPermissionMessage(input)
	threadTS, err := slack.SendMessage(cfg, text)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ctree: slack send failed: %v\n", err)
		return ""
	}

	reply, err := slack.WaitForReply(cfg, threadTS, 5*time.Minute)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ctree: slack poll failed: %v\n", err)
		return ""
	}
	if reply == "" {
		return "" // timeout, fall through to terminal
	}

	return parseDecision(reply)
}

// handleNotification sends a one-way Slack message when Claude needs input.
func handleNotification(input hookInput) {
	cfg, err := slack.LoadConfig()
	if cfg == nil || err != nil {
		return
	}
	text := ":bell: *Claude needs input*\nCheck your terminal — Claude is asking a question."
	_, _ = slack.SendMessage(cfg, text)
}

// writeDecision outputs a permission decision as JSON to stdout for Claude Code.
// PermissionRequest hooks use decision.behavior ("allow"/"deny"), not permissionDecision.
func writeDecision(decision string) {
	output := map[string]any{
		"behavior": decision,
	}
	if decision == "deny" {
		output["message"] = "Denied via Slack"
	}
	resp := map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName": "PermissionRequest",
			"decision":      output,
		},
	}
	_ = json.NewEncoder(os.Stdout).Encode(resp)
}

func formatPermissionMessage(input hookInput) string {
	var b strings.Builder
	b.WriteString(":lock: *Permission Request*\n")

	if input.ToolName != "" {
		b.WriteString(fmt.Sprintf("*Tool:* `%s`\n", input.ToolName))
	}
	if input.CWD != "" {
		b.WriteString(fmt.Sprintf("*Dir:* `%s`\n", input.CWD))
	}

	if input.ToolInput != nil {
		detail := formatToolInput(input.ToolInput)
		if detail != "" {
			b.WriteString("```\n")
			b.WriteString(detail)
			b.WriteString("\n```\n")
		}
	}

	b.WriteString("Reply in thread: *yes* or *no*")
	return b.String()
}

// formatToolInput extracts a readable summary from the tool input.
// For Bash, shows the command. For Edit/Write, shows the file path.
// Falls back to indented JSON, truncated to 500 chars.
func formatToolInput(input map[string]any) string {
	if cmd, ok := input["command"].(string); ok {
		return cmd
	}
	if fp, ok := input["file_path"].(string); ok {
		return fp
	}

	data, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		return ""
	}
	s := string(data)
	if len(s) > 500 {
		s = s[:500] + "\n..."
	}
	return s
}

// parseDecision normalizes a Slack reply to a Claude permission decision.
// PermissionRequest hooks only support "allow" or "deny".
func parseDecision(reply string) string {
	switch strings.ToLower(strings.TrimSpace(reply)) {
	case "allow", "yes", "y", "approve", "ok":
		return "allow"
	default:
		return "deny"
	}
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
