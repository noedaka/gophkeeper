package models

import (
	"fmt"
	grpcclient "gophkeeper/cmd/client/internal/grpc_client"
	"gophkeeper/cmd/client/internal/ui/styles"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MainModel - главный экран после успешной аутентификации
type MainModel struct {
	BaseModel
	token      string
	userID     string
	login      string
	grpcClient *grpcclient.GophKeeperClient
	menuItems  []string
	cursor     int
}

// NewMainModel создаёт главную модель
func NewMainModel(token, userID, login string, grpcClient *grpcclient.GophKeeperClient) MainModel {
	return MainModel{
		BaseModel:  NewBaseModel(),
		token:      token,
		userID:     userID,
		login:      login,
		grpcClient: grpcClient,
		menuItems: []string{
			"Логины/Пароли",
			"Банковские карты",
			"Текстовые данные",
			"Бинарные данные",
			"Настройки",
			"Выйти",
		},
		cursor: 0,
	}
}

// Init инициализирует модель
func (m MainModel) Init() tea.Cmd {
	return nil
}

// Update обрабатывает сообщения
func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.Cancel()
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.menuItems)-1 {
				m.cursor++
			}
		case "enter", " ":
			switch m.cursor {
			case 0:
				credsMenuModel := NewCredsMenuModel(m.token, m.userID, m.login, m.grpcClient)
				return credsMenuModel, credsMenuModel.Init()
			case 1: 
				cardMenuModel := NewCardMenuModel(m.token, m.userID, m.login, m.grpcClient)
				return cardMenuModel, cardMenuModel.Init()
			case 2: 
				m.SetError("Раздел 'Текстовые данные' в разработке")
			case 3: 
				m.SetError("Раздел 'Бинарные данные' в разработке")
			case 4: 
				authModel := NewAuthModel(m.grpcClient)
				return authModel, authModel.Init()
			}
		case "esc":
			m.ClearError()
		}
	case tea.WindowSizeMsg:
		m.UpdateSize(msg)
	}
	return m, nil
}

// View отображает интерфейс
func (m MainModel) View() string {
	if m.Width == 0 || m.Height == 0 {
		return "Загрузка..."
	}

	title := styles.TitleStyle.Render("GophKeeper")
	welcome := lipgloss.NewStyle().
		Foreground(styles.SecondaryColor).
		Render(fmt.Sprintf("Вы вошли как: %s", m.login))

	var menuItems []string
	for i, item := range m.menuItems {
		if i == m.cursor {
			menuItems = append(menuItems,
				styles.ButtonActiveStyle.
					Width(30).
					Render("> "+item),
			)
		} else {
			menuItems = append(menuItems,
				lipgloss.NewStyle().
					Padding(0, 3).
					Width(30).
					Render(" "+item),
			)
		}
	}
	menu := lipgloss.JoinVertical(lipgloss.Left, menuItems...)
	menu = lipgloss.NewStyle().MarginTop(2).Render(menu)

	status := lipgloss.NewStyle().
		Foreground(styles.SuccessColor).
		Render("✓ Аутентифицирован")

	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		welcome,
		"",
		menu,
		"",
		status,
		m.RenderError(),
		"",
		styles.HelpStyle.Render("↑/↓: навигация • Enter: выбрать • Esc: сброс"),
	)

	return m.Center(content)
}
