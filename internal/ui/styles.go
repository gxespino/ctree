package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/gxespino/cmux/internal/model"
)

var (
	colorPurple     = lipgloss.Color("#7C3AED")
	colorMagenta    = lipgloss.Color("#FF00FF")
	colorGreen      = lipgloss.Color("#10B981")
	colorYellow     = lipgloss.Color("#F59E0B")
	colorRed        = lipgloss.Color("#EF4444")
	colorBlue       = lipgloss.Color("#3B82F6")
	colorGray   = lipgloss.Color("#6B7280")
	colorDimmed = lipgloss.Color("#4B5563")
	colorWhite  = lipgloss.Color("#F9FAFB")
	colorAddGreen   = lipgloss.Color("#34D399")
	colorRemoveRed  = lipgloss.Color("#F87171")

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPurple).
			PaddingLeft(1).
			PaddingBottom(1)

	footerStyle = lipgloss.NewStyle().
			Foreground(colorGray).
			PaddingLeft(1).
			PaddingTop(1)

	normalItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				BorderLeft(true).
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(colorPurple)

	windowNumStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorMagenta)

	nameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite)

	dimmedStyle = lipgloss.NewStyle().
			Foreground(colorDimmed)

	branchStyle = lipgloss.NewStyle().
			Foreground(colorGray)

	addedStyle = lipgloss.NewStyle().
			Foreground(colorAddGreen)

	removedStyle = lipgloss.NewStyle().
			Foreground(colorRemoveRed)

	statusStyles = map[model.Status]lipgloss.Style{
		model.StatusWorking: lipgloss.NewStyle().Foreground(colorYellow).Bold(true),
		model.StatusIdle:    lipgloss.NewStyle().Foreground(colorGray),
		model.StatusUnread:  lipgloss.NewStyle().Foreground(colorBlue).Bold(true),
		model.StatusDone:    lipgloss.NewStyle().Foreground(colorGreen),
		model.StatusError:   lipgloss.NewStyle().Foreground(colorRed).Bold(true),
		model.StatusExited:  lipgloss.NewStyle().Foreground(colorDimmed),
		model.StatusUnknown: lipgloss.NewStyle().Foreground(colorDimmed),
	}

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPurple)

	borderDimStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorDimmed)

	previewHeaderStyle = lipgloss.NewStyle().
				Foreground(colorGray).
				Bold(true)

	previewContentStyle = lipgloss.NewStyle().
				Foreground(colorDimmed).
				PaddingLeft(1)
)
