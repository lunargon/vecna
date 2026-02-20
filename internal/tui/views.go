package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/shravan20/vecna/internal/config"
)

func (m Model) viewHome() string {
	hosts := config.GetHosts()

	title := Title.Render("VECNA")
	subtitle := Subtle.Render("SSH Manager")

	header := lipgloss.JoinVertical(lipgloss.Center, title, subtitle)

	var hostList string
	if len(hosts) == 0 {
		hostList = Subtle.Render("No hosts configured. Press 'a' to add one.")
	} else {
		for _, h := range hosts {
			hostList += fmt.Sprintf("  %s (%s@%s:%d)\n", h.Name, h.User, h.Hostname, h.Port)
		}
	}

	content := Box.Render(hostList)

	status := StatusBar.Render("q: quit • a: add host • ?: help")

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
