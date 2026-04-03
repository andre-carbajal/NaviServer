package ui

import (
	"fmt"
	"naviserver/pkg/sdk"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type BackupWizardStep int

const (
	BackupStepSelectServer BackupWizardStep = iota
	BackupStepName
)

type BackupCreateWizardModel struct {
	client             *sdk.Client
	step               BackupWizardStep
	serverList         list.Model
	backupName         textinput.Model
	creating           bool
	width              int
	height             int
	selectedServerID   string
	selectedServerName string
}

func NewBackupCreateWizard(client *sdk.Client) BackupCreateWizardModel {
	sl := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	sl.Title = "Select Server"
	sl.SetShowStatusBar(false)
	sl.Styles.Title = titleStyle

	tiName := textinput.New()
	tiName.Placeholder = "Backup Name (Optional, press Enter to skip)"
	tiName.CharLimit = 32
	tiName.Width = 30

	return BackupCreateWizardModel{
		client:     client,
		serverList: sl,
		backupName: tiName,
		step:       BackupStepSelectServer,
	}
}

func (m BackupCreateWizardModel) Init() tea.Cmd {
	return tea.Batch(
		fetchServers(m.client),
		textinput.Blink,
	)
}

func (m BackupCreateWizardModel) Update(msg tea.Msg) (BackupCreateWizardModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.creating {
			return m, nil
		}

		switch m.step {
		case BackupStepSelectServer:
			if msg.Type == tea.KeyEnter {
				if m.serverList.FilterState() != list.Filtering {
					i := m.serverList.SelectedItem()
					if i != nil {
						itm := i.(serverSelectListItem)
						m.selectedServerID = itm.id
						m.selectedServerName = itm.name
						m.step = BackupStepName
						m.backupName.Focus()
						return m, textinput.Blink
					}
				}
			}

		case BackupStepName:
			if msg.Type == tea.KeyEnter {
				m.creating = true
				return m, createBackup(m.client, m.selectedServerID, m.backupName.Value())
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.serverList.SetWidth(msg.Width - 4)
		m.serverList.SetHeight(msg.Height - 12)

	case serverListMsg:
		var items []list.Item
		for _, s := range msg {
			items = append(items, serverSelectListItem{id: s.ID, name: s.Name})
		}
		m.serverList.SetItems(items)
	}

	if m.step == BackupStepSelectServer {
		m.serverList, cmd = m.serverList.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.step == BackupStepName {
		m.backupName, cmd = m.backupName.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m BackupCreateWizardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	title := headerStyle.Width(m.width).Render("CREATE BACKUP")

	headerText := ""
	if m.step == BackupStepSelectServer {
		headerText = "Step 1: Select a server to backup."
	} else if m.step == BackupStepName {
		headerText = fmt.Sprintf("Step 2: Backup name for '%s' (Optional)", m.selectedServerName)
	}

	headerBox := baseStyle.
		Width(m.width-4).
		Align(lipgloss.Center).
		Padding(0, 1).
		Render(headerText)

	content := ""
	if m.creating {
		content = lipgloss.NewStyle().Align(lipgloss.Center).Width(m.width - 4).Render("Creating backup... Please wait.")
	} else {
		if m.step == BackupStepSelectServer {
			content = baseStyle.
				Width(m.width - 4).
				Height(m.height - 12).
				Render(m.serverList.View())
		} else {
			content = baseStyle.
				Width(m.width - 4).
				Height(m.height - 12).
				Align(lipgloss.Center).
				Render(
					lipgloss.JoinVertical(lipgloss.Center,
						"\n\nEnter a name for this backup (or leave empty for default):",
						"\n",
						m.backupName.View(),
					),
				)
		}
	}

	keys := []string{
		keyStyle.Render("esc") + descStyle.Render(": back"),
	}
	if m.step == BackupStepSelectServer {
		keys = append(keys, keyStyle.Render("enter")+descStyle.Render(": next"))
	} else {
		keys = append(keys, keyStyle.Render("enter")+descStyle.Render(": create"))
	}

	helpText := lipgloss.JoinHorizontal(lipgloss.Top, keys...)
	helpText = ""
	for i, k := range keys {
		helpText += k
		if i < len(keys)-1 {
			helpText += lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(" ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â¢ ")
		}
	}

	footerBox := footerStyle.
		Width(m.width - 4).
		Align(lipgloss.Center).
		Render(helpText)

	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		headerBox,
		content,
		footerBox,
	)
}

func (m *BackupCreateWizardModel) Reset() {
	m.creating = false
	m.step = BackupStepSelectServer
	m.backupName.SetValue("")
	m.serverList.ResetSelected()
	m.selectedServerID = ""
	m.selectedServerName = ""
}

func (m *BackupCreateWizardModel) Focus() {
	if m.step == BackupStepName {
		m.backupName.Focus()
	}
}
