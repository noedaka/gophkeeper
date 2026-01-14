package models

import (
	"fmt"
	grpcclient "gophkeeper/cmd/client/internal/grpc_client"
	"gophkeeper/cmd/client/internal/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CardMenuModel управляет меню работы с картами
type CardMenuModel struct {
	BaseModel
	token    string
	userID   string
	login    string
	menuItems []string
	cursor   int
	grpcClient *grpcclient.GophKeeperClient
}

// Сообщения для навигации
type (
	SwitchToAddCardMsg struct{}
	SwitchToCardListMsg struct{}
	CardDeletedMsg struct{}
)

// NewCardMenuModel создаёт новую модель меню карт
func NewCardMenuModel(token, userID, login string, grpcClient *grpcclient.GophKeeperClient) CardMenuModel {
	return CardMenuModel{
		BaseModel: NewBaseModel(),
		token:    token,
		userID:   userID,
		login:    login,
		menuItems: []string{
			"Добавить карту",
			"Список карт",
			"Назад",
		},
		cursor: 0,
		grpcClient: grpcClient,
	}
}

// Init инициализирует модель
func (m CardMenuModel) Init() tea.Cmd {
	return nil
}

// Update обрабатывает сообщения
func (m CardMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			case 0: // Добавить карту
				addCardModel := NewAddCardModel(m.token, m.userID, m.login, m.grpcClient)
				return addCardModel, addCardModel.Init()
			case 1: // Список карт
				cardListModel := NewCardListModel(m.token, m.userID, m.login, m.grpcClient)
				return cardListModel, cardListModel.Init()
			case 2: // Назад
				mainModel := NewMainModel(m.token, m.userID, m.login, m.grpcClient)
				return mainModel, mainModel.Init()
			}
		case "esc":
			mainModel := NewMainModel(m.token, m.userID, m.login, m.grpcClient)
			return mainModel, mainModel.Init()
		}
	case tea.WindowSizeMsg:
		m.UpdateSize(msg)
	case CardDeletedMsg:
		// Обновляем список карт после удаления
		m.ClearError()
		m.SetError("Карта успешно удалена")
	}
	return m, nil
}

// View отображает интерфейс
func (m CardMenuModel) View() string {
	if m.Width == 0 || m.Height == 0 {
		return "Загрузка..."
	}

	// Заголовок
	title := styles.TitleStyle.Render("Банковские карты")
	subtitle := lipgloss.NewStyle().
		Foreground(styles.SecondaryColor).
		Render(fmt.Sprintf("Пользователь: %s", m.login))

	// Меню
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

	// Сборка интерфейса
	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		subtitle,
		"",
		menu,
		"",
		m.RenderError(),
		"",
		styles.HelpStyle.Render("↑/↓: навигация • Enter: выбрать • Esc: назад"),
	)

	return m.Center(content)
}