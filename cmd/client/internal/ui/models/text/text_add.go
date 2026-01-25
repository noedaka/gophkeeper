package text

import (
	"context"
	"fmt"
	"gophkeeper/cmd/client/internal/crypto"
	grpcclient "gophkeeper/cmd/client/internal/grpc_client"
	"gophkeeper/cmd/client/internal/ui/components"
	"gophkeeper/cmd/client/internal/ui/models/base"
	message "gophkeeper/cmd/client/internal/ui/models/messages"
	"gophkeeper/cmd/client/internal/ui/navigation"
	"gophkeeper/cmd/client/internal/ui/styles"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/proto"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AddTextModel управляет работой добавлением текста
type AddTextModel struct {
	base.BaseModel
	token      string
	userID     string
	login      string
	grpcClient *grpcclient.GophKeeperClient

	crypt    *crypto.Crypt
	nav      navigation.Navigator
	text     components.InputField
	metadata components.InputField

	cursorPos int
	submitBtn string
}

type TextAddedMsg struct {
	CredID int32
}

// NewAddTextModel создает новую модель добавления произвольного текста
func NewAddTextModel(token, userID, login string, grpcClient *grpcclient.GophKeeperClient, crypt *crypto.Crypt, nav navigation.Navigator) AddTextModel {
	m := AddTextModel{
		BaseModel:  base.NewBaseModel(),
		token:      token,
		userID:     userID,
		login:      login,
		grpcClient: grpcClient,
		crypt:      crypt,
		nav:        nav,

		text:     components.NewInput("Текст", false),
		metadata: components.NewInput("Метка/Название", false),

		cursorPos: 0,
		submitBtn: "Сохранить запись",
	}

	m.text, _ = m.text.Focus()

	return m
}

// Init инициализирует модель
func (m AddTextModel) Init() tea.Cmd {
	return nil
}

// Update обновляет состоние модели
func (m AddTextModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.cursorPos == 2 {
				return m, m.submit()
			}
			m = m.moveCursor(false)
		case "esc":
			textMenuModel := NewTextMenuModel(m.token, m.userID, m.login, m.grpcClient, m.crypt, m.nav)
			return textMenuModel, textMenuModel.Init()
		}
	case tea.WindowSizeMsg:
		m.UpdateSize(msg)
	case TextAddedMsg:
		textMenuModel := NewTextMenuModel(m.token, m.userID, m.login, m.grpcClient,m.crypt, m.nav)
		textMenuModel.SetError(fmt.Sprintf("Запись успешно добавлена (ID: %d)", msg.CredID))
		return textMenuModel, textMenuModel.Init()
	case message.ErrorMsg:
		m.SetError(msg.Error)
	}

	var cmd tea.Cmd
	m.text, cmd = m.text.Update(msg)
	cmds = append(cmds, cmd)
	m.metadata, cmd = m.metadata.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// moveCursor обрабатывает логику изменения курсора
func (m AddTextModel) moveCursor(backward bool) AddTextModel {
	m = m.blurAll()

	if backward {
		m.cursorPos--
		if m.cursorPos < 0 {
			m.cursorPos = 2
		}
	} else {
		m.cursorPos++
		if m.cursorPos > 2 {
			m.cursorPos = 0
		}
	}

	if m.cursorPos < 2 {
		switch m.cursorPos {
		case 0:
			m.text, _ = m.text.Focus()
		case 1:
			m.metadata, _ = m.metadata.Focus()
		}
	}

	if m.cursorPos == 2 {
		m.submitBtn = "-> Сохранить запись"
	} else {
		m.submitBtn = "Сохранить запись"
	}

	return m
}

// blurAll снимает фокус с полей
func (m AddTextModel) blurAll() AddTextModel {
	m.text = m.text.Blur()
	m.metadata = m.metadata.Blur()
	return m
}

// View отображает интерфейс
func (m AddTextModel) View() string {
	title := styles.TitleStyle.Render("Добавить текст")

	form := lipgloss.JoinVertical(lipgloss.Left,
		m.renderField("Текст:", m.text.View(), 0),
		m.renderField("Метка/Название:", m.metadata.View(), 1),
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

// renderField отображает выбранное поле
func (m AddTextModel) renderField(label string, fieldView string, index int) string {
	if m.cursorPos == index {
		label = "-> " + label
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().MarginBottom(1).Render(label),
		fieldView,
		"",
	)
}

// sumbit обрабатыват нажатие кнопки отправить
func (m AddTextModel) submit() tea.Cmd {
	plainText := &domain.Text{
		Text:     m.text.Model.Value(),
		Metadata: m.metadata.Model.Value(),
	}

	plainBytes, err := plainText.MarshalPlain()
	if err != nil {
		return func() tea.Msg {
			return message.ErrorMsg{Error: fmt.Sprintf("marshal error: %v", err)}
		}
	}

	encrypted, nonce, err := m.crypt.Encrypt(plainBytes)
	if err != nil {
		return func() tea.Msg {
			return message.ErrorMsg{Error: fmt.Sprintf("encryption error: %v", err)}
		}
	}

	textName := "text"
	rec := proto.EncryptedRecord_builder{
		Ciphertext: encrypted,
		Nonce:      nonce,
		Metadata:   &plainText.Metadata,
		RecordType: &textName,
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		credID, err := m.grpcClient.StoreRecord(ctx, rec.Build())
		if err != nil {
			return message.ErrorMsg{Error: fmt.Sprintf("Ошибка сохранения: %v", err)}
		}
		return TextAddedMsg{CredID: credID}
	}
}
