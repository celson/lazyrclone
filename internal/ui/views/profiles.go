package views

import (
	"fmt"
	"strings"

	"github.com/celson/lazyrclone/internal/config"
	"github.com/celson/lazyrclone/internal/ui/components"
	"github.com/celson/lazyrclone/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

type ProfilesView struct {
	Config       *config.Config
	Remotes      []*RemoteItem
	SelectedIdx  int
	Width        int
	Height       int
	IsActive     bool
	ScrollOffset int
}

func NewProfilesView(cfg *config.Config) *ProfilesView {
	return &ProfilesView{
		Config:   cfg,
		IsActive: true,
	}
}

func (v *ProfilesView) SetSize(width, height int) {
	v.Width = width
	v.Height = height
}

func (v *ProfilesView) SetRemotes(remotes []*RemoteItem) {
	v.Remotes = remotes
}

func (v *ProfilesView) MoveUp() {
	if v.SelectedIdx > 0 {
		v.SelectedIdx--
		if v.SelectedIdx < v.ScrollOffset {
			v.ScrollOffset = v.SelectedIdx
		}
	}
}

func (v *ProfilesView) MoveDown() {
	if v.SelectedIdx < len(v.Config.Profiles)-1 {
		v.SelectedIdx++
		visibleLines := (v.Height - 5) / 2
		if visibleLines < 1 {
			visibleLines = 1
		}
		if v.SelectedIdx >= v.ScrollOffset+visibleLines {
			v.ScrollOffset = v.SelectedIdx - visibleLines + 1
		}
	}
}

func (v *ProfilesView) SelectedProfile() *config.Profile {
	if len(v.Config.Profiles) == 0 || v.SelectedIdx >= len(v.Config.Profiles) {
		return nil
	}
	return v.Config.Profiles[v.SelectedIdx]
}

func (v *ProfilesView) Render() string {
	innerWidth := v.Width - 2
	innerHeight := v.Height - 2
	if innerWidth < 5 {
		innerWidth = 5
	}
	if innerHeight < 2 {
		innerHeight = 2
	}

	var lines []string

	if len(v.Config.Profiles) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(" No profiles saved."))
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(" Press [n] to create."))
	} else {
		remotesFooterHeight := 0
		if len(v.Remotes) > 0 {
			remotesFooterHeight = 2
		}

		visibleCount := (innerHeight - remotesFooterHeight) / 2
		if visibleCount < 1 {
			visibleCount = 1
		}

		endIdx := v.ScrollOffset + visibleCount
		if endIdx > len(v.Config.Profiles) {
			endIdx = len(v.Config.Profiles)
		}

		for i := v.ScrollOffset; i < endIdx; i++ {
			p := v.Config.Profiles[i]
			opColor := styles.ColorPrimary
			switch p.Operation {
			case config.OpSync:
				opColor = styles.ColorWarning
			case config.OpCheck:
				opColor = styles.ColorSecondary
			case config.OpMove:
				opColor = styles.ColorDanger
			}

			opBadge := lipgloss.NewStyle().Bold(true).Foreground(opColor).Render(fmt.Sprintf("[%s]", strings.ToUpper(string(p.Operation))))

			maxNameWidth := innerWidth - 10
			if maxNameWidth < 6 {
				maxNameWidth = 6
			}
			displayName := p.Name
			if len(displayName) > maxNameWidth {
				displayName = displayName[:maxNameWidth-3] + "..."
			}

			headerLine := fmt.Sprintf("%s %s", opBadge, displayName)

			transferLine := fmt.Sprintf("%s ➔ %s", p.Source, p.Destination)
			if len(transferLine) > innerWidth-4 && innerWidth > 8 {
				transferLine = transferLine[:innerWidth-7] + "..."
			}

			if i == v.SelectedIdx {
				cursorStyle := styles.SelectedItemStyle.Width(innerWidth)
				lines = append(lines, cursorStyle.Render("▶ "+headerLine))
				lines = append(lines, cursorStyle.Render("  "+transferLine))
			} else {
				lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorWhite).Render("  "+headerLine))
				lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("  "+transferLine))
			}
		}

		// Remotes footer if space permits
		if len(v.Remotes) > 0 && len(lines) < innerHeight-1 {
			for len(lines) < innerHeight-2 {
				lines = append(lines, "")
			}
			lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat("─", innerWidth)))
			remotesHint := fmt.Sprintf(" Remotes: %d available • Press [R] or [4] to view", len(v.Remotes))
			lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorSecondary).Render(remotesHint))
		}
	}

	return components.RenderPanelBox(v.Width, v.Height, "[1] Profiles", nil, v.IsActive, lines)
}
