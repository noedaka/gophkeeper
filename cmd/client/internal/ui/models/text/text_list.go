package text

import (
	"context"
	"fmt"
	grpcclient "gophkeeper/cmd/client/internal/grpc_client"
	"gophkeeper/cmd/client/internal/ui/models/base"
	message "gophkeeper/cmd/client/internal/ui/models/messages"
	"gophkeeper/cmd/client/internal/ui/navigation"
	"gophkeeper/cmd/client/internal/ui/styles"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TextListModel модель списка текстов
type TextListModel struct {
	base.BaseModel
	token      string
	userID     string
	login      string
	grpcClient *grpcclient.GophKeeperClient
	nav        navigation.Navigator
	textIDs    []int32
	cursor     int
	loading    bool
	errorMsg   string
}

type TextIDLoadedMsg struct {
	textIDs []int32
}

func NewTextListModel(token, userID, login string, grpcClient *grpcclient.GophKeeperClient, nav navigation.Navigator) TextListModel {
	m := TextListModel{
		BaseModel:  base.NewBaseModel(),
		token:      token,
		userID:     userID,
		login:      login,
		nav:        nav,
		grpcClient: grpcClient,
		textIDs:    []int32{},
		cursor:     0,
	}

	return m
}

func (m TextListModel) Init() tea.Cmd {
	return m.loadText
}

func (m TextListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.cursor < len(m.textIDs)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.textIDs) > 0 && m.cursor < len(m.textIDs) {
				credID := m.textIDs[m.cursor]
				detailModel := NewTextDetailModel(m.token, m.userID, m.login, m.grpcClient, credID, m.nav)
				return detailModel, detailModel.Init()
			}
		case "r", "R":
			m.loading = true
			m.errorMsg = ""
			return m, m.loadText
		case "esc":
			textMenuModel := NewTextMenuModel(m.token, m.userID, m.login, m.grpcClient, m.nav)
			return textMenuModel, textMenuModel.Init()
		}
	case tea.WindowSizeMsg:
		m.UpdateSize(msg)
	case TextIDLoadedMsg:
		m.textIDs = msg.textIDs
		m.loading = false
		sort.Slice(m.textIDs, func(i, j int) bool {
			return m.textIDs[i] < m.textIDs[j]
		})
	case message.ErrorMsg:
		m.errorMsg = msg.Error
		m.loading = false
	}
	return m, nil
}

func (m TextListModel) View() string {
	title := styles.TitleStyle.Render("Список текстов")
	subtitle := lipgloss.NewStyle().
		Foreground(styles.SecondaryColor).
		Render(fmt.Sprintf("Найдено записей: %d", len(m.textIDs)))

	var content string

	if m.errorMsg != "" {
		content = lipgloss.JoinVertical(lipgloss.Center,
			title,
			"",
			styles.ErrorStyle.Render("Ошибка: "+m.errorMsg),
			"",
			styles.HelpStyle.Render("Нажмите R для повторной загрузки • Esc: назад"),
		)
	} else if len(m.textIDs) == 0 {
		content = lipgloss.JoinVertical(lipgloss.Center,
			title,
			"",
			lipgloss.NewStyle().
				Foreground(styles.SecondaryColor).
				Render("Нет сохраненных записей"),
			"",
			styles.HelpStyle.Render("Esc: назад"),
		)
	} else {
		var items []string
		for i, credID := range m.textIDs {
			item := fmt.Sprintf("Запись #%d", credID)
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

func (m TextListModel) loadText() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	infos, err := m.grpcClient.ListRecords(ctx, "text")
	if err != nil {
		return message.ErrorMsg{Error: fmt.Sprintf("Ошибка загрузки: %v", err)}
	}

	var ids []int32
	for _, info := range infos {
		ids = append(ids, info.GetId())
	}

	return TextIDLoadedMsg{textIDs: ids}
}
