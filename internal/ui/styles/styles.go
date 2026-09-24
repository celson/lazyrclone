package styles

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	ColorPrimary   = lipgloss.Color("#7D56F4")
	ColorSecondary = lipgloss.Color("#00D7D7")
	ColorActive    = lipgloss.Color("#88C0D0")
	ColorSuccess   = lipgloss.Color("#50FA7B")
	ColorWarning   = lipgloss.Color("#FFB86C")
	ColorDanger    = lipgloss.Color("#FF5555")
	ColorMuted     = lipgloss.Color("#6272A4")
	ColorBgDark    = lipgloss.Color("#1E1E2E")
	ColorSelection = lipgloss.Color("#3B4252")
	ColorWhite     = lipgloss.Color("#F8F8F2")

	// Base Panel Styles
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted).
			Padding(0, 1)

	ActivePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)

	PanelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	ActivePanelTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorWhite).
				Background(ColorPrimary).
				Padding(0, 1)

	// Tab Styles
	TabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite).
			Background(ColorPrimary).
			Padding(0, 2)

	TabInactiveStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Background(ColorSelection).
				Padding(0, 2)

	// List item styles
	SelectedItemStyle = lipgloss.NewStyle().
				Background(ColorSelection).
				Foreground(ColorWhite).
				Bold(true)

	NormalItemStyle = lipgloss.NewStyle().
			Foreground(ColorWhite)

	MutedItemStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Status & Action Badges
	BadgeAdd = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSuccess).
			SetString("+ ADD")

	BadgeUpdate = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWarning).
			SetString("~ UPDATE")

	BadgeDelete = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorDanger).
			SetString("- DELETE")

	BadgeEqual = lipgloss.NewStyle().
			Foreground(ColorMuted).
			SetString("= IDENTICAL")

	// Progress Styles
	ProgressFilled = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			SetString("█")

	ProgressEmpty = lipgloss.NewStyle().
			Foreground(ColorSelection).
			SetString("░")

	// Status Bar Styles
	StatusBarStyle = lipgloss.NewStyle().
			Background(ColorSelection).
			Foreground(ColorWhite).
			Padding(0, 1)

	ShortcutKeyStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorSecondary)

	ShortcutDescStyle = lipgloss.NewStyle().
				Foreground(ColorWhite)

	// Modal Styles
	ModalStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(ColorPrimary).
			Background(ColorBgDark).
			Padding(1, 2)

	ModalDangerStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(ColorDanger).
				Background(ColorBgDark).
				Padding(1, 2)

	ModalTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)
)

func RenderProgressBar(width int, percentage int) string {
	if width < 5 {
		width = 10
	}
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 100 {
		percentage = 100
	}

	filledLen := (width * percentage) / 100
	emptyLen := width - filledLen

	bar := ""
	for i := 0; i < filledLen; i++ {
		bar += ProgressFilled.String()
	}
	for i := 0; i < emptyLen; i++ {
		bar += ProgressEmpty.String()
	}
	return bar
}
