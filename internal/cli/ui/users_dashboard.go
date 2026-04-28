package ui

import (
	"fmt"
	"os"

	"naviserver/pkg/sdk"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type usersListItem struct {
	id          string
	title       string
	description string
}

func (i usersListItem) FilterValue() string { return i.id + " " + i.title + " " + i.description }
func (i usersListItem) Title() string       { return i.title }
func (i usersListItem) Description() string { return i.description }

type usersDataMsg []sdk.User

type usersDashboardModel struct {
	client    *sdk.Client
	list      list.Model
	users     []sdk.User
	width     int
	height    int
	err       error
	message   string
	showHelp  bool
	isLoading bool
}

func RunUsersDashboard(client *sdk.Client) {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Users"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	l.Styles.HelpStyle = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)

	m := usersDashboardModel{
		client:    client,
		list:      l,
		isLoading: true,
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running users dashboard: %v", err)
		os.Exit(1)
	}
}

func (m usersDashboardModel) Init() tea.Cmd {
	return fetchUsersCmd(m.client)
}

func (m usersDashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			break
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc":
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "r":
			m.message = "Refreshing users..."
			return m, fetchUsersCmd(m.client)
		case "c":
			m.message = "User creation flow starts in Sprint 03"
			return m, nil
		case "x":
			m.message = "Permission editor starts in Sprint 03"
			return m, nil
		case "p":
			m.message = "Password update flow starts in Sprint 03"
			return m, nil
		case "d":
			m.message = "Delete confirmation flow starts in Sprint 03"
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width - 4)
		m.list.SetHeight(msg.Height - 12)
	case usersDataMsg:
		m.isLoading = false
		m.users = msg
		m.updateList()
		return m, nil
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *usersDashboardModel) updateList() {
	items := make([]list.Item, 0, len(m.users))
	for _, user := range m.users {
		items = append(items, usersListItem{
			id:          user.ID,
			title:       user.Username,
			description: fmt.Sprintf("Role: %s | ID: %s", user.Role, user.ID),
		})
	}
	m.list.SetItems(items)
}

func (m usersDashboardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	title := headerStyle.Width(m.width).Render("USERS DASHBOARD")

	headerContent := fmt.Sprintf("Daemon: %s\nUsers: %d", m.client.BaseURL(), len(m.users))
	headerBox := baseStyle.
		Width(m.width - 4).
		Align(lipgloss.Center).
		Render(headerContent)

	listBox := baseStyle.
		Width(m.width - 4).
		Height(m.height - 12).
		Render(m.list.View())

	keys := []string{
		keyStyle.Render("/") + descStyle.Render(": filter"),
		keyStyle.Render("c") + descStyle.Render(": create (Sprint 03)"),
		keyStyle.Render("x") + descStyle.Render(": permissions (Sprint 03)"),
		keyStyle.Render("p") + descStyle.Render(": password (Sprint 03)"),
		keyStyle.Render("d") + descStyle.Render(": delete (Sprint 03)"),
		keyStyle.Render("r") + descStyle.Render(": refresh"),
		keyStyle.Render("?") + descStyle.Render(": help"),
		keyStyle.Render("q/esc") + descStyle.Render(": back"),
		keyStyle.Render("ctrl+c") + descStyle.Render(": exit"),
	}
	footerBox := footerStyle.Width(m.width - 4).Render(renderInlineKeys(keys))

	if m.showHelp {
		helpBody := lipgloss.JoinVertical(lipgloss.Left,
			"Users dashboard",
			"- This sprint provides list and navigation foundation",
			"- c/x/p/d flows are implemented in Sprint 03",
			"- Press r to refresh current users",
		)
		helpBox := helpBoxStyle.Width(m.width - 4).Render(helpBody)
		listBox = lipgloss.JoinVertical(lipgloss.Left, listBox, helpBox)
	}

	if m.message != "" {
		footerBox = fmt.Sprintf("%s\n%s", statusMessage(m.message), footerBox)
	}

	if m.err != nil {
		footerBox = fmt.Sprintf("%s\n%s", errorMessage(m.err.Error()), footerBox)
	}

	return lipgloss.JoinVertical(lipgloss.Center, title, headerBox, listBox, footerBox)
}

func fetchUsersCmd(client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		users, err := client.ListUsers()
		if err != nil {
			return errMsg(err)
		}
		return usersDataMsg(users)
	}
}
