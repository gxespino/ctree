package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gxespino/ctree/internal/hook"
	"github.com/gxespino/ctree/internal/setup"
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
			if setup.Check() {
				fmt.Println("ctree hooks already configured in ~/.claude/settings.json")
				return
			}
			if err := setup.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "ctree setup: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("ctree hooks configured in ~/.claude/settings.json")
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
