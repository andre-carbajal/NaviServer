package ui

import (
	"fmt"
	"log"
	"naviserver/pkg/sdk"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gorilla/websocket"
)

type logModel struct {
	sub           chan logStreamEvent
	conn          *websocket.Conn
	viewport      viewport.Model
	textInput     textinput.Model
	searchInput   textinput.Model
	err           error
	ready         bool
	serverID      string
	server        *sdk.Server
	logLines      []string
	quitting      bool
	back          bool
	client        *sdk.Client
	width         int
	height        int
	showHelp      bool
	autoScroll    bool
	searchMode    bool
	searchQuery   string
	searchMatches []int
	currentMatch  int
	wsState       wsConnState
	wsError       string
}

func initialLogModel(id string, sub chan logStreamEvent, client *sdk.Client) logModel {
	ti := textinput.New()
	ti.Placeholder = "Type a command..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	search := textinput.New()
	search.Placeholder = "Search logs..."
	search.CharLimit = 120
	search.Width = 30

	return logModel{
		sub:          sub,
		textInput:    ti,
		searchInput:  search,
		serverID:     id,
		client:       client,
		autoScroll:   true,
		currentMatch: -1,
		wsState:      wsStateConnecting,
	}
}

func (m logModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		waitForLogEvent(m.sub),
		getServerDetails(m.client, m.serverID),
		tickCmd(),
	)
}

type logMsg string
type errMsg2 error
type serverDetailsMsg *sdk.Server

type wsConnState int

const (
	wsStateUnknown wsConnState = iota
	wsStateConnecting
	wsStateConnected
	wsStateReconnecting
	wsStateDisconnected
)

type logStreamEvent struct {
	log          string
	conn         *websocket.Conn
	connUpdated  bool
	state        wsConnState
	stateUpdated bool
	err          error
}

func waitForLogEvent(sub chan logStreamEvent) tea.Cmd {
	return func() tea.Msg {
		if sub == nil {
			return nil
		}
		event, ok := <-sub
		if !ok {
			return nil
		}
		return event
	}
}

func getServerDetails(client *sdk.Client, id string) tea.Cmd {
	return func() tea.Msg {
		srv, err := client.GetServer(id)
		if err != nil {
			return errMsg2(err)
		}
		return serverDetailsMsg(srv)
	}
}

func (m logModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.searchMode {
			switch msg.String() {
			case "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "esc":
				m.searchMode = false
				m.searchInput.Blur()
				return m, nil
			case "enter":
				m.searchQuery = strings.TrimSpace(m.searchInput.Value())
				m.computeSearchMatches()
				m.rebuildViewportContent()
				m.searchMode = false
				m.searchInput.Blur()
				m.jumpToCurrentMatch()
				return m, nil
			}

			m.searchInput, tiCmd = m.searchInput.Update(msg)
			return m, tiCmd
		}

		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "esc", "q":
			m.back = true
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "a":
			m.autoScroll = !m.autoScroll
			if m.autoScroll {
				m.viewport.GotoBottom()
			}
			return m, nil
		case "/":
			m.searchMode = true
			m.searchInput.SetValue(m.searchQuery)
			m.searchInput.Focus()
			return m, textinput.Blink
		case "n", "N":
			if len(m.searchMatches) > 0 {
				if msg.String() == "N" {
					m.currentMatch--
					if m.currentMatch < 0 {
						m.currentMatch = len(m.searchMatches) - 1
					}
				} else {
					m.currentMatch = (m.currentMatch + 1) % len(m.searchMatches)
				}
				m.rebuildViewportContent()
				m.jumpToCurrentMatch()
			}
			return m, nil
		case "enter":
			if m.textInput.Value() != "" {
				cmd := m.textInput.Value()
				m.textInput.SetValue("")
				if m.conn != nil && m.wsState == wsStateConnected {
					_ = m.conn.WriteMessage(websocket.TextMessage, []byte(cmd+"\n"))
				} else {
					m.wsError = "Cannot send command: WebSocket disconnected"
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 16
		verticalMarginHeight := headerHeight

		contentWidth := msg.Width - 6

		if !m.ready {
			m.viewport = viewport.New(contentWidth, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.ready = true
		} else {
			m.viewport.Width = contentWidth
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

	case logStreamEvent:
		if msg.connUpdated {
			m.conn = msg.conn
		}
		if msg.stateUpdated {
			m.wsState = msg.state
			if msg.err != nil {
				m.wsError = msg.err.Error()
			} else if msg.state == wsStateConnected {
				m.wsError = ""
			}
		}
		if msg.log != "" {
			for _, line := range strings.Split(strings.TrimRight(msg.log, "\n"), "\n") {
				m.logLines = append(m.logLines, line)
			}
			m.computeSearchMatches()
			m.rebuildViewportContent()
			if m.autoScroll && m.searchQuery == "" {
				m.viewport.GotoBottom()
			}
		}
		return m, waitForLogEvent(m.sub)

	case serverDetailsMsg:
		m.server = msg

	case errMsg2:
		m.err = msg
		return m, tea.Quit
	case tickMsg:
		return m, tea.Batch(getServerDetails(m.client, m.serverID), tickCmd())
	}

	m.textInput, tiCmd = m.textInput.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m logModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	title := headerStyle.Width(m.width).Render("SERVER CONSOLE LOGS")

	serverInfoContent := ""
	if m.server != nil {
		statusColor := "160"
		statusIcon := "🔴"
		if m.server.Status == "RUNNING" {
			statusColor = "42"
			statusIcon = "🟢"
		} else if m.server.Status == "STARTING" {
			statusColor = "220"
			statusIcon = "🟡"
		} else if m.server.Status == "STOPPING" {
			statusColor = "208"
			statusIcon = "🟠"
		} else if m.server.Status == "CREATING" {
			statusColor = "51"
			statusIcon = "🔵"
		}

		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor))

		wsIcon, wsColor, wsLabel := m.wsStateInfo()
		wsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(wsColor))

		serverInfoContent = fmt.Sprintf(
			"Server: %s %s  •  ID: %s  •  Port: %d\nLoader: %s %s  •  RAM: %d MB\nWebSocket: %s %s",
			statusIcon,
			statusStyle.Render(m.server.Name),
			m.server.ID,
			m.server.Port,
			m.server.Loader,
			m.server.Version,
			m.server.RAM,
			wsIcon,
			wsStyle.Render(wsLabel),
		)
	} else {
		serverInfoContent = "Loading server details..."
	}

	headerBox := baseStyle.
		Width(m.width-4).
		Align(lipgloss.Center).
		Padding(0, 1).
		Render(serverInfoContent)

	console := baseStyle.
		Width(m.width - 4).
		Render(m.viewport.View())

	keys := []string{
		keyStyle.Render("enter") + descStyle.Render(": send"),
		keyStyle.Render("a") + descStyle.Render(": autoscroll"),
		keyStyle.Render("/") + descStyle.Render(": search"),
		keyStyle.Render("n/N") + descStyle.Render(": next/prev match"),
		keyStyle.Render("q/esc") + descStyle.Render(": back"),
		keyStyle.Render("?") + descStyle.Render(": help"),
		keyStyle.Render("ctrl+c") + descStyle.Render(": quit"),
	}
	helpText := renderInlineKeys(keys)

	inputLine := fmt.Sprintf("→ %s", m.textInput.View())
	if m.searchMode {
		inputLine = fmt.Sprintf("/ %s", m.searchInput.View())
	}

	searchStatus := ""
	if m.searchQuery != "" {
		if len(m.searchMatches) == 0 {
			searchStatus = "Search: 0 matches"
		} else {
			searchStatus = fmt.Sprintf("Search: \"%s\" (%d/%d)", m.searchQuery, m.currentMatch+1, len(m.searchMatches))
		}
	}

	autoscrollStatus := "Autoscroll: OFF"
	if m.autoScroll {
		autoscrollStatus = "Autoscroll: ON"
	}
	metaLine := autoscrollStatus
	if searchStatus != "" {
		metaLine += "  •  " + searchStatus
	}

	helpLine := lipgloss.NewStyle().
		Width(m.width - 6).
		Align(lipgloss.Center).
		Render(helpText)

	footerContent := lipgloss.JoinVertical(lipgloss.Left,
		metaLine,
		"",
		inputLine,
		"",
		helpLine,
	)

	footerBox := footerStyle.
		Width(m.width - 4).
		Align(lipgloss.Left).
		Render(footerContent)

	if m.showHelp {
		helpBody := lipgloss.JoinVertical(lipgloss.Left,
			"Logs view",
			"- Type command and press Enter to send to server console",
			"- Press a to pause/resume autoscroll",
			"- Press / to search logs and n/N to navigate matches",
			"- WebSocket status shows connected/reconnecting/disconnected state",
			"- q/Esc returns to servers list",
			"- Ctrl+C exits the entire TUI",
		)
		helpBox := helpBoxStyle.Width(m.width - 4).Render(helpBody)
		console = lipgloss.JoinVertical(lipgloss.Left, console, helpBox)
	}

	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		headerBox,
		console,
		footerBox,
	)
}

func RunLogs(client *sdk.Client, id string) bool {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	sub := make(chan logStreamEvent, 256)
	stop := make(chan struct{})

	go streamServerLogs(client, id, sub, stop)

	p := tea.NewProgram(
		initialLogModel(id, sub, client),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	m, err := p.Run()
	if err != nil {
		log.Printf("Error running logs UI: %v", err)
		return true
	}

	if logModel, ok := m.(logModel); ok {
		close(stop)
		if logModel.conn != nil {
			_ = logModel.conn.Close()
		}
		return logModel.back
	}
	close(stop)
	return false
}

func (m *logModel) computeSearchMatches() {
	if m.searchQuery == "" {
		m.searchMatches = nil
		m.currentMatch = -1
		return
	}

	q := strings.ToLower(m.searchQuery)
	matches := make([]int, 0)
	for i, line := range m.logLines {
		if strings.Contains(strings.ToLower(line), q) {
			matches = append(matches, i)
		}
	}

	m.searchMatches = matches
	if len(matches) == 0 {
		m.currentMatch = -1
		return
	}
	if m.currentMatch < 0 || m.currentMatch >= len(matches) {
		m.currentMatch = 0
	}
}

func (m *logModel) rebuildViewportContent() {
	if len(m.logLines) == 0 {
		m.viewport.SetContent("")
		return
	}

	if m.searchQuery == "" {
		m.viewport.SetContent(strings.Join(m.logLines, "\n"))
		return
	}

	lines := make([]string, 0, len(m.logLines))
	for idx, line := range m.logLines {
		rendered := highlightSearchTerm(line, m.searchQuery)
		if containsLine(m.searchMatches, idx) {
			prefix := "• "
			if m.currentMatch >= 0 && m.currentMatch < len(m.searchMatches) && m.searchMatches[m.currentMatch] == idx {
				prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true).Render("▶ ")
			} else {
				prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("• ")
			}
			rendered = prefix + rendered
		}
		lines = append(lines, rendered)
	}

	m.viewport.SetContent(strings.Join(lines, "\n"))
}

func (m *logModel) jumpToCurrentMatch() {
	if len(m.searchMatches) == 0 || m.currentMatch < 0 || m.currentMatch >= len(m.searchMatches) {
		return
	}
	line := m.searchMatches[m.currentMatch]
	offset := line - (m.viewport.Height / 2)
	if offset < 0 {
		offset = 0
	}
	m.viewport.SetYOffset(offset)
}

func (m logModel) wsStateInfo() (icon string, color string, label string) {
	switch m.wsState {
	case wsStateConnected:
		return "🟢", "42", "Connected"
	case wsStateReconnecting:
		return "🟡", "220", "Reconnecting"
	case wsStateDisconnected:
		return "🔴", "196", "Disconnected"
	default:
		return "🔵", "51", "Connecting"
	}
}

func highlightSearchTerm(line, query string) string {
	if query == "" {
		return line
	}

	lowerLine := strings.ToLower(line)
	lowerQuery := strings.ToLower(query)
	if !strings.Contains(lowerLine, lowerQuery) {
		return line
	}

	highlight := lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("220")).Bold(true)
	parts := make([]string, 0, 8)
	searchFrom := 0
	for {
		idx := strings.Index(lowerLine[searchFrom:], lowerQuery)
		if idx < 0 {
			parts = append(parts, line[searchFrom:])
			break
		}
		abs := searchFrom + idx
		parts = append(parts, line[searchFrom:abs])
		end := abs + len(query)
		parts = append(parts, highlight.Render(line[abs:end]))
		searchFrom = end
		if searchFrom >= len(line) {
			break
		}
	}
	return strings.Join(parts, "")
}

func containsLine(lines []int, target int) bool {
	for _, line := range lines {
		if line == target {
			return true
		}
	}
	return false
}

func streamServerLogs(client *sdk.Client, id string, out chan<- logStreamEvent, stop <-chan struct{}) {
	defer close(out)

	wsURL, err := client.GetWebSocketURL(fmt.Sprintf("/ws/servers/%s/console", id))
	if err != nil {
		sendLogEvent(out, stop, logStreamEvent{state: wsStateDisconnected, stateUpdated: true, err: err})
		return
	}

	header := http.Header{}
	header.Set("X-NaviServer-Client", "CLI")

	var conn *websocket.Conn
	attempt := 0

	for {
		select {
		case <-stop:
			if conn != nil {
				_ = conn.Close()
			}
			return
		default:
		}

		if conn == nil {
			state := wsStateConnecting
			if attempt > 0 {
				state = wsStateReconnecting
			}
			sendLogEvent(out, stop, logStreamEvent{state: state, stateUpdated: true})

			newConn, _, dialErr := websocket.DefaultDialer.Dial(wsURL, header)
			if dialErr != nil {
				attempt++
				sendLogEvent(out, stop, logStreamEvent{state: wsStateReconnecting, stateUpdated: true, err: dialErr})
				if !waitReconnectBackoff(attempt, stop) {
					return
				}
				continue
			}

			attempt = 0
			conn = newConn
			sendLogEvent(out, stop, logStreamEvent{conn: conn, connUpdated: true, state: wsStateConnected, stateUpdated: true})
		}

		_, message, readErr := conn.ReadMessage()
		if readErr != nil {
			_ = conn.Close()
			conn = nil
			attempt++
			sendLogEvent(out, stop, logStreamEvent{conn: nil, connUpdated: true, state: wsStateReconnecting, stateUpdated: true, err: readErr})
			if !waitReconnectBackoff(attempt, stop) {
				return
			}
			continue
		}

		sendLogEvent(out, stop, logStreamEvent{log: string(message)})
	}
}

func sendLogEvent(out chan<- logStreamEvent, stop <-chan struct{}, event logStreamEvent) {
	select {
	case out <- event:
	case <-stop:
	}
}

func waitReconnectBackoff(attempt int, stop <-chan struct{}) bool {
	delaySeconds := 1
	if attempt > 1 {
		delaySeconds = 2
	}
	if attempt > 2 {
		delaySeconds = 4
	}
	if attempt > 3 {
		delaySeconds = 8
	}

	timer := time.NewTimer(time.Duration(delaySeconds) * time.Second)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-stop:
		return false
	}
}
