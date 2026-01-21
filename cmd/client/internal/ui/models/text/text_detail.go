package text

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

// TextDetailModel управляет работой детализированного текста
type TextDetailModel struct {
	base.BaseModel
	token      string
	userID     string
	login      string
	grpcClient *grpcclient.GophKeeperClient
	credID     int32

	nav       navigation.Navigator
	plainText *domain.Text
	metadata  string

	loading   bool
	errorMsg  string
	menuItems []string
	cursor    int
}

type TextLoadedMsg struct {
	PlainText *domain.Text
	Metadata  string
}

type TextDeleteConfirmMsg struct{}

// NewTextDetailModel создает новую модель
func NewTextDetailModel(token, userID, login string, grpcClient *grpcclient.GophKeeperClient, credID int32, nav navigation.Navigator) TextDetailModel {
	m := TextDetailModel{
		BaseModel:  base.NewBaseModel(),
		token:      token,
		userID:     userID,
		login:      login,
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
func (m TextDetailModel) Init() tea.Cmd {
	return m.loadText
}

// Update обновляет состоние модели
func (m TextDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				return m, m.deleteText
			case 1:
				textListModel := NewTextListModel(m.token, m.userID, m.login, m.grpcClient, m.nav)
				return textListModel, textListModel.Init()
			}
		case "esc":
			textListModel := NewTextListModel(m.token, m.userID, m.login, m.grpcClient, m.nav)
			return textListModel, textListModel.Init()
		}
	case tea.WindowSizeMsg:
		m.UpdateSize(msg)
	case TextLoadedMsg:
		m.plainText = msg.PlainText
		m.metadata = msg.Metadata
		m.loading = false
	case message.ErrorMsg:
		m.errorMsg = msg.Error
		m.loading = false
	case TextDeleteConfirmMsg:
		textListModel := NewTextListModel(m.token, m.userID, m.login, m.grpcClient, m.nav)
		textListModel.SetError("Запись успешно удалена")
		return textListModel, textListModel.Init()
	}
	return m, nil
}

// View отображает интерфейс
func (m TextDetailModel) View() string {
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
			m.renderTextField("Текст:", m.plainText.Text),
			m.renderTextField("Метка:", m.metadata),
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

// renderField отображает выбранное поле
func (m TextDetailModel) renderTextField(label, value string) string {
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

func (m TextDetailModel) renderLoading() string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := frames[int(time.Now().UnixNano()/int64(time.Millisecond))%len(frames)]
	return lipgloss.NewStyle().
		Foreground(styles.PrimaryColor).
		Render(frame + " Загрузка данных записи...")
}

// Загружает выбранынй текст
func (m TextDetailModel) loadText() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	encRec, err := m.grpcClient.GetRecord(ctx, m.credID)
	if err != nil {
		return message.ErrorMsg{Error: fmt.Sprintf("Ошибка загрузки: %v", err)}
	}

	plainBytes, err := crypto.Decrypt(encRec.GetCiphertext(), encRec.GetNonce())
	if err != nil {
		return message.ErrorMsg{Error: "Ошибка расшифровки данных"}
	}

	text, err := domain.UnmarshalPlainText(plainBytes)
	if err != nil {
		return message.ErrorMsg{Error: "Ошибка десериализации данных"}
	}

	return TextLoadedMsg{
		PlainText: text,
		Metadata:  encRec.GetMetadata(),
	}
}

// deleteText удаляет выбранный текст
func (m TextDetailModel) deleteText() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := m.grpcClient.DeleteRecord(ctx, m.credID)
	if err != nil {
		return message.ErrorMsg{Error: fmt.Sprintf("Ошибка удаления: %v", err)}
	}
	return TextDeleteConfirmMsg{}
}
