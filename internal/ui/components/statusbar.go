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

func RenderStatusBar(width int, shortcuts []Shortcut, message string, isError bool) string {
	var parts []string
	for _, sc := range shortcuts {
		key := styles.ShortcutKeyStyle.Render(fmt.Sprintf("[%s]", sc.Key))
		desc := styles.ShortcutDescStyle.Render(sc.Desc)
		parts = append(parts, fmt.Sprintf("%s %s", key, desc))
	}
	left := strings.Join(parts, "  ")

	var right string
	if message != "" {
		msgStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorSuccess)
		if isError {
			msgStyle = lipgloss.NewStyle().Bold(true).Foreground(styles.ColorDanger)
		}
		right = msgStyle.Render(message)
	}

	spaceLen := width - lipgloss.Width(left) - lipgloss.Width(right) - 4
	if spaceLen < 1 {
		spaceLen = 1
	}
	spacer := lipgloss.NewStyle().Width(spaceLen).Render("")

	content := lipgloss.JoinHorizontal(lipgloss.Center, left, spacer, right)
	return styles.StatusBarStyle.Width(width).Render(content)
}
