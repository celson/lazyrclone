package components

import (
	"fmt"

	"github.com/celson/lazyrclone/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

type Tab int

const (
	TabProfiles Tab = iota
	TabExplorer
	TabTransfers
	TabRemotes
)

var TabNames = []string{"1: Profiles", "2: Explorer", "3: Transfers", "4: Remotes"}

func RenderHeader(width int, currentTab Tab, rcloneVer string, isMock bool) string {
	logo := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.MochaCrust).
		Background(styles.ColorPrimary).
		Padding(0, 1).
		Render("⚡ lazyrclone")

	var tabs []string
	for i, name := range TabNames {
		if Tab(i) == currentTab {
			tabs = append(tabs, styles.TabActiveStyle.Render(name))
		} else {
			tabs = append(tabs, styles.TabInactiveStyle.Render(name))
		}
	}
	tabsRow := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	statusText := fmt.Sprintf("● %s", rcloneVer)
	statusStyle := lipgloss.NewStyle().Foreground(styles.ColorSuccess).Bold(true)
	if isMock {
		statusText = "● Demo Mode"
		statusStyle = lipgloss.NewStyle().Foreground(styles.ColorWarning).Bold(true)
	}
	statusRendered := statusStyle.Render(statusText)

	leftPart := lipgloss.JoinHorizontal(lipgloss.Center, logo, "  ", tabsRow)

	spaceLen := width - lipgloss.Width(leftPart) - lipgloss.Width(statusRendered) - 2
	if spaceLen < 1 {
		spaceLen = 1
	}
	spacer := lipgloss.NewStyle().Width(spaceLen).Render("")

	return lipgloss.JoinHorizontal(lipgloss.Center, leftPart, spacer, statusRendered)
}
