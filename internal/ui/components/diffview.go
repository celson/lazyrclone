package components

import (
	"fmt"
	"strings"

	"github.com/celson/lazyrclone/internal/rclone"
	"github.com/celson/lazyrclone/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

type DiffView struct {
	Result       *rclone.DryRunResult
	Cursor       int
	ScrollOffset int
	Height       int
	Width        int
}

func NewDiffView(width, height int) *DiffView {
	return &DiffView{
		Width:  width,
		Height: height,
	}
}

func (dv *DiffView) SetResult(res *rclone.DryRunResult) {
	dv.Result = res
	dv.Cursor = 0
	dv.ScrollOffset = 0
}

func (dv *DiffView) MoveUp() {
	if dv.Result == nil || len(dv.Result.Items) == 0 {
		return
	}
	if dv.Cursor > 0 {
		dv.Cursor--
		if dv.Cursor < dv.ScrollOffset {
			dv.ScrollOffset = dv.Cursor
		}
	}
}

func (dv *DiffView) MoveDown() {
	if dv.Result == nil || len(dv.Result.Items) == 0 {
		return
	}
	if dv.Cursor < len(dv.Result.Items)-1 {
		dv.Cursor++
		visibleLines := dv.Height - 3
		if visibleLines < 1 {
			visibleLines = 1
		}
		if dv.Cursor >= dv.ScrollOffset+visibleLines {
			dv.ScrollOffset = dv.Cursor - visibleLines + 1
		}
	}
}

func (dv *DiffView) RenderLines() []string {
	if dv.Result == nil {
		return []string{
			"",
			lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("  No dry-run diff available."),
			lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("  Select a profile in [1] and press [d] to run an honest dry-run diff preview."),
		}
	}

	if dv.Result.Err != nil {
		return []string{
			"",
			lipgloss.NewStyle().Foreground(styles.ColorDanger).Render(fmt.Sprintf("  Dry-run error: %v", dv.Result.Err)),
		}
	}

	// Summary bar
	addBadge := lipgloss.NewStyle().Foreground(styles.ColorSuccess).Bold(true).Render(fmt.Sprintf("+ %d to add", dv.Result.ToAdd))
	updateBadge := lipgloss.NewStyle().Foreground(styles.ColorWarning).Bold(true).Render(fmt.Sprintf("~ %d to update", dv.Result.ToUpdate))
	deleteBadge := lipgloss.NewStyle().Foreground(styles.ColorDanger).Bold(true).Render(fmt.Sprintf("- %d to delete", dv.Result.ToDelete))
	equalBadge := lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(fmt.Sprintf("= %d unchanged", dv.Result.Unchanged))
	durationStr := lipgloss.NewStyle().Foreground(styles.ColorSecondary).Render(fmt.Sprintf("(diff in %v)", dv.Result.Duration.Round(100)))

	summary := lipgloss.JoinHorizontal(lipgloss.Center,
		"  Diff Summary: ", addBadge, " | ", updateBadge, " | ", deleteBadge, " | ", equalBadge, "  ", durationStr,
	)

	separator := lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat("─", dv.Width))

	visibleCount := dv.Height - 3
	if visibleCount < 1 {
		visibleCount = 1
	}

	lines := []string{summary, separator}
	items := dv.Result.Items

	if len(items) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render("  ✓ Destination is already synchronized! No changes needed."))
	} else {
		endIdx := dv.ScrollOffset + visibleCount
		if endIdx > len(items) {
			endIdx = len(items)
		}

		for i := dv.ScrollOffset; i < endIdx; i++ {
			item := items[i]
			var badge string
			var lineStyle lipgloss.Style

			switch item.Action {
			case rclone.ActionAdd:
				badge = styles.BadgeAdd.String()
				lineStyle = lipgloss.NewStyle().Foreground(styles.ColorSuccess)
			case rclone.ActionUpdate:
				badge = styles.BadgeUpdate.String()
				lineStyle = lipgloss.NewStyle().Foreground(styles.ColorWarning)
			case rclone.ActionDelete:
				badge = styles.BadgeDelete.String()
				lineStyle = lipgloss.NewStyle().Foreground(styles.ColorDanger)
			default:
				badge = styles.BadgeEqual.String()
				lineStyle = lipgloss.NewStyle().Foreground(styles.ColorMuted)
			}

			sizeStr := ""
			if item.Size > 0 {
				sizeStr = fmt.Sprintf("[%s]", rclone.FormatBytes(item.Size))
			}

			line := fmt.Sprintf(" %-10s %-30s %-10s %s", badge, item.Path, sizeStr, item.Message)
			if i == dv.Cursor {
				line = styles.SelectedItemStyle.Width(dv.Width).Render(line)
			} else {
				line = lineStyle.Render(line)
			}
			lines = append(lines, line)
		}
	}

	return lines
}

func (dv *DiffView) Render() string {
	return strings.Join(dv.RenderLines(), "\n")
}
