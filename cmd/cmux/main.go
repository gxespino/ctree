package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gxespino/cmux/internal/state"
	"github.com/gxespino/cmux/internal/ui"
)

func main() {
	// Verify we're inside tmux
	if os.Getenv("TMUX") == "" {
		fmt.Fprintln(os.Stderr, "cmux: must be run inside a tmux session")
		os.Exit(1)
	}

	// Verify tmux is available
	if _, err := exec.LookPath("tmux"); err != nil {
		fmt.Fprintln(os.Stderr, "cmux: tmux not found in PATH")
		os.Exit(1)
	}

	// Load persistent state
	persistedState, err := state.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cmux: failed to load state: %v\n", err)
		os.Exit(1)
	}

	// Create and run the Bubble Tea program
	app := ui.NewApp(persistedState)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithReportFocus())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "cmux: %v\n", err)
		os.Exit(1)
	}

	// Save state on clean exit
	if err := state.Save(persistedState); err != nil {
		fmt.Fprintf(os.Stderr, "cmux: failed to save state: %v\n", err)
	}
}
