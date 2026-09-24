package views

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/celson/lazyrclone/internal/rclone"
	"github.com/celson/lazyrclone/internal/ui/components"
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
	IsActive    bool
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
		items = append(items, item)
	}

	v.Remotes = items
	if v.SelectedIdx >= len(v.Remotes) {
		v.SelectedIdx = 0
	}

	// Fetch quota asynchronously in background so app starts instantly
	for _, it := range items {
		if it.Info.Name != "local:" {
			go func(target *RemoteItem) {
				ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
				defer cancel()
				if about, err := v.Client.AboutRemote(ctx, target.Info.Name); err == nil {
					target.About = about
				}
			}(it)
		}
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
	innerWidth := v.Width - 2
	innerHeight := v.Height - 2
	if innerWidth < 10 {
		innerWidth = 10
	}
	if innerHeight < 2 {
		innerHeight = 2
	}

	var lines []string
	if len(v.Remotes) == 0 {
		lines = append(lines, "  No remotes found.")
		lines = append(lines, "  Press [c] to configure.")
	} else {
		maxVisible := innerHeight
		scrollOffset := 0
		if v.SelectedIdx >= maxVisible {
			scrollOffset = v.SelectedIdx - maxVisible + 1
		}
		endIdx := scrollOffset + maxVisible
		if endIdx > len(v.Remotes) {
			endIdx = len(v.Remotes)
		}

		for i := scrollOffset; i < endIdx; i++ {
			r := v.Remotes[i]
			typeBadge := fmt.Sprintf("[%s]", r.Info.Type)
			name := r.Info.Name

			statusIcon := "•"
			if r.Testing {
				statusIcon = "⟳"
			} else if r.TestOK != nil {
				if *r.TestOK {
					statusIcon = "✓"
				} else {
					statusIcon = "✕"
				}
			}

			availName := innerWidth - len(typeBadge) - 7
			if availName < 6 {
				availName = 6
			}
			cleanName := name
			if len(cleanName) > availName {
				cleanName = cleanName[:availName-2] + ".."
			}

			row := fmt.Sprintf("%s %-*s %s", statusIcon, availName, cleanName, typeBadge)
			if i == v.SelectedIdx {
				lines = append(lines, styles.SelectedItemStyle.Width(innerWidth-2).Render("▶ "+row))
			} else {
				lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorWhite).Render("  "+row))
			}
		}
	}

	return components.RenderPanelBox(v.Width, v.Height, "[2] Remotes", nil, v.IsActive, lines)
}

func (v *RemotesView) RenderDetails(width, height int) []string {
	innerWidth := width - 2
	if innerWidth < 10 {
		innerWidth = 10
	}

	var lines []string
	lines = append(lines, "")

	sel := v.SelectedRemote()
	if sel == nil {
		lines = append(lines, "  No remote selected.")
		lines = append(lines, "  Select a remote in [2] Remotes to view quota and connection details.")
		return lines
	}

	statusStr := "Ready"
	statusStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	if sel.Testing {
		statusStr = "Testing connection..."
		statusStyle = lipgloss.NewStyle().Foreground(styles.ColorWarning)
	} else if sel.TestOK != nil {
		if *sel.TestOK {
			statusStr = "Online (" + sel.TestMsg + ")"
			statusStyle = lipgloss.NewStyle().Foreground(styles.ColorSuccess)
		} else {
			statusStr = "Offline (" + sel.TestMsg + ")"
			statusStyle = lipgloss.NewStyle().Foreground(styles.ColorDanger)
		}
	}

	titleLine := fmt.Sprintf("  Remote: %s    Provider: %s    Status: %s",
		lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render(sel.Info.Name),
		lipgloss.NewStyle().Bold(true).Foreground(styles.ColorSecondary).Render(sel.Info.Type),
		statusStyle.Render(statusStr),
	)
	lines = append(lines, titleLine)
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorBorder).Render("  "+strings.Repeat("─", innerWidth-4)))

	// Quota Bar
	if sel.About != nil && sel.About.Total > 0 {
		usedPct := int((float64(sel.About.Used) / float64(sel.About.Total)) * 100)
		barWidth := innerWidth - 36
		if barWidth < 10 {
			barWidth = 10
		}
		progressBar := styles.RenderProgressBar(barWidth, usedPct)
		lines = append(lines, fmt.Sprintf("  Quota: %s %3d%%  (%s / %s, Free: %s)",
			progressBar, usedPct,
			rclone.FormatBytes(sel.About.Used),
			rclone.FormatBytes(sel.About.Total),
			rclone.FormatBytes(sel.About.Free),
		))
	} else if sel.About != nil && sel.About.Used > 0 {
		lines = append(lines, fmt.Sprintf("  Quota: %s used (provider does not report total quota limit)", rclone.FormatBytes(sel.About.Used)))
	} else {
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("  Quota: Not reported or unlimited."))
	}

	// Safe config parameters (strictly exclude passwords, tokens, keys, secrets)
	if len(sel.Info.Details) > 0 {
		var configPairs []string
		var safeKeys []string
		for k := range sel.Info.Details {
			if !rclone.IsSensitiveConfigKey(k) && k != "type" {
				safeKeys = append(safeKeys, k)
			}
		}
		sort.Strings(safeKeys)

		for _, k := range safeKeys {
			val := sel.Info.Details[k]
			if len(val) > 28 {
				val = val[:25] + "..."
			}
			configPairs = append(configPairs, fmt.Sprintf("%s = %s", k, val))
		}

		if len(configPairs) > 0 {
			cfgLine := "  Config: " + strings.Join(configPairs, " • ")
			if len(cfgLine) > innerWidth-4 && innerWidth > 8 {
				cfgLine = cfgLine[:innerWidth-7] + "..."
			}
			lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorSubtext).Render(cfgLine))
		}
	}

	lines = append(lines, "")
	actionGuide := lipgloss.NewStyle().Foreground(styles.ColorSecondary).Render(
		"  [Enter] Browse Explorer  [t] Test  [c] Config (CLI)  [e] Reconnect  [d] Delete  [r] Refresh",
	)
	lines = append(lines, actionGuide)

	return lines
}

func (v *RemotesView) RenderLines(width, height int) []string {
	return v.RenderDetails(width, height)
}
