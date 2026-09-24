package views

import (
	"fmt"
	"strings"

	"github.com/celson/lazyrclone/internal/rclone"
	"github.com/celson/lazyrclone/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

type TransfersView struct {
	Jobs        []*rclone.TransferJob
	SelectedJob int
	Width       int
	Height      int
	IsActive    bool
}

func NewTransfersView() *TransfersView {
	return &TransfersView{
		Jobs: make([]*rclone.TransferJob, 0),
	}
}

func (v *TransfersView) SetSize(width, height int) {
	v.Width = width
	v.Height = height
}

func (v *TransfersView) AddJob(job *rclone.TransferJob) {
	v.Jobs = append([]*rclone.TransferJob{job}, v.Jobs...)
	v.SelectedJob = 0
}

func (v *TransfersView) ActiveJob() *rclone.TransferJob {
	if len(v.Jobs) == 0 || v.SelectedJob >= len(v.Jobs) {
		return nil
	}
	return v.Jobs[v.SelectedJob]
}

func (v *TransfersView) MoveUp() {
	if v.SelectedJob > 0 {
		v.SelectedJob--
	}
}

func (v *TransfersView) MoveDown() {
	if v.SelectedJob < len(v.Jobs)-1 {
		v.SelectedJob++
	}
}

func (v *TransfersView) ClearCompleted() {
	var active []*rclone.TransferJob
	for _, j := range v.Jobs {
		if j.Status == rclone.JobStatusRunning {
			active = append(active, j)
		}
	}
	v.Jobs = active
	v.SelectedJob = 0
}

func (v *TransfersView) Render() string {
	panelStyle := styles.PanelStyle
	title := styles.PanelTitleStyle.Render(" [2] Transfers & Runs ")
	if v.IsActive {
		panelStyle = styles.ActivePanelStyle
		title = styles.ActivePanelTitleStyle.Render(" [2] Transfers & Runs ")
	}

	var contentLines []string

	if len(v.Jobs) == 0 {
		contentLines = append(contentLines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Padding(1, 1).Render("No runs recorded yet.\nSelect profile and press [r]."))
	} else {
		for i, j := range v.Jobs {
			statusBadge := lipgloss.NewStyle().Foreground(styles.ColorWarning).Render("● RUN")
			switch j.Status {
			case rclone.JobStatusCompleted:
				statusBadge = lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render("✓ OK")
			case rclone.JobStatusFailed:
				statusBadge = lipgloss.NewStyle().Foreground(styles.ColorDanger).Render("✕ ERR")
			case rclone.JobStatusCancelled:
				statusBadge = lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("⊘ CAN")
			}

			pct := 0
			if j.LatestStats != nil {
				pct = j.LatestStats.Percentage
			} else if j.Status == rclone.JobStatusCompleted {
				pct = 100
			}

			maxNameWidth := v.Width - 14
			if maxNameWidth < 10 {
				maxNameWidth = 10
			}
			displayName := j.Name
			if len(displayName) > maxNameWidth {
				displayName = displayName[:maxNameWidth-3] + "..."
			}

			line := fmt.Sprintf("%s %-20s %3d%%", statusBadge, displayName, pct)

			if i == v.SelectedJob {
				contentLines = append(contentLines, styles.SelectedItemStyle.Width(v.Width-4).Render("▶ "+line))
			} else {
				contentLines = append(contentLines, lipgloss.NewStyle().Foreground(styles.ColorWhite).Render("  "+line))
			}

			// If this is the active selected job, show its mini progress bar & metrics
			if i == v.SelectedJob {
				barWidth := v.Width - 12
				if barWidth < 8 {
					barWidth = 8
				}
				progressBar := styles.RenderProgressBar(barWidth, pct)
				contentLines = append(contentLines, fmt.Sprintf("   %s %3d%%", progressBar, pct))

				if j.LatestStats != nil {
					metricLine := fmt.Sprintf("   %s • %s • ETA %s",
						rclone.FormatBytes(j.LatestStats.Bytes),
						rclone.FormatSpeed(j.LatestStats.Speed),
						rclone.FormatDuration(j.LatestStats.ETA),
					)
					if len(metricLine) > v.Width-4 && v.Width > 8 {
						metricLine = metricLine[:v.Width-7] + "..."
					}
					contentLines = append(contentLines, lipgloss.NewStyle().Foreground(styles.ColorActive).Render(metricLine))
				}
			}
		}
	}

	return panelStyle.
		Width(v.Width).
		Height(v.Height).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, "\n", strings.Join(contentLines, "\n")))
}
