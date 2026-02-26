package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ctreeHookPrefix identifies our hooks in the settings file.
const ctreeHookPrefix = "ctree hook"

// hookEvents defines the Claude Code hooks we inject.
// Each entry: event name → hook command argument.
var hookEvents = map[string]string{
	"UserPromptSubmit":  "prompt-submit",
	"Stop":              "stop",
	"Notification":      "notification",
	"PermissionRequest": "permission-request",
	"PostToolUse":       "post-tool-use",
	"SessionEnd":        "session-end",
}

// Run configures Claude Code hooks in ~/.claude/settings.json.
// Merges ctree hooks with existing hooks, preserving user-defined hooks.
func Run() error {
	path := settingsPath()

	binPath, err := resolveBinaryPath()
	if err != nil {
		return err
	}

	settings, err := readSettings(path)
	if err != nil {
		return err
	}

	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = make(map[string]any)
	}

	for event, arg := range hookEvents {
		command := fmt.Sprintf("%s hook %s", binPath, arg)
		hooks[event] = mergeEventHooks(hooks[event], command)
	}

	settings["hooks"] = hooks

	return writeSettings(path, settings)
}

// resolveBinaryPath returns the absolute path to the running ctree binary.
func resolveBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolving ctree binary path: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolving ctree binary path: %w", err)
	}
	return resolved, nil
}

// Check returns true if ctree hooks are already configured.
func Check() bool {
	settings, err := readSettings(settingsPath())
	if err != nil {
		return false
	}

	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		return false
	}

	for event := range hookEvents {
		if !eventHasCtreeHook(hooks[event]) {
			return false
		}
	}
	return true
}

func settingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "settings.json")
}

func readSettings(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]any), nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return settings, nil
}

func writeSettings(path string, settings map[string]any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// mergeEventHooks adds a ctree hook command to an event's matcher groups,
// replacing any existing ctree hook for that event.
func mergeEventHooks(existing any, command string) []any {
	groups, _ := existing.([]any)

	// Remove any existing ctree matcher groups
	var kept []any
	for _, g := range groups {
		if !matcherGroupHasCtree(g) {
			kept = append(kept, g)
		}
	}

	// Add our matcher group
	ctreeGroup := map[string]any{
		"hooks": []any{
			map[string]any{
				"type":    "command",
				"command": command,
				"timeout": 5,
			},
		},
	}

	return append(kept, ctreeGroup)
}

// matcherGroupHasCtree checks if a matcher group contains a ctree hook.
func matcherGroupHasCtree(group any) bool {
	g, _ := group.(map[string]any)
	if g == nil {
		return false
	}
	hooks, _ := g["hooks"].([]any)
	for _, h := range hooks {
		hm, _ := h.(map[string]any)
		if hm == nil {
			continue
		}
		cmd, _ := hm["command"].(string)
		if strings.Contains(cmd, ctreeHookPrefix) {
			return true
		}
	}
	return false
}

// eventHasCtreeHook checks if an event already has a ctree hook configured.
func eventHasCtreeHook(eventVal any) bool {
	groups, _ := eventVal.([]any)
	for _, g := range groups {
		if matcherGroupHasCtree(g) {
			return true
		}
	}
	return false
}
