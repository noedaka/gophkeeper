package creds

import (
	"context"
	"fmt"
	"gophkeeper/cmd/client/internal/crypto"
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

// CredsListModel модель списка пар логин/пароль
type CredsListModel struct {
	base.BaseModel
	token      string
	userID     string
	login      string
	grpcClient *grpcclient.GophKeeperClient

	crypt *crypto.Crypt

	nav navigation.Navigator

	credIDs  []int32
	cursor   int
	loading  bool
	errorMsg string
}

type CredsLoadedMsg struct {
	CredIDs []int32
}

// NewCredsListModel создает новую модель пары логин/пароль
func NewCredsListModel(token, userID, login string, grpcClient *grpcclient.GophKeeperClient, crypt *crypto.Crypt, nav navigation.Navigator) CredsListModel {
	m := CredsListModel{
		BaseModel:  base.NewBaseModel(),
		token:      token,
		userID:     userID,
		login:      login,
		crypt:      crypt,
		nav:        nav,
		grpcClient: grpcClient,
		credIDs:    []int32{},
		cursor:     0,
		loading:    true,
	}
	return m
}

// Init инициализирует модель
func (m CredsListModel) Init() tea.Cmd {
	return m.loadCreds
}

// Update обрабатывает сообщения
func (m CredsListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.cursor < len(m.credIDs)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.credIDs) > 0 && m.cursor < len(m.credIDs) {
				credID := m.credIDs[m.cursor]
				detailModel := NewCredsDetailModel(m.token, m.userID, m.login, m.grpcClient, credID, m.crypt, m.nav)
				return detailModel, detailModel.Init()
			}
		case "r", "R":
			m.loading = true
			m.errorMsg = ""
			return m, m.loadCreds
		case "esc":
			credsMenuModel := NewCredsMenuModel(m.token, m.userID, m.login, m.grpcClient, m.crypt, m.nav)
			return credsMenuModel, credsMenuModel.Init()
		}
	case tea.WindowSizeMsg:
		m.UpdateSize(msg)
	case CredsLoadedMsg:
		m.credIDs = msg.CredIDs
		m.loading = false
		sort.Slice(m.credIDs, func(i, j int) bool {
			return m.credIDs[i] < m.credIDs[j]
		})
	case message.ErrorMsg:
		m.errorMsg = msg.Error
		m.loading = false
	}
	return m, nil
}

// View отображает интерфейс
func (m CredsListModel) View() string {
	title := styles.TitleStyle.Render("Список логинов/паролей")
	subtitle := lipgloss.NewStyle().
		Foreground(styles.SecondaryColor).
		Render(fmt.Sprintf("Найдено записей: %d", len(m.credIDs)))

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
	} else if len(m.credIDs) == 0 {
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
		for i, credID := range m.credIDs {
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

func (m CredsListModel) renderLoading() string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := frames[int(time.Now().UnixNano()/int64(time.Millisecond))%len(frames)]
	return lipgloss.NewStyle().
		Foreground(styles.PrimaryColor).
		Render(frame + " Загрузка списка записей...")
}

// loadCreds отправляет запрос на загрузку пар логин/пароль
func (m CredsListModel) loadCreds() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	infos, err := m.grpcClient.ListRecords(ctx, "creds")
	if err != nil {
		return message.ErrorMsg{Error: fmt.Sprintf("Ошибка загрузки: %v", err)}
	}

	var ids []int32
	for _, info := range infos {
		ids = append(ids, info.GetId())
	}

	return CredsLoadedMsg{CredIDs: ids}
}
