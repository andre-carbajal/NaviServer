package ui

import (
	"os"

	"naviserver/pkg/sdk"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type mainMenuItem struct {
	title       string
	description string
	action      string
}

func (i mainMenuItem) FilterValue() string { return i.title + " " + i.description }
func (i mainMenuItem) Title() string       { return i.title }
func (i mainMenuItem) Description() string { return i.description }

type mainMenuModel struct {
	list   list.Model
	width  int
	height int
	choice string
}

func newMainMenuModel() mainMenuModel {
	items := []list.Item{
		mainMenuItem{title: "Servers", description: "Create, start, stop, delete and inspect", action: "servers"},
		mainMenuItem{title: "Backups", description: "Create, restore and delete backups", action: "backups"},
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Main Sections"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	l.Styles.HelpStyle = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)

	return mainMenuModel{list: l}
}

func (m mainMenuModel) Init() tea.Cmd { return nil }

func (m mainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.choice = "quit"
			return m, tea.Quit
		case "s":
			m.choice = "servers"
			return m, tea.Quit
		case "b":
			m.choice = "backups"
			return m, tea.Quit
		case "enter":
			i, ok := m.list.SelectedItem().(mainMenuItem)
			if ok {
				m.choice = i.action
				return m, tea.Quit
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width - 4)
		m.list.SetHeight(msg.Height - 10)
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m mainMenuModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	title := headerStyle.Width(m.width).Render("NAVISERVER TUI")

	headerContent := "Select a section to manage your infrastructure"
	headerBox := baseStyle.
		Width(m.width - 4).
		Align(lipgloss.Center).
		Render(headerContent)

	listBox := baseStyle.
		Width(m.width - 4).
		Height(m.height - 10).
		Render(m.list.View())

	keys := []string{
		keyStyle.Render("enter") + descStyle.Render(": open"),
		keyStyle.Render("s") + descStyle.Render(": servers"),
		keyStyle.Render("b") + descStyle.Render(": backups"),
		keyStyle.Render("q/esc") + descStyle.Render(": quit"),
	}
	statusLine := ""
	for i, k := range keys {
		statusLine += k
		if i < len(keys)-1 {
			statusLine += lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(" • ")
		}
	}

	footerBox := footerStyle.
		Width(m.width - 4).
		Render(statusLine)

	return lipgloss.JoinVertical(lipgloss.Center, title, headerBox, listBox, footerBox)
}

func runMainMenu() string {
	p := tea.NewProgram(newMainMenuModel(), tea.WithAltScreen(), tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
	fm, err := p.Run()
	if err != nil {
		return "quit"
	}

	m, ok := fm.(mainMenuModel)
	if !ok {
		return "quit"
	}

	if m.choice == "" {
		return "quit"
	}

	return m.choice
}

func RunMainTUI(client *sdk.Client) {
	for {
		switch runMainMenu() {
		case "servers":
			for {
				serverID := RunServerDashboard(client)
				if serverID == "" {
					break
				}
				if !RunLogs(client, serverID) {
					break
				}
			}
		case "backups":
			RunBackupDashboard(client)
		default:
			return
		}
	}
}
