package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestModalConfirmAligned(t *testing.T) {
	m := NewModalManager()
	m.ShowConfirm(
		"Execute Operation: bisync",
		"Operation 'bisync' target: gdrive:Documents\nDry-run preview: +7920 to add | ~0 to update | -0 to delete\n\nAre you sure you want to proceed?",
		true,
		func() {},
	)

	rendered := m.Render(100, 25)
	for i, line := range strings.Split(rendered, "\n") {
		if strings.TrimSpace(line) != "" {
			w := lipgloss.Width(line)
			if w != 100 {
				t.Errorf("line %d has width %d, expected 100", i, w)
			}
		}
	}
}
