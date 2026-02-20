package tui

import "github.com/charmbracelet/lipgloss"

var (
	subtle    = lipgloss.AdaptiveColor{Light: "#666", Dark: "#888"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(highlight)

	Subtle = lipgloss.NewStyle().
		Foreground(subtle)

	Active = lipgloss.NewStyle().
		Foreground(special).
		Bold(true)

	Box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(subtle).
		Padding(0, 1)

	StatusBar = lipgloss.NewStyle().
			Foreground(subtle).
			Padding(0, 1)
)
