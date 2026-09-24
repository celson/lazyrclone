package components

import (
	"fmt"
	"strings"

	"github.com/celson/lazyrclone/internal/config"
	"github.com/celson/lazyrclone/internal/ui/styles"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

type ModalType int

const (
	ModalNone ModalType = iota
	ModalConfirm
	ModalInput
	ModalProfile
	ModalHelp
)

type ModalManager struct {
	Type          ModalType
	Title         string
	Message       string
	IsDanger      bool
	OnConfirm     func()
	Input         textinput.Model
	ProfileInputs []textinput.Model
	ActiveField   int
	EditingID     string
	Width         int
	Height        int
}

func NewModalManager() *ModalManager {
	ti := textinput.New()
	ti.Focus()

	// 5 fields for Profile modal: Name, Source, Destination, Operation, Flags
	pInputs := make([]textinput.Model, 5)
	pInputs[0] = textinput.New()
	pInputs[0].Placeholder = "e.g., Backup Photos"
	pInputs[0].Prompt = "Name:        "

	pInputs[1] = textinput.New()
	pInputs[1].Placeholder = "e.g., local:~/Photos or /home/user/Photos"
	pInputs[1].Prompt = "Source:      "

	pInputs[2] = textinput.New()
	pInputs[2].Placeholder = "e.g., s3:bucket-backup or gdrive:Photos"
	pInputs[2].Prompt = "Destination: "

	pInputs[3] = textinput.New()
	pInputs[3].Placeholder = "copy / sync / move / bisync / check"
	pInputs[3].Prompt = "Operation:   "

	pInputs[4] = textinput.New()
	pInputs[4].Placeholder = "e.g. --transfers 4, --checkers 8"
	pInputs[4].Prompt = "Flags:       "

	return &ModalManager{
		Type:          ModalNone,
		Input:         ti,
		ProfileInputs: pInputs,
	}
}

func (m *ModalManager) ShowConfirm(title, message string, isDanger bool, onConfirm func()) {
	m.Type = ModalConfirm
	m.Title = title
	m.Message = message
	m.IsDanger = isDanger
	m.OnConfirm = onConfirm
}

func (m *ModalManager) ShowInput(title, placeholder string, onConfirm func()) {
	m.Type = ModalInput
	m.Title = title
	m.Input.SetValue("")
	m.Input.Placeholder = placeholder
	m.Input.Focus()
	m.OnConfirm = onConfirm
}

func (m *ModalManager) ShowProfileEdit(p *config.Profile, onConfirm func()) {
	m.Type = ModalProfile
	m.ActiveField = 0
	if p != nil {
		m.Title = "Edit Sync Profile"
		m.EditingID = p.ID
		m.ProfileInputs[0].SetValue(p.Name)
		m.ProfileInputs[1].SetValue(p.Source)
		m.ProfileInputs[2].SetValue(p.Destination)
		m.ProfileInputs[3].SetValue(string(p.Operation))
		m.ProfileInputs[4].SetValue(strings.Join(p.Flags, ", "))
	} else {
		m.Title = "New Sync Profile"
		m.EditingID = ""
		m.ProfileInputs[0].SetValue("")
		m.ProfileInputs[1].SetValue("")
		m.ProfileInputs[2].SetValue("")
		m.ProfileInputs[3].SetValue("copy")
		m.ProfileInputs[4].SetValue("")
	}
	for i := range m.ProfileInputs {
		if i == 0 {
			m.ProfileInputs[i].Focus()
		} else {
			m.ProfileInputs[i].Blur()
		}
	}
	m.OnConfirm = onConfirm
}

func (m *ModalManager) ShowHelp() {
	m.Type = ModalHelp
	m.Title = "lazyrclone — Keyboard Navigation & Help"
}

func (m *ModalManager) Close() {
	m.Type = ModalNone
	m.OnConfirm = nil
}

func (m *ModalManager) NextField() {
	if m.Type != ModalProfile {
		return
	}
	m.ProfileInputs[m.ActiveField].Blur()
	m.ActiveField = (m.ActiveField + 1) % len(m.ProfileInputs)
	m.ProfileInputs[m.ActiveField].Focus()
}

func (m *ModalManager) PrevField() {
	if m.Type != ModalProfile {
		return
	}
	m.ProfileInputs[m.ActiveField].Blur()
	m.ActiveField--
	if m.ActiveField < 0 {
		m.ActiveField = len(m.ProfileInputs) - 1
	}
	m.ProfileInputs[m.ActiveField].Focus()
}

func (m *ModalManager) GetProfileResult() *config.Profile {
	op := config.OperationType(strings.ToLower(strings.TrimSpace(m.ProfileInputs[3].Value())))
	if op == "" {
		op = config.OpCopy
	}

	flagsRaw := m.ProfileInputs[4].Value()
	var flags []string
	for _, f := range strings.Split(flagsRaw, ",") {
		f = strings.TrimSpace(f)
		if f != "" {
			flags = append(flags, f)
		}
	}

	id := m.EditingID
	if id == "" {
		id = fmt.Sprintf("profile-%s", strings.ToLower(strings.ReplaceAll(m.ProfileInputs[0].Value(), " ", "-")))
	}

	return &config.Profile{
		ID:          id,
		Name:        m.ProfileInputs[0].Value(),
		Source:      m.ProfileInputs[1].Value(),
		Destination: m.ProfileInputs[2].Value(),
		Operation:   op,
		Flags:       flags,
		Transfers:   4,
		Checkers:    8,
	}
}

func (m *ModalManager) Render(screenWidth, screenHeight int) string {
	if m.Type == ModalNone {
		return ""
	}

	var content string
	var boxStyle lipgloss.Style = styles.ModalStyle
	if m.IsDanger {
		boxStyle = styles.ModalDangerStyle
	}

	modalWidth := 64
	if screenWidth < 70 {
		modalWidth = screenWidth - 6
	}

	switch m.Type {
	case ModalConfirm:
		title := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorWarning).Render(m.Title)
		if m.IsDanger {
			title = lipgloss.NewStyle().Bold(true).Foreground(styles.ColorDanger).Render("⚠️  " + m.Title)
		}
		msg := lipgloss.NewStyle().Width(modalWidth - 4).Render(m.Message)
		buttons := lipgloss.NewStyle().MarginTop(1).Render(
			lipgloss.JoinHorizontal(lipgloss.Center,
				lipgloss.NewStyle().Foreground(styles.ColorSuccess).Bold(true).Render("[y] Confirm"),
				"    ",
				lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("[n/Esc] Cancel"),
			),
		)
		content = lipgloss.JoinVertical(lipgloss.Left, title, "\n", msg, buttons)

	case ModalInput:
		title := styles.ModalTitleStyle.Render(m.Title)
		buttons := lipgloss.NewStyle().MarginTop(1).Foreground(styles.ColorMuted).Render("[Enter] Confirm   [Esc] Cancel")
		content = lipgloss.JoinVertical(lipgloss.Left, title, m.Input.View(), buttons)

	case ModalProfile:
		title := styles.ModalTitleStyle.Render(m.Title)
		var fields []string
		for _, in := range m.ProfileInputs {
			fields = append(fields, in.View())
		}
		buttons := lipgloss.NewStyle().MarginTop(1).Foreground(styles.ColorMuted).Render("[Tab/Shift+Tab] Move   [Enter] Save Profile   [Esc] Cancel")
		content = lipgloss.JoinVertical(lipgloss.Left, title, strings.Join(fields, "\n"), buttons)

	case ModalHelp:
		title := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render("📖 lazyrclone Help & Keybindings")
		helpLines := []string{
			"Navigation:",
			"  1, 2, 3         Focus Panel [1] Profiles, [2] Runs, or [3] Main View",
			"  Tab, Shift+Tab  Cycle focus between panels",
			"  [ , ]           Switch Main View sub-tab (Diff | Explorer | Logs)",
			"  j/k, Up/Down    Navigate current panel items or scroll",
			"  q, Ctrl+C       Quit lazyrclone",
			"  ?               Toggle this Help dialog",
			"",
			"Panel [1] Profiles (Tasks):",
			"  d, p            Run honest dry-run diff preview (shows in Main View)",
			"  r, Enter        Execute sync/copy operation",
			"  n               Create new Profile",
			"  e               Edit selected Profile",
			"  x               Delete selected Profile",
			"",
			"Panel [2] Transfers & Runs:",
			"  j/k             Select transfer job",
			"  x               Cancel / Abort running transfer",
			"  c               Clear completed jobs",
			"",
			"Panel [3] Main View (Diff / Explorer / Logs):",
			"  [ , ]           Switch between Diff, Explorer, and Logs",
			"  j/k             Scroll diff items or logs",
			"  ←/→             Switch between Source / Dest in Explorer",
			"  Space           Select item in Explorer",
			"  c               Copy selected file(s) to target pane",
			"  s               Sync directory to target pane",
		}
		helpText := lipgloss.NewStyle().Foreground(styles.ColorWhite).Render(strings.Join(helpLines, "\n"))
		closeBtn := lipgloss.NewStyle().MarginTop(1).Foreground(styles.ColorSecondary).Render("Press [Esc] or [?] or [Enter] to close")
		content = lipgloss.JoinVertical(lipgloss.Left, title, "\n", helpText, closeBtn)
	}

	renderedBox := boxStyle.Width(modalWidth).Render(content)
	return lipgloss.Place(screenWidth, screenHeight, lipgloss.Center, lipgloss.Center, renderedBox)
}
