package ui

import (
	"fmt"
	"log"
	"naviserver/pkg/sdk"
	"net/http"
	"os"
	"os/signal"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gorilla/websocket"
)

type logModel struct {
	sub       chan string
	conn      *websocket.Conn
	viewport  viewport.Model
	textInput textinput.Model
	err       error
	ready     bool
	serverID  string
	server    *sdk.Server
	content   string
	quitting  bool
	back      bool
	client    *sdk.Client
	width     int
	height    int
}

func initialLogModel(id string, conn *websocket.Conn, sub chan string, client *sdk.Client) logModel {
	ti := textinput.New()
	ti.Placeholder = "Type a command..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	return logModel{
		sub:       sub,
		conn:      conn,
		textInput: ti,
		serverID:  id,
		client:    client,
	}
}

func (m logModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		waitForLog(m.sub),
		getServerDetails(m.client, m.serverID),
		tickCmd(),
	)
}

type logMsg string
type errMsg2 error
type serverDetailsMsg *sdk.Server

func waitForLog(sub chan string) tea.Cmd {
	return func() tea.Msg {
		if sub == nil {
			return nil
		}
		msg, ok := <-sub
		if !ok {
			return nil
		}
		return logMsg(msg)
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
		switch msg.Type {
		case tea.KeyCtrlC:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEsc:
			m.back = true
			return m, tea.Quit
		case tea.KeyEnter:
			if m.textInput.Value() != "" {
				cmd := m.textInput.Value()
				m.textInput.SetValue("")
				if m.conn != nil {
					_ = m.conn.WriteMessage(websocket.TextMessage, []byte(cmd+"\n"))
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

	case logMsg:
		m.content += string(msg) + "\n"
		m.viewport.SetContent(m.content)
		m.viewport.GotoBottom()
		return m, waitForLog(m.sub)

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

	title := headerStyle.Width(m.width).Render("SEVER CONSOLE LOGS")

	serverInfoContent := ""
	if m.server != nil {
		statusColor := "160"
		statusIcon := "ÃƒÂ°Ã…Â¸Ã¢â‚¬ÂÃ‚Â´"
		if m.server.Status == "RUNNING" {
			statusColor = "42"
			statusIcon = "ÃƒÂ°Ã…Â¸Ã…Â¸Ã‚Â¢"
		} else if m.server.Status == "STARTING" {
			statusColor = "220"
			statusIcon = "ÃƒÂ°Ã…Â¸Ã…Â¸Ã‚Â¡"
		}

		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor))

		serverInfoContent = fmt.Sprintf(
			"Server: %s %s  ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â¢  ID: %s  ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â¢  Port: %d\nLoader: %s %s  ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â¢  RAM: %d MB",
			statusIcon,
			statusStyle.Render(m.server.Name),
			m.server.ID,
			m.server.Port,
			m.server.Loader,
			m.server.Version,
			m.server.RAM,
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
		keyStyle.Render("esc") + descStyle.Render(": back"),
		keyStyle.Render("ctrl+c") + descStyle.Render(": quit"),
	}
	helpText := lipgloss.JoinHorizontal(lipgloss.Top, keys...)
	helpText = ""
	for i, k := range keys {
		helpText += k
		if i < len(keys)-1 {
			helpText += lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(" ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â¢ ")
		}
	}

	inputLine := fmt.Sprintf("ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ %s", m.textInput.View())

	helpLine := lipgloss.NewStyle().
		Width(m.width - 6).
		Align(lipgloss.Center).
		Render(helpText)

	footerContent := lipgloss.JoinVertical(lipgloss.Left,
		inputLine,
		"\n",
		helpLine,
	)

	footerBox := footerStyle.
		Width(m.width - 4).
		Align(lipgloss.Left).
		Render(footerContent)

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

	wsURL, err := client.GetWebSocketURL(fmt.Sprintf("/ws/servers/%s/console", id))
	if err != nil {
		log.Fatal("Error parsing base URL:", err)
	}

	header := http.Header{}
	header.Set("X-NaviServer-Client", "CLI")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		fmt.Printf("Error connecting to logs: %v\nPress Enter to continue...", err)
		fmt.Scanln()
		return true
	}
	defer conn.Close()

	sub := make(chan string)

	go func() {
		defer close(sub)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			sub <- string(message)
		}
	}()

	p := tea.NewProgram(
		initialLogModel(id, conn, sub, client),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	m, err := p.Run()
	if err != nil {
		log.Printf("Error running logs UI: %v", err)
		return true
	}

	if logModel, ok := m.(logModel); ok {
		return logModel.back
	}
	return false
}
