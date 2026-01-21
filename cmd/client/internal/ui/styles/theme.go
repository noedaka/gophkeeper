package styles

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	PrimaryColor   = lipgloss.Color("69")
	SecondaryColor = lipgloss.Color("245")
	SuccessColor   = lipgloss.Color("46")
	ErrorColor     = lipgloss.Color("160")
	WarningColor   = lipgloss.Color("214")

	AppStyle = lipgloss.NewStyle().
			Padding(1, 2)

	TitleStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true).
			Padding(0, 1).
			MarginBottom(1)

	InputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(SecondaryColor).
			Padding(0, 1).
			Width(30)

	InputFocusedStyle = InputStyle.Copy().
				BorderForeground(PrimaryColor)

	ButtonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(SecondaryColor).
			Padding(0, 3).
			Margin(0, 1)

	ButtonActiveStyle = ButtonStyle.Copy().
				Background(PrimaryColor).
				Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			MarginTop(1)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(SuccessColor).
			MarginTop(1)

	WarningStyle = lipgloss.NewStyle().
			Foreground(WarningColor).
			MarginTop(1)

	HelpStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			MarginTop(2)
)
