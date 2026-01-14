package models

import (
    "context"
    "fmt"
    grpcclient "gophkeeper/cmd/client/internal/grpc_client"
    "gophkeeper/cmd/client/internal/ui/styles"
    "gophkeeper/internal/proto"
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

// CardDetailModel управляет просмотром деталей карты
type CardDetailModel struct {
    BaseModel
    token      string
    userID     string
    login      string
    grpcClient *grpcclient.GophKeeperClient
    cardID     int32
    card       *proto.Card
    loading    bool
    errorMsg   string
    menuItems  []string
    cursor     int
}

// Сообщения
type (
    CardLoadedMsg struct {
        Card *proto.Card
    }
    CardLoadErrorMsg struct {
        Error string
    }
    CardDeleteConfirmMsg struct{}
)

// NewCardDetailModel создаёт новую модель деталей карты
func NewCardDetailModel(token, userID, login string, grpcClient *grpcclient.GophKeeperClient, cardID int32) CardDetailModel {
    m := CardDetailModel{
        BaseModel:  NewBaseModel(),
        token:      token,
        userID:     userID,
        login:      login,
        grpcClient: grpcClient,
        cardID:     cardID,
        card:       nil,
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
            case 0: // Удалить карту
                return m, m.deleteCard
            case 1: // Назад
                cardListModel := NewCardListModel(m.token, m.userID, m.login, m.grpcClient)
                return cardListModel, cardListModel.Init()
            }
        case "esc":
            cardListModel := NewCardListModel(m.token, m.userID, m.login, m.grpcClient)
            return cardListModel, cardListModel.Init()
        }
    case tea.WindowSizeMsg:
        m.UpdateSize(msg)
    case CardLoadedMsg:
        m.card = msg.Card
        m.loading = false
    case CardLoadErrorMsg:
        m.errorMsg = msg.Error
        m.loading = false
    case CardDeleteConfirmMsg:
        // Возвращаемся к списку карт с сообщением об удалении
        cardListModel := NewCardListModel(m.token, m.userID, m.login, m.grpcClient)
        cardListModel.SetError("Карта успешно удалена")
        return cardListModel, cardListModel.Init()
    }
    return m, nil
}

// View отображает интерфейс
func (m CardDetailModel) View() string {
    if m.Width == 0 || m.Height == 0 {
        return "Загрузка..."
    }

    // Заголовок
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
        // Детали карты — теперь показываем полные данные (без маскировки)
        cardInfo := lipgloss.JoinVertical(lipgloss.Left,
            m.renderCardField("Номер карты:", m.card.GetCardNumber()),
            m.renderCardField("Держатель:", m.card.GetCardHolder()),
            m.renderCardField("Срок действия:", m.card.GetExpiryDate()),
            m.renderCardField("CVV:", m.card.GetCvv()),
            m.renderCardField("Метка:", m.card.GetMetadata()),
        )
        cardInfo = lipgloss.NewStyle().
            Border(lipgloss.RoundedBorder()).
            BorderForeground(styles.PrimaryColor).
            Padding(1, 2).
            Width(40).
            Render(cardInfo)

        // Меню действий
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

// loadCard загружает данные карты с сервера
func (m CardDetailModel) loadCard() tea.Msg {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    card, err := m.grpcClient.GetCard(ctx, m.cardID)
    if err != nil {
        return CardLoadErrorMsg{Error: fmt.Sprintf("Ошибка загрузки: %v", err)}
    }
    return CardLoadedMsg{Card: card}
}

// deleteCard удаляет карту
func (m CardDetailModel) deleteCard() tea.Msg {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    err := m.grpcClient.DeleteCard(ctx, m.cardID)
    if err != nil {
        return CardLoadErrorMsg{Error: fmt.Sprintf("Ошибка удаления: %v", err)}
    }
    return CardDeleteConfirmMsg{}
}