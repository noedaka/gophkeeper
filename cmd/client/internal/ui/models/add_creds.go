// add_creds.go (новый файл, копия AddCardModel с адаптацией)

package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	grpcclient "gophkeeper/cmd/client/internal/grpc_client"
	"gophkeeper/cmd/client/internal/ui/components"
	"gophkeeper/cmd/client/internal/ui/styles"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/proto"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AddCredsModel управляет формой добавления логина/пароля
type AddCredsModel struct {
	BaseModel
	token      string
	userID     string
	login      string
	grpcClient *grpcclient.GophKeeperClient

	username components.InputField
	password components.InputField
	service  components.InputField
	metadata components.InputField

	cursorPos int // 0-3 поля, 4 кнопка
	submitBtn string
}

type (
	CredsAddedMsg struct {
		CredID int32
	}
	CredsAddErrorMsg struct {
		Error string
	}
)

func NewAddCredsModel(token, userID, login string, grpcClient *grpcclient.GophKeeperClient) AddCredsModel {
	m := AddCredsModel{
		BaseModel:  NewBaseModel(),
		token:      token,
		userID:     userID,
		login:      login,
		grpcClient: grpcClient,

		username: components.NewInput("Логин/Email", false),
		password: components.NewInput("Пароль", true),
		service:  components.NewInput("Сервис/Сайт", false),
		metadata: components.NewInput("Метка/Название", false),

		cursorPos: 0,
		submitBtn: "Сохранить запись",
	}

	m.username, _ = m.username.Focus()

	return m
}

func (m AddCredsModel) Init() tea.Cmd {
	return nil
}

func (m AddCredsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.Cancel()
			return m, tea.Quit
		case "tab":
			m = m.moveCursor(false)
		case "shift+tab":
			m = m.moveCursor(true)
		case "enter":
			if m.cursorPos == 4 {
				return m, m.submit()
			}
			m = m.moveCursor(false)
		case "esc":
			credsMenuModel := NewCredsMenuModel(m.token, m.userID, m.login, m.grpcClient)
			return credsMenuModel, credsMenuModel.Init()
		}
	case tea.WindowSizeMsg:
		m.UpdateSize(msg)
	case CredsAddedMsg:
		credsMenuModel := NewCredsMenuModel(m.token, m.userID, m.login, m.grpcClient)
		credsMenuModel.SetError(fmt.Sprintf("Запись успешно добавлена (ID: %d)", msg.CredID))
		return credsMenuModel, credsMenuModel.Init()
	case CredsAddErrorMsg:
		m.SetError(msg.Error)
	}

	// Обновляем все поля
	var cmd tea.Cmd
	m.username, cmd = m.username.Update(msg)
	cmds = append(cmds, cmd)
	m.password, cmd = m.password.Update(msg)
	cmds = append(cmds, cmd)
	m.service, cmd = m.service.Update(msg)
	cmds = append(cmds, cmd)
	m.metadata, cmd = m.metadata.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m AddCredsModel) View() string {
	title := styles.TitleStyle.Render("Добавить логин/пароль")

	form := lipgloss.JoinVertical(lipgloss.Left,
		m.renderField("Логин/Email:", m.username.View(), 0),
		m.renderField("Пароль:", m.password.View(), 1),
		m.renderField("Сервис/Сайт:", m.service.View(), 2),
		m.renderField("Метка/Название:", m.metadata.View(), 3),
	)
	form = lipgloss.NewStyle().Width(40).Render(form)

	submitBtnView := styles.ButtonActiveStyle.Render(m.submitBtn)

	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		form,
		"",
		submitBtnView,
		m.RenderError(),
		m.RenderLoading(),
		"",
		styles.HelpStyle.Render("Tab/Shift+Tab: перемещение • Enter: подтвердить • Esc: назад"),
	)

	return m.Center(content)
}

func (m AddCredsModel) renderField(label string, fieldView string, index int) string {
	if m.cursorPos == index {
		label = "-> " + label
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().MarginBottom(1).Render(label),
		fieldView,
		"",
	)
}

func (m AddCredsModel) moveCursor(backward bool) AddCredsModel {
	m = m.blurAll()

	if backward {
		m.cursorPos--
		if m.cursorPos < 0 {
			m.cursorPos = 4
		}
	} else {
		m.cursorPos++
		if m.cursorPos > 4 {
			m.cursorPos = 0
		}
	}

	if m.cursorPos < 4 {
		switch m.cursorPos {
		case 0:
			m.username, _ = m.username.Focus()
		case 1:
			m.password, _ = m.password.Focus()
		case 2:
			m.service, _ = m.service.Focus()
		case 3:
			m.metadata, _ = m.metadata.Focus()
		}
	}

	if m.cursorPos == 4 {
		m.submitBtn = "-> Сохранить запись"
	} else {
		m.submitBtn = "Сохранить запись"
	}

	return m
}

func (m AddCredsModel) blurAll() AddCredsModel {
	m.username = m.username.Blur()
	m.password = m.password.Blur()
	m.service = m.service.Blur()
	m.metadata = m.metadata.Blur()
	return m
}

func (m AddCredsModel) submit() tea.Cmd {
	if err := m.validate(); err != nil {
		return func() tea.Msg {
			return CredsAddErrorMsg{Error: err.Error()}
		}
	}

	domainCreds := domain.Creds{
		Login:       m.username.Model.Value(),
		Password:    m.password.Model.Value(),
		ServiceName: m.service.Model.Value(),
		Metadata:    m.metadata.Model.Value(),
	}

	creds := &proto.Credentials_builder{
		Username:    &domainCreds.Login,
		Password:    &domainCreds.Password,
		ServiceName: &domainCreds.ServiceName,
		Metadata:    &domainCreds.Metadata,
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		credID, err := m.grpcClient.StoreCredentials(ctx, creds.Build())
		if err != nil {
			return CredsAddErrorMsg{Error: fmt.Sprintf("Ошибка сохранения: %v", err)}
		}
		return CredsAddedMsg{CredID: credID}
	}
}

func (m AddCredsModel) validate() error {
	if strings.TrimSpace(m.username.Model.Value()) == "" {
		return fmt.Errorf("Укажите логин/email")
	}
	if strings.TrimSpace(m.password.Model.Value()) == "" {
		return fmt.Errorf("Укажите пароль")
	}
	return nil
}
