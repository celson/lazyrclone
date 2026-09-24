package components

import (
	"fmt"

	"github.com/celson/lazyrclone/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

type PanelID int

const (
	PanelProfiles PanelID = 0 // [1] Profiles & Remotes
	PanelRuns     PanelID = 1 // [2] Transfers & Runs
	PanelMain     PanelID = 2 // [3] Main View (Diff / Explorer / Logs)
)

var PanelNames = []string{"1: Profiles", "2: Runs", "3: Main View"}

func RenderHeader(width int, focusedPanel PanelID, rcloneVer string, isMock bool) string {
	logo := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.MochaCrust).
		Background(styles.ColorPrimary).
		Padding(0, 1).
		Render("⚡ lazyrclone")

	var panels []string
	for i, name := range PanelNames {
		if PanelID(i) == focusedPanel {
			panels = append(panels, styles.TabActiveStyle.Render(name))
		} else {
			panels = append(panels, styles.TabInactiveStyle.Render(name))
		}
	}
	panelsRow := lipgloss.JoinHorizontal(lipgloss.Top, panels...)

	statusText := fmt.Sprintf("● %s", rcloneVer)
	statusStyle := lipgloss.NewStyle().Foreground(styles.ColorSuccess).Bold(true)
	if isMock {
		statusText = "● Demo Mode"
		statusStyle = lipgloss.NewStyle().Foreground(styles.ColorWarning).Bold(true)
	}
	statusRendered := statusStyle.Render(statusText)

	leftPart := lipgloss.JoinHorizontal(lipgloss.Center, logo, "  ", panelsRow)

	spaceLen := width - lipgloss.Width(leftPart) - lipgloss.Width(statusRendered) - 2
	if spaceLen < 1 {
		spaceLen = 1
	}
	spacer := lipgloss.NewStyle().Width(spaceLen).Render("")

	return lipgloss.JoinHorizontal(lipgloss.Center, leftPart, spacer, statusRendered)
}
