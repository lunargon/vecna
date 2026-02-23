package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var vecnaArt = []string{
	`██╗   ██╗███████╗ ██████╗███╗   ██╗ █████╗ `,
	`██║   ██║██╔════╝██╔════╝████╗  ██║██╔══██╗`,
	`██║   ██║█████╗  ██║     ██╔██╗ ██║███████║`,
	`╚██╗ ██╔╝██╔══╝  ██║     ██║╚██╗██║██╔══██║`,
	` ╚████╔╝ ███████╗╚██████╗██║ ╚████║██║  ██║`,
	`  ╚═══╝  ╚══════╝ ╚═════╝╚═╝  ╚═══╝╚═╝  ╚═╝`,
}

var loaderSpinner = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}

var loaderArtColors = []lipgloss.Style{
	lipgloss.NewStyle().Foreground(colorPurple),
	lipgloss.NewStyle().Foreground(colorHighlight),
	lipgloss.NewStyle().Foreground(colorCyan),
}

func renderLoader(width, height, frame int, message string) string {
	if width < 10 {
		width = 40
	}
	if height < 8 {
		height = 12
	}

	artStyle := loaderArtColors[(frame/3)%len(loaderArtColors)]

	var artLines []string
	revealCount := frame * 4
	for _, line := range vecnaArt {
		runes := []rune(line)
		if revealCount < len(runes) {
			artLines = append(artLines, artStyle.Render(string(runes[:revealCount])))
		} else {
			artLines = append(artLines, artStyle.Render(line))
		}
	}

	contentWidth := width - 6
	if contentWidth < 10 {
		contentWidth = width
	}

	centeredArt := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Center).
		Render(strings.Join(artLines, "\n"))

	spinner := loaderSpinner[frame%len(loaderSpinner)]
	spinnerLine := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Center).
		Render(stylePurple.Render(spinner) + styleDim.Render("  "+message))

	dots := ""
	for i := 0; i < frame%4; i++ {
		dots += " ◈"
	}
	dotsLine := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Center).
		Render(stylePurple.Render(dots))

	padTop := (height - len(vecnaArt) - 4) / 2
	if padTop < 1 {
		padTop = 1
	}

	var lines []string
	for i := 0; i < padTop; i++ {
		lines = append(lines, "")
	}
	lines = append(lines, centeredArt, "", spinnerLine, dotsLine)

	return strings.Join(lines, "\n")
}

func renderLoaderFullscreen(width, height, frame, panelHeight int, message string) string {
	content := renderLoader(width, height, frame, message)

	terminal := stylePanelActive.
		Width(width - 4).
		Height(panelHeight).
		Render(content)

	return fmt.Sprintf("\n%s", terminal)
}
