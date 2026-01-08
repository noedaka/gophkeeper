package models

import (
	"fmt"

	"gophkeeper/cmd/client/internal/ui/styles"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MainModel - главный экран после успешной аутентификации
type MainModel struct {
	BaseModel
	token     string
	userID    string
	login     string
	menuItems []string
	cursor    int
}

// NewMainModel создаёт главную модель
func NewMainModel(token, userID, login string) MainModel {
	return MainModel{
		BaseModel: NewBaseModel(),
		token:     token,
		userID:    userID,
		login:     login,
		menuItems: []string{
			"Мои пароли",
			"Текстовые данные",
			"Банковские карты",
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
			case 5: // Выйти
				// Возвращаемся к экрану авторизации
				// TODO: Сделать Logout запрос к серверу
				authModel := NewAuthModel(nil) // Здесь нужно передать реального клиента
				return authModel, authModel.Init()
			default:
				// Пока просто показываем заглушку
				m.SetError(fmt.Sprintf("Раздел '%s' в разработке", m.menuItems[m.cursor]))
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

	// Заголовок
	title := styles.TitleStyle.Render("GophKeeper")
	welcome := lipgloss.NewStyle().
		Foreground(styles.SecondaryColor).
		Render(fmt.Sprintf("Вы вошли как: %s (ID: %s)", m.login, m.userID))

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
					Render("  "+item),
			)
		}
	}

	menu := lipgloss.JoinVertical(lipgloss.Left, menuItems...)
	menu = lipgloss.NewStyle().MarginTop(2).Render(menu)

	// Статус
	status := lipgloss.NewStyle().
		Foreground(styles.SuccessColor).
		Render("✓ Аутентифицирован")

	// Токен (скрытый)
	tokenPreview := lipgloss.NewStyle().
		Foreground(styles.SecondaryColor).
		Faint(true).
		Render(fmt.Sprintf("Токен: %s...", m.token[:min(10, len(m.token))]))

	// Сборка интерфейса
	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		welcome,
		"",
		menu,
		"",
		status,
		"",
		tokenPreview,
		m.RenderError(),
		"",
		styles.HelpStyle.Render("↑/↓: навигация • Enter: выбрать • Esc: сброс"),
	)

	return m.Center(content)
}

// Вспомогательная функция для безопасного получения подстроки
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}