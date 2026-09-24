package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/celson/lazyrclone/internal/rclone"
	"github.com/celson/lazyrclone/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

type RemoteItem struct {
	Info     rclone.RemoteInfo
	About    *rclone.AboutInfo
	Testing  bool
	TestOK   *bool
	TestMsg  string
}

type RemotesView struct {
	Client      rclone.RcloneClient
	Remotes     []*RemoteItem
	SelectedIdx int
	Loading     bool
	Width       int
	Height      int
}

func NewRemotesView(client rclone.RcloneClient) *RemotesView {
	return &RemotesView{
		Client:  client,
		Remotes: make([]*RemoteItem, 0),
	}
}

func (v *RemotesView) SetSize(width, height int) {
	v.Width = width
	v.Height = height
}

func (v *RemotesView) Refresh() {
	v.Loading = true
	remotes, err := v.Client.ListRemotes(context.Background())
	v.Loading = false
	if err != nil {
		return
	}

	items := make([]*RemoteItem, 0, len(remotes))
	for _, r := range remotes {
		item := &RemoteItem{
			Info: r,
		}
		if about, err := v.Client.AboutRemote(context.Background(), r.Name); err == nil {
			item.About = about
		}
		items = append(items, item)
	}

	v.Remotes = items
	if v.SelectedIdx >= len(v.Remotes) {
		v.SelectedIdx = 0
	}
}

func (v *RemotesView) MoveUp() {
	if v.SelectedIdx > 0 {
		v.SelectedIdx--
	}
}

func (v *RemotesView) MoveDown() {
	if v.SelectedIdx < len(v.Remotes)-1 {
		v.SelectedIdx++
	}
}

func (v *RemotesView) SelectedRemote() *RemoteItem {
	if len(v.Remotes) == 0 || v.SelectedIdx >= len(v.Remotes) {
		return nil
	}
	return v.Remotes[v.SelectedIdx]
}

func (v *RemotesView) TestConnection() {
	r := v.SelectedRemote()
	if r == nil {
		return
	}
	r.Testing = true
	r.TestMsg = "Testing reachability..."

	go func(target *RemoteItem) {
		_, err := v.Client.ListDir(context.Background(), target.Info.Name)
		target.Testing = false
		ok := (err == nil)
		target.TestOK = &ok
		if ok {
			target.TestMsg = "Connection successful!"
		} else {
			target.TestMsg = fmt.Sprintf("Failed: %v", err)
		}
	}(r)
}

func (v *RemotesView) Render() string {
	topHeight := (v.Height / 2) - 1
	bottomHeight := v.Height - topHeight - 2
	if topHeight < 5 {
		topHeight = 5
	}
	if bottomHeight < 5 {
		bottomHeight = 5
	}

	var listLines []string
	if len(v.Remotes) == 0 {
		listLines = append(listLines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("No remotes configured or detected. Press [r] to refresh."))
	} else {
		for i, r := range v.Remotes {
			typeBadge := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render(fmt.Sprintf("[%s]", r.Info.Type))
			testStatus := ""
			if r.Testing {
				testStatus = lipgloss.NewStyle().Foreground(styles.ColorWarning).Render("⟳ testing...")
			} else if r.TestOK != nil {
				if *r.TestOK {
					testStatus = lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render("✓ Online")
				} else {
					testStatus = lipgloss.NewStyle().Foreground(styles.ColorDanger).Render("✕ Offline")
				}
			}

			quotaShort := ""
			if r.About != nil && r.About.Total > 0 {
				usedPct := int((float64(r.About.Used) / float64(r.About.Total)) * 100)
				quotaShort = fmt.Sprintf("%s / %s (%d%%)", rclone.FormatBytes(r.About.Used), rclone.FormatBytes(r.About.Total), usedPct)
			}

			line := fmt.Sprintf("%-20s %-14s %-28s %s", r.Info.Name, typeBadge, quotaShort, testStatus)
			if i == v.SelectedIdx {
				listLines = append(listLines, styles.SelectedItemStyle.Width(v.Width-6).Render("▶ "+line))
			} else {
				listLines = append(listLines, styles.NormalItemStyle.Render("  "+line))
			}
		}
	}

	topPanel := styles.ActivePanelStyle.
		Width(v.Width - 4).
		Height(topHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left, styles.ActivePanelTitleStyle.Render(" Configured Rclone Remotes "), "\n", strings.Join(listLines, "\n")))

	// Details / Quota Panel
	sel := v.SelectedRemote()
	var detailsContent string
	if sel == nil {
		detailsContent = lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Select a remote to view credentials, configuration, and quota details.")
	} else {
		header := fmt.Sprintf("Remote: %s   Type: %s",
			lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render(sel.Info.Name),
			lipgloss.NewStyle().Bold(true).Foreground(styles.ColorSecondary).Render(sel.Info.Type),
		)

		var quotaLines []string
		if sel.About != nil && sel.About.Total > 0 {
			usedPct := int((float64(sel.About.Used) / float64(sel.About.Total)) * 100)
			barWidth := v.Width - 32
			if barWidth < 10 {
				barWidth = 10
			}
			bar := styles.RenderProgressBar(barWidth, usedPct)

			quotaLines = append(quotaLines,
				fmt.Sprintf("Storage Quota: %s %3d%%", bar, usedPct),
				fmt.Sprintf("Used: %s   Free: %s   Total: %s   Objects: %d",
					rclone.FormatBytes(sel.About.Used),
					rclone.FormatBytes(sel.About.Free),
					rclone.FormatBytes(sel.About.Total),
					sel.About.Objects,
				),
			)
		} else {
			quotaLines = append(quotaLines, "Storage Quota: Unlimited or not reported by provider.")
		}

		var detailLines []string
		if len(sel.Info.Details) > 0 {
			detailLines = append(detailLines, "Config Parameters:")
			for k, val := range sel.Info.Details {
				detailLines = append(detailLines, fmt.Sprintf("  • %-16s: %s", k, val))
			}
		}

		if sel.TestMsg != "" {
			detailLines = append(detailLines, fmt.Sprintf("Reachability: %s", sel.TestMsg))
		}

		parts := []string{header, "\n", strings.Join(quotaLines, "\n"), "\n", strings.Join(detailLines, "\n")}
		detailsContent = strings.Join(parts, "\n")
	}

	bottomPanel := styles.PanelStyle.
		Width(v.Width - 4).
		Height(bottomHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left, styles.PanelTitleStyle.Render(" Remote Details & Quota "), "\n", detailsContent))

	return lipgloss.JoinVertical(lipgloss.Left, topPanel, bottomPanel)
}
