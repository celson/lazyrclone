package views

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

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

func (v *RemotesView) RenderLines(width, height int) []string {
	innerWidth := width - 2
	innerHeight := height - 2
	if innerWidth < 10 {
		innerWidth = 10
	}
	if innerHeight < 4 {
		innerHeight = 4
	}

	var lines []string

	// Header row
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorSecondary)
	nameColWidth := 20
	typeColWidth := 14
	statusColWidth := 14
	quotaColWidth := innerWidth - nameColWidth - typeColWidth - statusColWidth - 4
	if quotaColWidth < 16 {
		quotaColWidth = 16
	}

	headerLine := fmt.Sprintf("  %-*s %-*s %-*s %s",
		nameColWidth, "REMOTE",
		typeColWidth, "TYPE",
		quotaColWidth, "QUOTA / STORAGE",
		"STATUS",
	)
	lines = append(lines, headerStyle.Render(headerLine))
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorBorder).Render("  "+strings.Repeat("─", innerWidth-4)))

	if len(v.Remotes) == 0 {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("  No remotes found in rclone.conf."))
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("  Press [r] to refresh or configure remotes using 'rclone config'."))
		return lines
	}

	// How many lines to allocate for list vs details
	listHeight := (innerHeight * 55) / 100
	if listHeight < 5 {
		listHeight = 5
	}
	maxVisibleRemotes := listHeight - 2
	if maxVisibleRemotes < 1 {
		maxVisibleRemotes = 1
	}

	scrollOffset := 0
	if v.SelectedIdx >= maxVisibleRemotes {
		scrollOffset = v.SelectedIdx - maxVisibleRemotes + 1
	}
	endIdx := scrollOffset + maxVisibleRemotes
	if endIdx > len(v.Remotes) {
		endIdx = len(v.Remotes)
	}

	for i := scrollOffset; i < endIdx; i++ {
		r := v.Remotes[i]
		typeBadge := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render(fmt.Sprintf("[%s]", r.Info.Type))

		testStatus := lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("• Ready")
		if r.Testing {
			testStatus = lipgloss.NewStyle().Foreground(styles.ColorWarning).Render("⟳ testing...")
		} else if r.TestOK != nil {
			if *r.TestOK {
				testStatus = lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render("✓ Online")
			} else {
				testStatus = lipgloss.NewStyle().Foreground(styles.ColorDanger).Render("✕ Offline")
			}
		}

		quotaShort := lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("--")
		if r.About != nil && r.About.Total > 0 {
			usedPct := int((float64(r.About.Used) / float64(r.About.Total)) * 100)
			quotaShort = fmt.Sprintf("%s / %s (%d%%)", rclone.FormatBytes(r.About.Used), rclone.FormatBytes(r.About.Total), usedPct)
		} else if r.About != nil && r.About.Used > 0 {
			quotaShort = fmt.Sprintf("%s used", rclone.FormatBytes(r.About.Used))
		}

		cleanName := r.Info.Name
		if len(cleanName) > nameColWidth-2 {
			cleanName = cleanName[:nameColWidth-4] + ".."
		}

		rowContent := fmt.Sprintf("%-*s %-*s %-*s %s",
			nameColWidth, cleanName,
			typeColWidth, typeBadge,
			quotaColWidth, quotaShort,
			testStatus,
		)

		if i == v.SelectedIdx {
			lines = append(lines, styles.SelectedItemStyle.Width(innerWidth-2).Render("▶ "+rowContent))
		} else {
			lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorWhite).Render("  "+rowContent))
		}
	}

	// Pad between list and details if needed
	for len(lines) < listHeight {
		lines = append(lines, "")
	}

	// Divider
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat("─", innerWidth)))

	// Details Card for Selected Remote
	sel := v.SelectedRemote()
	if sel != nil {
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

		titleLine := fmt.Sprintf("  Remote: %s   Provider: %s   Status: %s",
			lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render(sel.Info.Name),
			lipgloss.NewStyle().Bold(true).Foreground(styles.ColorSecondary).Render(sel.Info.Type),
			statusStyle.Render(statusStr),
		)
		lines = append(lines, titleLine)

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
			"  [Enter] Browse  [t] Test  [c] Config (CLI)  [e] Reconnect  [d] Delete  [r] Refresh",
		)
		lines = append(lines, actionGuide)
	}

	return lines
}

func (v *RemotesView) Render() string {
	return strings.Join(v.RenderLines(v.Width, v.Height), "\n")
}
