package ui

import (
	"strings"
	"testing"

	"github.com/celson/lazyrclone/internal/config"
	"github.com/celson/lazyrclone/internal/rclone"
	tea "github.com/charmbracelet/bubbletea"
)

func TestAppModelBorderEnclosure(t *testing.T) {
	t.Setenv("LAZYRCLONE_CONFIG_DIR", t.TempDir())
	cfg := config.DefaultConfig()
	client := rclone.NewMockClient()
	app := NewAppModel(cfg, client)

	// Simulate window size 100x30
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	view := app.View()
	lines := strings.Split(view, "\n")

	if len(lines) != 30 {
		t.Fatalf("expected view to have exactly 30 lines (terminal height), got %d lines", len(lines))
	}

	// Check line 0 (top line of panels): must contain top-left corner '╭' and top-right corner '╮'
	topLine := lines[0]
	if !strings.Contains(topLine, "╭") || !strings.Contains(topLine, "╮") {
		t.Errorf("top line does not have closed corners: %s", topLine)
	}

	// Check panel titles are in top line
	if !strings.Contains(topLine, "[1] Profiles") {
		t.Errorf("expected '[1] Profiles' embedded in top line, got: %s", topLine)
	}
	if !strings.Contains(topLine, "[4] Main View") {
		t.Errorf("expected '[4] Main View' embedded in top line, got: %s", topLine)
	}

	// Check line 28 (bottom of panels): must contain bottom corners '╰' and '╯'
	bottomPanelsLine := lines[28]
	if !strings.Contains(bottomPanelsLine, "╰") || !strings.Contains(bottomPanelsLine, "╯") {
		t.Errorf("bottom line of panels does not have closed corners: %s", bottomPanelsLine)
	}

	// Check line 29 (bottom status bar)
	statusBarLine := lines[29]
	if !strings.Contains(statusBarLine, "lazyrclone v0.1.0") {
		t.Errorf("expected status bar with version, got: %s", statusBarLine)
	}

	// Test SubTabRemotes rendering and border enclosure
	app.MainSubTab = SubTabRemotes
	remotesView := app.View()
	remotesLines := strings.Split(remotesView, "\n")
	if len(remotesLines) != 30 {
		t.Fatalf("expected SubTabRemotes view to have exactly 30 lines, got %d", len(remotesLines))
	}
	if !strings.Contains(remotesLines[0], "Remote") {
		t.Errorf("expected 'Remote' tab in top line, got: %s", remotesLines[0])
	}
}

func TestPanelNavigation(t *testing.T) {
	t.Setenv("LAZYRCLONE_CONFIG_DIR", t.TempDir())
	cfg := config.DefaultConfig()
	client := rclone.NewMockClient()
	app := NewAppModel(cfg, client)

	// Press 2 -> PanelRemotes
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if app.FocusedPanel != 1 {
		t.Fatalf("expected FocusedPanel to be 1 (PanelRemotes), got %d", app.FocusedPanel)
	}
	if app.MainSubTab != SubTabRemotes {
		t.Fatalf("expected MainSubTab to be SubTabRemotes, got %d", app.MainSubTab)
	}

	// Press 1 -> PanelProfiles
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	if app.FocusedPanel != 0 {
		t.Fatalf("expected FocusedPanel to be 0 (PanelProfiles), got %d", app.FocusedPanel)
	}
	if app.MainSubTab != SubTabDiff {
		t.Fatalf("expected MainSubTab to be SubTabDiff, got %d", app.MainSubTab)
	}

	// Press 4 -> PanelMain
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	if app.FocusedPanel != 3 {
		t.Fatalf("expected FocusedPanel to be 3 (PanelMain), got %d", app.FocusedPanel)
	}

	// Press Esc -> Returns to LastSidePanel (0)
	app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if app.FocusedPanel != 0 {
		t.Fatalf("expected FocusedPanel to return to 0 (PanelProfiles), got %d", app.FocusedPanel)
	}

	// Press ] -> next tab (SubTabExplorer)
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	if app.MainSubTab != SubTabExplorer {
		t.Fatalf("expected MainSubTab to be SubTabExplorer, got %d", app.MainSubTab)
	}
}

func TestStopExecution(t *testing.T) {
	t.Setenv("LAZYRCLONE_CONFIG_DIR", t.TempDir())
	cfg := config.DefaultConfig()
	client := rclone.NewMockClient()
	app := NewAppModel(cfg, client)

	// Add a running job for the first profile
	p := cfg.Profiles[0]
	cancelled := false
	job := &rclone.TransferJob{
		ID:        "job-test-1",
		ProfileID: p.ID,
		Name:      p.Name,
		Status:    rclone.JobStatusRunning,
		CancelFunc: func() {
			cancelled = true
		},
	}
	app.TransfersView.AddJob(job)

	// 1. In Profiles panel, press 's'
	app.FocusedPanel = 0 // PanelProfiles
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	if !cancelled {
		t.Errorf("expected CancelFunc to be called when pressing 's' in Profiles")
	}
	st, _ := job.GetStatus()
	if st != rclone.JobStatusCancelled {
		t.Errorf("expected job status to be JobStatusCancelled, got %v", st)
	}
	if p.LastStatus != "cancelled" {
		t.Errorf("expected profile LastStatus to be 'cancelled', got %s", p.LastStatus)
	}

	// 2. In Transfers panel, press 's' on a running job
	cancelled2 := false
	job2 := &rclone.TransferJob{
		ID:     "job-test-2",
		Name:   "Transfer 2",
		Status: rclone.JobStatusRunning,
		CancelFunc: func() {
			cancelled2 = true
		},
	}
	app.TransfersView.AddJob(job2)
	app.FocusedPanel = 2 // PanelRuns
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	if !cancelled2 {
		t.Errorf("expected CancelFunc to be called when pressing 's' in Transfers")
	}
	st2, _ := job2.GetStatus()
	if st2 != rclone.JobStatusCancelled {
		t.Errorf("expected job2 status to be JobStatusCancelled, got %v", st2)
	}

	// 3. In Live Logs (MainSubTab = SubTabLogs), press 's'
	cancelled3 := false
	job3 := &rclone.TransferJob{
		ID:     "job-test-3",
		Name:   "Transfer 3",
		Status: rclone.JobStatusRunning,
		CancelFunc: func() {
			cancelled3 = true
		},
	}
	app.TransfersView.AddJob(job3)
	app.FocusedPanel = 3 // PanelMain
	app.MainSubTab = SubTabLogs
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	if !cancelled3 {
		t.Errorf("expected CancelFunc to be called when pressing 's' in Live Logs")
	}
	st3, _ := job3.GetStatus()
	if st3 != rclone.JobStatusCancelled {
		t.Errorf("expected job3 status to be JobStatusCancelled, got %v", st3)
	}

	// 4. In Diff Preview (SubTabDiff), press 's' with DryRunCancel active
	dryRunCancelled := false
	app.DryRunCancel = func() {
		dryRunCancelled = true
	}
	app.MainSubTab = SubTabDiff
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	if !dryRunCancelled {
		t.Errorf("expected DryRunCancel to be called when pressing 's' in Diff View")
	}
	if app.DryRunCancel != nil {
		t.Errorf("expected DryRunCancel to be cleared")
	}
}
