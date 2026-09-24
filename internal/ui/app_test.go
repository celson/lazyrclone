package ui

import (
	"strings"
	"testing"

	"github.com/celson/lazyrclone/internal/config"
	"github.com/celson/lazyrclone/internal/rclone"
	tea "github.com/charmbracelet/bubbletea"
)

func TestAppModelBorderEnclosure(t *testing.T) {
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
	if !strings.Contains(topLine, "[3] Main View") {
		t.Errorf("expected '[3] Main View' embedded in top line, got: %s", topLine)
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
}
