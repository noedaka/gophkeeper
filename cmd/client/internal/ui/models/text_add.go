package models

import (
	"context"
	"fmt"
	"gophkeeper/cmd/client/internal/crypto"
	grpcclient "gophkeeper/cmd/client/internal/grpc_client"
	"gophkeeper/cmd/client/internal/ui/components"
	"gophkeeper/cmd/client/internal/ui/styles"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/proto"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type AddTextModel struct {
	BaseModel
	token      string
	userID     string
	login      string
	grpcClient *grpcclient.GophKeeperClient

	text     components.InputField
	metadata components.InputField

	cursorPos int
	submitBtn string
}

type (
	TextAddedMsg struct {
		CredID int32
	}
	TextAddErrorMsg struct {
		Error string
	}
)

func NewAddTextModel(token, userID, login string, grpcClient *grpcclient.GophKeeperClient) AddTextModel {
	m := AddTextModel{
		BaseModel:  NewBaseModel(),
		token:      token,
		userID:     userID,
		login:      login,
		grpcClient: grpcClient,

		text:     components.NewInput("Текст", false),
		metadata: components.NewInput("Метка/Название", false),

		cursorPos: 0,
		submitBtn: "Сохранить запись",
	}

	m.text, _ = m.text.Focus()

	return m
}

func (m AddTextModel) Init() tea.Cmd {
	return nil
}

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
			textMenuModel := NewTextMenuModel(m.token, m.userID, m.login, m.grpcClient)
			return textMenuModel, textMenuModel.Init()
		}
	case tea.WindowSizeMsg:
		m.UpdateSize(msg)
	case TextAddedMsg:
		textMenuModel := NewTextMenuModel(m.token, m.userID, m.login, m.grpcClient)
		textMenuModel.SetError(fmt.Sprintf("Запись успешно добавлена (ID: %d)", msg.CredID))
		return textMenuModel, textMenuModel.Init()
	case TextAddErrorMsg:
		m.SetError(msg.Error)
	}

	var cmd tea.Cmd
	m.text, cmd = m.text.Update(msg)
	cmds = append(cmds, cmd)
	m.metadata, cmd = m.metadata.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

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

func (m AddTextModel) blurAll() AddTextModel {
	m.text = m.text.Blur()
	m.metadata = m.metadata.Blur()
	return m
}

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

func (m AddTextModel) submit() tea.Cmd {
	plainText := &domain.Text{
		Text:       m.text.Model.Value(),
		Metadata:    m.metadata.Model.Value(),
	}

	plainBytes, err := plainText.MarshalPlain()
	if err != nil {
		return func() tea.Msg {
			return TextAddErrorMsg{Error: fmt.Sprintf("marshal error: %v", err)}
		}
	}

	encrypted, nonce, err := crypto.Encrypt(plainBytes)
	if err != nil {
		return func() tea.Msg {
			return TextAddErrorMsg{Error: fmt.Sprintf("encryption error: %v", err)}
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
			return TextAddErrorMsg{Error: fmt.Sprintf("Ошибка сохранения: %v", err)}
		}
		return TextAddedMsg{CredID: credID}
	}
}