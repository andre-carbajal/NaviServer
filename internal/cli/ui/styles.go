package ui

import "github.com/charmbracelet/lipgloss"

var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62")).
			Bold(true).
			Padding(0, 1).
			Align(lipgloss.Center)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62")).
			Bold(true).
			Padding(0, 1)

	keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true)

	descStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	footerStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1).
			Align(lipgloss.Center)

	helpBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("244")).
			Padding(0, 1)

	statusMessageStyle = lipgloss.NewStyle().
				MarginLeft(2).
				Foreground(lipgloss.Color("42")).
				Bold(true)

	errorMessageStyle = lipgloss.NewStyle().
				MarginLeft(2).
				Foreground(lipgloss.Color("196")).
				Bold(true)
)

func renderInlineKeys(keys []string) string {
	line := ""
	for i, k := range keys {
		line += k
		if i < len(keys)-1 {
			line += lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(" • ")
		}
	}
	return line
}

func statusMessage(msg string) string {
	if msg == "" {
		return ""
	}
	return statusMessageStyle.Render("Status: " + msg)
}

func errorMessage(msg string) string {
	if msg == "" {
		return ""
	}
	return errorMessageStyle.Render("Error: " + msg)
}

func confirmHint() string {
	return "Confirm: y/Enter | Cancel: n/Esc"
}
