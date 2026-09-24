package ui

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/celson/lazyrclone/internal/config"
	"github.com/celson/lazyrclone/internal/rclone"
	"github.com/celson/lazyrclone/internal/ui/components"
	"github.com/celson/lazyrclone/internal/ui/views"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DryRunFinishedMsg struct {
	Result *rclone.DryRunResult
}

type TransferUpdateMsg struct {
	JobID string
	Stats *rclone.StatsMsg
	Log   string
	Done  bool
	Err   error
}

type ClearStatusMsg struct{}

func clearStatusCmd() tea.Cmd {
	return tea.Tick(4*time.Second, func(time.Time) tea.Msg {
		return ClearStatusMsg{}
	})
}

type AppModel struct {
	Config        *config.Config
	Client        rclone.RcloneClient
	CurrentTab    components.Tab
	ProfilesView  *views.ProfilesView
	ExplorerView  *views.ExplorerView
	TransfersView *views.TransfersView
	RemotesView   *views.RemotesView
	Modal         *components.ModalManager
	Width         int
	Height        int
	StatusMsg     string
	IsStatusErr   bool
	RcloneVer     string
	IsMock        bool
}

func NewAppModel(cfg *config.Config, client rclone.RcloneClient) *AppModel {
	ver, _ := client.Version()

	pView := views.NewProfilesView(cfg)
	eView := views.NewExplorerView(client)
	tView := views.NewTransfersView()
	rView := views.NewRemotesView(client)
	modal := components.NewModalManager()

	initialTab := components.TabProfiles
	switch cfg.Settings.DefaultView {
	case "explorer":
		initialTab = components.TabExplorer
	case "transfers":
		initialTab = components.TabTransfers
	case "remotes":
		initialTab = components.TabRemotes
	}

	app := &AppModel{
		Config:        cfg,
		Client:        client,
		CurrentTab:    initialTab,
		ProfilesView:  pView,
		ExplorerView:  eView,
		TransfersView: tView,
		RemotesView:   rView,
		Modal:         modal,
		RcloneVer:     ver,
		IsMock:        client.IsMock(),
	}

	return app
}

func (m *AppModel) Init() tea.Cmd {
	// Initialize remotes and explorer panes
	m.RemotesView.Refresh()
	m.ExplorerView.LoadPane(0)
	m.ExplorerView.LoadPane(1)
	return nil
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

		contentHeight := msg.Height - 3
		m.ProfilesView.SetSize(msg.Width, contentHeight)
		m.ExplorerView.SetSize(msg.Width, contentHeight)
		m.TransfersView.SetSize(msg.Width, contentHeight)
		m.RemotesView.SetSize(msg.Width, contentHeight)
		return m, nil

	case DryRunFinishedMsg:
		m.ProfilesView.SetDryRunResult(msg.Result)
		if msg.Result.Err != nil {
			m.StatusMsg = fmt.Sprintf("Dry-run error: %v", msg.Result.Err)
			m.IsStatusErr = true
		} else {
			m.StatusMsg = fmt.Sprintf("Dry-run complete: +%d to add, ~%d to update, -%d to delete",
				msg.Result.ToAdd, msg.Result.ToUpdate, msg.Result.ToDelete)
			m.IsStatusErr = false
		}
		return m, clearStatusCmd()

	case TransferUpdateMsg:
		for _, j := range m.TransfersView.Jobs {
			if j.ID == msg.JobID {
				if msg.Stats != nil {
					j.LatestStats = msg.Stats
				}
				if msg.Log != "" {
					j.Logs = append(j.Logs, msg.Log)
				}
				if msg.Done {
					now := time.Now()
					j.EndTime = &now
					if msg.Err != nil {
						j.Status = rclone.JobStatusFailed
						j.ErrorMsg = msg.Err.Error()
					} else {
						j.Status = rclone.JobStatusCompleted
					}
				}
				break
			}
		}
		return m, nil

	case ClearStatusMsg:
		m.StatusMsg = ""
		return m, nil

	case tea.KeyMsg:
		// 1. Handle active modal dialogs first
		if m.Modal.Type != components.ModalNone {
			return m.handleModalKey(msg)
		}

		// 2. Global Hotkeys
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "1":
			m.CurrentTab = components.TabProfiles
			return m, nil
		case "2":
			m.CurrentTab = components.TabExplorer
			return m, nil
		case "3":
			m.CurrentTab = components.TabTransfers
			return m, nil
		case "4":
			m.CurrentTab = components.TabRemotes
			return m, nil
		case "?":
			m.Modal.ShowHelp()
			return m, nil
		}

		// 3. Tab-specific keybindings
		switch m.CurrentTab {
		case components.TabProfiles:
			return m.handleProfilesKey(msg)
		case components.TabExplorer:
			return m.handleExplorerKey(msg)
		case components.TabTransfers:
			return m.handleTransfersKey(msg)
		case components.TabRemotes:
			return m.handleRemotesKey(msg)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *AppModel) handleModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch m.Modal.Type {
	case components.ModalConfirm:
		switch key {
		case "y", "Y", "enter":
			if m.Modal.OnConfirm != nil {
				m.Modal.OnConfirm()
			}
			m.Modal.Close()
		case "n", "N", "esc", "q":
			m.Modal.Close()
		}
		return m, nil

	case components.ModalHelp:
		switch key {
		case "esc", "q", "enter", "?":
			m.Modal.Close()
		}
		return m, nil

	case components.ModalInput:
		switch key {
		case "enter":
			if m.Modal.OnConfirm != nil {
				m.Modal.OnConfirm()
			}
			m.Modal.Close()
			return m, nil
		case "esc":
			m.Modal.Close()
			return m, nil
		default:
			var cmd tea.Cmd
			m.Modal.Input, cmd = m.Modal.Input.Update(msg)
			return m, cmd
		}

	case components.ModalProfile:
		switch key {
		case "tab", "down":
			m.Modal.NextField()
			return m, nil
		case "shift+tab", "up":
			m.Modal.PrevField()
			return m, nil
		case "enter":
			if m.Modal.ActiveField == len(m.Modal.ProfileInputs)-1 {
				if m.Modal.OnConfirm != nil {
					m.Modal.OnConfirm()
				}
				m.Modal.Close()
				return m, nil
			}
			m.Modal.NextField()
			return m, nil
		case "ctrl+s":
			if m.Modal.OnConfirm != nil {
				m.Modal.OnConfirm()
			}
			m.Modal.Close()
			return m, nil
		case "esc":
			m.Modal.Close()
			return m, nil
		default:
			var cmd tea.Cmd
			m.Modal.ProfileInputs[m.Modal.ActiveField], cmd = m.Modal.ProfileInputs[m.Modal.ActiveField].Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m *AppModel) handleProfilesKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.ProfilesView.MoveUp()
	case "down", "j":
		m.ProfilesView.MoveDown()
	case "f":
		m.ProfilesView.ToggleDiffFocus()
	case "tab":
		m.CurrentTab = components.TabExplorer
	case "d":
		// Honest Dry Run
		p := m.ProfilesView.SelectedProfile()
		if p == nil {
			return m, nil
		}
		m.StatusMsg = fmt.Sprintf("Calculating dry-run diff for '%s'...", p.Name)
		return m, m.runDryRunCmd(p)
	case "enter", "r":
		// Execute Profile
		p := m.ProfilesView.SelectedProfile()
		if p == nil {
			return m, nil
		}
		if m.Config.Settings.ConfirmDestructive && (p.Operation == config.OpSync || p.Operation == config.OpMove) {
			m.Modal.ShowConfirm(
				"Execute Operation: "+string(p.Operation),
				fmt.Sprintf("Warning: Operation '%s' may modify or delete files at %s.\nAre you sure you want to proceed?", p.Operation, p.Destination),
				true,
				func() {
					m.startProfileJob(p)
				},
			)
			return m, nil
		}
		m.startProfileJob(p)
		m.CurrentTab = components.TabTransfers
	case "n":
		// New Profile
		m.Modal.ShowProfileEdit(nil, func() {
			newP := m.Modal.GetProfileResult()
			m.Config.Profiles = append(m.Config.Profiles, newP)
			_ = config.SaveConfig(m.Config)
			m.StatusMsg = fmt.Sprintf("Profile '%s' created.", newP.Name)
		})
	case "e":
		// Edit Profile
		p := m.ProfilesView.SelectedProfile()
		if p == nil {
			return m, nil
		}
		m.Modal.ShowProfileEdit(p, func() {
			updated := m.Modal.GetProfileResult()
			p.Name = updated.Name
			p.Source = updated.Source
			p.Destination = updated.Destination
			p.Operation = updated.Operation
			p.Flags = updated.Flags
			_ = config.SaveConfig(m.Config)
			m.StatusMsg = fmt.Sprintf("Profile '%s' updated.", p.Name)
		})
	case "x":
		// Delete Profile
		p := m.ProfilesView.SelectedProfile()
		if p == nil {
			return m, nil
		}
		m.Modal.ShowConfirm(
			"Delete Profile",
			fmt.Sprintf("Are you sure you want to delete profile '%s'?", p.Name),
			true,
			func() {
				idx := m.ProfilesView.SelectedIdx
				m.Config.Profiles = append(m.Config.Profiles[:idx], m.Config.Profiles[idx+1:]...)
				if m.ProfilesView.SelectedIdx >= len(m.Config.Profiles) && m.ProfilesView.SelectedIdx > 0 {
					m.ProfilesView.SelectedIdx--
				}
				_ = config.SaveConfig(m.Config)
				m.StatusMsg = "Profile deleted."
			},
		)
	}
	return m, nil
}

func (m *AppModel) handleExplorerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "h", "l", "left", "right":
		m.ExplorerView.SwitchPane()
	case "up", "k":
		m.ExplorerView.MoveUp()
	case "down", "j":
		m.ExplorerView.MoveDown()
	case "enter":
		m.ExplorerView.EnterDir()
	case "backspace", "-":
		m.ExplorerView.GoUp()
	case " ":
		m.ExplorerView.ToggleSelect()
	case "R":
		m.ExplorerView.LoadPane(m.ExplorerView.ActivePane)
	case "r":
		// Change remote of active pane
		activeIdx := m.ExplorerView.ActivePane
		currRemote := m.ExplorerView.Panes[activeIdx].Remote
		m.Modal.ShowInput("Change Remote (e.g. gdrive:, s3:, local:)", currRemote, func() {
			val := strings.TrimSpace(m.Modal.Input.Value())
			if val != "" {
				if !strings.HasSuffix(val, ":") && !strings.HasPrefix(val, "/") && !strings.HasPrefix(val, ".") {
					val = val + ":"
				}
				m.ExplorerView.SetPaneRemote(activeIdx, val)
				m.StatusMsg = fmt.Sprintf("Pane remote switched to %s", val)
			}
		})
	case "c":
		// Copy selected or hovered items to opposite pane
		items := m.ExplorerView.GetSelectedOrMarkedItems()
		if len(items) == 0 {
			m.StatusMsg = "No item selected to copy."
			m.IsStatusErr = true
			return m, clearStatusCmd()
		}
		srcPane := m.ExplorerView.CurrentActivePane()
		dstPane := m.ExplorerView.InactivePane()

		srcPath := path.Join(srcPane.Remote, srcPane.CurrentDir, items[0].Name)
		dstPath := path.Join(dstPane.Remote, dstPane.CurrentDir)

		job := &rclone.TransferJob{
			ID:          fmt.Sprintf("copy-%d", time.Now().UnixNano()),
			Name:        fmt.Sprintf("Copy %s -> %s", items[0].Name, dstPane.Remote),
			Source:      srcPath,
			Destination: dstPath,
			Operation:   "copy",
			StartTime:   time.Now(),
			Status:      rclone.JobStatusRunning,
			Logs:        make([]string, 0),
		}
		m.TransfersView.AddJob(job)
		m.CurrentTab = components.TabTransfers
		m.StatusMsg = fmt.Sprintf("Started copy of %d item(s)", len(items))
		return m, m.startTransferCmd(job)

	case "s":
		// Sync current active folder to destination pane
		srcPane := m.ExplorerView.CurrentActivePane()
		dstPane := m.ExplorerView.InactivePane()
		srcPath := path.Join(srcPane.Remote, srcPane.CurrentDir)
		dstPath := path.Join(dstPane.Remote, dstPane.CurrentDir)

		m.Modal.ShowConfirm(
			"Sync Folder to Target Pane",
			fmt.Sprintf("Sync from %s to %s?\nFiles in destination not in source might be deleted!", srcPath, dstPath),
			true,
			func() {
				job := &rclone.TransferJob{
					ID:          fmt.Sprintf("sync-%d", time.Now().UnixNano()),
					Name:        fmt.Sprintf("Sync %s -> %s", srcPane.Remote, dstPane.Remote),
					Source:      srcPath,
					Destination: dstPath,
					Operation:   "sync",
					StartTime:   time.Now(),
					Status:      rclone.JobStatusRunning,
					Logs:        make([]string, 0),
				}
				m.TransfersView.AddJob(job)
				m.CurrentTab = components.TabTransfers
			},
		)

	case "n":
		// Mkdir
		activePane := m.ExplorerView.CurrentActivePane()
		m.Modal.ShowInput("Create New Directory", "folder_name", func() {
			val := strings.TrimSpace(m.Modal.Input.Value())
			if val != "" {
				full := path.Join(activePane.Remote, activePane.CurrentDir, val)
				_ = m.Client.CreateDir(context.Background(), full)
				activePane.Items = append(activePane.Items, rclone.FileItem{
					Name:    val,
					Path:    val,
					IsDir:   true,
					ModTime: time.Now(),
				})
				m.StatusMsg = fmt.Sprintf("Directory '%s' created.", val)
			}
		})

	case "x":
		// Delete selected
		items := m.ExplorerView.GetSelectedOrMarkedItems()
		if len(items) == 0 {
			return m, nil
		}
		activePane := m.ExplorerView.CurrentActivePane()
		targetName := items[0].Name
		m.Modal.ShowConfirm(
			"Delete File / Folder",
			fmt.Sprintf("Are you sure you want to permanently delete '%s'?", targetName),
			true,
			func() {
				full := path.Join(activePane.Remote, activePane.CurrentDir, targetName)
				_ = m.Client.Delete(context.Background(), full, items[0].IsDir)
				m.ExplorerView.LoadPane(m.ExplorerView.ActivePane)
				m.StatusMsg = fmt.Sprintf("Deleted '%s'.", targetName)
			},
		)
	}
	return m, nil
}

func (m *AppModel) handleTransfersKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.TransfersView.MoveUp()
	case "down", "j":
		m.TransfersView.MoveDown()
	case "c":
		m.TransfersView.ClearCompleted()
		m.StatusMsg = "Cleared completed transfers."
	case "x", "X":
		job := m.TransfersView.ActiveJob()
		if job != nil && job.Status == rclone.JobStatusRunning {
			if job.CancelFunc != nil {
				job.CancelFunc()
			}
			job.Status = rclone.JobStatusCancelled
			m.StatusMsg = "Job cancelled."
		}
	}
	return m, nil
}

func (m *AppModel) handleRemotesKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.RemotesView.MoveUp()
	case "down", "j":
		m.RemotesView.MoveDown()
	case "t":
		m.RemotesView.TestConnection()
		m.StatusMsg = "Testing remote connection..."
	case "r":
		m.RemotesView.Refresh()
		m.StatusMsg = "Remotes refreshed."
	}
	return m, nil
}

func (m *AppModel) runDryRunCmd(p *config.Profile) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		res, err := m.Client.DryRun(ctx, string(p.Operation), p.Source, p.Destination, p.Flags)
		if err != nil {
			return DryRunFinishedMsg{Result: &rclone.DryRunResult{Err: err}}
		}
		return DryRunFinishedMsg{Result: res}
	}
}

func (m *AppModel) startProfileJob(p *config.Profile) {
	ctx, cancel := context.WithCancel(context.Background())
	job := &rclone.TransferJob{
		ID:          fmt.Sprintf("job-%d", time.Now().UnixNano()),
		ProfileID:   p.ID,
		Name:        p.Name,
		Source:      p.Source,
		Destination: p.Destination,
		Operation:   string(p.Operation),
		StartTime:   time.Now(),
		Status:      rclone.JobStatusRunning,
		Logs:        make([]string, 0),
		CancelFunc:  cancel,
	}
	m.TransfersView.AddJob(job)

	// Update profile last run status
	now := time.Now()
	p.LastRun = &now
	p.LastStatus = "running"
	_ = config.SaveConfig(m.Config)

	go func() {
		err := m.Client.StartTransfer(ctx, job, func(stats *rclone.StatsMsg) {
			job.LatestStats = stats
		}, func(log string) {
			job.Logs = append(job.Logs, log)
		})

		endTime := time.Now()
		job.EndTime = &endTime
		if err != nil {
			job.Status = rclone.JobStatusFailed
			job.ErrorMsg = err.Error()
			p.LastStatus = "failed"
		} else {
			job.Status = rclone.JobStatusCompleted
			p.LastStatus = "success"
		}
		_ = config.SaveConfig(m.Config)
	}()
}

func (m *AppModel) startTransferCmd(job *rclone.TransferJob) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	job.CancelFunc = cancel

	return func() tea.Msg {
		err := m.Client.StartTransfer(ctx, job, func(stats *rclone.StatsMsg) {
			job.LatestStats = stats
		}, func(log string) {
			job.Logs = append(job.Logs, log)
		})

		return TransferUpdateMsg{
			JobID: job.ID,
			Done:  true,
			Err:   err,
		}
	}
}

func (m *AppModel) View() string {
	if m.Width == 0 || m.Height == 0 {
		return "Initializing lazyrclone..."
	}

	header := components.RenderHeader(m.Width, m.CurrentTab, m.RcloneVer, m.IsMock)

	var content string
	var shortcuts []components.Shortcut

	switch m.CurrentTab {
	case components.TabProfiles:
		content = m.ProfilesView.Render()
		shortcuts = []components.Shortcut{
			{Key: "d", Desc: "Dry-Run"},
			{Key: "r/Enter", Desc: "Run"},
			{Key: "n", Desc: "New"},
			{Key: "e", Desc: "Edit"},
			{Key: "x", Desc: "Delete"},
			{Key: "f", Desc: "Focus Diff"},
			{Key: "Tab", Desc: "Next Tab"},
			{Key: "?", Desc: "Help"},
			{Key: "q", Desc: "Quit"},
		}

	case components.TabExplorer:
		content = m.ExplorerView.Render()
		shortcuts = []components.Shortcut{
			{Key: "Tab/h/l", Desc: "Switch Pane"},
			{Key: "Enter", Desc: "Open"},
			{Key: "Bksp", Desc: "Up"},
			{Key: "Space", Desc: "Select"},
			{Key: "r", Desc: "Remote"},
			{Key: "c", Desc: "Copy"},
			{Key: "s", Desc: "Sync"},
			{Key: "n", Desc: "Mkdir"},
			{Key: "x", Desc: "Delete"},
			{Key: "?", Desc: "Help"},
		}

	case components.TabTransfers:
		content = m.TransfersView.Render()
		shortcuts = []components.Shortcut{
			{Key: "j/k", Desc: "Select Job"},
			{Key: "x", Desc: "Cancel Job"},
			{Key: "c", Desc: "Clear Done"},
			{Key: "Tab", Desc: "Next Tab"},
			{Key: "?", Desc: "Help"},
			{Key: "q", Desc: "Quit"},
		}

	case components.TabRemotes:
		content = m.RemotesView.Render()
		shortcuts = []components.Shortcut{
			{Key: "j/k", Desc: "Select Remote"},
			{Key: "t", Desc: "Test Connection"},
			{Key: "r", Desc: "Refresh"},
			{Key: "Tab", Desc: "Next Tab"},
			{Key: "?", Desc: "Help"},
			{Key: "q", Desc: "Quit"},
		}
	}

	statusBar := components.RenderStatusBar(m.Width, shortcuts, m.StatusMsg, m.IsStatusErr)

	mainView := lipgloss.JoinVertical(lipgloss.Left, header, "\n", content, statusBar)

	if m.Modal.Type != components.ModalNone {
		modalOverlay := m.Modal.Render(m.Width, m.Height)
		return modalOverlay
	}

	return mainView
}
