package components

import (
	"strings"

	"github.com/celson/lazyrclone/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

type SubTabItem struct {
	Name     string
	IsActive bool
}

// RenderPanelBox draws a completely enclosed, rectangular box with the title
// and optional sub-tabs embedded directly into the top border line,
// identical to Lazygit and Lazydocker.
// It is guaranteed to render EXACTLY `height` lines and EXACTLY `width` columns.
func RenderPanelBox(width, height int, title string, subTabs []SubTabItem, isActive bool, contentLines []string) string {
	if width < 6 {
		width = 6
	}
	if height < 3 {
		height = 3
	}

	borderColor := styles.MochaSurface1
	if isActive {
		borderColor = styles.MochaMauve
	}
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	// 1. Build Top Border Line
	var titleParts []string
	if title != "" {
		if isActive {
			titleParts = append(titleParts, lipgloss.NewStyle().Bold(true).Foreground(styles.MochaMauve).Render(title))
		} else {
			titleParts = append(titleParts, lipgloss.NewStyle().Bold(true).Foreground(styles.MochaSubtext0).Render(title))
		}
	}

	if len(subTabs) > 0 {
		var tabGroupParts []string
		for _, tab := range subTabs {
			if tab.IsActive {
				tabStr := lipgloss.NewStyle().Bold(true).Foreground(styles.MochaCrust).Background(styles.MochaMauve).Render(" " + tab.Name + " ")
				tabGroupParts = append(tabGroupParts, tabStr)
			} else {
				tabStr := lipgloss.NewStyle().Foreground(styles.MochaOverlay0).Render(" " + tab.Name + " ")
				tabGroupParts = append(tabGroupParts, tabStr)
			}
		}
		tabsJoined := strings.Join(tabGroupParts, borderStyle.Render("─"))
		tabIndicator := borderStyle.Render("< [ ] ") + tabsJoined + borderStyle.Render(" [ ] >")
		titleParts = append(titleParts, tabIndicator)
	}

	titleCombined := strings.Join(titleParts, borderStyle.Render("─"))
	leftCap := borderStyle.Render("╭─")
	rightCap := borderStyle.Render("╮")

	usedWidth := 2 + lipgloss.Width(titleCombined) + 1 // "╭─" + title + "─...─" + "╮"
	var topLine string
	if usedWidth >= width {
		// Title is too long, clamp it
		maxTitleW := width - 4
		if maxTitleW < 1 {
			maxTitleW = 1
		}
		clampedTitle := lipgloss.NewStyle().MaxWidth(maxTitleW).Inline(true).Render(titleCombined)
		topLine = leftCap + clampedTitle + borderStyle.Render("─") + rightCap
	} else {
		remaining := width - usedWidth
		dashes := borderStyle.Render(strings.Repeat("─", remaining))
		topLine = leftCap + titleCombined + dashes + rightCap
	}

	// 2. Build Content Lines
	innerWidth := width - 2
	innerHeight := height - 2
	cellStyle := lipgloss.NewStyle().Width(innerWidth).MaxWidth(innerWidth).Inline(true)
	vBorder := borderStyle.Render("│")

	rows := make([]string, 0, height)
	rows = append(rows, topLine)

	for i := 0; i < innerHeight; i++ {
		var line string
		if i < len(contentLines) {
			line = contentLines[i]
		}
		renderedCell := cellStyle.Render(line)
		rows = append(rows, vBorder+renderedCell+vBorder)
	}

	// 3. Build Bottom Border Line
	bottomLine := borderStyle.Render("╰" + strings.Repeat("─", innerWidth) + "╯")
	rows = append(rows, bottomLine)

	return strings.Join(rows, "\n")
}
