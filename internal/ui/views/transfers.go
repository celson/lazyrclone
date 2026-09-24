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
	LogOffset   int
	Width       int
	Height      int
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
	topHeight := 7
	midHeight := 10
	bottomHeight := v.Height - topHeight - midHeight - 4
	if bottomHeight < 5 {
		bottomHeight = 5
	}

	job := v.ActiveJob()

	// 1. Top Panel: Jobs List
	var jobLines []string
	if len(v.Jobs) == 0 {
		jobLines = append(jobLines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("No active or recent transfers. Start a sync/copy from Profiles or Explorer."))
	} else {
		for i, j := range v.Jobs {
			statusBadge := lipgloss.NewStyle().Foreground(styles.ColorWarning).Render("● RUNNING")
			switch j.Status {
			case rclone.JobStatusCompleted:
				statusBadge = lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render("✓ COMPLETED")
			case rclone.JobStatusFailed:
				statusBadge = lipgloss.NewStyle().Foreground(styles.ColorDanger).Render("✕ FAILED")
			case rclone.JobStatusCancelled:
				statusBadge = lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("⊘ CANCELLED")
			}

			pct := 0
			if j.LatestStats != nil {
				pct = j.LatestStats.Percentage
			} else if j.Status == rclone.JobStatusCompleted {
				pct = 100
			}

			line := fmt.Sprintf("%-24s %-12s %s -> %s (%d%%)", j.Name, statusBadge, j.Source, j.Destination, pct)
			if i == v.SelectedJob {
				jobLines = append(jobLines, styles.SelectedItemStyle.Width(v.Width-6).Render("▶ "+line))
			} else {
				jobLines = append(jobLines, styles.NormalItemStyle.Render("  "+line))
			}
		}
	}

	topPanel := styles.ActivePanelStyle.
		Width(v.Width - 4).
		Height(topHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left, styles.ActivePanelTitleStyle.Render(" Transfer Jobs "), "\n", strings.Join(jobLines, "\n")))

	// 2. Middle Panel: Live Progress & Active Files
	var progressContent string
	if job == nil {
		progressContent = lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Select a job above to view real-time metrics and file stream.")
	} else {
		stats := job.LatestStats
		pct := 0
		bytesText := "-- / --"
		speedText := "--/s"
		etaText := "--:--"

		if stats != nil {
			pct = stats.Percentage
			bytesText = fmt.Sprintf("%s / %s", rclone.FormatBytes(stats.Bytes), rclone.FormatBytes(stats.TotalBytes))
			speedText = rclone.FormatSpeed(stats.Speed)
			etaText = rclone.FormatDuration(stats.ETA)
		} else if job.Status == rclone.JobStatusCompleted {
			pct = 100
		}

		barWidth := v.Width - 24
		if barWidth < 10 {
			barWidth = 10
		}
		progressBar := styles.RenderProgressBar(barWidth, pct)
		pctStr := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render(fmt.Sprintf("%3d%%", pct))
		progressRow := lipgloss.JoinHorizontal(lipgloss.Center, progressBar, "  ", pctStr)

		metricsRow := fmt.Sprintf("Transferred: %s   Speed: %s   ETA: %s   Status: %s",
			lipgloss.NewStyle().Bold(true).Foreground(styles.ColorWhite).Render(bytesText),
			lipgloss.NewStyle().Bold(true).Foreground(styles.ColorSuccess).Render(speedText),
			lipgloss.NewStyle().Bold(true).Foreground(styles.ColorSecondary).Render(etaText),
			lipgloss.NewStyle().Bold(true).Foreground(styles.ColorWarning).Render(string(job.Status)),
		)

		var activeFilesLines []string
		if stats != nil && len(stats.Transferring) > 0 {
			for _, af := range stats.Transferring {
				afBar := styles.RenderProgressBar(15, af.Percentage)
				line := fmt.Sprintf("  • %-32s %s %3d%%  (%s)", af.Name, afBar, af.Percentage, rclone.FormatSpeed(af.Speed))
				activeFilesLines = append(activeFilesLines, lipgloss.NewStyle().Foreground(styles.ColorWhite).Render(line))
			}
		}

		parts := []string{progressRow, metricsRow}
		if len(activeFilesLines) > 0 {
			parts = append(parts, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Active Transfers:"), strings.Join(activeFilesLines, "\n"))
		}
		progressContent = strings.Join(parts, "\n")
	}

	midPanel := styles.PanelStyle.
		Width(v.Width - 4).
		Height(midHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left, styles.PanelTitleStyle.Render(" Real-time Transfer Metrics "), "\n", progressContent))

	// 3. Bottom Panel: Job Terminal Logs
	var logsContent string
	if job == nil || len(job.Logs) == 0 {
		logsContent = lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("No log output.")
	} else {
		visibleLogs := bottomHeight - 4
		if visibleLogs < 1 {
			visibleLogs = 1
		}
		start := len(job.Logs) - visibleLogs
		if start < 0 {
			start = 0
		}
		logsSlice := job.Logs[start:]
		logsContent = strings.Join(logsSlice, "\n")
	}

	bottomPanel := styles.PanelStyle.
		Width(v.Width - 4).
		Height(bottomHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left, styles.PanelTitleStyle.Render(" Operation Logs "), "\n", logsContent))

	return lipgloss.JoinVertical(lipgloss.Left, topPanel, midPanel, bottomPanel)
}
