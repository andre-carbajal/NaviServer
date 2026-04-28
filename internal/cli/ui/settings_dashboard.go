package ui

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"naviserver/pkg/sdk"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const estimatedBytesPerLogLine = 200

type settingsListItem struct {
	field       settingsField
	title       string
	description string
}

func (i settingsListItem) FilterValue() string { return i.title + " " + i.description }
func (i settingsListItem) Title() string       { return i.title }
func (i settingsListItem) Description() string { return i.description }

type settingsIPItem struct {
	value       string
	title       string
	description string
}

func (i settingsIPItem) FilterValue() string { return i.value + " " + i.title + " " + i.description }
func (i settingsIPItem) Title() string       { return i.title }
func (i settingsIPItem) Description() string { return i.description }

type settingsMode int

const (
	settingsModeList settingsMode = iota
	settingsModeNetworkConfig
	settingsModeEditNumber
	settingsModeSelectPublicIP
)

type settingsField int

const (
	settingsFieldNetwork settingsField = iota
	settingsFieldPublicAddress
	settingsFieldLogBuffer
	settingsFieldStartPort
	settingsFieldEndPort
)

type settingsDataMsg struct {
	portRange  sdk.PortRange
	logBuffer  sdk.LogBufferSettings
	publicIP   sdk.PublicAddressSettings
	interfaces sdk.NetworkInterfaces
}

type settingsDashboardModel struct {
	client         *sdk.Client
	list           list.Model
	networkList    list.Model
	ipList         list.Model
	input          textinput.Model
	width          int
	height         int
	err            error
	message        string
	showHelp       bool
	mode           settingsMode
	editReturnMode settingsMode
	networkMessage string
	editField      settingsField
	portRange      sdk.PortRange
	logBuffer      sdk.LogBufferSettings
	publicIP       sdk.PublicAddressSettings
	interfaces     sdk.NetworkInterfaces
}

func RunSettingsDashboard(client *sdk.Client) {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Settings"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	l.Styles.HelpStyle = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)

	ipList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	ipList.Title = "Select Public Address"
	ipList.SetShowStatusBar(false)
	ipList.SetFilteringEnabled(false)
	ipList.SetShowHelp(false)
	ipList.Styles.Title = titleStyle
	ipList.Styles.PaginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	ipList.Styles.HelpStyle = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)

	in := textinput.New()
	in.CharLimit = 12
	in.Width = 20

	m := settingsDashboardModel{
		client:      client,
		list:        l,
		networkList: l,
		ipList:      ipList,
		input:       in,
		mode:        settingsModeList,
	}
	m.networkList = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	m.networkList.Title = "Network Configuration"
	m.networkList.SetShowStatusBar(false)
	m.networkList.SetFilteringEnabled(false)
	m.networkList.SetShowHelp(false)
	m.networkList.Styles.Title = titleStyle
	m.networkList.Styles.PaginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	m.networkList.Styles.HelpStyle = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running settings dashboard: %v", err)
		os.Exit(1)
	}
}

func (m settingsDashboardModel) Init() tea.Cmd {
	return fetchSettingsDataCmd(m.client)
}

func (m settingsDashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		switch m.mode {
		case settingsModeEditNumber:
			switch msg.String() {
			case "esc":
				m.mode = m.editReturnMode
				m.input.Blur()
				m.message = "Edit cancelled"
				return m, nil
			case "enter":
				return m.saveNumericField()
			}

			m.input, cmd = m.input.Update(msg)
			return m, cmd

		case settingsModeSelectPublicIP:
			switch msg.String() {
			case "esc":
				m.mode = settingsModeList
				m.message = "Selection cancelled"
				return m, nil
			case "enter":
				item, ok := m.ipList.SelectedItem().(settingsIPItem)
				if !ok {
					return m, nil
				}
				if err := m.client.SetPublicIP(item.value); err != nil {
					m.err = err
					return m, nil
				}
				m.err = nil
				m.publicIP.PublicIP = item.value
				m.mode = settingsModeList
				m.message = fmt.Sprintf("Public address updated to %s", item.value)
				m.updateList()
				return m, nil
			case "?":
				m.showHelp = !m.showHelp
				return m, nil
			}

			m.ipList, cmd = m.ipList.Update(msg)
			return m, cmd
		case settingsModeNetworkConfig:
			switch msg.String() {
			case "esc":
				m.mode = settingsModeList
				m.message = ""
				return m, nil
			case "enter":
				i, ok := m.networkList.SelectedItem().(settingsListItem)
				if !ok {
					return m, nil
				}
				switch i.field {
				case settingsFieldStartPort:
					m.editField = settingsFieldStartPort
					m.input.SetValue(fmt.Sprintf("%d", m.portRange.Start))
					m.input.Placeholder = "Start Port"
					m.input.Focus()
					m.mode = settingsModeEditNumber
					m.editReturnMode = settingsModeNetworkConfig
					m.networkMessage = "Editing start port"
					return m, textinput.Blink
				case settingsFieldEndPort:
					m.editField = settingsFieldEndPort
					m.input.SetValue(fmt.Sprintf("%d", m.portRange.End))
					m.input.Placeholder = "End Port"
					m.input.Focus()
					m.mode = settingsModeEditNumber
					m.editReturnMode = settingsModeNetworkConfig
					m.networkMessage = "Editing end port"
					return m, textinput.Blink
				}
			case "?":
				m.showHelp = !m.showHelp
				return m, nil
			}

			m.networkList, cmd = m.networkList.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "r":
			m.message = "Refreshing settings..."
			return m, fetchSettingsDataCmd(m.client)
		case "enter":
			i, ok := m.list.SelectedItem().(settingsListItem)
			if !ok {
				return m, nil
			}
			switch i.field {
			case settingsFieldNetwork:
				m.mode = settingsModeNetworkConfig
				m.message = ""
				m.updateNetworkList()
				return m, nil
			case settingsFieldPublicAddress:
				m.mode = settingsModeSelectPublicIP
				m.buildIPList()
				m.message = "Select a public address"
				return m, nil
			case settingsFieldLogBuffer:
				m.editField = settingsFieldLogBuffer
				m.input.SetValue(fmt.Sprintf("%d", m.logBuffer.LogBufferSize))
				m.input.Placeholder = "Log buffer lines"
				m.input.Focus()
				m.mode = settingsModeEditNumber
				m.editReturnMode = settingsModeList
				m.message = "Editing console log buffer"
				return m, textinput.Blink
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width - 4)
		m.list.SetHeight(msg.Height - 12)
		m.networkList.SetWidth(msg.Width - 4)
		m.networkList.SetHeight(msg.Height - 12)
		m.ipList.SetWidth(msg.Width - 4)
		m.ipList.SetHeight(msg.Height - 12)
	case settingsDataMsg:
		m.portRange = msg.portRange
		m.logBuffer = msg.logBuffer
		m.publicIP = msg.publicIP
		m.interfaces = msg.interfaces
		m.err = nil
		m.updateList()
		m.updateNetworkList()
		return m, nil
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m settingsDashboardModel) saveNumericField() (tea.Model, tea.Cmd) {
	value := strings.TrimSpace(m.input.Value())
	n, err := strconv.Atoi(value)
	if err != nil {
		m.err = fmt.Errorf("value must be a valid integer")
		return m, nil
	}

	switch m.editField {
	case settingsFieldStartPort:
		if n <= 0 {
			m.err = fmt.Errorf("start port must be greater than 0")
			return m, nil
		}
		portRange, err := m.client.GetPortRange()
		if err != nil {
			m.err = err
			return m, nil
		}
		if n > portRange.End {
			m.err = fmt.Errorf("start port cannot be greater than end port")
			return m, nil
		}
		if err := m.client.SetPortRange(n, portRange.End); err != nil {
			m.err = err
			return m, nil
		}
		m.portRange.Start = n
		m.portRange.End = portRange.End
		m.mode = m.editReturnMode
		m.input.Blur()
		m.err = nil
		m.message = fmt.Sprintf("Start port updated to %d", n)
		m.networkMessage = m.message
		m.updateList()
		m.updateNetworkList()
		return m, nil

	case settingsFieldEndPort:
		if n <= 0 {
			m.err = fmt.Errorf("end port must be greater than 0")
			return m, nil
		}
		portRange, err := m.client.GetPortRange()
		if err != nil {
			m.err = err
			return m, nil
		}
		if n < portRange.Start {
			m.err = fmt.Errorf("end port cannot be less than start port")
			return m, nil
		}
		if err := m.client.SetPortRange(portRange.Start, n); err != nil {
			m.err = err
			return m, nil
		}
		m.portRange.Start = portRange.Start
		m.portRange.End = n
		m.mode = m.editReturnMode
		m.input.Blur()
		m.err = nil
		m.message = fmt.Sprintf("Port range updated to %d-%d", m.portRange.Start, m.portRange.End)
		m.updateList()
		m.networkMessage = m.message
		m.updateNetworkList()
		return m, nil

	case settingsFieldLogBuffer:
		if n < 0 {
			m.err = fmt.Errorf("console log buffer must be >= 0")
			return m, nil
		}
		if err := m.client.SetLogBufferSize(n); err != nil {
			m.err = err
			return m, nil
		}
		m.logBuffer.LogBufferSize = n
		m.mode = settingsModeList
		m.input.Blur()
		m.err = nil
		m.message = fmt.Sprintf("Console log buffer updated to %d", n)
		m.updateList()
		return m, nil
	}

	return m, nil
}

func (m *settingsDashboardModel) updateList() {
	estimatedBytes := int64(m.logBuffer.LogBufferSize * estimatedBytesPerLogLine)
	publicAddress := m.publicIP.PublicIP
	if publicAddress == "" {
		publicAddress = "localhost"
	}
	if publicAddress != "localhost" && !containsIP(m.interfaces.Interfaces, publicAddress) {
		publicAddress = publicAddress + " (unavailable)"
	}

	items := []list.Item{
		settingsListItem{
			field:       settingsFieldNetwork,
			title:       "Network Configuration",
			description: fmt.Sprintf("Current range: %d-%d (%d ports)", m.portRange.Start, m.portRange.End, m.portRange.End-m.portRange.Start+1),
		},
		settingsListItem{
			field:       settingsFieldPublicAddress,
			title:       "Public Address",
			description: fmt.Sprintf("Current: %s | Interfaces: %d | Enter to select", publicAddress, len(uniqueSortedIPs(m.interfaces.Interfaces))),
		},
		settingsListItem{
			field:       settingsFieldLogBuffer,
			title:       "Console Log Buffer",
			description: fmt.Sprintf("Lines: %d | Estimated memory: %s (~%d bytes/line) | Enter to edit", m.logBuffer.LogBufferSize, formatBytesShort(estimatedBytes), estimatedBytesPerLogLine),
		},
	}
	m.list.SetItems(items)
}

func (m *settingsDashboardModel) updateNetworkList() {
	items := []list.Item{
		settingsListItem{
			field:       settingsFieldStartPort,
			title:       "Start Port",
			description: fmt.Sprintf("Current: %d | Enter to edit", m.portRange.Start),
		},
		settingsListItem{
			field:       settingsFieldEndPort,
			title:       "End Port",
			description: fmt.Sprintf("Current: %d | Enter to edit", m.portRange.End),
		},
	}
	m.networkList.SetItems(items)
}

func (m *settingsDashboardModel) buildIPList() {
	items := make([]list.Item, 0)
	items = append(items, settingsIPItem{
		value:       "localhost",
		title:       "localhost",
		description: "default",
	})

	for _, ip := range uniqueSortedIPs(m.interfaces.Interfaces) {
		items = append(items, settingsIPItem{
			value:       ip,
			title:       ip,
			description: "available",
		})
	}

	if m.publicIP.PublicIP != "" && m.publicIP.PublicIP != "localhost" && !containsIP(m.interfaces.Interfaces, m.publicIP.PublicIP) {
		items = append(items, settingsIPItem{
			value:       m.publicIP.PublicIP,
			title:       m.publicIP.PublicIP,
			description: "unavailable",
		})
	}

	m.ipList.SetItems(items)

	selectedIndex := 0
	for i, item := range items {
		ipItem, ok := item.(settingsIPItem)
		if ok && ipItem.value == m.publicIP.PublicIP {
			selectedIndex = i
			break
		}
	}
	m.ipList.Select(selectedIndex)
}

func (m settingsDashboardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	title := headerStyle.Width(m.width).Render("SETTINGS DASHBOARD")

	estimatedBytes := int64(m.logBuffer.LogBufferSize * estimatedBytesPerLogLine)
	summary := fmt.Sprintf("Daemon: %s\nEstimated log buffer memory: %s", m.client.BaseURL(), formatBytesShort(estimatedBytes))
	headerBox := baseStyle.
		Width(m.width - 4).
		Align(lipgloss.Center).
		Render(summary)

	listBox := ""
	keys := []string{}

	switch m.mode {
	case settingsModeList:
		listBox = baseStyle.
			Width(m.width - 4).
			Height(m.height - 12).
			Render(m.list.View())
		keys = []string{
			keyStyle.Render("enter") + descStyle.Render(": edit/select"),
			keyStyle.Render("r") + descStyle.Render(": refresh"),
			keyStyle.Render("?") + descStyle.Render(": help"),
			keyStyle.Render("q/esc") + descStyle.Render(": back"),
			keyStyle.Render("ctrl+c") + descStyle.Render(": exit"),
		}

	case settingsModeNetworkConfig:
		networkSummary := fmt.Sprintf("Current range: %d-%d (%d ports)", m.portRange.Start, m.portRange.End, m.portRange.End-m.portRange.Start+1)
		content := lipgloss.JoinVertical(lipgloss.Left, networkSummary, "", m.networkList.View())
		listBox = baseStyle.
			Width(m.width - 4).
			Height(m.height - 12).
			Render(content)
		keys = []string{
			keyStyle.Render("enter") + descStyle.Render(": edit"),
			keyStyle.Render("?") + descStyle.Render(": help"),
			keyStyle.Render("esc") + descStyle.Render(": back"),
			keyStyle.Render("ctrl+c") + descStyle.Render(": exit"),
		}

	case settingsModeEditNumber:
		titleText := "Edit Value"
		if m.editField == settingsFieldStartPort {
			titleText = "Edit Start Port"
		} else if m.editField == settingsFieldEndPort {
			titleText = "Edit End Port"
		} else if m.editField == settingsFieldLogBuffer {
			titleText = "Edit Console Log Buffer"
		}

		content := lipgloss.JoinVertical(lipgloss.Left,
			titleText,
			"",
			m.input.View(),
		)
		listBox = baseStyle.
			Width(m.width-4).
			Height(m.height-12).
			Align(lipgloss.Center, lipgloss.Center).
			Render(content)

		keys = []string{
			keyStyle.Render("enter") + descStyle.Render(": save"),
			keyStyle.Render("esc") + descStyle.Render(": cancel"),
			keyStyle.Render("ctrl+c") + descStyle.Render(": exit"),
		}

	case settingsModeSelectPublicIP:
		listBox = baseStyle.
			Width(m.width - 4).
			Height(m.height - 12).
			Render(m.ipList.View())
		keys = []string{
			keyStyle.Render("enter") + descStyle.Render(": select and save"),
			keyStyle.Render("esc") + descStyle.Render(": cancel"),
			keyStyle.Render("?") + descStyle.Render(": help"),
			keyStyle.Render("ctrl+c") + descStyle.Render(": exit"),
		}
	}

	footerBox := footerStyle.Width(m.width - 4).Render(renderInlineKeys(keys))

	if m.showHelp {
		helpBody := ""
		switch m.mode {
		case settingsModeList:
			helpBody = lipgloss.JoinVertical(lipgloss.Left,
				"Settings dashboard",
				"- Enter edits or selects the current setting",
				"- Network Configuration opens Start/End port editor",
				"- Public Address: select localhost or an interface IP",
				"- Log Buffer: uses estimated memory of ~200 bytes/line",
			)
		case settingsModeNetworkConfig:
			helpBody = lipgloss.JoinVertical(lipgloss.Left,
				"Network configuration",
				"- Start and End Port are edited independently",
				"- Enter saves each field immediately",
				"- Esc returns to settings list",
			)
		case settingsModeEditNumber:
			helpBody = lipgloss.JoinVertical(lipgloss.Left,
				"Edit mode",
				"- Type a numeric value",
				"- Enter saves immediately",
				"- Esc cancels editing",
			)
		case settingsModeSelectPublicIP:
			helpBody = lipgloss.JoinVertical(lipgloss.Left,
				"Public address selection",
				"- Choose localhost or an available interface",
				"- Unavailable label means saved value is not currently detected",
				"- Enter saves selection immediately",
			)
		}
		helpBox := helpBoxStyle.Width(m.width - 4).Render(helpBody)
		listBox = lipgloss.JoinVertical(lipgloss.Left, listBox, helpBox)
	}

	if m.message != "" {
		footerBox = fmt.Sprintf("%s\n%s", statusMessage(m.message), footerBox)
	}

	if m.mode == settingsModeNetworkConfig && m.networkMessage != "" && m.networkMessage != m.message {
		footerBox = fmt.Sprintf("%s\n%s", statusMessage(m.networkMessage), footerBox)
	}

	if m.err != nil {
		footerBox = fmt.Sprintf("%s\n%s", errorMessage(m.err.Error()), footerBox)
	}

	return lipgloss.JoinVertical(lipgloss.Center, title, headerBox, listBox, footerBox)
}

func uniqueSortedIPs(ips []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(ips))
	for _, ip := range ips {
		if ip == "" {
			continue
		}
		if _, ok := seen[ip]; ok {
			continue
		}
		seen[ip] = struct{}{}
		out = append(out, ip)
	}
	sort.Strings(out)
	return out
}

func containsIP(ips []string, target string) bool {
	for _, ip := range ips {
		if ip == target {
			return true
		}
	}
	return false
}

func fetchSettingsDataCmd(client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		portRange, err := client.GetPortRange()
		if err != nil {
			return errMsg(err)
		}

		logBuffer, err := client.GetLogBufferSize()
		if err != nil {
			return errMsg(err)
		}

		publicIP, err := client.GetPublicIP()
		if err != nil {
			return errMsg(err)
		}

		interfaces, err := client.GetNetworkInterfaces()
		if err != nil {
			return errMsg(err)
		}

		return settingsDataMsg{
			portRange:  *portRange,
			logBuffer:  *logBuffer,
			publicIP:   *publicIP,
			interfaces: *interfaces,
		}
	}
}
