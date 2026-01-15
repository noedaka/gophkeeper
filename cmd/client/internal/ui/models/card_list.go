package models

import (
	"context"
	"fmt"
	grpcclient "gophkeeper/cmd/client/internal/grpc_client"
	"gophkeeper/cmd/client/internal/ui/styles"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CardListModel управляет списком карт
type CardListModel struct {
	BaseModel
	token      string
	userID     string
	login      string
	grpcClient *grpcclient.GophKeeperClient

	cardIDs  []int32
	cursor   int
	loading  bool
	errorMsg string
}

type (
	CardsLoadedMsg struct {
		CardIDs []int32
	}
	CardsLoadErrorMsg struct {
		Error string
	}
	CardSelectedMsg struct {
		CardID int32
	}
)

// NewCardListModel создаёт новую модель списка карт
func NewCardListModel(token, userID, login string, grpcClient *grpcclient.GophKeeperClient) CardListModel {
	m := CardListModel{
		BaseModel:  NewBaseModel(),
		token:      token,
		userID:     userID,
		login:      login,
		grpcClient: grpcClient,
		cardIDs:    []int32{},
		cursor:     0,
		loading:    true,
	}

	return m
}

// Init инициализирует модель и загружает список карт
func (m CardListModel) Init() tea.Cmd {
	return m.loadCards
}

// Update обрабатывает сообщения
func (m CardListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.cursor < len(m.cardIDs)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.cardIDs) > 0 && m.cursor < len(m.cardIDs) {
				cardID := m.cardIDs[m.cursor]
				cardDetailModel := NewCardDetailModel(m.token, m.userID, m.login, m.grpcClient, cardID)
				return cardDetailModel, cardDetailModel.Init()
			}
		case "r", "R":
			m.loading = true
			m.errorMsg = ""
			return m, m.loadCards
		case "esc":
			cardMenuModel := NewCardMenuModel(m.token, m.userID, m.login, m.grpcClient)
			return cardMenuModel, cardMenuModel.Init()
		}
	case tea.WindowSizeMsg:
		m.UpdateSize(msg)
	case CardsLoadedMsg:
		m.cardIDs = msg.CardIDs
		m.loading = false
		sort.Slice(m.cardIDs, func(i, j int) bool {
			return m.cardIDs[i] < m.cardIDs[j]
		})
	case CardsLoadErrorMsg:
		m.errorMsg = msg.Error
		m.loading = false
	}

	return m, nil
}

// View отображает интерфейс
func (m CardListModel) View() string {
	title := styles.TitleStyle.Render("Список карт")
	subtitle := lipgloss.NewStyle().
		Foreground(styles.SecondaryColor).
		Render(fmt.Sprintf("Найдено карт: %d", len(m.cardIDs)))

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
			styles.HelpStyle.Render("Нажмите R для повторной загрузки • Esc: назад"),
		)
	} else if len(m.cardIDs) == 0 {
		content = lipgloss.JoinVertical(lipgloss.Center,
			title,
			"",
			lipgloss.NewStyle().
				Foreground(styles.SecondaryColor).
				Render("Нет сохраненных карт"),
			"",
			styles.HelpStyle.Render("Esc: назад"),
		)
	} else {
		var items []string
		for i, cardID := range m.cardIDs {
			item := fmt.Sprintf("Карта #%d", cardID)
			if i == m.cursor {
				items = append(items,
					styles.ButtonActiveStyle.
						Width(30).
						Render("> "+item),
				)
			} else {
				items = append(items,
					lipgloss.NewStyle().
						Padding(0, 3).
						Width(30).
						Render(" "+item),
				)
			}
		}

		list := lipgloss.JoinVertical(lipgloss.Left, items...)
		list = lipgloss.NewStyle().MarginTop(2).Render(list)

		content = lipgloss.JoinVertical(lipgloss.Center,
			title,
			subtitle,
			"",
			list,
			"",
			styles.HelpStyle.Render("↑/↓: навигация • Enter: просмотр • R: обновить • Esc: назад"),
		)
	}

	return m.Center(content)
}

// renderLoading отображает индикатор загрузки
func (m CardListModel) renderLoading() string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := frames[int(time.Now().UnixNano()/int64(time.Millisecond))%len(frames)]
	return lipgloss.NewStyle().
		Foreground(styles.PrimaryColor).
		Render(frame + " Загрузка списка карт...")
}

func (m CardListModel) loadCards() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	infos, err := m.grpcClient.ListRecords(ctx, "card")
	if err != nil {
		return CardsLoadErrorMsg{Error: fmt.Sprintf("Ошибка загрузки: %v", err)}
	}

	var ids []int32
	for _, info := range infos {
		ids = append(ids, info.GetId())
	}

	return CardsLoadedMsg{CardIDs: ids}
}
