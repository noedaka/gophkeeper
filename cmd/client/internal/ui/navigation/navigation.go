package navigation

import tea "github.com/charmbracelet/bubbletea"

type Navigator interface {
    BackToMain() (tea.Model, tea.Cmd)
}