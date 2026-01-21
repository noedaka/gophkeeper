package binary

import (
	"context"
	"fmt"
	"sort"
	"time"

	grpcclient "gophkeeper/cmd/client/internal/grpc_client"
	"gophkeeper/cmd/client/internal/ui/models/base"
	message "gophkeeper/cmd/client/internal/ui/models/messages"
	"gophkeeper/cmd/client/internal/ui/navigation"
	"gophkeeper/cmd/client/internal/ui/styles"
	"gophkeeper/internal/proto"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// BinaryListModel управляет списком бинарных файлов
type BinaryListModel struct {
	base.BaseModel
	token      string
	userID     string
	login      string
	grpcClient *grpcclient.GophKeeperClient
	nav        navigation.Navigator
	records    []*proto.BinaryRecordInfo
	cursor     int
	loading    bool
	errorMsg   string
}

type BinariesLoadedMsg struct {
	Records []*proto.BinaryRecordInfo
}

func NewBinaryListModel(token, userID, login string, grpcClient *grpcclient.GophKeeperClient, nav navigation.Navigator) BinaryListModel {
	return BinaryListModel{
		BaseModel:  base.NewBaseModel(),
		token:      token,
		userID:     userID,
		login:      login,
		nav:        nav,
		grpcClient: grpcClient,
		loading:    false,
	}
}

func (m BinaryListModel) Init() tea.Cmd {
	return m.loadBinaries()
}

func (m BinaryListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":

			m.Cancel()
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if len(m.records) > 0 && m.cursor < len(m.records)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.records) > 0 {
				rec := m.records[m.cursor]
				return NewBinaryDetailModel(m.token, m.userID, m.login, m.grpcClient, rec.GetId(), rec.GetMetadata(), m.nav), nil
			}
		case "r", "R":
			m.loading = true
			m.errorMsg = ""
			m.records = nil
			m.cursor = 0
			return m, m.loadBinaries()
		case "esc":
			return NewBinaryMenuModel(m.token, m.userID, m.login, m.grpcClient, m.nav), nil
		}
	case tea.WindowSizeMsg:
		m.UpdateSize(msg)
	case BinariesLoadedMsg:
		m.records = msg.Records
		m.loading = false
		m.errorMsg = ""
		sort.Slice(m.records, func(i, j int) bool {
			return m.records[i].GetId() < m.records[j].GetId()
		})
		if len(m.records) == 0 {
			m.cursor = 0
		} else if m.cursor >= len(m.records) {
			m.cursor = len(m.records) - 1
		}
	case message.ErrorMsg:
		m.errorMsg = msg.Error
		m.loading = false
		m.records = nil
		m.cursor = 0
	}
	return m, nil
}

func (m BinaryListModel) View() string {
	title := styles.TitleStyle.Render("Список бинарных файлов")

	var content string
	if m.loading {
		content = lipgloss.JoinVertical(lipgloss.Center,
			title,
			"",
			m.renderLoading(),
			"",
			styles.HelpStyle.Render("Ожидайте • Esc: назад"),
		)
	} else if m.errorMsg != "" {
		content = lipgloss.JoinVertical(lipgloss.Center,
			title,
			"",
			styles.ErrorStyle.Render("Ошибка: "+m.errorMsg),
			"",
			styles.HelpStyle.Render("R: повторить загрузку • Esc: назад"),
		)
	} else if len(m.records) == 0 {
		content = lipgloss.JoinVertical(lipgloss.Center,
			title,
			"",
			lipgloss.NewStyle().Foreground(styles.SecondaryColor).Render("Нет загруженных файлов"),
			"",
			styles.HelpStyle.Render("R: обновить • Esc: назад"),
		)
	} else {
		subtitle := lipgloss.NewStyle().
			Foreground(styles.SecondaryColor).
			Render(fmt.Sprintf("Найдено файлов: %d", len(m.records)))

		var items []string
		for i, rec := range m.records {
			item := fmt.Sprintf("Файл #%d", rec.GetId())
			if i == m.cursor {
				items = append(items,
					styles.ButtonActiveStyle.
						Width(50).
						Render("> "+item),
				)
			} else {
				items = append(items,
					lipgloss.NewStyle().
						Padding(0, 4).
						Width(50).
						Render("  "+item),
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
			styles.HelpStyle.Render("↑/↓: навигация • Enter: просмотреть/скачать • R: обновить • Esc: назад"),
		)
	}

	return m.Center(content)
}

func (m BinaryListModel) renderLoading() string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := frames[int(time.Now().UnixNano()/int64(time.Millisecond))%len(frames)]
	return lipgloss.NewStyle().
		Foreground(styles.PrimaryColor).
		Render(frame + " Загрузка списка файлов...")
}

func (m BinaryListModel) loadBinaries() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		records, err := m.grpcClient.ListBinaries(ctx)
		if err != nil {
			return message.ErrorMsg{Error: fmt.Sprintf("Ошибка загрузки списка: %v", err)}
		}
		return BinariesLoadedMsg{Records: records}
	}
}
