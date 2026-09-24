package views

import (
	"fmt"
	"strings"

	"github.com/celson/lazyrclone/internal/config"
	"github.com/celson/lazyrclone/internal/rclone"
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
	panelStyle := styles.PanelStyle
	title := styles.PanelTitleStyle.Render(" [1] Profiles ")
	if v.IsActive {
		panelStyle = styles.ActivePanelStyle
		title = styles.ActivePanelTitleStyle.Render(" [1] Profiles (Tasks) ")
	}

	innerContentHeight := v.Height - 3
	if innerContentHeight < 3 {
		innerContentHeight = 3
	}

	var lines []string

	if len(v.Config.Profiles) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Padding(1, 1).Render("No profiles saved.\nPress [n] to create."))
	} else {
		visibleCount := (innerContentHeight - 3) / 2
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

			maxNameWidth := v.Width - 14
			if maxNameWidth < 10 {
				maxNameWidth = 10
			}
			displayName := p.Name
			if len(displayName) > maxNameWidth {
				displayName = displayName[:maxNameWidth-3] + "..."
			}

			headerLine := fmt.Sprintf("%s %s", opBadge, displayName)

			maxPathWidth := v.Width - 6
			if maxPathWidth < 12 {
				maxPathWidth = 12
			}
			transferLine := fmt.Sprintf("%s ➔ %s", p.Source, p.Destination)
			if len(transferLine) > maxPathWidth {
				transferLine = transferLine[:maxPathWidth-3] + "..."
			}
			subLine := lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("  " + transferLine)

			if i == v.SelectedIdx {
				cursorStyle := styles.SelectedItemStyle.Width(v.Width - 4)
				lines = append(lines, cursorStyle.Render("▶ "+headerLine))
				lines = append(lines, cursorStyle.Render("  "+transferLine))
			} else {
				lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorWhite).Render("  "+headerLine))
				lines = append(lines, subLine)
			}
		}
	}

	// Remotes footer in Panel 1
	var remotesFooter string
	if len(v.Remotes) > 0 {
		var rTokens []string
		for _, r := range v.Remotes {
			quota := ""
			if r.About != nil && r.About.Total > 0 {
				quota = fmt.Sprintf(" (%s)", rclone.FormatBytes(r.About.Used))
			}
			rTokens = append(rTokens, fmt.Sprintf("%s%s", r.Info.Name, quota))
		}
		remotesLine := strings.Join(rTokens, " • ")
		if len(remotesLine) > v.Width-6 && v.Width > 10 {
			remotesLine = remotesLine[:v.Width-9] + "..."
		}
		remotesFooter = lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat("─", v.Width-4)),
			lipgloss.NewStyle().Foreground(styles.ColorSecondary).Render("Remotes: "+remotesLine),
		)
	}

	fullBody := strings.Join(lines, "\n")
	if remotesFooter != "" {
		fullBody = lipgloss.JoinVertical(lipgloss.Left, fullBody, "\n", remotesFooter)
	}

	return panelStyle.
		Width(v.Width).
		Height(v.Height).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, "\n", fullBody))
}
