package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gxespino/ctree/internal/model"
)

type windowDelegate struct{}

func newWindowDelegate() windowDelegate {
	return windowDelegate{}
}

func (d windowDelegate) Height() int                             { return 3 }
func (d windowDelegate) Spacing() int                            { return 1 }
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
	badge := statusStyle.Render(win.Status.String())
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

	content := line1 + "\n" + line2 + "\n" + line3

	if isSelected {
		fmt.Fprint(w, selectedItemStyle.Render(content))
	} else {
		fmt.Fprint(w, normalItemStyle.Render(content))
	}
}
