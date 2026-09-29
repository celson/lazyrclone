package views

import (
	"fmt"

	"github.com/celson/lazyrclone/internal/rclone"
	"github.com/celson/lazyrclone/internal/ui/components"
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
		st, _ := j.GetStatus()
		if st == rclone.JobStatusRunning {
			active = append(active, j)
		}
	}
	v.Jobs = active
	v.SelectedJob = 0
}

func (v *TransfersView) Render() string {
	innerWidth := v.Width - 2
	if innerWidth < 5 {
		innerWidth = 5
	}

	var contentLines []string

	if len(v.Jobs) == 0 {
		contentLines = append(contentLines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(" No runs recorded yet."))
		contentLines = append(contentLines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(" Select profile and press [r]."))
	} else {
		for i, j := range v.Jobs {
			st, jobErr := j.GetStatus()
			latestStats := j.GetStats()

			statusBadge := lipgloss.NewStyle().Foreground(styles.ColorWarning).Render("● RUN")
			switch st {
			case rclone.JobStatusCompleted:
				statusBadge = lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render("✓ OK")
			case rclone.JobStatusFailed:
				statusBadge = lipgloss.NewStyle().Foreground(styles.ColorDanger).Render("✕ ERR")
			case rclone.JobStatusCancelled:
				statusBadge = lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("⊘ CAN")
			}

			pct := 0
			if st == rclone.JobStatusCompleted {
				pct = 100
			} else if latestStats != nil {
				if latestStats.Percentage > 0 {
					pct = latestStats.Percentage
				} else if latestStats.TotalBytes > 0 {
					pct = int((float64(latestStats.Bytes) / float64(latestStats.TotalBytes)) * 100)
				} else if latestStats.TotalTransfers > 0 {
					pct = int((float64(latestStats.Transfers) / float64(latestStats.TotalTransfers)) * 100)
				}
				if pct > 100 {
					pct = 100
				}
			}

			maxNameWidth := innerWidth - 14
			if maxNameWidth < 6 {
				maxNameWidth = 6
			}
			displayName := j.Name
			if len(displayName) > maxNameWidth {
				displayName = displayName[:maxNameWidth-3] + "..."
			}

			line := fmt.Sprintf("%s %-*s %3d%%", statusBadge, maxNameWidth, displayName, pct)

			if i == v.SelectedJob {
				contentLines = append(contentLines, styles.SelectedItemStyle.Width(innerWidth).Render("▶ "+line))
			} else {
				contentLines = append(contentLines, lipgloss.NewStyle().Foreground(styles.ColorWhite).Render("  "+line))
			}

			// If this is the active selected job, show its mini progress bar & metrics
			if i == v.SelectedJob {
				barWidth := innerWidth - 10
				if barWidth < 6 {
					barWidth = 6
				}
				progressBar := styles.RenderProgressBar(barWidth, pct)
				contentLines = append(contentLines, fmt.Sprintf("   %s %3d%%", progressBar, pct))

				if st == rclone.JobStatusFailed {
					errMsg := "Failed"
					if jobErr != "" {
						errMsg = "Failed: " + jobErr
					}
					if len(errMsg) > innerWidth-2 && innerWidth > 6 {
						errMsg = errMsg[:innerWidth-5] + "..."
					}
					contentLines = append(contentLines, lipgloss.NewStyle().Foreground(styles.ColorDanger).Render("   "+errMsg))
				} else if latestStats != nil {
					etaOrDone := "ETA " + rclone.FormatDuration(latestStats.ETA)
					if st == rclone.JobStatusCompleted {
						etaOrDone = "Done"
					}
					metricLine := fmt.Sprintf("   %s • %s • %s",
						rclone.FormatBytes(latestStats.Bytes),
						rclone.FormatSpeed(latestStats.Speed),
						etaOrDone,
					)
					if len(metricLine) > innerWidth-2 && innerWidth > 6 {
						metricLine = metricLine[:innerWidth-5] + "..."
					}
					contentLines = append(contentLines, lipgloss.NewStyle().Foreground(styles.ColorActive).Render(metricLine))
				} else if st == rclone.JobStatusCompleted {
					contentLines = append(contentLines, lipgloss.NewStyle().Foreground(styles.ColorActive).Render("   Completed • Done"))
				}
			}
		}
	}

	return components.RenderPanelBox(v.Width, v.Height, "[3] Transfers", nil, v.IsActive, contentLines)
}
