package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gxespino/cmux/internal/model"
	"github.com/gxespino/cmux/internal/state"
)

// App is the top-level Bubble Tea model.
type App struct {
	list         list.Model
	windows      []model.Window
	prevStatuses map[string]model.Status // windowID → last known status
	width        int
	height       int
	keys         keyMap
	state        *state.PersistentState
	err          error
	focused      bool
}

// NewApp creates a new App.
func NewApp(s *state.PersistentState) App {
	delegate := newWindowDelegate()
	l := list.New([]list.Item{}, delegate, 40, 20)
	l.Title = "cmux"
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()
	l.Styles.Title = headerStyle
	l.SetStatusBarItemName("session", "sessions")

	return App{
		list:         l,
		keys:         defaultKeyMap(),
		state:        s,
		prevStatuses: make(map[string]model.Status),
		focused:      true,
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(pollTmuxCmd(), tickCmd())
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return a.handleKey(msg)

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		// Reserve space for border (2 top/bottom) + footer (2 lines)
		a.list.SetSize(msg.Width-2, msg.Height-6)
		return a, nil

	case tickMsg:
		return a, pollTmuxCmd()

	case pollResultMsg:
		return a.handlePollResult(msg)

	case gitResultMsg:
		return a.handleGitResult(msg)

	case jumpedMsg:
		a.state.MarkSeen(msg.windowID)
		_ = state.Save(a.state)
		return a, tickCmd()

	case newWorkspaceResultMsg:
		if msg.err != nil {
			a.err = msg.err
		}
		return a, pollTmuxCmd()

	case errMsg:
		a.err = msg.err
		return a, tickCmd()

	case tea.FocusMsg:
		a.focused = true
		return a, nil

	case tea.BlurMsg:
		a.focused = false
		return a, nil
	}

	var cmd tea.Cmd
	a.list, cmd = a.list.Update(msg)
	return a, cmd
}

func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If the list is filtering, let it handle all keys
	if a.list.FilterState() == list.Filtering {
		var cmd tea.Cmd
		a.list, cmd = a.list.Update(msg)
		return a, cmd
	}

	switch {
	case key.Matches(msg, a.keys.Quit):
		return a, tea.Quit

	case key.Matches(msg, a.keys.Enter):
		if item, ok := a.list.SelectedItem().(model.Window); ok {
			return a, jumpToWindowCmd(item.SessionName, item.WindowIndex)
		}

	case key.Matches(msg, a.keys.NewWorkspace):
		return a, newWorkspaceCmd()

	case key.Matches(msg, a.keys.Refresh):
		return a, pollTmuxCmd()

	case key.Matches(msg, a.keys.Escape):
		if a.list.FilterState() == list.FilterApplied {
			a.list.ResetFilter()
			return a, nil
		}
		return a, tea.Quit
	}

	// Delegate to bubbles list for j/k/arrow nav and / filtering
	var cmd tea.Cmd
	a.list, cmd = a.list.Update(msg)
	return a, cmd
}

func (a App) handlePollResult(msg pollResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		a.err = msg.err
		return a, tickCmd()
	}

	a.err = nil
	incoming := msg.windows

	// State machine for idle sessions. The detect layer returns Working or Idle.
	// We refine Idle into Unread / Done / Idle based on transitions:
	//
	//   Working → Idle  =  Unread  (just finished, user should review)
	//   Unread  + user views  =  Done
	//   Unread  + not viewed  =  stays Unread
	//   Done    → stays Done  (until Working again or 5 min timeout → Idle)
	//
	// This avoids relying on window_activity timestamps which drift.
	for i := range incoming {
		w := &incoming[i]

		// Working and Error pass through from the detect layer untouched.
		if w.Status != model.StatusIdle {
			continue
		}

		prev, hasPrev := a.prevStatuses[w.WindowID]

		switch {
		case hasPrev && prev == model.StatusWorking:
			// Just finished working → mark Unread, clear "seen" flag
			w.Status = model.StatusUnread
			delete(a.state.LastSeen, w.Target())

		case hasPrev && prev == model.StatusUnread:
			// Was Unread — did the user look at it?
			if w.IsActiveWindow {
				a.state.MarkSeen(w.Target())
				w.Status = model.StatusDone
			} else if _, seen := a.state.LastSeen[w.Target()]; seen {
				// User jumped to it since last poll
				w.Status = model.StatusDone
			} else {
				w.Status = model.StatusUnread
			}

		case hasPrev && prev == model.StatusDone:
			if w.IsActiveWindow {
				a.state.MarkSeen(w.Target())
			}
			if time.Since(w.LastActivity) < 5*time.Minute {
				w.Status = model.StatusDone
			}
			// else stays Idle (quiet for a while)

		default:
			// First poll or was already Idle — stay Idle
			if w.IsActiveWindow {
				a.state.MarkSeen(w.Target())
			}
		}
	}

	// Persist state changes (seen flags, deletions) from the state machine above.
	_ = state.Save(a.state)

	// Detect Working → Unread transitions (chime notification)
	shouldChime := false
	for _, w := range incoming {
		prev, ok := a.prevStatuses[w.WindowID]
		if ok && prev == model.StatusWorking && w.Status == model.StatusUnread {
			shouldChime = true
		}
	}

	// Update previous statuses for next poll
	for _, w := range incoming {
		a.prevStatuses[w.WindowID] = w.Status
	}

	// Preserve git data from previous poll (git results arrive async)
	for i := range incoming {
		for j := range a.windows {
			if incoming[i].WindowID == a.windows[j].WindowID {
				incoming[i].GitBranch = a.windows[j].GitBranch
				incoming[i].GitAdded = a.windows[j].GitAdded
				incoming[i].GitRemoved = a.windows[j].GitRemoved
				incoming[i].GitDirty = a.windows[j].GitDirty
				break
			}
		}
	}

	// Only update list items if something actually changed (prevents flash)
	changed := len(incoming) != len(a.windows)
	if !changed {
		for i := range incoming {
			if windowFingerprint(incoming[i]) != windowFingerprint(a.windows[i]) {
				changed = true
				break
			}
		}
	}

	a.windows = incoming

	var cmds []tea.Cmd
	if shouldChime {
		cmds = append(cmds, bellCmd())
	}
	if changed {
		items := make([]list.Item, len(a.windows))
		for i, w := range a.windows {
			items[i] = w
		}
		cmd := a.list.SetItems(items)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Fire git commands for each window
	for _, w := range a.windows {
		if w.WorkingDir != "" {
			cmds = append(cmds, pollGitCmd(w.WindowID, w.WorkingDir))
		}
	}
	cmds = append(cmds, tickCmd())

	return a, tea.Batch(cmds...)
}

// windowFingerprint creates a comparable string for change detection.
func windowFingerprint(w model.Window) string {
	return fmt.Sprintf("%s:%d:%s:%d:%s:%d:%d",
		w.SessionName, w.WindowIndex, w.Status,
		w.LastActivity.Unix(), w.GitBranch, w.GitAdded, w.GitRemoved)
}

func (a App) handleGitResult(msg gitResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		return a, nil
	}

	// Find and update the matching window, only if data changed
	changed := false
	for i := range a.windows {
		if a.windows[i].WindowID == msg.windowID {
			if a.windows[i].GitBranch != msg.branch ||
				a.windows[i].GitAdded != msg.added ||
				a.windows[i].GitRemoved != msg.removed {
				a.windows[i].GitBranch = msg.branch
				a.windows[i].GitAdded = msg.added
				a.windows[i].GitRemoved = msg.removed
				a.windows[i].GitDirty = msg.dirty
				changed = true
			}
			break
		}
	}

	if !changed {
		return a, nil
	}

	items := make([]list.Item, len(a.windows))
	for i, w := range a.windows {
		items[i] = w
	}
	cmd := a.list.SetItems(items)
	return a, cmd
}

func (a App) View() string {
	var b strings.Builder
	b.WriteString(a.list.View())
	b.WriteString("\n")

	if a.err != nil {
		b.WriteString(statusStyles[model.StatusError].Render("  Error: " + a.err.Error()))
		b.WriteString("\n")
	}

	b.WriteString(footerStyle.Render("j/k nav  enter jump  n new  q quit"))

	content := b.String()

	if a.focused {
		return borderStyle.Width(a.width - 2).Height(a.height - 2).Render(content)
	}
	return borderDimStyle.Width(a.width - 2).Height(a.height - 2).Render(content)
}
