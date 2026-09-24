package views

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/celson/lazyrclone/internal/rclone"
	"github.com/celson/lazyrclone/internal/ui/components"
	"github.com/celson/lazyrclone/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

type ExplorerPane struct {
	Remote       string
	CurrentDir   string
	Items        []rclone.FileItem
	SelectedIdx  int
	ScrollOffset int
	Marked       map[string]bool
	Loading      bool
	ErrorMsg     string
}

type ExplorerView struct {
	Panes      [2]*ExplorerPane
	ActivePane int // 0 = Left, 1 = Right
	Client     rclone.RcloneClient
	Width      int
	Height     int
}

func NewExplorerView(client rclone.RcloneClient) *ExplorerView {
	ev := &ExplorerView{
		Client:     client,
		ActivePane: 0,
		Panes: [2]*ExplorerPane{
			{
				Remote:     "local:",
				CurrentDir: ".",
				Marked:     make(map[string]bool),
			},
			{
				Remote:     "gdrive:",
				CurrentDir: "",
				Marked:     make(map[string]bool),
			},
		},
	}
	return ev
}

func (v *ExplorerView) LoadPane(paneIdx int) {
	p := v.Panes[paneIdx]
	p.Loading = true
	p.ErrorMsg = ""

	remotePath := p.Remote
	if p.CurrentDir != "" && p.CurrentDir != "." {
		if !strings.HasSuffix(remotePath, "/") && !strings.HasSuffix(remotePath, ":") {
			remotePath += "/"
		}
		remotePath += strings.TrimPrefix(p.CurrentDir, "/")
	}

	go func(targetPane *ExplorerPane, path string) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		items, err := v.Client.ListDir(ctx, path)
		targetPane.Loading = false
		if err != nil {
			targetPane.ErrorMsg = err.Error()
			targetPane.Items = nil
			return
		}

		targetPane.Items = items
		if targetPane.SelectedIdx >= len(items) {
			targetPane.SelectedIdx = 0
		}
		targetPane.ScrollOffset = 0
	}(p, remotePath)
}

func (v *ExplorerView) SetSize(width, height int) {
	v.Width = width
	v.Height = height
}

func (v *ExplorerView) CurrentActivePane() *ExplorerPane {
	return v.Panes[v.ActivePane]
}

func (v *ExplorerView) InactivePane() *ExplorerPane {
	return v.Panes[1-v.ActivePane]
}

func (v *ExplorerView) SwitchPane() {
	v.ActivePane = 1 - v.ActivePane
}

func (v *ExplorerView) MoveUp() {
	p := v.CurrentActivePane()
	if p.SelectedIdx > 0 {
		p.SelectedIdx--
		if p.SelectedIdx < p.ScrollOffset {
			p.ScrollOffset = p.SelectedIdx
		}
	}
}

func (v *ExplorerView) MoveDown() {
	p := v.CurrentActivePane()
	if p.SelectedIdx < len(p.Items)-1 {
		p.SelectedIdx++
		visibleLines := v.Height - 4
		if visibleLines < 1 {
			visibleLines = 1
		}
		if p.SelectedIdx >= p.ScrollOffset+visibleLines {
			p.ScrollOffset = p.SelectedIdx - visibleLines + 1
		}
	}
}

func (v *ExplorerView) ToggleSelect() {
	p := v.CurrentActivePane()
	if len(p.Items) == 0 || p.SelectedIdx >= len(p.Items) {
		return
	}
	item := p.Items[p.SelectedIdx]
	if p.Marked[item.Name] {
		delete(p.Marked, item.Name)
	} else {
		p.Marked[item.Name] = true
	}
}

func (v *ExplorerView) EnterDir() bool {
	p := v.CurrentActivePane()
	if len(p.Items) == 0 || p.SelectedIdx >= len(p.Items) {
		return false
	}
	item := p.Items[p.SelectedIdx]
	if item.IsDir {
		p.CurrentDir = path.Join(p.CurrentDir, item.Name)
		p.SelectedIdx = 0
		p.ScrollOffset = 0
		p.Marked = make(map[string]bool)
		v.LoadPane(v.ActivePane)
		return true
	}
	return false
}

func (v *ExplorerView) GoUp() bool {
	p := v.CurrentActivePane()
	if p.CurrentDir == "" || p.CurrentDir == "." || p.CurrentDir == "/" {
		return false
	}
	parent := path.Dir(p.CurrentDir)
	if parent == "." || parent == "/" {
		parent = ""
	}
	p.CurrentDir = parent
	p.SelectedIdx = 0
	p.ScrollOffset = 0
	p.Marked = make(map[string]bool)
	v.LoadPane(v.ActivePane)
	return true
}

func (v *ExplorerView) SetPaneRemote(paneIdx int, remote string) {
	v.Panes[paneIdx].Remote = remote
	v.Panes[paneIdx].CurrentDir = ""
	v.Panes[paneIdx].SelectedIdx = 0
	v.Panes[paneIdx].ScrollOffset = 0
	v.Panes[paneIdx].Marked = make(map[string]bool)
	v.LoadPane(paneIdx)
}

func (v *ExplorerView) GetSelectedOrMarkedItems() []rclone.FileItem {
	p := v.CurrentActivePane()
	var selected []rclone.FileItem
	if len(p.Marked) > 0 {
		for _, it := range p.Items {
			if p.Marked[it.Name] {
				selected = append(selected, it)
			}
		}
	} else if len(p.Items) > 0 && p.SelectedIdx < len(p.Items) {
		selected = append(selected, p.Items[p.SelectedIdx])
	}
	return selected
}

func (v *ExplorerView) renderPane(idx int, paneWidth, paneHeight int) string {
	p := v.Panes[idx]
	isActive := (idx == v.ActivePane)

	paneLabel := "Source"
	if idx == 1 {
		paneLabel = "Dest"
	}
	headerText := fmt.Sprintf("[%s: %s%s]", paneLabel, p.Remote, p.CurrentDir)

	innerWidth := paneWidth - 2
	visibleLines := paneHeight - 2
	if visibleLines < 1 {
		visibleLines = 1
	}

	var lines []string

	if p.Loading {
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorSecondary).Render(" Loading contents..."))
	} else if p.ErrorMsg != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorDanger).Render(" Error: "+p.ErrorMsg))
	} else if len(p.Items) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(" (Directory empty)"))
	} else {
		endIdx := p.ScrollOffset + visibleLines
		if endIdx > len(p.Items) {
			endIdx = len(p.Items)
		}

		for i := p.ScrollOffset; i < endIdx; i++ {
			item := p.Items[i]
			icon := "📄"
			nameColor := styles.ColorWhite
			if item.IsDir {
				icon = "📁"
				nameColor = styles.ColorSecondary
			}

			check := "[ ]"
			if p.Marked[item.Name] {
				check = "[x]"
			}

			sizeStr := ""
			if !item.IsDir {
				sizeStr = rclone.FormatBytes(item.Size)
			} else {
				sizeStr = "<DIR>"
			}

			maxNameLen := innerWidth - 18
			if maxNameLen < 6 {
				maxNameLen = 6
			}
			displayName := item.Name
			if len(displayName) > maxNameLen {
				displayName = displayName[:maxNameLen-3] + "..."
			}

			itemRow := fmt.Sprintf("%s %s %-*s %8s", check, icon, maxNameLen, displayName, sizeStr)

			if i == p.SelectedIdx && isActive {
				itemRow = styles.SelectedItemStyle.Width(innerWidth).Render(itemRow)
			} else {
				if p.Marked[item.Name] {
					itemRow = lipgloss.NewStyle().Foreground(styles.ColorSuccess).Bold(true).Render(itemRow)
				} else {
					itemRow = lipgloss.NewStyle().Foreground(nameColor).Render(itemRow)
				}
			}
			lines = append(lines, itemRow)
		}
	}

	return components.RenderPanelBox(paneWidth, paneHeight, headerText, nil, isActive, lines)
}

func (v *ExplorerView) Render() string {
	paneWidth := v.Width / 2
	if paneWidth < 15 {
		paneWidth = 15
	}
	rightPaneWidth := v.Width - paneWidth

	left := v.renderPane(0, paneWidth, v.Height)
	right := v.renderPane(1, rightPaneWidth, v.Height)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}
