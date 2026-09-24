package views

import (
	"fmt"
	"strings"

	"github.com/celson/lazyrclone/internal/config"
	"github.com/celson/lazyrclone/internal/rclone"
	"github.com/celson/lazyrclone/internal/ui/components"
	"github.com/celson/lazyrclone/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

type ProfilesView struct {
	Config       *config.Config
	SelectedIdx  int
	DiffView     *components.DiffView
	Width        int
	Height       int
	IsActive     bool
	IsDiffActive bool
}

func NewProfilesView(cfg *config.Config) *ProfilesView {
	return &ProfilesView{
		Config:   cfg,
		DiffView: components.NewDiffView(80, 15),
		IsActive: true,
	}
}

func (v *ProfilesView) SetSize(width, height int) {
	v.Width = width
	v.Height = height
	if v.DiffView != nil {
		v.DiffView.Width = width - 4
		v.DiffView.Height = (height / 2) - 3
	}
}

func (v *ProfilesView) MoveUp() {
	if v.IsDiffActive {
		v.DiffView.MoveUp()
		return
	}
	if v.SelectedIdx > 0 {
		v.SelectedIdx--
	}
}

func (v *ProfilesView) MoveDown() {
	if v.IsDiffActive {
		v.DiffView.MoveDown()
		return
	}
	if v.SelectedIdx < len(v.Config.Profiles)-1 {
		v.SelectedIdx++
	}
}

func (v *ProfilesView) SelectedProfile() *config.Profile {
	if len(v.Config.Profiles) == 0 || v.SelectedIdx >= len(v.Config.Profiles) {
		return nil
	}
	return v.Config.Profiles[v.SelectedIdx]
}

func (v *ProfilesView) SetDryRunResult(res *rclone.DryRunResult) {
	v.DiffView.SetResult(res)
}

func (v *ProfilesView) ToggleDiffFocus() {
	v.IsDiffActive = !v.IsDiffActive
}

func (v *ProfilesView) Render() string {
	topHeight := (v.Height / 2) - 1
	bottomHeight := v.Height - topHeight - 2
	if topHeight < 4 {
		topHeight = 4
	}
	if bottomHeight < 4 {
		bottomHeight = 4
	}

	// 1. Profiles List (Top Pane)
	var listItems []string
	if len(v.Config.Profiles) == 0 {
		listItems = append(listItems, lipgloss.NewStyle().Foreground(styles.ColorMuted).Padding(1, 2).Render("No profiles saved. Press [n] to create your first sync/copy profile."))
	} else {
		for i, p := range v.Config.Profiles {
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
			arrow := lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("➔")
			transferText := fmt.Sprintf("%s %s %s", p.Source, arrow, p.Destination)

			statusBadge := ""
			if p.LastStatus != "" {
				statusBadge = lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render("✓ " + p.LastStatus)
			}

			line := fmt.Sprintf("%-28s %-10s %-40s %s", p.Name, opBadge, transferText, statusBadge)

			if i == v.SelectedIdx && !v.IsDiffActive {
				listItems = append(listItems, styles.SelectedItemStyle.Width(v.Width-6).Render("▶ "+line))
			} else {
				listItems = append(listItems, styles.NormalItemStyle.Render("  "+line))
			}
		}
	}

	topContent := strings.Join(listItems, "\n")
	topPanelStyle := styles.PanelStyle
	topTitle := styles.PanelTitleStyle.Render(" Profiles (Saved Sync / Copy Tasks) ")
	if !v.IsDiffActive {
		topPanelStyle = styles.ActivePanelStyle
		topTitle = styles.ActivePanelTitleStyle.Render(" Profiles (Saved Sync / Copy Tasks) ")
	}

	topPanel := topPanelStyle.
		Width(v.Width - 4).
		Height(topHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left, topTitle, "\n", topContent))

	// 2. Profile Details & Dry-Run Preview (Bottom Pane)
	p := v.SelectedProfile()
	var detailsSection string
	if p != nil {
		flagsStr := "none"
		if len(p.Flags) > 0 {
			flagsStr = strings.Join(p.Flags, " ")
		}
		excludeStr := "none"
		if len(p.Exclude) > 0 {
			excludeStr = strings.Join(p.Exclude, ", ")
		}

		infoLine1 := fmt.Sprintf("Source: %s   Destination: %s   Op: %s",
			lipgloss.NewStyle().Bold(true).Foreground(styles.ColorActive).Render(p.Source),
			lipgloss.NewStyle().Bold(true).Foreground(styles.ColorActive).Render(p.Destination),
			lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render(string(p.Operation)),
		)
		infoLine2 := fmt.Sprintf("Flags: %s   Excludes: %s   Transfers: %d",
			flagsStr, excludeStr, p.Transfers,
		)
		detailsSection = lipgloss.JoinVertical(lipgloss.Left, infoLine1, infoLine2, "")
	}

	bottomPanelStyle := styles.PanelStyle
	bottomTitle := styles.PanelTitleStyle.Render(" Dry-Run Honest Diff Preview ")
	if v.IsDiffActive {
		bottomPanelStyle = styles.ActivePanelStyle
		bottomTitle = styles.ActivePanelTitleStyle.Render(" Dry-Run Honest Diff Preview (Focused - j/k to scroll) ")
	}

	diffContent := v.DiffView.Render()
	bottomFullContent := lipgloss.JoinVertical(lipgloss.Left, bottomTitle, detailsSection, diffContent)

	bottomPanel := bottomPanelStyle.
		Width(v.Width - 4).
		Height(bottomHeight).
		Render(bottomFullContent)

	return lipgloss.JoinVertical(lipgloss.Left, topPanel, bottomPanel)
}
