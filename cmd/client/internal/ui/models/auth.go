package models

import (
	"fmt"

	grpcclient "gophkeeper/cmd/client/internal/grpc_client"
	"gophkeeper/cmd/client/internal/ui/components"
	"gophkeeper/cmd/client/internal/ui/styles"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AuthModel управляет экраном аутентификации
type AuthModel struct {
	BaseModel
	grpcClient *grpcclient.GophKeeperClient // Добавляем gRPC клиент
	activeTab  string                   // "login" или "register"
	loginEmail components.InputField
	loginPass  components.InputField
	regEmail   components.InputField
	regPass    components.InputField
	regConfirm components.InputField
	submitBtn  string
	cursorPos  int // 0-2 для полей, 3 для кнопки
}

// Сообщения
type (
	SwitchTabMsg   string
	AuthSuccessMsg struct {
		Token  string
		UserID string
		Login  string
	}
	AuthErrorMsg struct {
		Error string
	}
)

// NewAuthModel создаёт новую модель аутентификации
func NewAuthModel(grpcClient *grpcclient.GophKeeperClient) AuthModel {
	m := AuthModel{
		BaseModel:  NewBaseModel(),
		grpcClient: grpcClient,
		activeTab:  "login",
		loginEmail: components.NewInput("Логин", false),
		loginPass:  components.NewInput("Пароль", true),
		regEmail:   components.NewInput("Логин", false),
		regPass:    components.NewInput("Пароль", true),
		regConfirm: components.NewInput("Подтверждение пароля", true),
		submitBtn:  "Войти",
		cursorPos:  0,
	}

	// Начинаем с фокуса на первом поле
	m.loginEmail, _ = m.loginEmail.Focus()

	return m
}

// Init инициализирует модель
func (m AuthModel) Init() tea.Cmd {
	return nil
}

// Update обрабатывает сообщения
func (m AuthModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.Cancel()
			return m, tea.Quit

		case "tab", "shift+tab":
			// Переключение между полями
			m = m.moveCursor(msg.String() == "shift+tab")

		case "left", "right":
			// Переключение между вкладками
			if msg.String() == "left" || msg.String() == "right" {
				if m.activeTab == "login" {
					m.activeTab = "register"
				} else {
					m.activeTab = "login"
				}
				m = m.resetForm()
			}

		case "enter":
			// Нажатие на кнопку отправки
			return m, m.submit()

		case "esc":
			// Сброс фокуса
			m = m.blurAll()
		}

	case tea.WindowSizeMsg:
		m.UpdateSize(msg)

	case SwitchTabMsg:
		m.activeTab = string(msg)
		m = m.resetForm()

	case AuthErrorMsg:
		m.SetError(msg.Error)
		m.SetLoading(false)

	case AuthSuccessMsg:
		// Переход к главному экрану
		mainModel := NewMainModel(msg.Token, msg.UserID, msg.Login)
		return mainModel, mainModel.Init()
	}

	// Обновляем активные поля ввода
	var cmd tea.Cmd
	if m.activeTab == "login" {
		m.loginEmail, cmd = m.loginEmail.Update(msg)
		cmds = append(cmds, cmd)
		m.loginPass, cmd = m.loginPass.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		m.regEmail, cmd = m.regEmail.Update(msg)
		cmds = append(cmds, cmd)
		m.regPass, cmd = m.regPass.Update(msg)
		cmds = append(cmds, cmd)
		m.regConfirm, cmd = m.regConfirm.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View отображает интерфейс
func (m AuthModel) View() string {
	if m.Width == 0 || m.Height == 0 {
		return "Загрузка..."
	}

	// Заголовок
	title := styles.TitleStyle.Render("GophKeeper")
	subtitle := lipgloss.NewStyle().
		Foreground(styles.SecondaryColor).
		MarginBottom(2).
		Render("Безопасное хранение паролей и данных")

	// Статус подключения
	status := ""
	if m.grpcClient == nil {
		status = lipgloss.NewStyle().
			Foreground(styles.WarningColor).
			MarginBottom(1).
			Render("⚠️ Offline режим. Сервер недоступен")
	}

	// Вкладки
	loginTab := " Вход "
	registerTab := " Регистрация "

	if m.activeTab == "login" {
		loginTab = styles.ButtonActiveStyle.Render(loginTab)
		registerTab = styles.ButtonStyle.Render(registerTab)
	} else {
		loginTab = styles.ButtonStyle.Render(loginTab)
		registerTab = styles.ButtonActiveStyle.Render(registerTab)
	}

	tabs := lipgloss.JoinHorizontal(lipgloss.Center, loginTab, registerTab)

	// Форма
	var form string
	if m.activeTab == "login" {
		form = m.renderLoginForm()
	} else {
		form = m.renderRegisterForm()
	}

	// Кнопка отправки
	submitBtn := styles.ButtonActiveStyle.Render(m.submitBtn)

	// Сборка интерфейса
	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		subtitle,
		status,
		"",
		tabs,
		"",
		form,
		"",
		submitBtn,
		m.RenderError(),
		m.RenderLoading(),
		"",
		styles.HelpStyle.Render("Tab/Shift+Tab: перемещение • Enter: подтвердить • ←/→: переключение вкладок"),
	)

	return m.Center(content)
}

// renderLoginForm отображает форму входа
func (m AuthModel) renderLoginForm() string {
	loginLabel := "Логин:"
	passLabel := "Пароль:"

	if m.cursorPos == 0 {
		loginLabel = "->" + loginLabel
	}
	if m.cursorPos == 1 {
		passLabel = "->" + passLabel
	}

	form := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().MarginBottom(1).Render(loginLabel),
		m.loginEmail.View(),
		"",
		lipgloss.NewStyle().MarginBottom(1).Render(passLabel),
		m.loginPass.View(),
	)

	return lipgloss.NewStyle().Width(40).Render(form)
}

// renderRegisterForm отображает форму регистрации
func (m AuthModel) renderRegisterForm() string {
	loginLabel := "Логин:"
	passLabel := "Пароль:"
	confirmLabel := "Подтверждение:"

	switch m.cursorPos {
	case 0:
		loginLabel = "->" + loginLabel
	case 1:
		passLabel = "->" + passLabel
	case 2:
		confirmLabel = "->" + confirmLabel
	}

	form := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().MarginBottom(1).Render(loginLabel),
		m.regEmail.View(),
		"",
		lipgloss.NewStyle().MarginBottom(1).Render(passLabel),
		m.regPass.View(),
		"",
		lipgloss.NewStyle().MarginBottom(1).Render(confirmLabel),
		m.regConfirm.View(),
	)

	return lipgloss.NewStyle().Width(40).Render(form)
}

// moveCursor перемещает курсор между полями
func (m AuthModel) moveCursor(backward bool) AuthModel {
	// Снимаем фокус со всех полей
	m = m.blurAll()

	// Определяем количество полей в активной форме
	fieldCount := 2
	if m.activeTab == "register" {
		fieldCount = 3
	}

	// Обновляем позицию курсора
	if backward {
		m.cursorPos--
		if m.cursorPos < 0 {
			m.cursorPos = fieldCount // Переход на кнопку
		}
	} else {
		m.cursorPos++
		if m.cursorPos > fieldCount {
			m.cursorPos = 0
		}
	}

	// Устанавливаем фокус на соответствующее поле
	if m.cursorPos < fieldCount {
		switch {
		case m.activeTab == "login" && m.cursorPos == 0:
			m.loginEmail, _ = m.loginEmail.Focus()
		case m.activeTab == "login" && m.cursorPos == 1:
			m.loginPass, _ = m.loginPass.Focus()
		case m.activeTab == "register" && m.cursorPos == 0:
			m.regEmail, _ = m.regEmail.Focus()
		case m.activeTab == "register" && m.cursorPos == 1:
			m.regPass, _ = m.regPass.Focus()
		case m.activeTab == "register" && m.cursorPos == 2:
			m.regConfirm, _ = m.regConfirm.Focus()
		}
	}

	// Обновляем текст кнопки
	if m.cursorPos == fieldCount {
		if m.activeTab == "login" {
			m.submitBtn = "Войти " + "->"
		} else {
			m.submitBtn = "Зарегистрироваться " + "->"
		}
	} else {
		if m.activeTab == "login" {
			m.submitBtn = "Войти"
		} else {
			m.submitBtn = "Зарегистрироваться"
		}
	}

	return m
}

// blurAll снимает фокус со всех полей
func (m AuthModel) blurAll() AuthModel {
	m.loginEmail = m.loginEmail.Blur()
	m.loginPass = m.loginPass.Blur()
	m.regEmail = m.regEmail.Blur()
	m.regPass = m.regPass.Blur()
	m.regConfirm = m.regConfirm.Blur()
	return m
}

// resetForm сбрасывает форму при переключении вкладок
func (m AuthModel) resetForm() AuthModel {
	m = m.blurAll()
	m.cursorPos = 0
	m.ClearError()

	if m.activeTab == "login" {
		m.loginEmail, _ = m.loginEmail.Focus()
		m.submitBtn = "Войти"
	} else {
		m.regEmail, _ = m.regEmail.Focus()
		m.submitBtn = "Зарегистрироваться"
	}

	return m
}

// submit отправляет форму
func (m AuthModel) submit() tea.Cmd {
	m.ClearError()

	if m.activeTab == "login" {
		return m.login()
	}
	return m.register()
}

// login выполняет вход
func (m AuthModel) login() tea.Cmd {
	login := m.loginEmail.Model.Value()
	password := m.loginPass.Model.Value()

	// Валидация
	if login == "" || password == "" {
		return func() tea.Msg {
			return AuthErrorMsg{Error: "Заполните все поля"}
		}
	}

	// Проверка подключения
	if m.grpcClient == nil {
		return func() tea.Msg {
			return AuthErrorMsg{Error: "Нет подключения к серверу"}
		}
	}

	m.SetLoading(true, "Выполняется вход...")

	// Асинхронный вызов gRPC
	return func() tea.Msg {
		token, err := m.grpcClient.Login(m.Ctx, login, password)
		if err != nil {
			return AuthErrorMsg{Error: fmt.Sprintf("Ошибка входа: %v", err)}
		}

		return AuthSuccessMsg{
			Token:  token,
			Login:  login,
		}
	}
}

// register выполняет регистрацию
func (m AuthModel) register() tea.Cmd {
	login := m.regEmail.Model.Value()
	password := m.regPass.Model.Value()
	confirm := m.regConfirm.Model.Value()

	// Валидация
	if login == "" || password == "" || confirm == "" {
		return func() tea.Msg {
			return AuthErrorMsg{Error: "Заполните все поля"}
		}
	}

	if len(password) < 6 {
		return func() tea.Msg {
			return AuthErrorMsg{Error: "Пароль должен содержать минимум 6 символов"}
		}
	}

	if password != confirm {
		return func() tea.Msg {
			return AuthErrorMsg{Error: "Пароли не совпадают"}
		}
	}

	// Проверка подключения
	if m.grpcClient == nil {
		return func() tea.Msg {
			return AuthErrorMsg{Error: "Нет подключения к серверу"}
		}
	}

	m.SetLoading(true, "Регистрация...")

	// Асинхронный вызов gRPC
	return func() tea.Msg {
		token, err := m.grpcClient.Register(m.Ctx, login, password)
		if err != nil {
			return AuthErrorMsg{Error: fmt.Sprintf("Ошибка регистрации: %v", err)}
		}

		return AuthSuccessMsg{
			Token:  token,
			Login:  login,
		}
	}
}
