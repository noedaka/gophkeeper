package card

import (
	"context"
	"fmt"
	"gophkeeper/cmd/client/internal/crypto"
	grpcclient "gophkeeper/cmd/client/internal/grpc_client"
	"gophkeeper/cmd/client/internal/ui/models/base"
	message "gophkeeper/cmd/client/internal/ui/models/messages"
	"gophkeeper/cmd/client/internal/ui/navigation"
	"gophkeeper/cmd/client/internal/ui/styles"
	"gophkeeper/internal/domain"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CardDetailModel управляет просмотром деталей карты
type CardDetailModel struct {
	base.BaseModel
	token      string
	userID     string
	login      string
	grpcClient *grpcclient.GophKeeperClient
	cardID     int32

	crypt *crypto.Crypt
	nav   navigation.Navigator

	plainCard *domain.Card
	metadata  string

	loading   bool
	errorMsg  string
	menuItems []string
	cursor    int
}

type CardLoadedMsg struct {
	PlainCard *domain.Card
	Metadata  string
}
type CardDeleteConfirmMsg struct{}

// NewCardDetailModel создаёт новую модель деталей карты
func NewCardDetailModel(token, userID, login string, grpcClient *grpcclient.GophKeeperClient, cardID int32, crypt *crypto.Crypt, nav navigation.Navigator) CardDetailModel {
	m := CardDetailModel{
		BaseModel:  base.NewBaseModel(),
		token:      token,
		userID:     userID,
		login:      login,
		grpcClient: grpcClient,
		crypt:      crypt,
		nav:        nav,
		cardID:     cardID,
		plainCard:  nil,
		loading:    true,
		menuItems:  []string{"Удалить карту", "Назад"},
		cursor:     0,
	}
	return m
}

// Init инициализирует модель и загружает данные карты
func (m CardDetailModel) Init() tea.Cmd {
	return m.loadCard
}

// Update обрабатывает сообщения
func (m CardDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}
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
		case "enter":
			switch m.cursor {
			case 0:
				return m, m.deleteCard
			case 1:
				cardListModel := NewCardListModel(m.token, m.userID, m.login, m.grpcClient, m.crypt, m.nav)
				return cardListModel, cardListModel.Init()
			}
		case "esc":
			cardListModel := NewCardListModel(m.token, m.userID, m.login, m.grpcClient, m.crypt, m.nav)
			return cardListModel, cardListModel.Init()
		}
	case tea.WindowSizeMsg:
		m.UpdateSize(msg)
	case CardLoadedMsg:
		m.plainCard = msg.PlainCard
		m.metadata = msg.Metadata
		m.loading = false
	case message.ErrorMsg:
		m.errorMsg = msg.Error
		m.loading = false
	case CardDeleteConfirmMsg:
		cardListModel := NewCardListModel(m.token, m.userID, m.login, m.grpcClient, m.crypt, m.nav)
		cardListModel.SetError("Карта успешно удалена")
		return cardListModel, cardListModel.Init()
	}
	return m, nil
}

// View отображает интерфейс
func (m CardDetailModel) View() string {
	title := styles.TitleStyle.Render(fmt.Sprintf("Карта #%d", m.cardID))

	var content string
	if m.loading {
		content = lipgloss.JoinVertical(lipgloss.Center,
			title,
			"",
			m.renderLoading(),
		)
	} else if m.errorMsg != "" {
		content = lipgloss.JoinVertical(lipgloss.Center,
			title,
			"",
			styles.ErrorStyle.Render("Ошибка: "+m.errorMsg),
			"",
			styles.HelpStyle.Render("Esc: назад"),
		)
	} else {
		cardInfo := lipgloss.JoinVertical(lipgloss.Left,
			m.renderCardField("Номер карты:", m.plainCard.CardNumber),
			m.renderCardField("Держатель:", m.plainCard.CardHolderName),
			m.renderCardField("Срок действия:", m.plainCard.ExpiryDate),
			m.renderCardField("CVV:", m.plainCard.CVV),
			m.renderCardField("Метка:", m.metadata),
		)
		cardInfo = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.PrimaryColor).
			Padding(1, 2).
			Width(40).
			Render(cardInfo)

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

		content = lipgloss.JoinVertical(lipgloss.Center,
			title,
			"",
			cardInfo,
			"",
			menu,
			"",
			styles.HelpStyle.Render("↑/↓: навигация • Enter: выбрать • Esc: назад"),
		)
	}

	return m.Center(content)
}

// renderCardField отображает поле карты
func (m CardDetailModel) renderCardField(label, value string) string {
	return lipgloss.JoinHorizontal(lipgloss.Left,
		lipgloss.NewStyle().
			Foreground(styles.SecondaryColor).
			Width(20).
			Render(label),
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Render(value),
		"\n\n",
	)
}

// renderLoading отображает индикатор загрузки
func (m CardDetailModel) renderLoading() string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := frames[int(time.Now().UnixNano()/int64(time.Millisecond))%len(frames)]
	return lipgloss.NewStyle().
		Foreground(styles.PrimaryColor).
		Render(frame + " Загрузка данных карты...")
}

// loadCard загружает зашифрованную запись, расшифровывает и десериализует
func (m CardDetailModel) loadCard() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	encRec, err := m.grpcClient.GetRecord(ctx, m.cardID)
	if err != nil {
		return message.ErrorMsg{Error: fmt.Sprintf("Ошибка загрузки: %v", err)}
	}

	plainBytes, err := m.crypt.Decrypt(encRec.GetCiphertext(), encRec.GetNonce())
	if err != nil {
		return message.ErrorMsg{Error: "Ошибка расшифровки данных"}
	}

	card, err := domain.UnmarshalPlainCard(plainBytes)
	if err != nil {
		return message.ErrorMsg{Error: "Ошибка десериализации данных"}
	}

	return CardLoadedMsg{
		PlainCard: card,
		Metadata:  encRec.GetMetadata(),
	}
}

// deleteCard удаляет запись (унифицированный метод)
func (m CardDetailModel) deleteCard() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := m.grpcClient.DeleteRecord(ctx, m.cardID)
	if err != nil {
		return message.ErrorMsg{Error: fmt.Sprintf("Ошибка удаления: %v", err)}
	}
	return CardDeleteConfirmMsg{}
}
