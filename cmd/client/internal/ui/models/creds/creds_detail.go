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
	"gophkeeper/internal/domain"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CredsMenuModel управляет работе с детальными логинами/паролями
type CredsDetailModel struct {
	base.BaseModel
	token      string
	userID     string
	login      string
	grpcClient *grpcclient.GophKeeperClient
	credID     int32

	crypt *crypto.Crypt

	nav navigation.Navigator

	plainCreds *domain.Creds
	metadata   string

	loading   bool
	errorMsg  string
	menuItems []string
	cursor    int
}

type CredLoadedMsg struct {
	PlainCreds *domain.Creds
	Metadata   string
}

type CredDeleteConfirmMsg struct{}

// NewCredsMenuModel создаёт новую модель детализированных логинов/паролей
func NewCredsDetailModel(token, userID, login string, grpcClient *grpcclient.GophKeeperClient, credID int32, crypt *crypto.Crypt, nav navigation.Navigator) CredsDetailModel {
	m := CredsDetailModel{
		BaseModel:  base.NewBaseModel(),
		token:      token,
		userID:     userID,
		login:      login,
		crypt:      crypt,
		nav:        nav,
		grpcClient: grpcClient,
		credID:     credID,
		loading:    true,
		menuItems:  []string{"Удалить запись", "Назад"},
		cursor:     0,
	}
	return m
}

// Init инициализирует модель
func (m CredsDetailModel) Init() tea.Cmd {
	return m.loadCred
}

// Update обрабатывает сообщения
func (m CredsDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				return m, m.deleteCred
			case 1:
				credsListModel := NewCredsListModel(m.token, m.userID, m.login, m.grpcClient, m.crypt, m.nav)
				return credsListModel, credsListModel.Init()
			}
		case "esc":
			credsListModel := NewCredsListModel(m.token, m.userID, m.login, m.grpcClient, m.crypt, m.nav)
			return credsListModel, credsListModel.Init()
		}
	case tea.WindowSizeMsg:
		m.UpdateSize(msg)
	case CredLoadedMsg:
		m.plainCreds = msg.PlainCreds
		m.metadata = msg.Metadata
		m.loading = false
	case message.ErrorMsg:
		m.errorMsg = msg.Error
		m.loading = false
	case CredDeleteConfirmMsg:
		credsListModel := NewCredsListModel(m.token, m.userID, m.login, m.grpcClient, m.crypt, m.nav)
		credsListModel.SetError("Запись успешно удалена")
		return credsListModel, credsListModel.Init()
	}
	return m, nil
}

// View отображает интерфейс
func (m CredsDetailModel) View() string {
	title := styles.TitleStyle.Render(fmt.Sprintf("Запись #%d", m.credID))

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
		credInfo := lipgloss.JoinVertical(lipgloss.Left,
			m.renderCredField("Логин:", m.plainCreds.Login),
			m.renderCredField("Пароль:", m.plainCreds.Password),
			m.renderCredField("Сервис:", m.plainCreds.ServiceName),
			m.renderCredField("Метка:", m.metadata),
		)
		credInfo = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.PrimaryColor).
			Padding(1, 2).
			Width(40).
			Render(credInfo)

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
			credInfo,
			"",
			menu,
			"",
			styles.HelpStyle.Render("↑/↓: навигация • Enter: выбрать • Esc: назад"),
		)
	}

	return m.Center(content)
}

// renderCredField применяет стили к полям
func (m CredsDetailModel) renderCredField(label, value string) string {
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

func (m CredsDetailModel) renderLoading() string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := frames[int(time.Now().UnixNano()/int64(time.Millisecond))%len(frames)]
	return lipgloss.NewStyle().
		Foreground(styles.PrimaryColor).
		Render(frame + " Загрузка данных записи...")
}

// loadCred загружает детализированную пару
func (m CredsDetailModel) loadCred() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	encRec, err := m.grpcClient.GetRecord(ctx, m.credID)
	if err != nil {
		return message.ErrorMsg{Error: fmt.Sprintf("Ошибка загрузки: %v", err)}
	}

	plainBytes, err := m.crypt.Decrypt(encRec.GetCiphertext(), encRec.GetNonce())
	if err != nil {
		return message.ErrorMsg{Error: "Ошибка расшифровки данных"}
	}

	creds, err := domain.UnmarshalPlainCreds(plainBytes)
	if err != nil {
		return message.ErrorMsg{Error: "Ошибка десериализации данных"}
	}

	return CredLoadedMsg{
		PlainCreds: creds,
		Metadata:   encRec.GetMetadata(),
	}
}

func (m CredsDetailModel) deleteCred() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := m.grpcClient.DeleteRecord(ctx, m.credID)
	if err != nil {
		return message.ErrorMsg{Error: fmt.Sprintf("Ошибка удаления: %v", err)}
	}
	return CredDeleteConfirmMsg{}
}
