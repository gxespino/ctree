package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gxespino/cmux/internal/hook"
	"github.com/gxespino/cmux/internal/setup"
	"github.com/gxespino/cmux/internal/state"
	"github.com/gxespino/cmux/internal/ui"
)

func main() {
	// Subcommand dispatch — these do not require tmux
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "hook":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: cmux hook <event>")
				os.Exit(1)
			}
			if err := hook.Run(os.Args[2]); err != nil {
				fmt.Fprintf(os.Stderr, "cmux hook: %v\n", err)
				os.Exit(1)
			}
			return
		case "setup":
			if setup.Check() {
				fmt.Println("cmux hooks already configured in ~/.claude/settings.json")
				return
			}
			if err := setup.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "cmux setup: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("cmux hooks configured in ~/.claude/settings.json")
			return
		}
	}

	// TUI mode — requires tmux
	if os.Getenv("TMUX") == "" {
		fmt.Fprintln(os.Stderr, "cmux: must be run inside a tmux session")
		os.Exit(1)
	}

	if _, err := exec.LookPath("tmux"); err != nil {
		fmt.Fprintln(os.Stderr, "cmux: tmux not found in PATH")
		os.Exit(1)
	}

	persistedState, err := state.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cmux: failed to load state: %v\n", err)
		os.Exit(1)
	}

	app := ui.NewApp(persistedState)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithReportFocus())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "cmux: %v\n", err)
		os.Exit(1)
	}

	if err := state.Save(persistedState); err != nil {
		fmt.Fprintf(os.Stderr, "cmux: failed to save state: %v\n", err)
	}
}
