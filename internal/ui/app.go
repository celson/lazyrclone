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
)

type MainSubTab int

const (
	SubTabDiff MainSubTab = iota
	SubTabExplorer
	SubTabLogs
	SubTabRemotes
)

var MainSubTabNames = []string{"Diff Preview", "Dual Explorer", "Live Logs", "Remotes"}

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

type TickMsg time.Time

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

type AppModel struct {
	Config        *config.Config
	Client        rclone.RcloneClient
	FocusedPanel  components.PanelID
	MainSubTab    MainSubTab
	ProfilesView  *views.ProfilesView
	ExplorerView  *views.ExplorerView
	TransfersView *views.TransfersView
	RemotesView   *views.RemotesView
	DiffView      *components.DiffView
	Modal         *components.ModalManager
	Width         int
	Height        int
	StatusMsg     string
	IsStatusErr   bool
	RcloneVer     string
	IsMock        bool
	LogsOffset    int
	DryRunCancel  context.CancelFunc
}

func NewAppModel(cfg *config.Config, client rclone.RcloneClient) *AppModel {
	ver, _ := client.Version()

	pView := views.NewProfilesView(cfg)
	eView := views.NewExplorerView(client)
	tView := views.NewTransfersView()
	rView := views.NewRemotesView(client)
	dView := components.NewDiffView(80, 20)
	modal := components.NewModalManager()

	app := &AppModel{
		Config:        cfg,
		Client:        client,
		FocusedPanel:  components.PanelProfiles,
		MainSubTab:    SubTabDiff,
		ProfilesView:  pView,
		ExplorerView:  eView,
		TransfersView: tView,
		RemotesView:   rView,
		DiffView:      dView,
		Modal:         modal,
		RcloneVer:     ver,
		IsMock:        client.IsMock(),
	}

	return app
}

func (m *AppModel) hasRunningJobs() bool {
	if m.TransfersView == nil {
		return false
	}
	for _, j := range m.TransfersView.Jobs {
		if j.Status == rclone.JobStatusRunning {
			return true
		}
	}
	return false
}

func (m *AppModel) Init() tea.Cmd {
	m.RemotesView.Refresh()
	m.ProfilesView.SetRemotes(m.RemotesView.Remotes)
	m.syncExplorerWithSelectedProfile()
	return tickCmd(500 * time.Millisecond)
}

func (m *AppModel) syncExplorerWithSelectedProfile() {
	p := m.ProfilesView.SelectedProfile()
	if p == nil {
		m.ExplorerView.LoadPane(0)
		m.ExplorerView.LoadPane(1)
		return
	}

	srcRemote, srcDir := splitRemoteAndPath(p.Source)
	dstRemote, dstDir := splitRemoteAndPath(p.Destination)

	m.ExplorerView.Panes[0].Remote = srcRemote
	m.ExplorerView.Panes[0].CurrentDir = srcDir
	m.ExplorerView.Panes[0].SelectedIdx = 0
	m.ExplorerView.Panes[0].ScrollOffset = 0
	m.ExplorerView.Panes[0].Marked = make(map[string]bool)

	m.ExplorerView.Panes[1].Remote = dstRemote
	m.ExplorerView.Panes[1].CurrentDir = dstDir
	m.ExplorerView.Panes[1].SelectedIdx = 0
	m.ExplorerView.Panes[1].ScrollOffset = 0
	m.ExplorerView.Panes[1].Marked = make(map[string]bool)

	m.ExplorerView.LoadPane(0)
	m.ExplorerView.LoadPane(1)
}

func splitRemoteAndPath(full string) (string, string) {
	if strings.Contains(full, ":") {
		parts := strings.SplitN(full, ":", 2)
		return parts[0] + ":", parts[1]
	}
	return "local:", full
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case DryRunFinishedMsg:
		m.DryRunCancel = nil
		m.DiffView.SetResult(msg.Result)
		if msg.Result.Err != nil {
			m.StatusMsg = fmt.Sprintf("Dry-run error: %v", msg.Result.Err)
			m.IsStatusErr = true
		} else {
			m.StatusMsg = fmt.Sprintf("Diff ready: +%d to add, ~%d to update, -%d to delete",
				msg.Result.ToAdd, msg.Result.ToUpdate, msg.Result.ToDelete)
			m.IsStatusErr = false
		}
		return m, clearStatusCmd()

	case TickMsg:
		d := 1000 * time.Millisecond
		if m.hasRunningJobs() {
			d = 200 * time.Millisecond
		}
		return m, tickCmd(d)

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
						if j.LatestStats != nil {
							j.LatestStats.Percentage = 100
						}
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
		// 1. Modals handle keys first
		if m.Modal.Type != components.ModalNone {
			return m.handleModalKey(msg)
		}

		// 2. Global Hotkeys
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "1":
			m.FocusedPanel = components.PanelProfiles
			m.updatePanelsActive()
			return m, nil

		case "2":
			m.FocusedPanel = components.PanelRuns
			m.updatePanelsActive()
			return m, nil

		case "3":
			m.FocusedPanel = components.PanelMain
			m.updatePanelsActive()
			return m, nil

		case "4":
			m.FocusedPanel = components.PanelMain
			m.MainSubTab = SubTabRemotes
			m.updatePanelsActive()
			return m, nil

		case "tab":
			// Cycle panels: 0 -> 1 -> 2 -> 0
			m.FocusedPanel = (m.FocusedPanel + 1) % 3
			m.updatePanelsActive()
			return m, nil

		case "shift+tab":
			if m.FocusedPanel == 0 {
				m.FocusedPanel = 2
			} else {
				m.FocusedPanel--
			}
			m.updatePanelsActive()
			return m, nil

		case "[":
			if m.MainSubTab == 0 {
				m.MainSubTab = SubTabRemotes
			} else {
				m.MainSubTab--
			}
			return m, nil

		case "]":
			m.MainSubTab = (m.MainSubTab + 1) % 4
			return m, nil

		case "?":
			m.Modal.ShowHelp()
			return m, nil
		}

		// 3. Panel-Specific Keybindings
		switch m.FocusedPanel {
		case components.PanelProfiles:
			return m.handleProfilesKey(msg)
		case components.PanelRuns:
			return m.handleRunsKey(msg)
		case components.PanelMain:
			return m.handleMainKey(msg)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *AppModel) updatePanelsActive() {
	m.ProfilesView.IsActive = (m.FocusedPanel == components.PanelProfiles)
	m.TransfersView.IsActive = (m.FocusedPanel == components.PanelRuns)
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
		m.syncExplorerWithSelectedProfile()
	case "down", "j":
		m.ProfilesView.MoveDown()
		m.syncExplorerWithSelectedProfile()
	case "l", "right":
		m.FocusedPanel = components.PanelMain
		m.updatePanelsActive()
	case "R":
		m.FocusedPanel = components.PanelMain
		m.MainSubTab = SubTabRemotes
		m.updatePanelsActive()
		return m, nil
	case "d", "p":
		p := m.ProfilesView.SelectedProfile()
		if p == nil {
			return m, nil
		}
		m.MainSubTab = SubTabDiff
		m.StatusMsg = fmt.Sprintf("Running dry-run diff for '%s'...", p.Name)
		return m, m.runDryRunCmd(p)

	case "enter", "r":
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

	case "n":
		m.Modal.ShowProfileEdit(nil, func() {
			newP := m.Modal.GetProfileResult()
			m.Config.Profiles = append(m.Config.Profiles, newP)
			_ = config.SaveConfig(m.Config)
			m.StatusMsg = fmt.Sprintf("Profile '%s' created.", newP.Name)
			m.syncExplorerWithSelectedProfile()
		})

	case "e":
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
			m.syncExplorerWithSelectedProfile()
		})

	case "x":
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
				m.syncExplorerWithSelectedProfile()
			},
		)
	}
	return m, nil
}

func (m *AppModel) handleRunsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.TransfersView.MoveUp()
	case "down", "j":
		m.TransfersView.MoveDown()
	case "l", "right":
		m.FocusedPanel = components.PanelMain
		m.updatePanelsActive()
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
			m.StatusMsg = "Transfer cancelled."
		}
	}
	return m, nil
}

func (m *AppModel) handleMainKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "h", "left":
		if m.MainSubTab != SubTabExplorer {
			m.FocusedPanel = components.PanelProfiles
			m.updatePanelsActive()
			return m, nil
		}
	}

	switch m.MainSubTab {
	case SubTabDiff:
		switch msg.String() {
		case "up", "k":
			m.DiffView.MoveUp()
		case "down", "j":
			m.DiffView.MoveDown()
		case "d", "p":
			p := m.ProfilesView.SelectedProfile()
			if p != nil {
				m.StatusMsg = fmt.Sprintf("Re-running dry-run for '%s'...", p.Name)
				return m, m.runDryRunCmd(p)
			}
		case "x":
			if m.DryRunCancel != nil {
				m.DryRunCancel()
				m.DryRunCancel = nil
				m.StatusMsg = "Dry-run cancelled."
				return m, clearStatusCmd()
			}
		}

	case SubTabExplorer:
		switch msg.String() {
		case "left", "right":
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
			m.StatusMsg = fmt.Sprintf("Started copy of %s", items[0].Name)
			return m, m.startTransferCmd(job)

		case "s":
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
					go func() {
						err := m.Client.StartTransfer(context.Background(), job, func(stats *rclone.StatsMsg) {
							job.LatestStats = stats
						}, func(log string) {
							job.Logs = append(job.Logs, log)
						})
						endTime := time.Now()
						job.EndTime = &endTime
						if err != nil {
							job.Status = rclone.JobStatusFailed
							job.ErrorMsg = err.Error()
						} else {
							job.Status = rclone.JobStatusCompleted
							if job.LatestStats != nil {
								job.LatestStats.Percentage = 100
							}
						}
					}()
				},
			)

		case "n":
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

	case SubTabLogs:
		switch msg.String() {
		case "up", "k":
			if m.LogsOffset > 0 {
				m.LogsOffset--
			}
		case "down", "j":
			m.LogsOffset++
		}

	case SubTabRemotes:
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
			m.ProfilesView.SetRemotes(m.RemotesView.Remotes)
			m.StatusMsg = "Remotes refreshed."
		case "enter":
			r := m.RemotesView.SelectedRemote()
			if r != nil {
				m.ExplorerView.Panes[m.ExplorerView.ActivePane].Remote = r.Info.Name
				m.ExplorerView.Panes[m.ExplorerView.ActivePane].CurrentDir = ""
				m.ExplorerView.Panes[m.ExplorerView.ActivePane].SelectedIdx = 0
				m.ExplorerView.Panes[m.ExplorerView.ActivePane].ScrollOffset = 0
				m.ExplorerView.LoadPane(m.ExplorerView.ActivePane)
				m.MainSubTab = SubTabExplorer
				m.StatusMsg = fmt.Sprintf("Opened remote '%s' in Dual Explorer.", r.Info.Name)
			}
		}
	}

	return m, nil
}

func (m *AppModel) runDryRunCmd(p *config.Profile) tea.Cmd {
	if m.DryRunCancel != nil {
		m.DryRunCancel()
		m.DryRunCancel = nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	m.DryRunCancel = cancel

	return func() tea.Msg {
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

	now := time.Now()
	p.LastRun = &now
	p.LastStatus = "running"
	_ = config.SaveConfig(m.Config)
	m.StatusMsg = fmt.Sprintf("Job '%s' started.", p.Name)

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
			m.StatusMsg = fmt.Sprintf("Job '%s' failed: %v", p.Name, err)
			m.IsStatusErr = true
		} else {
			job.Status = rclone.JobStatusCompleted
			if job.LatestStats != nil {
				job.LatestStats.Percentage = 100
			}
			p.LastStatus = "success"
			m.StatusMsg = fmt.Sprintf("Job '%s' completed successfully.", p.Name)
			m.IsStatusErr = false
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

func (m *AppModel) renderMainPanel(width, height int) string {
	var subTabs []components.SubTabItem
	if width < 75 {
		subTabs = []components.SubTabItem{
			{Name: "Diff", IsActive: m.MainSubTab == SubTabDiff},
			{Name: "Explorer", IsActive: m.MainSubTab == SubTabExplorer},
			{Name: "Logs", IsActive: m.MainSubTab == SubTabLogs},
			{Name: "Remotes", IsActive: m.MainSubTab == SubTabRemotes},
		}
	} else {
		subTabs = []components.SubTabItem{
			{Name: "Diff Preview", IsActive: m.MainSubTab == SubTabDiff},
			{Name: "Dual Explorer", IsActive: m.MainSubTab == SubTabExplorer},
			{Name: "Live Logs", IsActive: m.MainSubTab == SubTabLogs},
			{Name: "Remotes", IsActive: m.MainSubTab == SubTabRemotes},
		}
	}

	isActive := (m.FocusedPanel == components.PanelMain)

	switch m.MainSubTab {
	case SubTabDiff:
		m.DiffView.Width = width - 4
		m.DiffView.Height = height - 2
		lines := m.DiffView.RenderLines()
		return components.RenderPanelBox(width, height, "[3] Main View", subTabs, isActive, lines)

	case SubTabExplorer:
		m.ExplorerView.SetSize(width-2, height-2)
		explorerContent := m.ExplorerView.Render()
		lines := strings.Split(explorerContent, "\n")
		return components.RenderPanelBox(width, height, "[3] Main View", subTabs, isActive, lines)

	case SubTabLogs:
		job := m.TransfersView.ActiveJob()
		var lines []string
		if job == nil || len(job.Logs) == 0 {
			lines = append(lines, "")
			lines = append(lines, "  No active logs.")
			lines = append(lines, "  Start a transfer with [r] in [1] Profiles to stream live output.")
		} else {
			visibleLogs := height - 4
			if visibleLogs < 1 {
				visibleLogs = 1
			}
			start := len(job.Logs) - visibleLogs - m.LogsOffset
			if start < 0 {
				start = 0
			}
			end := start + visibleLogs
			if end > len(job.Logs) {
				end = len(job.Logs)
			}
			for _, l := range job.Logs[start:end] {
				lines = append(lines, "  "+l)
			}
		}
		return components.RenderPanelBox(width, height, "[3] Main View", subTabs, isActive, lines)

	case SubTabRemotes:
		lines := m.RemotesView.RenderLines(width, height)
		return components.RenderPanelBox(width, height, "[3] Main View", subTabs, isActive, lines)
	}

	return ""
}

func (m *AppModel) View() string {
	if m.Width == 0 || m.Height == 0 {
		return "Initializing lazyrclone..."
	}

	// EXACT VERTICAL BUDGET:
	// Total screen height = m.Height
	// Status bar = 1 row at the very bottom
	// Panels occupy all remaining rows: m.Height - 1
	// Starts at row 0 (exact top edge, identical to Lazygit & Lazydocker)
	availHeight := m.Height - 1
	if availHeight < 6 {
		availHeight = 6
	}

	leftWidth := (m.Width * 38) / 100
	if leftWidth < 34 {
		leftWidth = 34
	}
	if leftWidth > 52 {
		leftWidth = 52
	}
	rightWidth := m.Width - leftWidth

	leftTopHeight := (availHeight * 55) / 100
	if leftTopHeight < 6 {
		leftTopHeight = 6
	}
	leftBottomHeight := availHeight - leftTopHeight

	m.ProfilesView.SetSize(leftWidth, leftTopHeight)
	m.TransfersView.SetSize(leftWidth, leftBottomHeight)

	leftTop := m.ProfilesView.Render()
	leftBottom := m.TransfersView.Render()
	leftCol := leftTop + "\n" + leftBottom

	rightCol := m.renderMainPanel(rightWidth, availHeight)

	// Combine left and right columns row by row
	leftRows := strings.Split(leftCol, "\n")
	rightRows := strings.Split(rightCol, "\n")

	maxRows := len(leftRows)
	if len(rightRows) > maxRows {
		maxRows = len(rightRows)
	}

	var combinedRows []string
	for i := 0; i < maxRows; i++ {
		lRow := ""
		if i < len(leftRows) {
			lRow = leftRows[i]
		}
		rRow := ""
		if i < len(rightRows) {
			rRow = rightRows[i]
		}
		combinedRows = append(combinedRows, lRow+rRow)
	}
	mainBody := strings.Join(combinedRows, "\n")

	var shortcuts []components.Shortcut
	switch m.FocusedPanel {
	case components.PanelProfiles:
		shortcuts = []components.Shortcut{
			{Key: "1-3", Desc: "Focus"},
			{Key: "d", Desc: "Dry-Run"},
			{Key: "r/Enter", Desc: "Run"},
			{Key: "R", Desc: "Remotes"},
			{Key: "n", Desc: "New"},
			{Key: "e", Desc: "Edit"},
			{Key: "x", Desc: "Delete"},
			{Key: "[/]", Desc: "Sub-tab"},
			{Key: "Tab", Desc: "Next"},
			{Key: "?", Desc: "Help"},
			{Key: "q", Desc: "Quit"},
		}
	case components.PanelRuns:
		shortcuts = []components.Shortcut{
			{Key: "1-3", Desc: "Focus"},
			{Key: "j/k", Desc: "Select"},
			{Key: "x", Desc: "Cancel"},
			{Key: "c", Desc: "Clear Done"},
			{Key: "[/]", Desc: "Sub-tab"},
			{Key: "Tab", Desc: "Next"},
			{Key: "?", Desc: "Help"},
			{Key: "q", Desc: "Quit"},
		}
	case components.PanelMain:
		switch m.MainSubTab {
		case SubTabDiff:
			shortcuts = []components.Shortcut{
				{Key: "1-3", Desc: "Focus"},
				{Key: "j/k", Desc: "Scroll"},
				{Key: "d", Desc: "Re-run"},
				{Key: "[/]", Desc: "Sub-tab"},
				{Key: "Tab", Desc: "Next"},
				{Key: "?", Desc: "Help"},
				{Key: "q", Desc: "Quit"},
			}
		case SubTabExplorer:
			shortcuts = []components.Shortcut{
				{Key: "←/→", Desc: "Switch Pane"},
				{Key: "Enter", Desc: "Open"},
				{Key: "Bksp", Desc: "Up"},
				{Key: "Space", Desc: "Select"},
				{Key: "c", Desc: "Copy"},
				{Key: "s", Desc: "Sync"},
				{Key: "[/]", Desc: "Sub-tab"},
				{Key: "Tab", Desc: "Next"},
				{Key: "?", Desc: "Help"},
			}
		case SubTabLogs:
			shortcuts = []components.Shortcut{
				{Key: "1-3", Desc: "Focus"},
				{Key: "j/k", Desc: "Scroll"},
				{Key: "[/]", Desc: "Sub-tab"},
				{Key: "Tab", Desc: "Next"},
				{Key: "?", Desc: "Help"},
				{Key: "q", Desc: "Quit"},
			}
		case SubTabRemotes:
			shortcuts = []components.Shortcut{
				{Key: "1-3", Desc: "Focus"},
				{Key: "j/k", Desc: "Select"},
				{Key: "t", Desc: "Test Conn"},
				{Key: "Enter", Desc: "Explorer"},
				{Key: "r", Desc: "Refresh"},
				{Key: "[/]", Desc: "Sub-tab"},
				{Key: "Tab", Desc: "Next"},
				{Key: "?", Desc: "Help"},
				{Key: "q", Desc: "Quit"},
			}
		}
	}

	statusBar := components.RenderStatusBar(m.Width, shortcuts, m.StatusMsg, m.IsStatusErr, m.RcloneVer, m.IsMock)

	fullScreen := mainBody + "\n" + statusBar

	if m.Modal.Type != components.ModalNone {
		modalOverlay := m.Modal.Render(m.Width, m.Height)
		return modalOverlay
	}

	return fullScreen
}
