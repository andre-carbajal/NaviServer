package ui

import (
	"fmt"
	"naviserver/pkg/sdk"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type StepState int

const (
	StepPending StepState = iota
	StepRunning
	StepDone
	StepFailed
)

type ProgressStep struct {
	Label       string
	State       StepState
	HasProgress bool
}

type WizardStep int

const (
	StepName WizardStep = iota
	StepLoader
	StepIncludeSnapshots
	StepIncludeUnstable
	StepMCVersion
	StepBuildVersion
	StepLoaderVersion
	StepInstallerVersion
	StepRAM
	StepConfirm
)

type WizardModel struct {
	client                   *sdk.Client
	step                     WizardStep
	nameInput                textinput.Model
	ramInput                 textinput.Model
	loaderList               list.Model
	versionList              list.Model
	boolList                 list.Model
	selectedLoader           string
	selectedVersion          string
	selectedBuildVersion     string
	selectedLoaderVersion    string
	selectedInstallerVersion string
	includeSnapshots         bool
	includeUnstable          bool
	metadata                 *sdk.LoaderMetadata
	width                    int
	height                   int
	err                      error
	creating                 bool
	progress                 progress.Model
	spinner                  spinner.Model
	steps                    []ProgressStep
	progressConn             *websocket.Conn
	requestID                string
	showHelp                 bool
	createState              asyncFlowState
	createStatus             string
}

type asyncFlowState int

const (
	asyncStateIdle asyncFlowState = iota
	asyncStateRunning
	asyncStateDone
	asyncStateFailed
)

type WizardDoneMsg struct{}
type WizardCancelMsg struct{}

type loadersMsg []string
type versionsMsg []string
type loaderMetadataMsg struct {
	data *sdk.LoaderMetadata
}
type serverCreatedMsg struct{}
type progressMsg sdk.ProgressEvent
type progressConnMsg *websocket.Conn
type progressClosedMsg struct {
	err error
}

func NewWizardModel(client *sdk.Client, width, height int) WizardModel {
	tiName := textinput.New()
	tiName.Placeholder = "My Awesome Server"
	tiName.Focus()
	tiName.CharLimit = 32
	tiName.Width = 30

	tiRam := textinput.New()
	tiRam.Placeholder = "2048"
	tiRam.CharLimit = 6
	tiRam.Width = 10

	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 30, 20)
	l.Title = "Select Loader"
	l.SetShowHelp(false)

	v := list.New([]list.Item{}, list.NewDefaultDelegate(), 30, 20)
	v.Title = "Select Option"
	v.SetShowHelp(false)

	b := list.New([]list.Item{item("No"), item("Yes")}, list.NewDefaultDelegate(), 30, 8)
	b.Title = "Select"
	b.SetShowHelp(false)

	prog := progress.New(progress.WithDefaultGradient())
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return WizardModel{
		client:      client,
		step:        StepName,
		nameInput:   tiName,
		ramInput:    tiRam,
		loaderList:  l,
		versionList: v,
		boolList:    b,
		width:       width,
		height:      height,
		progress:    prog,
		spinner:     s,
		createState: asyncStateIdle,
	}
}

func (m WizardModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func (m WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.creating {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			// Allow leaving failed/running screen with Esc.
			if msg.String() == "esc" {
				m.creating = false
				if m.progressConn != nil {
					_ = m.progressConn.Close()
					m.progressConn = nil
				}
				return m, nil
			}
			return m, nil
		case serverCreatedMsg:
			return m, nil
		case progressConnMsg:
			m.progressConn = msg
			return m, waitForProgress(m.progressConn)
		case progressMsg:
			if len(m.steps) == 0 {
				m.steps = append(m.steps, ProgressStep{Label: msg.Message, State: StepRunning})
			} else {
				lastIdx := len(m.steps) - 1
				if m.steps[lastIdx].Label != msg.Message {
					m.steps[lastIdx].State = StepDone
					m.steps = append(m.steps, ProgressStep{Label: msg.Message, State: StepRunning})
				}
			}

			if msg.Message == "Server created successfully" {
				m.createState = asyncStateDone
				m.createStatus = "Done"
				m.steps[len(m.steps)-1].State = StepDone
				time.Sleep(500 * time.Millisecond)
				if m.progressConn != nil {
					_ = m.progressConn.Close()
					m.progressConn = nil
				}
				return m, func() tea.Msg { return WizardDoneMsg{} }
			}
			if msg.Progress == -1 || strings.HasPrefix(strings.ToLower(strings.TrimSpace(msg.Message)), "error:") {
				m.creating = false
				m.createState = asyncStateFailed
				m.createStatus = "Failed"
				m.err = fmt.Errorf("server creation failed: %s", msg.Message)
				if len(m.steps) > 0 {
					m.steps[len(m.steps)-1].State = StepFailed
				}
				if m.progressConn != nil {
					_ = m.progressConn.Close()
					m.progressConn = nil
				}
				return m, nil
			}

			if msg.Progress > 0 {
				m.steps[len(m.steps)-1].HasProgress = true
				cmd = m.progress.SetPercent(msg.Progress / 100)
				return m, tea.Batch(cmd, waitForProgress(m.progressConn))
			}

			m.steps[len(m.steps)-1].HasProgress = false

			return m, waitForProgress(m.progressConn)

		case progressClosedMsg:
			if m.createState == asyncStateDone {
				m.creating = false
				return m, nil
			}

			if msg.err != nil {
				m.err = fmt.Errorf("progress connection closed: %w", msg.err)
			} else {
				m.err = fmt.Errorf("progress connection closed before completion")
			}
			m.creating = false
			m.createState = asyncStateFailed
			m.createStatus = "Failed"
			if len(m.steps) > 0 {
				lastIdx := len(m.steps) - 1
				if m.steps[lastIdx].State == StepRunning {
					m.steps[lastIdx].State = StepFailed
				}
			}
			if m.progressConn != nil {
				_ = m.progressConn.Close()
				m.progressConn = nil
			}
			return m, nil

		case spinner.TickMsg:
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd

		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			m.progress.Width = msg.Width - 20
			return m, nil

		case errMsg:
			m.creating = false
			m.createState = asyncStateFailed
			m.createStatus = "Failed"
			m.err = msg.(error)
			if len(m.steps) > 0 {
				lastIdx := len(m.steps) - 1
				if m.steps[lastIdx].State == StepRunning {
					m.steps[lastIdx].State = StepFailed
				}
			}
			if m.progressConn != nil {
				_ = m.progressConn.Close()
				m.progressConn = nil
			}
			return m, nil
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "esc":
			if m.creating {
				return m, nil
			}
			if m.step > StepName {
				m.step--
				return m, nil
			}
			return m, func() tea.Msg { return WizardCancelMsg{} }
		case "ctrl+c":
			return m, tea.Quit
		}
	case loadersMsg:
		var items []list.Item
		for _, l := range msg {
			items = append(items, item(l))
		}
		m.loaderList.SetItems(items)
		m.loaderList.SetSize(m.width-8, m.height-14)
		m.step = StepLoader
		return m, nil
	case versionsMsg:
		var items []list.Item
		for _, v := range msg {
			items = append(items, item(v))
		}
		m.versionList.SetItems(items)
		m.versionList.SetSize(m.width-8, m.height-14)
		m.step = StepMCVersion
		m.versionList.ResetSelected()
		return m, nil
	case loaderMetadataMsg:
		m.metadata = msg.data
		return m, setupCurrentStepList(&m)
	case errMsg:
		m.err = msg
		return m, nil
	}

	switch m.step {
	case StepName:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEnter {
				if m.nameInput.Value() == "" {
					m.err = fmt.Errorf("name cannot be empty")
					return m, nil
				}
				m.err = nil
				return m, fetchLoaders(m.client)
			}
		}
		m.nameInput, cmd = m.nameInput.Update(msg)
		return m, cmd

	case StepLoader:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEnter {
				i, ok := m.loaderList.SelectedItem().(item)
				if ok {
					m.selectedLoader = string(i)
					m.includeSnapshots = false
					m.includeUnstable = false
					m.selectedVersion = ""
					m.selectedBuildVersion = ""
					m.selectedLoaderVersion = ""
					m.selectedInstallerVersion = ""
					return m, fetchLoaderMetadata(m.client, m.selectedLoader, sdk.LoaderOptions{})
				}
			}
		}
		m.loaderList, cmd = m.loaderList.Update(msg)
		return m, cmd

	case StepIncludeSnapshots:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEnter {
				i, ok := m.boolList.SelectedItem().(item)
				if ok {
					m.includeSnapshots = string(i) == "Yes"
					return m, fetchLoaderMetadata(m.client, m.selectedLoader, m.currentOptions())
				}
			}
		}
		m.boolList, cmd = m.boolList.Update(msg)
		return m, cmd

	case StepIncludeUnstable:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEnter {
				i, ok := m.boolList.SelectedItem().(item)
				if ok {
					m.includeUnstable = string(i) == "Yes"
					return m, fetchLoaderMetadata(m.client, m.selectedLoader, m.currentOptions())
				}
			}
		}
		m.boolList, cmd = m.boolList.Update(msg)
		return m, cmd

	case StepMCVersion:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEnter {
				i, ok := m.versionList.SelectedItem().(item)
				if ok {
					m.selectedVersion = string(i)
					m.step = nextStepAfterSelection(m)
					if m.step == StepRAM {
						m.ramInput.Focus()
						return m, textinput.Blink
					}
					return m, fetchLoaderMetadata(m.client, m.selectedLoader, m.currentOptions())
				}
			}
		}
		m.versionList, cmd = m.versionList.Update(msg)
		return m, cmd

	case StepBuildVersion, StepLoaderVersion, StepInstallerVersion:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEnter {
				i, ok := m.versionList.SelectedItem().(item)
				if ok {
					switch m.step {
					case StepBuildVersion:
						m.selectedBuildVersion = string(i)
					case StepLoaderVersion:
						m.selectedLoaderVersion = string(i)
					case StepInstallerVersion:
						m.selectedInstallerVersion = string(i)
					}
					m.step = nextStepAfterSelection(m)
					if m.step == StepRAM {
						m.ramInput.Focus()
						return m, textinput.Blink
					}
					return m, setupCurrentStepList(&m)
				}
			}
		}
		m.versionList, cmd = m.versionList.Update(msg)
		return m, cmd

	case StepRAM:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEnter {
				val, err := strconv.Atoi(m.ramInput.Value())
				if err != nil || val <= 0 {
					m.err = fmt.Errorf("invalid RAM amount")
					return m, nil
				}
				m.err = nil
				m.step = StepConfirm
				return m, nil
			}
		}
		m.ramInput, cmd = m.ramInput.Update(msg)
		return m, cmd

	case StepConfirm:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "y" || msg.Type == tea.KeyEnter {
				ram, _ := strconv.Atoi(m.ramInput.Value())
				m.creating = true
				m.createState = asyncStateRunning
				m.createStatus = "Running"
				m.err = nil
				m.steps = nil
				m.requestID = uuid.New().String()

				return m, tea.Batch(
					connectToProgress(m.client, m.requestID),
					createServer(m.client, sdk.CreateServerRequest{
						Name:   m.nameInput.Value(),
						Loader: m.selectedLoader,
						LoaderOptions: sdk.LoaderOptions{
							MCVersion:        m.selectedVersion,
							IncludeSnapshots: m.includeSnapshots,
							IncludeUnstable:  m.includeUnstable,
							BuildVersion:     m.selectedBuildVersion,
							LoaderVersion:    m.selectedLoaderVersion,
							InstallerVersion: m.selectedInstallerVersion,
						},
						Ram:       ram,
						RequestID: m.requestID,
					}),
					m.progress.SetPercent(0),
				)
			} else if msg.String() == "n" {
				return m, func() tea.Msg { return WizardCancelMsg{} }
			}
		}
	}

	return m, nil
}

func (m WizardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	title := headerStyle.Width(m.width).Render("CREATE NEW SERVER")

	stepTitle := ""
	content := ""

	if m.err != nil {
		content += lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("Error: %v\n\n", m.err))
	}

	switch m.step {
	case StepName:
		stepTitle = "Enter Server Name"
		content += fmt.Sprintf("\n%s", m.nameInput.View())
	case StepLoader:
		stepTitle = "Select Loader"
		content += "\n" + m.loaderList.View()
	case StepIncludeSnapshots:
		stepTitle = "Show snapshots?"
		content += "\n" + m.boolList.View()
	case StepIncludeUnstable:
		stepTitle = "Show unstable?"
		content += "\n" + m.boolList.View()
	case StepMCVersion:
		stepTitle = fmt.Sprintf("Select MC version for %s", m.selectedLoader)
		content += "\n" + m.versionList.View()
	case StepBuildVersion:
		stepTitle = "Select Paper build version"
		content += "\n" + m.versionList.View()
	case StepLoaderVersion:
		stepTitle = "Select loader version"
		content += "\n" + m.versionList.View()
	case StepInstallerVersion:
		stepTitle = "Select installer version"
		content += "\n" + m.versionList.View()
	case StepRAM:
		stepTitle = "Enter RAM (MB)"
		content += fmt.Sprintf("\n%s", m.ramInput.View())
	case StepConfirm:
		stepTitle = "Confirm Creation"
		content += fmt.Sprintf("\nName: %s\nLoader: %s\nMC Version: %s",
			m.nameInput.Value(), m.selectedLoader, m.selectedVersion)
		if m.selectedBuildVersion != "" {
			content += fmt.Sprintf("\nBuild: %s", m.selectedBuildVersion)
		}
		if m.selectedLoaderVersion != "" {
			content += fmt.Sprintf("\nLoader Version: %s", m.selectedLoaderVersion)
		}
		if m.selectedInstallerVersion != "" {
			content += fmt.Sprintf("\nInstaller Version: %s", m.selectedInstallerVersion)
		}
		if m.includeSnapshots {
			content += "\nInclude snapshots: yes"
		}
		if m.includeUnstable {
			content += "\nInclude unstable: yes"
		}
		content += fmt.Sprintf("\nRAM: %s MB", m.ramInput.Value())
		content = content + "\n" + confirmHint()
	}

	if m.creating {
		content = fmt.Sprintf("\n\nCreating server '%s'...\n\n", m.nameInput.Value())

		statusColor := lipgloss.Color("220")
		statusText := "Running"
		switch m.createState {
		case asyncStateDone:
			statusColor = lipgloss.Color("42")
			statusText = "Done"
		case asyncStateFailed:
			statusColor = lipgloss.Color("196")
			statusText = "Failed"
		}
		if m.createStatus != "" {
			statusText = m.createStatus
		}
		content += lipgloss.NewStyle().Bold(true).Foreground(statusColor).Render("Status: " + statusText)
		content += "\n\n"

		for _, step := range m.steps {
			icon := " "
			labelStyle := lipgloss.NewStyle()

			switch step.State {
			case StepDone:
				icon = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("✓")
				labelStyle = labelStyle.Foreground(lipgloss.Color("240"))
			case StepRunning:
				icon = m.spinner.View()
				labelStyle = labelStyle.Bold(true)
			case StepFailed:
				icon = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("✗")
				labelStyle = labelStyle.Foreground(lipgloss.Color("196"))
			default:
				icon = "•"
			}

			content += fmt.Sprintf(" %s %s\n", icon, labelStyle.Render(step.Label))

			if step.State == StepRunning && step.HasProgress {
				content += fmt.Sprintf("   %s\n", m.progress.View())
			}
		}
	}

	headerBox := baseStyle.
		Width(m.width - 4).
		Align(lipgloss.Center).
		Padding(1).
		Render(titleStyle.Render(stepTitle))

	mainContainer := baseStyle.
		Width(m.width - 4).
		Height(m.height - 12).
		Align(lipgloss.Center).
		Render(content)

	keys := []string{
		keyStyle.Render("esc") + descStyle.Render(": back/cancel"),
		keyStyle.Render("enter") + descStyle.Render(": next"),
		keyStyle.Render("?") + descStyle.Render(": help"),
		keyStyle.Render("ctrl+c") + descStyle.Render(": exit"),
	}
	statusLine := renderInlineKeys(keys)
	footerBox := footerStyle.
		Width(m.width - 4).
		Render(statusLine)

	if m.showHelp {
		helpBody := lipgloss.JoinVertical(lipgloss.Left,
			"Create server wizard",
			"- Enter accepts the current value or selection",
			"- Esc goes back one step or cancels at first step",
			"- Final confirmation uses y/Enter to confirm and n/Esc to cancel",
		)
		helpBox := helpBoxStyle.Width(m.width - 4).Render(helpBody)
		mainContainer = lipgloss.JoinVertical(lipgloss.Left, mainContainer, helpBox)
	}

	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		headerBox,
		mainContainer,
		footerBox,
	)
}

type item string

func (i item) FilterValue() string { return string(i) }
func (i item) Title() string       { return string(i) }
func (i item) Description() string { return "" }

func fetchLoaders(client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		loaders, err := client.ListLoaders()
		if err != nil {
			return errMsg(err)
		}
		return loadersMsg(loaders)
	}
}

func fetchVersions(client *sdk.Client, loader string) tea.Cmd {
	return func() tea.Msg {
		versions, err := client.ListLoaderVersions(loader)
		if err != nil {
			return errMsg(err)
		}
		return versionsMsg(versions)
	}
}

func fetchLoaderMetadata(client *sdk.Client, loader string, options sdk.LoaderOptions) tea.Cmd {
	return func() tea.Msg {
		md, err := client.GetLoaderMetadata(loader, options)
		if err != nil {
			return errMsg(err)
		}
		return loaderMetadataMsg{data: md}
	}
}

func (m WizardModel) currentOptions() sdk.LoaderOptions {
	return sdk.LoaderOptions{
		MCVersion:        m.selectedVersion,
		IncludeSnapshots: m.includeSnapshots,
		IncludeUnstable:  m.includeUnstable,
		BuildVersion:     m.selectedBuildVersion,
		LoaderVersion:    m.selectedLoaderVersion,
		InstallerVersion: m.selectedInstallerVersion,
	}
}

func setupCurrentStepList(m *WizardModel) tea.Cmd {
	if m.metadata == nil {
		return nil
	}
	// step routing
	if m.selectedLoader == "vanilla" && m.step <= StepMCVersion {
		if m.step != StepIncludeSnapshots && m.step != StepMCVersion {
			m.step = StepIncludeSnapshots
		} else if m.step == StepIncludeSnapshots {
			m.step = StepMCVersion
		}
	} else if (m.selectedLoader == "fabric" || m.selectedLoader == "neoforge") && m.step <= StepMCVersion {
		if m.step != StepIncludeUnstable && m.step != StepMCVersion {
			m.step = StepIncludeUnstable
		} else if m.step == StepIncludeUnstable {
			m.step = StepMCVersion
		}
	} else if m.step <= StepMCVersion {
		m.step = StepMCVersion
	}

	if m.selectedVersion == "" && m.metadata.LatestVersion != "" {
		m.selectedVersion = m.metadata.LatestVersion
	}

	switch m.step {
	case StepMCVersion:
		setListItems(&m.versionList, m.metadata.MinecraftVersions, m.width, m.height)
	case StepBuildVersion:
		if m.selectedBuildVersion == "" && len(m.metadata.BuildVersions) > 0 {
			m.selectedBuildVersion = m.metadata.BuildVersions[0]
		}
		setListItems(&m.versionList, m.metadata.BuildVersions, m.width, m.height)
	case StepLoaderVersion:
		if m.selectedLoaderVersion == "" && len(m.metadata.LoaderVersions) > 0 {
			m.selectedLoaderVersion = m.metadata.LoaderVersions[0]
		}
		setListItems(&m.versionList, m.metadata.LoaderVersions, m.width, m.height)
	case StepInstallerVersion:
		if m.selectedInstallerVersion == "" && len(m.metadata.InstallerVersions) > 0 {
			m.selectedInstallerVersion = m.metadata.InstallerVersions[0]
		}
		setListItems(&m.versionList, m.metadata.InstallerVersions, m.width, m.height)
	}
	return nil
}

func nextStepAfterSelection(m WizardModel) WizardStep {
	switch m.selectedLoader {
	case "paper":
		if m.step <= StepMCVersion {
			return StepBuildVersion
		}
	case "fabric", "forge", "neoforge":
		if m.step <= StepMCVersion {
			return StepLoaderVersion
		}
	}
	return StepRAM
}

func setListItems(l *list.Model, values []string, width, height int) {
	items := make([]list.Item, 0, len(values))
	for _, v := range values {
		items = append(items, item(v))
	}
	l.SetItems(items)
	l.SetSize(width-8, height-14)
	l.ResetSelected()
}

func createServer(client *sdk.Client, req sdk.CreateServerRequest) tea.Cmd {
	return func() tea.Msg {
		err := client.CreateServer(req)
		if err != nil {
			return errMsg(err)
		}
		return serverCreatedMsg{}
	}
}

func connectToProgress(client *sdk.Client, id string) tea.Cmd {
	return func() tea.Msg {
		wsURL, err := client.GetWebSocketURL(fmt.Sprintf("/ws/progress/%s", id))
		if err != nil {
			return errMsg(err)
		}

		header := http.Header{}
		header.Set("X-NaviServer-Client", "CLI")

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			conn, _, err = websocket.DefaultDialer.Dial(wsURL, header)
			if err != nil {
				return errMsg(err)
			}
		}
		return progressConnMsg(conn)
	}
}

func waitForProgress(conn *websocket.Conn) tea.Cmd {
	return func() tea.Msg {
		if conn == nil {
			return progressClosedMsg{err: nil}
		}
		var event sdk.ProgressEvent
		err := conn.ReadJSON(&event)
		if err != nil {
			return progressClosedMsg{err: err}
		}
		return progressMsg(event)
	}
}
