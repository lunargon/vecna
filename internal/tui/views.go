package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewHome() string {
	title := Title.Render("VECNA")
	subtitle := Subtle.Render("SSH Manager")
	header := lipgloss.JoinVertical(lipgloss.Center, title, subtitle)

	var hostList string
	if len(m.hosts) == 0 {
		hostList = Subtle.Render("No hosts configured. Press 'a' to add one.")
	} else {
		var lines []string
		for i, h := range m.hosts {
			line := fmt.Sprintf("%s (%s@%s:%d)", h.Name, h.User, h.Hostname, h.Port)
			if i == m.cursor {
				line = Active.Render("> " + line)
			} else {
				line = "  " + line
			}
			lines = append(lines, line)
		}
		hostList = strings.Join(lines, "\n")
	}

	content := Box.Render(hostList)
	status := StatusBar.Render("q: quit • a: add • d: delete • c: connect • ↑↓: navigate")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		"",
		header,
		"",
		content,
		"",
		status,
	)
}

func (m Model) viewAddHost() string {
	title := Title.Render("Add Host")

	var fields []string
	for _, input := range m.inputs {
		fields = append(fields, input.View())
	}

	form := Box.Render(strings.Join(fields, "\n"))
	status := StatusBar.Render("enter: next/save • esc: cancel • ↑↓: navigate fields")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		"",
		title,
		"",
		form,
		"",
		status,
	)
}
