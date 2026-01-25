package card

import (
	"fmt"
	"gophkeeper/cmd/client/internal/crypto"
	grpcclient "gophkeeper/cmd/client/internal/grpc_client"
	"gophkeeper/cmd/client/internal/ui/models/base"
	"gophkeeper/cmd/client/internal/ui/navigation"
	"gophkeeper/cmd/client/internal/ui/styles"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CardMenuModel управляет меню работы с картами
type CardMenuModel struct {
	base.BaseModel
	token      string
	userID     string
	login      string
	crypt      *crypto.Crypt
	nav        navigation.Navigator
	menuItems  []string
	cursor     int
	grpcClient *grpcclient.GophKeeperClient
}

type (
	SwitchToAddCardMsg  struct{}
	SwitchToCardListMsg struct{}
	CardDeletedMsg      struct{}
)

// NewCardMenuModel создаёт новую модель меню карт
func NewCardMenuModel(token, userID, login string, grpcClient *grpcclient.GophKeeperClient, crypt *crypto.Crypt, nav navigation.Navigator) CardMenuModel {
	return CardMenuModel{
		BaseModel: base.NewBaseModel(),
		token:     token,
		userID:    userID,
		login:     login,
		crypt:     crypt,
		nav:       nav,
		menuItems: []string{
			"Добавить карту",
			"Список карт",
			"Назад",
		},
		cursor:     0,
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
			case 0:
				addCardModel := NewAddCardModel(m.token, m.userID, m.login, m.grpcClient, m.crypt, m.nav)
				return addCardModel, addCardModel.Init()
			case 1:
				cardListModel := NewCardListModel(m.token, m.userID, m.login, m.grpcClient, m.crypt, m.nav)
				return cardListModel, cardListModel.Init()
			case 2:
				return m.nav.BackToMain()
			}
		case "esc":
			return m.nav.BackToMain()
		}
	case tea.WindowSizeMsg:
		m.UpdateSize(msg)
	case CardDeletedMsg:
		m.ClearError()
		m.SetError("Карта успешно удалена")
	}
	return m, nil
}

// View отображает интерфейс
func (m CardMenuModel) View() string {
	title := styles.TitleStyle.Render("Банковские карты")
	subtitle := lipgloss.NewStyle().
		Foreground(styles.SecondaryColor).
		Render(fmt.Sprintf("Пользователь: %s", m.login))

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
