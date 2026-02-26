package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gxespino/ctree/internal/model"
)

// spinnerFrames are braille dot characters that cycle to form a spinner animation.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type windowDelegate struct {
	spinnerFrame *int
}

func newWindowDelegate(frame *int) windowDelegate {
	return windowDelegate{spinnerFrame: frame}
}

func (d windowDelegate) Height() int                             { return 4 }
func (d windowDelegate) Spacing() int                            { return 0 }
func (d windowDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d windowDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	win, ok := item.(model.Window)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	// Line 1: window number + name + status badge
	idx := windowNumStyle.Render(fmt.Sprintf("%d", win.WindowIndex))
	name := nameStyle.Render(win.Title())
	statusStyle, ok := statusStyles[win.Status]
	if !ok {
		statusStyle = statusStyles[model.StatusUnknown]
	}
	badgeText := win.Status.String()
	if win.Status == model.StatusWorking && d.spinnerFrame != nil {
		frame := spinnerFrames[*d.spinnerFrame%len(spinnerFrames)]
		badgeText = frame + " " + badgeText
	} else if win.Status == model.StatusPaused {
		badgeText = "▶ " + badgeText
	}
	badge := statusStyle.Render(badgeText)
	line1 := fmt.Sprintf("%s %s  %s", idx, name, badge)

	// Line 2: git branch + diff stats
	var line2 string
	if win.GitBranch != "" {
		line2 = branchStyle.Render(" " + win.GitBranch)
		if win.GitAdded > 0 || win.GitRemoved > 0 {
			line2 += "  " + addedStyle.Render(fmt.Sprintf("+%d", win.GitAdded)) +
				" " + removedStyle.Render(fmt.Sprintf("-%d", win.GitRemoved))
		}
	} else {
		line2 = dimmedStyle.Render(" no repo")
	}

	// Line 3: relative time
	var line3 string
	if !win.LastActivity.IsZero() {
		line3 = dimmedStyle.Render(" " + model.RelativeTime(win.LastActivity))
	}

	// Line 0: group header (if first in group) or blank spacer
	var line0 string
	isGroupHead := true
	if index > 0 {
		if prev, ok := m.Items()[index-1].(model.Window); ok {
			isGroupHead = prev.WorkingDir != win.WorkingDir
		}
	}
	if isGroupHead {
		groupName := win.Title()
		rule := strings.Repeat("─", 20)
		line0 = groupHeaderStyle.Render("── " + groupName + " " + rule)
	}

	content := line0 + "\n" + line1 + "\n" + line2 + "\n" + line3

	if isSelected {
		fmt.Fprint(w, selectedItemStyle.Render(content))
	} else if win.Status == model.StatusPaused {
		fmt.Fprint(w, needsInputItemStyle.Render(content))
	} else {
		fmt.Fprint(w, normalItemStyle.Render(content))
	}
}
