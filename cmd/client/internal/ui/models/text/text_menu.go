package text

import (
	"fmt"
	grpcclient "gophkeeper/cmd/client/internal/grpc_client"
	"gophkeeper/cmd/client/internal/ui/models/base"
	"gophkeeper/cmd/client/internal/ui/navigation"
	"gophkeeper/cmd/client/internal/ui/styles"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TextMenuModel управляет меню работы с произвольным текстом
type TextMenuModel struct {
	base.BaseModel
	token      string
	userID     string
	login      string
	nav        navigation.Navigator
	menuItems  []string
	cursor     int
	grpcClient *grpcclient.GophKeeperClient
}

// NewTextMenuModel создает новую модель меню работы с произвольным текстом
func NewTextMenuModel(token, userID, login string, grpcClient *grpcclient.GophKeeperClient, nav navigation.Navigator) TextMenuModel {
	return TextMenuModel{
		BaseModel: base.NewBaseModel(),
		token:     token,
		userID:    userID,
		login:     login,
		nav:       nav,
		menuItems: []string{
			"Добавить произвольный текст",
			"Список записей",
			"Назад",
		},
		cursor:     0,
		grpcClient: grpcClient,
	}
}

// Init инициализирует модель
func (m TextMenuModel) Init() tea.Cmd {
	return nil
}

func (m TextMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				addTextModel := NewAddTextModel(m.token, m.userID, m.login, m.grpcClient, m.nav)
				return addTextModel, addTextModel.Init()
			case 1:
				credsListModel := NewTextListModel(m.token, m.userID, m.login, m.grpcClient, m.nav)
				return credsListModel, credsListModel.Init()
			case 2:
				return m.nav.BackToMain()
			}
		case "esc":
			return m.nav.BackToMain()
		}
	case tea.WindowSizeMsg:
		m.UpdateSize(msg)
	}
	return m, nil
}

// View отображает интерфейс
func (m TextMenuModel) View() string {
	title := styles.TitleStyle.Render("Произвольный текст")
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
