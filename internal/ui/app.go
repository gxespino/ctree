package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gxespino/ctree/internal/detect"
	"github.com/gxespino/ctree/internal/model"
	"github.com/gxespino/ctree/internal/state"
	"github.com/gxespino/ctree/internal/tmux"
)


// doneTimeout is how long Done persists before decaying to Idle.
// Short enough to not get stuck, long enough to be visible.
const doneTimeout = 15 * time.Second

// App is the top-level Bubble Tea model.
type App struct {
	list         list.Model
	windows      []model.Window
	prevStatuses map[string]model.Status // windowID → last known status
	doneAt       map[string]time.Time    // windowID → when session entered Done
	width        int
	height       int
	keys         keyMap
	state        *state.PersistentState
	err          error
	focused      bool

	showPreview    bool
	previewContent string
	previewPaneID  string

	spinnerFrame *int
}

// NewApp creates a new App.
func NewApp(s *state.PersistentState) App {
	frame := new(int)
	delegate := newWindowDelegate(frame)
	l := list.New([]list.Item{}, delegate, 40, 20)
	l.Title = "CTree"
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()
	l.Styles.Title = headerStyle
	l.SetStatusBarItemName("session", "sessions")

	// Bootstrap: detect current statuses so reopening the sidebar
	// doesn't reset everything to Idle.
	prev := make(map[string]model.Status)
	if panes, err := tmux.ListAllPanes(); err == nil {
		detect.EnrichAll(panes)
		for _, w := range panes {
			if w.IsClaudePane {
				prev[w.WindowID] = w.Status
			}
		}
	}

	return App{
		list:         l,
		keys:         defaultKeyMap(),
		state:        s,
		prevStatuses: prev,
		doneAt:       make(map[string]time.Time),
		focused:      true,
		showPreview:  state.GetPreview(),
		spinnerFrame: frame,
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
		a.updateListSize()
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

	case previewResultMsg:
		if msg.err != nil || msg.paneID != a.previewPaneID {
			return a, nil
		}
		a.previewContent = msg.content
		return a, nil

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

	case key.Matches(msg, a.keys.JumpUnread):
		// Find the most recent session needing attention and jump to it
		for _, w := range a.windows {
			if w.Status == model.StatusPaused || w.Status == model.StatusUnread || w.Status == model.StatusDone {
				return a, jumpToWindowCmd(w.SessionName, w.WindowIndex)
			}
		}
		return a, nil

	case key.Matches(msg, a.keys.NewWorkspace):
		return a, newWorkspaceCmd()

	case key.Matches(msg, a.keys.Preview):
		a.showPreview = !a.showPreview
		state.SetPreview(a.showPreview)
		a.updateListSize()
		if a.showPreview {
			if item, ok := a.list.SelectedItem().(model.Window); ok {
				a.previewPaneID = item.PaneID
				return a, capturePreviewCmd(item.PaneID, a.previewHeight(), a.width-4)
			}
		}
		return a, nil

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
	prevIndex := a.list.Index()
	var cmd tea.Cmd
	a.list, cmd = a.list.Update(msg)

	// If selection changed while preview is open, fetch new preview
	if a.showPreview && a.list.Index() != prevIndex {
		if item, ok := a.list.SelectedItem().(model.Window); ok {
			a.previewPaneID = item.PaneID
			a.previewContent = ""
			return a, tea.Batch(cmd, capturePreviewCmd(item.PaneID, a.previewHeight(), a.width-4))
		}
	}

	return a, cmd
}

func (a App) handlePollResult(msg pollResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		a.err = msg.err
		return a, tickCmd()
	}

	a.err = nil
	*a.spinnerFrame++

	// Sync preview toggle from disk so all ctree instances stay in sync
	if diskPreview := state.GetPreview(); diskPreview != a.showPreview {
		a.showPreview = diskPreview
		a.previewContent = ""
		a.updateListSize()
	}

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

		// Non-idle statuses pass through untouched.
		// Paused = waiting for user input (permission, question).
		if w.Status != model.StatusIdle {
			delete(a.doneAt, w.WindowID)
			continue
		}

		prev, hasPrev := a.prevStatuses[w.WindowID]

		switch {
		case hasPrev && (prev == model.StatusWorking || prev == model.StatusPaused):
			// Just finished working/paused → mark Unread, clear "seen" flag
			w.Status = model.StatusUnread
			delete(a.state.LastSeen, w.Target())

		case hasPrev && prev == model.StatusUnread:
			// Was Unread — did the user look at it?
			if w.IsActiveWindow {
				a.state.MarkSeen(w.Target())
				w.Status = model.StatusDone
				a.doneAt[w.WindowID] = time.Now()
			} else if _, seen := a.state.LastSeen[w.Target()]; seen {
				// User jumped to it since last poll
				w.Status = model.StatusDone
				a.doneAt[w.WindowID] = time.Now()
			} else {
				w.Status = model.StatusUnread
			}

		case hasPrev && prev == model.StatusDone:
			if w.IsActiveWindow {
				a.state.MarkSeen(w.Target())
			}
			if t, ok := a.doneAt[w.WindowID]; ok && time.Since(t) < doneTimeout {
				w.Status = model.StatusDone
			} else {
				delete(a.doneAt, w.WindowID)
			}
			// else decays to Idle

		default:
			// First poll or was already Idle — stay Idle
			if w.IsActiveWindow {
				a.state.MarkSeen(w.Target())
			}
		}
	}

	// Persist state changes (seen flags, deletions) from the state machine above.
	_ = state.Save(a.state)

	// Detect transitions that need user attention (chime notification)
	shouldChime := false
	for _, w := range incoming {
		prev, ok := a.prevStatuses[w.WindowID]
		if !ok {
			continue
		}
		// Working/Paused → Unread (just finished)
		if (prev == model.StatusWorking || prev == model.StatusPaused) && w.Status == model.StatusUnread {
			shouldChime = true
		}
		// Working → Paused (needs input mid-task)
		if prev == model.StatusWorking && w.Status == model.StatusPaused {
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

	// Group sessions by project directory (stable sort preserves tmux order within groups)
	sort.SliceStable(incoming, func(i, j int) bool {
		return incoming[i].WorkingDir < incoming[j].WorkingDir
	})

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

	// Refresh preview if open
	if a.showPreview {
		if item, ok := a.list.SelectedItem().(model.Window); ok {
			a.previewPaneID = item.PaneID
			cmds = append(cmds, capturePreviewCmd(item.PaneID, a.previewHeight(), a.width-4))
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

// previewHeight returns how many lines the preview panel content area gets.
func (a App) previewHeight() int {
	// border(2) + footer(2) + preview header(1) = 5 lines overhead
	available := a.height - 7
	if available < 4 {
		return 4
	}
	return available / 2
}

// updateListSize recalculates the list dimensions based on whether preview is shown.
func (a *App) updateListSize() {
	if a.showPreview {
		// List gets top half: total - border(2) - footer(2) - previewHeader(1) - previewContent
		listHeight := a.height - 6 - a.previewHeight() - 1
		if listHeight < 4 {
			listHeight = 4
		}
		a.list.SetSize(a.width-2, listHeight)
	} else {
		a.list.SetSize(a.width-2, a.height-6)
	}
}

func (a App) View() string {
	var b strings.Builder
	b.WriteString(a.list.View())
	b.WriteString("\n")

	if a.err != nil {
		b.WriteString(statusStyles[model.StatusError].Render("  Error: " + a.err.Error()))
		b.WriteString("\n")
	}

	if a.showPreview {
		// Divider line with label
		divider := previewHeaderStyle.Render("── Preview ")
		pad := a.width - 4 - len("── Preview ")
		if pad > 0 {
			divider += previewHeaderStyle.Render(strings.Repeat("─", pad))
		}
		b.WriteString(divider)
		b.WriteString("\n")

		// Preview content, truncated to fit
		lines := strings.Split(a.previewContent, "\n")
		maxLines := a.previewHeight()
		if len(lines) > maxLines {
			lines = lines[len(lines)-maxLines:]
		}
		for _, line := range lines {
			b.WriteString(previewContentStyle.Render(line))
			b.WriteString("\n")
		}

		b.WriteString(footerStyle.Render("j/k nav  tab unread  enter jump  p close  q quit"))
	} else {
		b.WriteString(footerStyle.Render("j/k nav  tab unread  enter jump  p preview  q quit"))
	}

	content := b.String()

	if a.focused {
		return borderStyle.Width(a.width - 2).Height(a.height - 2).Render(content)
	}
	return borderDimStyle.Width(a.width - 2).Height(a.height - 2).Render(content)
}
