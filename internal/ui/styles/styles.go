package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// Catppuccin Mocha Official Palette
var (
	// Accents
	MochaRosewater = lipgloss.Color("#f5e0dc")
	MochaFlamingo  = lipgloss.Color("#f2cdcd")
	MochaPink      = lipgloss.Color("#f5c2e7")
	MochaMauve     = lipgloss.Color("#cba6f7")
	MochaRed       = lipgloss.Color("#f38ba8")
	MochaMaroon    = lipgloss.Color("#eba0ac")
	MochaPeach     = lipgloss.Color("#fab387")
	MochaYellow    = lipgloss.Color("#f9e2af")
	MochaGreen     = lipgloss.Color("#a6e3a1")
	MochaTeal      = lipgloss.Color("#94e2d5")
	MochaSky       = lipgloss.Color("#89dceb")
	MochaSapphire  = lipgloss.Color("#74c7ec")
	MochaBlue      = lipgloss.Color("#89b4fa")
	MochaLavender  = lipgloss.Color("#b4befe")

	// Surfaces & Bases
	MochaText     = lipgloss.Color("#cdd6f4")
	MochaSubtext1 = lipgloss.Color("#bac2de")
	MochaSubtext0 = lipgloss.Color("#a6adc8")
	MochaOverlay2 = lipgloss.Color("#9399b2")
	MochaOverlay1 = lipgloss.Color("#7f849c")
	MochaOverlay0 = lipgloss.Color("#6c7086")
	MochaSurface2 = lipgloss.Color("#585b70")
	MochaSurface1 = lipgloss.Color("#45475a")
	MochaSurface0 = lipgloss.Color("#313244")
	MochaBase     = lipgloss.Color("#1e1e2e")
	MochaMantle   = lipgloss.Color("#181825")
	MochaCrust    = lipgloss.Color("#11111b")

	// Semantic UI Colors mapped to Catppuccin Mocha
	ColorPrimary   = MochaMauve    // Primary accent / active tabs / main headers
	ColorSecondary = MochaSapphire // Secondary accent / directory names / commands
	ColorActive    = MochaLavender // Focus indicator / active highlights
	ColorSuccess   = MochaGreen    // Additions (+), online status, success messages
	ColorWarning   = MochaPeach    // Updates (~), warnings, running status
	ColorDanger    = MochaRed      // Deletions (-), errors, destructive actions
	ColorMuted     = MochaOverlay0 // Dim items, inactive borders
	ColorBorder    = MochaSurface1 // Panel borders
	ColorBgDark    = MochaBase     // Background for modals and panels
	ColorSelection = MochaSurface0 // Selected row background
	ColorWhite     = MochaText     // Main foreground text
	ColorSubtext   = MochaSubtext0 // Secondary text

	// Base Panel Styles
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	ActivePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)

	PanelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(MochaSubtext1)

	ActivePanelTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(MochaCrust).
				Background(ColorPrimary).
				Padding(0, 1)

	// Tab Styles
	TabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(MochaCrust).
			Background(ColorPrimary).
			Padding(0, 2)

	TabInactiveStyle = lipgloss.NewStyle().
				Foreground(MochaSubtext0).
				Background(MochaSurface0).
				Padding(0, 2)

	// List item styles
	SelectedItemStyle = lipgloss.NewStyle().
				Background(MochaSurface0).
				Foreground(MochaLavender).
				Bold(true)

	NormalItemStyle = lipgloss.NewStyle().
			Foreground(MochaText)

	MutedItemStyle = lipgloss.NewStyle().
			Foreground(MochaOverlay0)

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
			Foreground(MochaMauve).
			SetString("█")

	ProgressEmpty = lipgloss.NewStyle().
			Foreground(MochaSurface0).
			SetString("░")

	// Status Bar Styles
	StatusBarStyle = lipgloss.NewStyle().
			Background(MochaMantle).
			Foreground(MochaText).
			Padding(0, 1)

	ShortcutKeyStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(MochaMauve)

	ShortcutDescStyle = lipgloss.NewStyle().
				Foreground(MochaSubtext1)

	// Modal Styles
	ModalStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(MochaMauve).
			Background(MochaBase).
			Padding(1, 2)

	ModalDangerStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(MochaRed).
				Background(MochaBase).
				Padding(1, 2)

	ModalTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(MochaMauve).
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
