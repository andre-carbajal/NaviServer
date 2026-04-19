package ui

import (
	"fmt"
	"naviserver/pkg/sdk"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type BackupDashboardMode int

const (
	BackupViewList BackupDashboardMode = iota
	BackupViewCreate
	BackupViewRestore
	BackupViewDeleteConfirm
)

type BackupRestoreMode int

const (
	RestoreModeSelectServer BackupRestoreMode = iota
	RestoreModeNewServer
)

type BackupRestoreStep int

const (
	RestoreStepSelectTarget BackupRestoreStep = iota
	RestoreStepName
	RestoreStepLoader
	RestoreStepVersion
	RestoreStepRAM
	RestoreStepConfirm
)

type BackupDashboardModel struct {
	client    *sdk.Client
	mode      BackupDashboardMode
	list      list.Model
	backups   []sdk.BackupInfo
	width     int
	height    int
	err       error
	message   string
	isLoading bool

	serverList list.Model

	createWizard BackupCreateWizardModel

	deleteBackupName string

	restoreMode     BackupRestoreMode
	restoreStep     BackupRestoreStep
	restoreBackup   string
	restoreTarget   string
	restoreName     textinput.Model
	restoreRam      textinput.Model
	restoreLoader   list.Model
	restoreVersion  list.Model
	selectedLoader  string
	selectedVersion string
	restoring       bool
}

type backupListItem struct {
	name string
	size int64
}

func (i backupListItem) FilterValue() string { return i.name }
func (i backupListItem) Title() string       { return i.name }
func (i backupListItem) Description() string {
	return fmt.Sprintf("%.2f MB", float64(i.size)/1024/1024)
}

type serverSelectListItem struct {
	id   string
	name string
}

func (i serverSelectListItem) FilterValue() string { return i.name }
func (i serverSelectListItem) Title() string       { return i.name }
func (i serverSelectListItem) Description() string { return i.id }

type backupDataMsg []sdk.BackupInfo
type serverListMsg []sdk.Server
type backupCreatedMsg struct{}
type backupRestoredMsg struct{}

func RunBackupDashboard(client *sdk.Client) {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Backups"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	l.Styles.HelpStyle = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)

	sl := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	sl.Title = "Select Server"
	sl.SetShowStatusBar(false)
	sl.Styles.Title = titleStyle

	tiRestoreName := textinput.New()
	tiRestoreName.Placeholder = "New Server Name"
	tiRestoreName.CharLimit = 32
	tiRestoreName.Width = 30

	tiRestoreRam := textinput.New()
	tiRestoreRam.Placeholder = "2048"
	tiRestoreRam.CharLimit = 6
	tiRestoreRam.Width = 10

	rl := list.New([]list.Item{}, list.NewDefaultDelegate(), 30, 20)
	rl.Title = "Select Loader"
	rl.SetShowHelp(false)
	rl.Styles.Title = titleStyle

	rv := list.New([]list.Item{}, list.NewDefaultDelegate(), 30, 20)
	rv.Title = "Select Version"
	rv.SetShowHelp(false)
	rv.Styles.Title = titleStyle

	wizard := NewBackupCreateWizard(client)

	m := BackupDashboardModel{
		client:         client,
		mode:           BackupViewList,
		list:           l,
		serverList:     sl,
		createWizard:   wizard,
		restoreName:    tiRestoreName,
		restoreRam:     tiRestoreRam,
		restoreLoader:  rl,
		restoreVersion: rv,
		isLoading:      true,
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running backup dashboard: %v", err)
		os.Exit(1)
	}
}

func (m BackupDashboardModel) Init() tea.Cmd {
	return tea.Batch(fetchBackups(m.client), tickCmd())
}

func (m BackupDashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.mode == BackupViewList && m.list.FilterState() == list.Filtering {
			break
		}
		if m.mode == BackupViewCreate && m.serverList.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc":
			if m.mode == BackupViewList {
				return m, tea.Quit
			}
			if m.mode == BackupViewCreate || m.mode == BackupViewRestore || m.mode == BackupViewDeleteConfirm {
				m.mode = BackupViewList
				m.err = nil
				m.message = ""
				m.deleteBackupName = ""
				return m, fetchBackups(m.client)
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width - 4)
		m.list.SetHeight(msg.Height - 12)

		m.serverList.SetWidth(msg.Width - 4)
		m.serverList.SetHeight(msg.Height - 8)

		m.restoreLoader.SetSize(msg.Width-2, msg.Height-8)
		m.restoreVersion.SetSize(msg.Width-2, msg.Height-8)

		var cmd tea.Cmd
		m.createWizard, cmd = m.createWizard.Update(msg)
		return m, cmd
	case backupDataMsg:
		m.isLoading = false
		var items []list.Item
		for _, b := range msg {
			name := fmt.Sprintf("💾 %s", b.Name)
			items = append(items, backupListItem{name: name, size: b.Size})
		}
		m.list.SetItems(items)
		m.backups = msg
		return m, nil
	case serverListMsg:
		var items []list.Item
		if m.mode == BackupViewRestore {
			items = append(items, serverSelectListItem{id: "new_server", name: "Create New Server"})
		}
		for _, s := range msg {
			items = append(items, serverSelectListItem{id: s.ID, name: s.Name})
		}
		m.serverList.SetItems(items)

		if m.mode == BackupViewCreate {
			var cmd tea.Cmd
			m.createWizard, cmd = m.createWizard.Update(msg)
			return m, cmd
		}

		return m, nil
	case loadersMsg:
		var items []list.Item
		for _, l := range msg {
			items = append(items, item(l))
		}
		m.restoreLoader.SetItems(items)
		return m, nil
	case versionsMsg:
		var items []list.Item
		for _, v := range msg {
			items = append(items, item(v))
		}
		m.restoreVersion.SetItems(items)
		return m, nil
	case backupCreatedMsg:
		m.createWizard.creating = false
		m.mode = BackupViewList
		m.message = "Backup created successfully"
		return m, tea.Batch(fetchBackups(m.client), tickCmd())
	case backupRestoredMsg:
		m.restoring = false
		m.mode = BackupViewList
		m.message = "Backup restored successfully"
		return m, tea.Batch(fetchBackups(m.client), tickCmd())
	case errMsg:
		m.err = msg
		m.createWizard.creating = false
		m.restoring = false
		return m, nil
	case string:
		if msg == "clear_message" {
			m.message = ""
			return m, nil
		}
	case tickMsg:
		return m, tea.Batch(fetchBackups(m.client), tickCmd())
	}

	switch m.mode {
	case BackupViewList:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "c":
				m.mode = BackupViewCreate
				m.createWizard.Reset()
				m.createWizard.Focus()
				return m, m.createWizard.Init()
			case "r":
				i := m.list.SelectedItem()
				if i != nil {
					m.mode = BackupViewRestore
					name := i.(backupListItem).name
					name = strings.TrimPrefix(name, "💾 ")

					m.restoreBackup = name
					m.restoreStep = RestoreStepSelectTarget
					return m, fetchServers(m.client)
				}
			case "d":
				i := m.list.SelectedItem()
				if i != nil {
					itm := i.(backupListItem)
					name := itm.name
					name = strings.TrimPrefix(name, "💾 ")

					m.deleteBackupName = name
					m.mode = BackupViewDeleteConfirm
					return m, nil
				}
			}
		}
		m.list, cmd = m.list.Update(msg)
		return m, cmd

	case BackupViewDeleteConfirm:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "y", "enter":
				go m.client.DeleteBackup(m.deleteBackupName)
				m.message = fmt.Sprintf("Deleting backup %s...", m.deleteBackupName)
				m.mode = BackupViewList
				return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
					return fetchBackups(m.client)()
				})
			case "n", "esc":
				m.mode = BackupViewList
				m.message = "Deletion cancelled"
				return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
					return "clear_message"
				})
			}
		}
		return m, nil

	case BackupViewCreate:
		var cmd tea.Cmd
		m.createWizard, cmd = m.createWizard.Update(msg)
		return m, cmd

	case BackupViewRestore:
		if m.restoring {
			return m, nil
		}

		switch m.restoreStep {
		case RestoreStepSelectTarget:
			switch msg := msg.(type) {
			case tea.KeyMsg:
				if msg.Type == tea.KeyEnter {
					i := m.serverList.SelectedItem()
					if i != nil {
						itm := i.(serverSelectListItem)
						if itm.id == "new_server" {
							m.restoreMode = RestoreModeNewServer
							m.restoreStep = RestoreStepName
							m.restoreName.Focus()
							return m, textinput.Blink
						} else {
							m.restoreMode = RestoreModeSelectServer
							m.restoreTarget = itm.id
							m.restoring = true
							return m, restoreBackup(m.client, m.restoreBackup, sdk.RestoreBackupRequest{
								TargetServerID: m.restoreTarget,
							})
						}
					}
				}
			}
			m.serverList, cmd = m.serverList.Update(msg)
			return m, cmd

		case RestoreStepName:
			switch msg := msg.(type) {
			case tea.KeyMsg:
				if msg.Type == tea.KeyEnter {
					if m.restoreName.Value() == "" {
						m.err = fmt.Errorf("name cannot be empty")
						return m, nil
					}
					m.err = nil
					m.restoreStep = RestoreStepLoader
					return m, fetchLoaders(m.client)
				}
			}
			m.restoreName, cmd = m.restoreName.Update(msg)
			return m, cmd

		case RestoreStepLoader:
			switch msg := msg.(type) {
			case tea.KeyMsg:
				if msg.Type == tea.KeyEnter {
					i, ok := m.restoreLoader.SelectedItem().(item)
					if ok {
						m.selectedLoader = string(i)
						m.restoreStep = RestoreStepVersion
						return m, fetchVersions(m.client, m.selectedLoader)
					}
				}
			}
			m.restoreLoader, cmd = m.restoreLoader.Update(msg)
			return m, cmd

		case RestoreStepVersion:
			switch msg := msg.(type) {
			case tea.KeyMsg:
				if msg.Type == tea.KeyEnter {
					i, ok := m.restoreVersion.SelectedItem().(item)
					if ok {
						m.selectedVersion = string(i)
						m.restoreStep = RestoreStepRAM
						m.restoreRam.Focus()
						return m, textinput.Blink
					}
				}
			}
			m.restoreVersion, cmd = m.restoreVersion.Update(msg)
			return m, cmd

		case RestoreStepRAM:
			switch msg := msg.(type) {
			case tea.KeyMsg:
				if msg.Type == tea.KeyEnter {
					val, err := strconv.Atoi(m.restoreRam.Value())
					if err != nil || val <= 0 {
						m.err = fmt.Errorf("invalid RAM amount")
						return m, nil
					}
					m.err = nil
					m.restoreStep = RestoreStepConfirm
					return m, nil
				}
			}
			m.restoreRam, cmd = m.restoreRam.Update(msg)
			return m, cmd

		case RestoreStepConfirm:
			switch msg := msg.(type) {
			case tea.KeyMsg:
				if msg.String() == "y" || msg.Type == tea.KeyEnter {
					ram, _ := strconv.Atoi(m.restoreRam.Value())
					m.restoring = true
					return m, restoreBackup(m.client, m.restoreBackup, sdk.RestoreBackupRequest{
						NewServerName:    m.restoreName.Value(),
						NewServerLoader:  m.selectedLoader,
						NewServerVersion: m.selectedVersion,
						NewServerRam:     ram,
					})
				} else if msg.String() == "n" {
					m.mode = BackupViewList
					return m, nil
				}
			}
		}
	}

	return m, nil
}

func (m BackupDashboardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	title := headerStyle.Width(m.width).Render("BACKUPS DASHBOARD")

	if m.err != nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("Error: %v\n\n", m.err))
	}

	switch m.mode {
	case BackupViewList:
		listContainer := baseStyle.
			Width(m.width - 4).
			Height(m.height - 8).
			Render(m.list.View())

		keys := []string{
			keyStyle.Render("/") + descStyle.Render(": filter"),
			keyStyle.Render("c") + descStyle.Render(": create"),
			keyStyle.Render("r") + descStyle.Render(": restore"),
			keyStyle.Render("d") + descStyle.Render(": delete"),
			keyStyle.Render("q/esc") + descStyle.Render(": quit"),
		}
		statusLine := lipgloss.JoinHorizontal(lipgloss.Top, keys...)
		statusLine = ""
		for i, k := range keys {
			statusLine += k
			if i < len(keys)-1 {
				statusLine += lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(" • ")
			}
		}

		footerBox := footerStyle.
			Width(m.width - 4).
			Render(statusLine)

		if m.message != "" {
			footerBox = fmt.Sprintf("%s\n%s",
				lipgloss.NewStyle().MarginLeft(2).Foreground(lipgloss.Color("205")).Bold(true).Render(m.message),
				footerBox)
		}

		return lipgloss.JoinVertical(lipgloss.Center,
			title,
			listContainer,
			footerBox,
		)

	case BackupViewCreate:
		return m.createWizard.View()

	case BackupViewRestore:
		header := headerStyle.Render("Restore Backup: " + m.restoreBackup)
		content := ""

		if m.restoring {
			content = "Restoring backup... Please wait."
		} else {
			switch m.restoreStep {
			case RestoreStepSelectTarget:
				content += "Select Target Server:\n" + m.serverList.View()
			case RestoreStepName:
				content += fmt.Sprintf("Enter New Server Name:\n\n%s", m.restoreName.View())
			case RestoreStepLoader:
				content += "Select Loader:\n\n" + m.restoreLoader.View()
			case RestoreStepVersion:
				content += fmt.Sprintf("Select Version for %s:\n\n%s", m.selectedLoader, m.restoreVersion.View())
			case RestoreStepRAM:
				content += fmt.Sprintf("Enter RAM (MB):\n\n%s", m.restoreRam.View())
			case RestoreStepConfirm:
				content += fmt.Sprintf("Confirm Restore?\n\nBackup: %s\nNew Server Name: %s\nLoader: %s\nVersion: %s\nRAM: %s MB\n\n(y/n)",
					m.restoreBackup, m.restoreName.Value(), m.selectedLoader, m.selectedVersion, m.restoreRam.Value())
			}
		}

		contentBox := baseStyle.
			Width(m.width - 4).
			Height(m.height - 5).
			Align(lipgloss.Center).
			Render(content)

		return lipgloss.JoinVertical(lipgloss.Center, title, header, contentBox)

	case BackupViewDeleteConfirm:
		header := headerStyle.Render("DELETE CONFIRMATION")
		content := fmt.Sprintf("\nAre you sure you want to delete backup:\n\n%s\n\n(y/n)",
			lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render(m.deleteBackupName))

		confirmBox := baseStyle.
			Width(m.width-4).
			Height(m.height-4).
			Align(lipgloss.Center, lipgloss.Center).
			Render(content)

		return lipgloss.JoinVertical(lipgloss.Center, title, header, confirmBox)
	}

	return ""
}

func fetchBackups(client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		backups, err := client.ListAllBackups()
		if err != nil {
			return errMsg(err)
		}
		return backupDataMsg(backups)
	}
}

func fetchServers(client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		servers, err := client.ListServers()
		if err != nil {
			return errMsg(err)
		}
		return serverListMsg(servers)
	}
}

func createBackup(client *sdk.Client, serverID, name string) tea.Cmd {
	return func() tea.Msg {
		_, err := client.CreateBackup(serverID, name)
		if err != nil {
			return errMsg(err)
		}
		return backupCreatedMsg{}
	}
}

func restoreBackup(client *sdk.Client, backupName string, req sdk.RestoreBackupRequest) tea.Cmd {
	return func() tea.Msg {
		err := client.RestoreBackup(backupName, req)
		if err != nil {
			return errMsg(err)
		}
		return backupRestoredMsg{}
	}
}
