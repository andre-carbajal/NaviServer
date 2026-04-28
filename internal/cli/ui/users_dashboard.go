package ui

import (
	"fmt"
	"os"
	"strings"

	"naviserver/pkg/sdk"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
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
type serversDataMsg []sdk.Server
type permissionsDataMsg []sdk.Permission
type userCreatedMsg struct{}
type passwordUpdatedMsg struct{}
type permissionsUpdatedMsg struct{}
type userDeletedMsg struct{}

type usersMode int

const (
	UsersViewList usersMode = iota
	UsersViewCreate
	UsersViewEditPermissions
	UsersViewChangePassword
	UsersViewDeleteConfirm
)

type permissionRow struct {
	serverID        string
	serverName      string
	canControlPower bool
	canViewConsole  bool
}

type permissionListItem struct {
	row permissionRow
	col int
}

func (i permissionListItem) FilterValue() string { return i.row.serverName + " " + i.row.serverID }
func (i permissionListItem) Title() string {
	power := "[ ]"
	if i.row.canControlPower {
		power = "[x]"
	}
	console := "[ ]"
	if i.row.canViewConsole {
		console = "[x]"
	}

	powerLabel := "Power Control"
	consoleLabel := "Console & Files"
	if i.col == 0 {
		powerLabel = lipgloss.NewStyle().Bold(true).Render("‹ " + powerLabel + " ›")
	}
	if i.col == 1 {
		consoleLabel = lipgloss.NewStyle().Bold(true).Render("‹ " + consoleLabel + " ›")
	}

	return fmt.Sprintf("%s  %s %s  %s %s", i.row.serverName, power, powerLabel, console, consoleLabel)
}
func (i permissionListItem) Description() string { return i.row.serverID }

type usersDashboardModel struct {
	client         *sdk.Client
	list           list.Model
	permissionList list.Model
	users          []sdk.User
	servers        []sdk.Server
	permissionRows []permissionRow
	activeUser     *sdk.User
	width          int
	height         int
	err            error
	message        string
	showHelp       bool
	isLoading      bool
	mode           usersMode
	createUsername textinput.Model
	createPassword textinput.Model
	createStep     int
	passwordInput  textinput.Model
	permissionCol  int
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

	pl := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	pl.Title = "Server Permissions"
	pl.SetShowStatusBar(false)
	pl.SetFilteringEnabled(false)
	pl.SetShowHelp(false)
	pl.Styles.Title = titleStyle
	pl.Styles.PaginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	pl.Styles.HelpStyle = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)

	createUsername := textinput.New()
	createUsername.Placeholder = "Username"
	createUsername.CharLimit = 32
	createUsername.Width = 30

	createPassword := textinput.New()
	createPassword.Placeholder = "Password"
	createPassword.CharLimit = 128
	createPassword.Width = 30
	createPassword.EchoMode = textinput.EchoPassword
	createPassword.EchoCharacter = '*'

	passwordInput := textinput.New()
	passwordInput.Placeholder = "New password"
	passwordInput.CharLimit = 128
	passwordInput.Width = 30
	passwordInput.EchoMode = textinput.EchoPassword
	passwordInput.EchoCharacter = '*'

	m := usersDashboardModel{
		client:         client,
		list:           l,
		permissionList: pl,
		createUsername: createUsername,
		createPassword: createPassword,
		passwordInput:  passwordInput,
		isLoading:      true,
		mode:           UsersViewList,
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
		if m.mode == UsersViewList && m.list.FilterState() == list.Filtering {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			break
		}

		switch m.mode {
		case UsersViewCreate:
			return m.updateCreateUser(msg)
		case UsersViewChangePassword:
			return m.updateChangePassword(msg)
		case UsersViewEditPermissions:
			return m.updateEditPermissions(msg)
		case UsersViewDeleteConfirm:
			return m.updateDeleteConfirm(msg)
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
			m.mode = UsersViewCreate
			m.createStep = 0
			m.createUsername.SetValue("")
			m.createPassword.SetValue("")
			m.createUsername.Focus()
			m.createPassword.Blur()
			m.message = "Create user"
			m.err = nil
			return m, textinput.Blink
		case "x":
			selected, ok := m.selectedUser()
			if !ok {
				m.message = "Select a user first"
				return m, nil
			}
			m.activeUser = &selected
			m.mode = UsersViewEditPermissions
			m.permissionCol = 0
			m.permissionRows = nil
			m.permissionList.SetItems([]list.Item{})
			m.message = fmt.Sprintf("Loading permissions for %s...", selected.Username)
			m.err = nil
			return m, tea.Batch(fetchServersCmd(m.client), fetchPermissionsCmd(m.client, selected.ID))
		case "p":
			selected, ok := m.selectedUser()
			if !ok {
				m.message = "Select a user first"
				return m, nil
			}
			m.activeUser = &selected
			m.mode = UsersViewChangePassword
			m.passwordInput.SetValue("")
			m.passwordInput.Focus()
			m.message = fmt.Sprintf("Updating password for %s", selected.Username)
			m.err = nil
			return m, textinput.Blink
		case "d":
			selected, ok := m.selectedUser()
			if !ok {
				m.message = "Select a user first"
				return m, nil
			}
			if strings.EqualFold(selected.Role, "admin") {
				m.message = "Admin user cannot be deleted"
				m.err = nil
				return m, nil
			}
			m.activeUser = &selected
			m.mode = UsersViewDeleteConfirm
			m.message = "Confirm deletion"
			m.err = nil
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width - 4)
		m.list.SetHeight(msg.Height - 12)
		m.permissionList.SetWidth(msg.Width - 4)
		m.permissionList.SetHeight(msg.Height - 12)
	case usersDataMsg:
		m.isLoading = false
		m.users = msg
		if m.activeUser != nil {
			for i := range m.users {
				if m.users[i].ID == m.activeUser.ID {
					m.activeUser = &m.users[i]
					break
				}
			}
		}
		m.updateList()
		return m, nil
	case serversDataMsg:
		m.servers = msg
		m.rebuildPermissionRows()
		return m, nil
	case permissionsDataMsg:
		m.applyPermissions(msg)
		m.rebuildPermissionRows()
		return m, nil
	case userCreatedMsg:
		m.mode = UsersViewList
		m.createUsername.Blur()
		m.createPassword.Blur()
		m.message = "User created successfully"
		m.err = nil
		return m, fetchUsersCmd(m.client)
	case passwordUpdatedMsg:
		m.mode = UsersViewList
		m.passwordInput.Blur()
		m.message = "Password updated successfully"
		m.err = nil
		return m, nil
	case permissionsUpdatedMsg:
		m.mode = UsersViewList
		m.message = "Permissions updated successfully"
		m.err = nil
		return m, nil
	case userDeletedMsg:
		m.mode = UsersViewList
		m.message = "User deleted successfully"
		m.err = nil
		m.activeUser = nil
		return m, fetchUsersCmd(m.client)
	case errMsg:
		m.err = msg
		return m, nil
	}

	if m.mode == UsersViewList {
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	if m.mode == UsersViewEditPermissions {
		m.permissionList, cmd = m.permissionList.Update(msg)
		return m, cmd
	}

	return m, nil
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

func (m usersDashboardModel) selectedUser() (sdk.User, bool) {
	i := m.list.SelectedItem()
	if i == nil {
		return sdk.User{}, false
	}
	itm, ok := i.(usersListItem)
	if !ok {
		return sdk.User{}, false
	}
	for _, user := range m.users {
		if user.ID == itm.id {
			return user, true
		}
	}
	return sdk.User{}, false
}

func (m *usersDashboardModel) rebuildPermissionRows() {
	if m.mode != UsersViewEditPermissions || m.activeUser == nil || len(m.servers) == 0 {
		return
	}

	current := make(map[string]permissionRow, len(m.permissionRows))
	for _, row := range m.permissionRows {
		current[row.serverID] = row
	}

	rows := make([]permissionRow, 0, len(m.servers))
	for _, server := range m.servers {
		row := permissionRow{serverID: server.ID, serverName: server.Name}
		if existing, ok := current[server.ID]; ok {
			row.canControlPower = existing.canControlPower
			row.canViewConsole = existing.canViewConsole
		}
		rows = append(rows, row)
	}

	m.permissionRows = rows
	m.syncPermissionListItems()
}

func (m *usersDashboardModel) applyPermissions(perms []sdk.Permission) {
	if m.activeUser == nil {
		return
	}

	permByServer := make(map[string]sdk.Permission)
	for _, perm := range perms {
		if perm.UserID == m.activeUser.ID {
			permByServer[perm.ServerID] = perm
		}
	}

	for i := range m.permissionRows {
		m.permissionRows[i].canControlPower = false
		m.permissionRows[i].canViewConsole = false
		if perm, ok := permByServer[m.permissionRows[i].serverID]; ok {
			m.permissionRows[i].canControlPower = perm.CanControlPower
			m.permissionRows[i].canViewConsole = perm.CanViewConsole
		}
	}
}

func (m *usersDashboardModel) syncPermissionListItems() {
	items := make([]list.Item, 0, len(m.permissionRows))
	for _, row := range m.permissionRows {
		items = append(items, permissionListItem{row: row, col: m.permissionCol})
	}
	m.permissionList.SetItems(items)
}

func (m usersDashboardModel) updateCreateUser(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	switch msg.String() {
	case "esc":
		m.mode = UsersViewList
		m.createUsername.Blur()
		m.createPassword.Blur()
		m.message = "User creation cancelled"
		return m, nil
	case "tab", "shift+tab", "up", "down":
		if m.createStep == 0 {
			m.createStep = 1
			m.createUsername.Blur()
			m.createPassword.Focus()
		} else {
			m.createStep = 0
			m.createPassword.Blur()
			m.createUsername.Focus()
		}
		return m, nil
	case "enter":
		if m.createStep == 0 {
			m.createStep = 1
			m.createUsername.Blur()
			m.createPassword.Focus()
			return m, textinput.Blink
		}

		username := strings.TrimSpace(m.createUsername.Value())
		password := m.createPassword.Value()
		if username == "" || password == "" {
			m.err = fmt.Errorf("username and password are required")
			return m, nil
		}
		m.err = nil
		m.message = "Creating user..."
		return m, createUserCmd(m.client, username, password)
	}

	var cmd tea.Cmd
	if m.createStep == 0 {
		m.createUsername, cmd = m.createUsername.Update(msg)
	} else {
		m.createPassword, cmd = m.createPassword.Update(msg)
	}
	return m, cmd
}

func (m usersDashboardModel) updateChangePassword(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	switch msg.String() {
	case "esc":
		m.mode = UsersViewList
		m.passwordInput.Blur()
		m.message = "Password update cancelled"
		return m, nil
	case "enter":
		if m.activeUser == nil {
			m.mode = UsersViewList
			m.err = fmt.Errorf("no user selected")
			return m, nil
		}
		password := m.passwordInput.Value()
		if password == "" {
			m.err = fmt.Errorf("password cannot be empty")
			return m, nil
		}
		m.err = nil
		m.message = "Updating password..."
		return m, updatePasswordCmd(m.client, m.activeUser.ID, password)
	}

	var cmd tea.Cmd
	m.passwordInput, cmd = m.passwordInput.Update(msg)
	return m, cmd
}

func (m usersDashboardModel) updateEditPermissions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	switch msg.String() {
	case "esc":
		m.mode = UsersViewList
		m.message = "Permission edit cancelled"
		return m, nil
	case "tab", "left", "right":
		if m.permissionCol == 0 {
			m.permissionCol = 1
		} else {
			m.permissionCol = 0
		}
		m.syncPermissionListItems()
		return m, nil
	case "r":
		if m.activeUser == nil {
			return m, nil
		}
		m.message = "Refreshing permissions..."
		return m, tea.Batch(fetchServersCmd(m.client), fetchPermissionsCmd(m.client, m.activeUser.ID))
	case "s":
		if m.activeUser == nil {
			m.err = fmt.Errorf("no user selected")
			return m, nil
		}
		m.err = nil
		m.message = "Saving permissions..."
		return m, setPermissionsCmd(m.client, m.activeUser.ID, m.permissionRows)
	case "enter":
		if m.activeUser == nil {
			m.err = fmt.Errorf("no user selected")
			return m, nil
		}
		m.err = nil
		m.message = "Saving permissions..."
		return m, setPermissionsCmd(m.client, m.activeUser.ID, m.permissionRows)
	case " ":
		idx := m.permissionList.Index()
		if idx < 0 || idx >= len(m.permissionRows) {
			return m, nil
		}
		row := m.permissionRows[idx]
		if m.permissionCol == 0 {
			row.canControlPower = !row.canControlPower
			if !row.canControlPower {
				row.canViewConsole = false
			}
		} else {
			row.canViewConsole = !row.canViewConsole
			if row.canViewConsole {
				row.canControlPower = true
			}
		}
		m.permissionRows[idx] = row
		m.syncPermissionListItems()
		return m, nil
	}

	var cmd tea.Cmd
	m.permissionList, cmd = m.permissionList.Update(msg)
	return m, cmd
}

func (m usersDashboardModel) updateDeleteConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if m.activeUser == nil {
		m.mode = UsersViewList
		m.err = fmt.Errorf("no user selected")
		return m, nil
	}
	if strings.EqualFold(m.activeUser.Role, "admin") {
		m.mode = UsersViewList
		m.message = "Admin user cannot be deleted"
		m.err = nil
		return m, nil
	}

	switch msg.String() {
	case "n", "esc":
		m.mode = UsersViewList
		m.message = "Deletion cancelled"
		return m, nil
	case "y", "enter":
		m.message = "Deleting user..."
		m.err = nil
		return m, deleteUserCmd(m.client, m.activeUser.ID)
	}

	return m, nil
}

func (m usersDashboardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	title := headerStyle.Width(m.width).Render("USERS DASHBOARD")

	headerContent := fmt.Sprintf("Daemon: %s\nUsers: %d", m.client.BaseURL(), len(m.users))
	if m.activeUser != nil {
		headerContent = fmt.Sprintf("%s\nSelected: %s", headerContent, m.activeUser.Username)
	}
	headerBox := baseStyle.
		Width(m.width - 4).
		Align(lipgloss.Center).
		Render(headerContent)

	listContent := m.list.View()
	if m.mode == UsersViewCreate {
		listContent = m.renderCreateUser()
	}
	if m.mode == UsersViewEditPermissions {
		listContent = m.renderEditPermissions()
	}
	if m.mode == UsersViewChangePassword {
		listContent = m.renderChangePassword()
	}
	if m.mode == UsersViewDeleteConfirm {
		listContent = m.renderDeleteConfirm()
	}

	listBox := baseStyle.
		Width(m.width - 4).
		Height(m.height - 12).
		Render(listContent)

	keys := []string{
		keyStyle.Render("/") + descStyle.Render(": filter"),
		keyStyle.Render("c") + descStyle.Render(": create"),
		keyStyle.Render("x") + descStyle.Render(": permissions"),
		keyStyle.Render("p") + descStyle.Render(": password"),
		keyStyle.Render("d") + descStyle.Render(": delete"),
		keyStyle.Render("r") + descStyle.Render(": refresh"),
		keyStyle.Render("?") + descStyle.Render(": help"),
		keyStyle.Render("q/esc") + descStyle.Render(": back"),
		keyStyle.Render("ctrl+c") + descStyle.Render(": exit"),
	}
	if m.mode == UsersViewCreate {
		keys = []string{
			keyStyle.Render("tab") + descStyle.Render(": switch field"),
			keyStyle.Render("enter") + descStyle.Render(": next/create"),
			keyStyle.Render("esc") + descStyle.Render(": cancel"),
			keyStyle.Render("ctrl+c") + descStyle.Render(": exit"),
		}
	}
	if m.mode == UsersViewEditPermissions {
		keys = []string{
			keyStyle.Render("up/down") + descStyle.Render(": select server"),
			keyStyle.Render("tab/left/right") + descStyle.Render(": switch column"),
			keyStyle.Render("space") + descStyle.Render(": toggle"),
			keyStyle.Render("enter") + descStyle.Render(": save"),
			keyStyle.Render("r") + descStyle.Render(": reload"),
			keyStyle.Render("esc") + descStyle.Render(": back"),
		}
	}
	if m.mode == UsersViewChangePassword {
		keys = []string{
			keyStyle.Render("enter") + descStyle.Render(": update"),
			keyStyle.Render("esc") + descStyle.Render(": cancel"),
			keyStyle.Render("ctrl+c") + descStyle.Render(": exit"),
		}
	}
	if m.mode == UsersViewDeleteConfirm {
		keys = []string{
			keyStyle.Render("y/enter") + descStyle.Render(": confirm"),
			keyStyle.Render("n/esc") + descStyle.Render(": cancel"),
			keyStyle.Render("ctrl+c") + descStyle.Render(": exit"),
		}
	}
	footerBox := footerStyle.Width(m.width - 4).Render(renderInlineKeys(keys))

	if m.showHelp {
		helpBody := lipgloss.JoinVertical(lipgloss.Left,
			"Users dashboard",
			"- c creates user with username and password",
			"- x edits permissions per server with dependency rules",
			"- p updates password, d confirms delete",
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

func (m usersDashboardModel) renderCreateUser() string {
	active := "username"
	if m.createStep == 1 {
		active = "password"
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		"Create User",
		"",
		fmt.Sprintf("Username: %s", m.createUsername.View()),
		fmt.Sprintf("Password: %s", m.createPassword.View()),
		"",
		fmt.Sprintf("Active field: %s", active),
	)
}

func (m usersDashboardModel) renderEditPermissions() string {
	userLabel := ""
	if m.activeUser != nil {
		userLabel = fmt.Sprintf("Editing permissions for %s", m.activeUser.Username)
	}
	if len(m.servers) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			userLabel,
			"",
			"No servers available. Create at least one server first.",
		)
	}

	activeToggle := "Active toggle: Power Control"
	if m.permissionCol == 1 {
		activeToggle = "Active toggle: Console & Files"
	}

	dependencyHint := "Rule: Console & Files => Power Control, and disabling Power Control disables Console & Files"
	return lipgloss.JoinVertical(lipgloss.Left,
		userLabel,
		"",
		activeToggle,
		"",
		m.permissionList.View(),
		"",
		dependencyHint,
	)
}

func (m usersDashboardModel) renderChangePassword() string {
	username := ""
	if m.activeUser != nil {
		username = m.activeUser.Username
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		fmt.Sprintf("Change Password: %s", username),
		"",
		fmt.Sprintf("New password: %s", m.passwordInput.View()),
	)
}

func (m usersDashboardModel) renderDeleteConfirm() string {
	username := ""
	if m.activeUser != nil {
		username = m.activeUser.Username
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		"Delete User",
		"",
		fmt.Sprintf("Are you sure you want to delete user '%s'?", username),
		confirmHint(),
	)
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

func fetchServersCmd(client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		servers, err := client.ListServers()
		if err != nil {
			return errMsg(err)
		}
		return serversDataMsg(servers)
	}
}

func fetchPermissionsCmd(client *sdk.Client, userID string) tea.Cmd {
	return func() tea.Msg {
		perms, err := client.GetPermissions(userID)
		if err != nil {
			return errMsg(err)
		}
		return permissionsDataMsg(perms)
	}
}

func createUserCmd(client *sdk.Client, username, password string) tea.Cmd {
	return func() tea.Msg {
		_, err := client.CreateUser(username, password)
		if err != nil {
			return errMsg(err)
		}
		return userCreatedMsg{}
	}
}

func updatePasswordCmd(client *sdk.Client, userID, password string) tea.Cmd {
	return func() tea.Msg {
		err := client.UpdatePassword(userID, password)
		if err != nil {
			return errMsg(err)
		}
		return passwordUpdatedMsg{}
	}
}

func setPermissionsCmd(client *sdk.Client, userID string, rows []permissionRow) tea.Cmd {
	return func() tea.Msg {
		payload := make([]sdk.Permission, 0, len(rows))
		for _, row := range rows {
			payload = append(payload, sdk.Permission{
				UserID:          userID,
				ServerID:        row.serverID,
				CanControlPower: row.canControlPower,
				CanViewConsole:  row.canViewConsole,
			})
		}
		err := client.SetPermissions(payload)
		if err != nil {
			return errMsg(err)
		}
		return permissionsUpdatedMsg{}
	}
}

func deleteUserCmd(client *sdk.Client, userID string) tea.Cmd {
	return func() tea.Msg {
		err := client.DeleteUser(userID)
		if err != nil {
			return errMsg(err)
		}
		return userDeletedMsg{}
	}
}
