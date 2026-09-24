package components

import (
	"fmt"
	"strings"

	"github.com/celson/lazyrclone/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

type Shortcut struct {
	Key  string
	Desc string
}

func RenderStatusBar(width int, shortcuts []Shortcut, message string, isError bool, rcloneVer string, isMock bool) string {
	statusColor := styles.ColorSuccess
	statusText := "● " + rcloneVer
	if isMock {
		statusColor = styles.ColorWarning
		statusText = "● Demo Mode"
	}
	rcloneBadge := lipgloss.NewStyle().Foreground(statusColor).Render(statusText)
	appBadge := lipgloss.NewStyle().Foreground(styles.MochaOverlay0).Render("lazyrclone v0.1.0")
	right := lipgloss.JoinHorizontal(lipgloss.Center, rcloneBadge, "  ", appBadge)
	rightWidth := lipgloss.Width(right)

	availForLeft := width - rightWidth - 2
	if availForLeft < 0 {
		availForLeft = 0
	}

	var leftParts []string
	currentWidth := 0

	if message != "" {
		msgStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorSuccess)
		if isError {
			msgStyle = lipgloss.NewStyle().Bold(true).Foreground(styles.ColorDanger)
		}
		msgRendered := msgStyle.Render(message) + "  |  "
		leftParts = append(leftParts, msgRendered)
		currentWidth += lipgloss.Width(msgRendered)
	}

	for _, sc := range shortcuts {
		key := styles.ShortcutKeyStyle.Render(fmt.Sprintf("[%s]", sc.Key))
		desc := styles.ShortcutDescStyle.Render(sc.Desc)
		item := fmt.Sprintf("%s %s", key, desc)
		itemWidth := lipgloss.Width(item) + 2

		if currentWidth+itemWidth > availForLeft {
			break
		}
		leftParts = append(leftParts, item)
		currentWidth += itemWidth
	}

	left := strings.Join(leftParts, "  ")

	spaceLen := width - lipgloss.Width(left) - rightWidth
	if spaceLen < 1 {
		spaceLen = 1
	}
	spacer := strings.Repeat(" ", spaceLen)

	content := left + spacer + right
	return styles.StatusBarStyle.Width(width).MaxWidth(width).Inline(true).Render(content)
}
