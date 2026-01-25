package binary

import (
	"context"
	"fmt"
	"gophkeeper/cmd/client/internal/crypto"
	grpcclient "gophkeeper/cmd/client/internal/grpc_client"
	"gophkeeper/cmd/client/internal/ui/components"
	"gophkeeper/cmd/client/internal/ui/models/base"
	message "gophkeeper/cmd/client/internal/ui/models/messages"
	"gophkeeper/cmd/client/internal/ui/navigation"
	"gophkeeper/cmd/client/internal/ui/styles"
	"io"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// BinaryDetailModel управляет отображением детализированного бинарника
type BinaryDetailModel struct {
	base.BaseModel
	token        string
	userID       string
	login        string
	grpcClient   *grpcclient.GophKeeperClient
	crypt        *crypto.Crypt
	nav          navigation.Navigator
	recordID     int32
	metadata     string
	savePath     components.InputField
	menuItems    []string
	cursor       int
	loading      bool
	downloadPath string
}

type BinaryDownloadedMsg struct {
	Path string
}

type BinaryDeletedMsg struct{}

func NewBinaryDetailModel(token, userID, login string, grpcClient *grpcclient.GophKeeperClient, recordID int32, metadata string, crypt *crypto.Crypt, nav navigation.Navigator) BinaryDetailModel {
	suggestedPath := metadata
	if suggestedPath == "" {
		suggestedPath = fmt.Sprintf("file_%d.bin", recordID)
	}
	return BinaryDetailModel{
		BaseModel:  base.NewBaseModel(),
		token:      token,
		userID:     userID,
		login:      login,
		grpcClient: grpcClient,
		crypt:      crypt,
		nav:        nav,
		recordID:   recordID,
		metadata:   metadata,
		savePath:   components.NewInput("Путь для сохранения ("+suggestedPath+")", false),
		menuItems:  []string{"Скачать файл", "Удалить файл", "Назад"},
		cursor:     0,
		loading:    false,
	}
}

func (m BinaryDetailModel) Init() tea.Cmd {
	m.savePath, _ = m.savePath.Focus()
	return nil
}

func (m BinaryDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

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
				if m.savePath.Value() == "" {
					m.SetError("Укажите путь для сохранения")
					return m, nil
				}
				m.loading = true
				return m, m.downloadFile(m.savePath.Value())
			case 1:
				m.loading = true
				return m, m.deleteFile()
			case 2:
				return NewBinaryListModel(m.token, m.userID, m.login, m.grpcClient, m.crypt, m.nav), NewBinaryListModel(m.token, m.userID, m.login, m.grpcClient, m.crypt, m.nav).Init()
			}
		case "esc":
			return NewBinaryListModel(m.token, m.userID, m.login, m.grpcClient, m.crypt, m.nav), nil
		case "tab":
			if m.savePath.Focused {
				m.savePath = m.savePath.Blur()
			} else {
				m.savePath, _ = m.savePath.Focus()
			}
		}
	case tea.WindowSizeMsg:
		m.UpdateSize(msg)
	case BinaryDownloadedMsg:
		m.loading = false
		listModel := NewBinaryListModel(m.token, m.userID, m.login, m.grpcClient, m.crypt, m.nav)
		listModel.SetError(fmt.Sprintf("Файл успешно скачан: %s", msg.Path))
		return listModel, listModel.Init()
	case message.ErrorMsg:
		m.loading = false
		m.SetError(msg.Error)
	case BinaryDeletedMsg:
		m.loading = false
		listModel := NewBinaryListModel(m.token, m.userID, m.login, m.grpcClient, m.crypt, m.nav)
		listModel.SetError("Файл успешно удалён")
		return listModel, listModel.Init()
	}

	var cmd tea.Cmd
	m.savePath, cmd = m.savePath.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m BinaryDetailModel) View() string {
	title := styles.TitleStyle.Render(fmt.Sprintf("Файл #%d", m.recordID))
	name := m.metadata
	if name == "" {
		name = "Без названия"
	}
	info := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Width(20).Foreground(styles.SecondaryColor).Render("Название:"),
		lipgloss.NewStyle().Render(name),
		"",
		lipgloss.NewStyle().Width(20).Foreground(styles.SecondaryColor).Render("Путь сохранения:"),
		m.savePath.View(),
	)
	info = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(styles.PrimaryColor).Padding(1, 2).Width(60).Render(info)

	var menu []string
	for i, item := range m.menuItems {
		if i == m.cursor {
			menu = append(menu, styles.ButtonActiveStyle.Width(40).Render("> "+item))
		} else {
			menu = append(menu, lipgloss.NewStyle().Padding(0, 3).Width(40).Render(" "+item))
		}
	}
	menuView := lipgloss.JoinVertical(lipgloss.Left, menu...)
	menuView = lipgloss.NewStyle().MarginTop(2).Render(menuView)

	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		info,
		"",
		menuView,
		"",
		m.RenderError(),
		m.renderLoading(),
		"",
		styles.HelpStyle.Render("↑/↓: меню • Tab: поле пути • Enter: действие • Esc: назад"),
	)
	return m.Center(content)
}

func (m BinaryDetailModel) renderLoading() string {
	if !m.loading {
		return ""
	}
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := frames[int(time.Now().UnixNano()/int64(time.Millisecond))%len(frames)]
	return lipgloss.NewStyle().Foreground(styles.PrimaryColor).Render(frame + " Обработка...")
}

// downloadFile streaming скачивание с расшифровкой по чанкам
func (m BinaryDetailModel) downloadFile(savePath string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
			return message.ErrorMsg{Error: fmt.Sprintf("Ошибка создания директории: %v", err)}
		}

		file, err := os.Create(savePath)
		if err != nil {
			return message.ErrorMsg{Error: fmt.Sprintf("Не удалось создать файл: %v", err)}
		}
		defer file.Close()

		stream, err := m.grpcClient.DownloadBinary(ctx, m.recordID)
		if err != nil {
			return message.ErrorMsg{Error: fmt.Sprintf("Ошибка открытия stream: %v", err)}
		}

		const nonceSize = 24
		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				return BinaryDownloadedMsg{Path: savePath}
			}
			if err != nil {
				return message.ErrorMsg{Error: fmt.Sprintf("Ошибка получения чанка: %v", err)}
			}

			data := chunk.GetData()
			if len(data) < nonceSize+16 {
				return message.ErrorMsg{Error: "Повреждённый чанк (слишком короткий)"}
			}

			nonce := data[:nonceSize]
			ciphertext := data[nonceSize:]

			plain, decErr := m.crypt.Decrypt(ciphertext, nonce)
			if decErr != nil {
				return message.ErrorMsg{Error: fmt.Sprintf("Ошибка расшифровки: %v", decErr)}
			}

			if _, writeErr := file.Write(plain); writeErr != nil {
				return message.ErrorMsg{Error: fmt.Sprintf("Ошибка записи в файл: %v", writeErr)}
			}

			if chunk.GetIsLast() {
				return BinaryDownloadedMsg{Path: savePath}
			}
		}
	}
}

func (m BinaryDetailModel) deleteFile() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := m.grpcClient.DeleteBinary(ctx, m.recordID); err != nil {
			return message.ErrorMsg{Error: fmt.Sprintf("Ошибка удаления: %v", err)}
		}
		return BinaryDeletedMsg{}
	}
}
