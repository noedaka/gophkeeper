package card

import (
	"context"

	"fmt"
	"strings"
	"time"

	"gophkeeper/cmd/client/internal/crypto"
	grpcclient "gophkeeper/cmd/client/internal/grpc_client"
	"gophkeeper/cmd/client/internal/ui/components"
	"gophkeeper/cmd/client/internal/ui/models/base"
	message "gophkeeper/cmd/client/internal/ui/models/messages"
	"gophkeeper/cmd/client/internal/ui/navigation"
	"gophkeeper/cmd/client/internal/ui/styles"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/proto"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AddCardModel управляет формой добавления карты
type AddCardModel struct {
	base.BaseModel
	token      string
	userID     string
	login      string
	grpcClient *grpcclient.GophKeeperClient

	crypt *crypto.Crypt
	nav   navigation.Navigator

	cardNumber components.InputField
	cardHolder components.InputField
	expiryDate components.InputField
	cvv        components.InputField
	metadata   components.InputField

	cursorPos int
	submitBtn string
}

type CardAddedMsg struct {
	CardID int32
}

// NewAddCardModel создаёт новую модель добавления карты
func NewAddCardModel(token, userID, login string, grpcClient *grpcclient.GophKeeperClient, crypt *crypto.Crypt, nav navigation.Navigator) AddCardModel {
	m := AddCardModel{
		BaseModel:  base.NewBaseModel(),
		token:      token,
		userID:     userID,
		login:      login,
		grpcClient: grpcClient,
		crypt:      crypt,
		nav:        nav,

		cardNumber: components.NewInput("Номер карты (16 цифр)", false),
		cardHolder: components.NewInput("Держатель карты", false),
		expiryDate: components.NewInput("Срок действия (ММ/ГГ)", false),
		cvv:        components.NewInput("CVV (3 цифры)", true),
		metadata:   components.NewInput("Метка/Название", false),

		cursorPos: 0,
		submitBtn: "Сохранить карту",
	}

	m.cardNumber.Model.CharLimit = 19
	m.expiryDate.Model.CharLimit = 5
	m.cvv.Model.CharLimit = 3

	m.cardNumber, _ = m.cardNumber.Focus()

	return m
}

// Init инициализирует модель
func (m AddCardModel) Init() tea.Cmd {
	return nil
}

// Update обрабатывает сообщения
func (m AddCardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.cursorPos == 5 {
				return m, m.submit()
			}
			m = m.moveCursor(false)

		case "esc":
			cardMenuModel := NewCardMenuModel(m.token, m.userID, m.login, m.grpcClient, m.crypt, m.nav)
			return cardMenuModel, cardMenuModel.Init()
		}

	case tea.WindowSizeMsg:
		m.UpdateSize(msg)

	case CardAddedMsg:
		cardMenuModel := NewCardMenuModel(m.token, m.userID, m.login, m.grpcClient, m.crypt, m.nav)
		cardMenuModel.SetError(fmt.Sprintf("Карта успешно добавлена (ID: %d)", msg.CardID))
		return cardMenuModel, cardMenuModel.Init()

	case message.ErrorMsg:
		m.SetError(msg.Error)
	}

	var cmd tea.Cmd
	m.cardNumber, cmd = m.cardNumber.Update(msg)
	cmds = append(cmds, cmd)
	m.cardHolder, cmd = m.cardHolder.Update(msg)
	cmds = append(cmds, cmd)
	m.expiryDate, cmd = m.expiryDate.Update(msg)
	cmds = append(cmds, cmd)
	m.cvv, cmd = m.cvv.Update(msg)
	cmds = append(cmds, cmd)
	m.metadata, cmd = m.metadata.Update(msg)
	cmds = append(cmds, cmd)

	m.formatCardNumberLive()
	m.formatExpiryDateLive()

	return m, tea.Batch(cmds...)
}

// View отображает интерфейс
func (m AddCardModel) View() string {
	title := styles.TitleStyle.Render("Добавить банковскую карту")

	form := lipgloss.JoinVertical(lipgloss.Left,
		m.renderField("Номер карты:", m.cardNumber.View(), 0),
		m.renderField("Держатель карты:", m.cardHolder.View(), 1),
		m.renderField("Срок действия (ММ/ГГ):", m.expiryDate.View(), 2),
		m.renderField("CVV:", m.cvv.View(), 3),
		m.renderField("Метка/Название:", m.metadata.View(), 4),
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

func (m AddCardModel) renderField(label string, fieldView string, index int) string {
	if m.cursorPos == index {
		label = "-> " + label
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().MarginBottom(1).Render(label),
		fieldView,
		"",
	)
}

// moveCursor перемещает фокус между полями и кнопкой
func (m AddCardModel) moveCursor(backward bool) AddCardModel {
	m = m.blurAll()

	if backward {
		m.cursorPos--
		if m.cursorPos < 0 {
			m.cursorPos = 5
		}
	} else {
		m.cursorPos++
		if m.cursorPos > 5 {
			m.cursorPos = 0
		}
	}

	if m.cursorPos < 5 {
		switch m.cursorPos {
		case 0:
			m.cardNumber, _ = m.cardNumber.Focus()
		case 1:
			m.cardHolder, _ = m.cardHolder.Focus()
		case 2:
			m.expiryDate, _ = m.expiryDate.Focus()
		case 3:
			m.cvv, _ = m.cvv.Focus()
		case 4:
			m.metadata, _ = m.metadata.Focus()
		}
	}

	if m.cursorPos == 5 {
		m.submitBtn = "-> Сохранить карту"
	} else {
		m.submitBtn = "Сохранить карту"
	}

	return m
}

// blurAll снимает фокус со всех полей
func (m AddCardModel) blurAll() AddCardModel {
	m.cardNumber = m.cardNumber.Blur()
	m.cardHolder = m.cardHolder.Blur()
	m.expiryDate = m.expiryDate.Blur()
	m.cvv = m.cvv.Blur()
	m.metadata = m.metadata.Blur()
	return m
}

func (m AddCardModel) submit() tea.Cmd {
	if err := m.validate(); err != nil {
		return func() tea.Msg {
			return message.ErrorMsg{Error: err.Error()}
		}
	}

	plainCard := &domain.Card{
		CardNumber:     m.formatCardNumber(m.cardNumber.Model.Value()),
		CardHolderName: m.cardHolder.Model.Value(),
		ExpiryDate:     m.expiryDate.Model.Value(),
		CVV:            m.cvv.Model.Value(),
		Metadata:       m.metadata.Model.Value(),
	}

	plainBytes, err := plainCard.MarshalPlain()
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

	cardName := "card"
	rec := proto.EncryptedRecord_builder{
		Ciphertext: encrypted,
		Nonce:      nonce,
		Metadata:   &plainCard.Metadata,
		RecordType: &cardName,
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cardID, err := m.grpcClient.StoreRecord(ctx, rec.Build())
		if err != nil {
			return message.ErrorMsg{Error: fmt.Sprintf("Ошибка сохранения: %v", err)}
		}
		return CardAddedMsg{CardID: cardID}
	}
}

// validate проверяет корректность данных
func (m AddCardModel) validate() error {
	rawNumber := strings.ReplaceAll(m.cardNumber.Model.Value(), " ", "")
	if len(rawNumber) != 16 || !isDigitsOnly(rawNumber) {
		return fmt.Errorf("Номер карты должен содержать ровно 16 цифр")
	}
	if strings.TrimSpace(m.cardHolder.Model.Value()) == "" {
		return fmt.Errorf("Укажите держателя карты")
	}
	if !isValidExpiryDate(m.expiryDate.Model.Value()) {
		return fmt.Errorf("Неверный формат срока действия (ММ/ГГ)")
	}
	if len(m.cvv.Model.Value()) != 3 || !isDigitsOnly(m.cvv.Model.Value()) {
		return fmt.Errorf("CVV должен содержать 3 цифры")
	}
	return nil
}

// formatCardNumber форматирует номер карты для отправки
func (m AddCardModel) formatCardNumber(number string) string {
	number = strings.ReplaceAll(number, " ", "")
	if len(number) != 16 {
		return number
	}
	parts := []string{number[0:4], number[4:8], number[8:12], number[12:16]}
	return strings.Join(parts, " ")
}

// formatCardNumberLive — автоформатирование номера карты при вводе
func (m *AddCardModel) formatCardNumberLive() {
	raw := strings.ReplaceAll(m.cardNumber.Model.Value(), " ", "")
	if len(raw) > 16 {
		raw = raw[:16]
	}
	var formatted strings.Builder
	for i := 0; i < len(raw); i += 4 {
		if i > 0 {
			formatted.WriteString(" ")
		}
		end := i + 4
		if end > len(raw) {
			end = len(raw)
		}
		formatted.WriteString(raw[i:end])
	}
	m.cardNumber.Model.SetValue(formatted.String())
	m.cardNumber.Model.SetCursor(len(formatted.String()))
}

// formatExpiryDateLive — автоформатирование срока действия
func (m *AddCardModel) formatExpiryDateLive() {
	val := strings.ReplaceAll(m.expiryDate.Model.Value(), "/", "")
	if len(val) > 4 {
		val = val[:4]
	}
	if len(val) >= 2 {
		val = val[:2] + "/" + val[2:]
	}
	m.expiryDate.Model.SetValue(val)
	m.expiryDate.Model.SetCursor(len(val))
}

// isValidExpiryDate проверяет формат ММ/ГГ
func isValidExpiryDate(date string) bool {
	if len(date) != 5 || date[2] != '/' {
		return false
	}
	month := date[0:2]
	if month < "01" || month > "12" {
		return false
	}
	return true
}

// isDigitsOnly вспомогательная функция
func isDigitsOnly(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
