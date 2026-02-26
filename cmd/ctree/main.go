package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gxespino/ctree/internal/hook"
	"github.com/gxespino/ctree/internal/setup"
	"github.com/gxespino/ctree/internal/slack"
	"github.com/gxespino/ctree/internal/state"
	"github.com/gxespino/ctree/internal/ui"
)

func main() {
	// Subcommand dispatch — these do not require tmux
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "hook":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: ctree hook <event>")
				os.Exit(1)
			}
			if err := hook.Run(os.Args[2]); err != nil {
				fmt.Fprintf(os.Stderr, "ctree hook: %v\n", err)
				os.Exit(1)
			}
			return
		case "setup":
			force := len(os.Args) > 2 && os.Args[2] == "--force"
			if !force && setup.Check() {
				fmt.Println("ctree hooks already configured in ~/.claude/settings.json")
				return
			}
			if err := setup.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "ctree setup: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("ctree hooks configured in ~/.claude/settings.json")
			return
		case "slack-setup":
			if err := runSlackSetup(); err != nil {
				fmt.Fprintf(os.Stderr, "ctree slack-setup: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	// TUI mode — requires tmux
	if os.Getenv("TMUX") == "" {
		fmt.Fprintln(os.Stderr, "ctree: must be run inside a tmux session")
		os.Exit(1)
	}

	if _, err := exec.LookPath("tmux"); err != nil {
		fmt.Fprintln(os.Stderr, "ctree: tmux not found in PATH")
		os.Exit(1)
	}

	persistedState, err := state.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ctree: failed to load state: %v\n", err)
		os.Exit(1)
	}

	app := ui.NewApp(persistedState)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithReportFocus())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ctree: %v\n", err)
		os.Exit(1)
	}

	if err := state.Save(persistedState); err != nil {
		fmt.Fprintf(os.Stderr, "ctree: failed to save state: %v\n", err)
	}
}

func runSlackSetup() error {
	fmt.Println("ctree Slack Integration Setup")
	fmt.Println("─────────────────────────────")
	fmt.Println()
	fmt.Println("Create a Slack App at https://api.slack.com/apps")
	fmt.Println("Required bot token scopes: chat:write, channels:history")
	fmt.Println("Install the app to your workspace, then invite the bot to your channel.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Bot Token (xoxb-...): ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	fmt.Print("Channel ID (e.g. C0123456789): ")
	channelID, _ := reader.ReadString('\n')
	channelID = strings.TrimSpace(channelID)

	if token == "" || channelID == "" {
		return fmt.Errorf("both bot token and channel ID are required")
	}

	cfg := slack.Config{BotToken: token, ChannelID: channelID}

	fmt.Print("\nTesting connection... ")
	if _, err := slack.SendMessage(&cfg, "ctree connected! Permission requests will appear here."); err != nil {
		return fmt.Errorf("slack test failed: %w", err)
	}
	fmt.Println("OK")

	if err := slack.SaveConfig(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Println("\nSaved to ~/.config/ctree/slack.json")
	fmt.Println("Permission requests will now be forwarded to Slack.")
	return nil
}
