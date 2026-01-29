package binary

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"gophkeeper/cmd/client/internal/crypto"
	grpcclient "gophkeeper/cmd/client/internal/grpc_client"
	"gophkeeper/cmd/client/internal/ui/components"
	"gophkeeper/cmd/client/internal/ui/models/base"
	message "gophkeeper/cmd/client/internal/ui/models/messages"
	"gophkeeper/cmd/client/internal/ui/navigation"
	"gophkeeper/cmd/client/internal/ui/styles"
	"gophkeeper/internal/proto"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AddBinaryModel управляет добавлением бинарника
type AddBinaryModel struct {
	base.BaseModel
	token      string
	userID     string
	login      string
	grpcClient *grpcclient.GophKeeperClient
	crypt      *crypto.Crypt
	nav        navigation.Navigator

	pathInput components.InputField
	metadata  components.InputField

	fileExists bool
	fileSize   int64

	loading bool
}

type BinaryAddedMsg struct {
	ID int32
}

func NewAddBinaryModel(token, userID, login string, grpcClient *grpcclient.GophKeeperClient, crypt *crypto.Crypt, nav navigation.Navigator) AddBinaryModel {
	m := AddBinaryModel{
		BaseModel:  base.NewBaseModel(),
		token:      token,
		userID:     userID,
		login:      login,
		grpcClient: grpcClient,
		crypt:      crypt,
		nav:        nav,
		pathInput:  components.NewInput("Полный путь к файлу (C:\\... или /...)", false),
		metadata:   components.NewInput("Метка/Описание (опционально)", false),
	}

	m.pathInput, _ = m.pathInput.Focus()

	return m
}

func (m AddBinaryModel) Init() tea.Cmd {
	return nil
}

func (m AddBinaryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			return NewBinaryMenuModel(m.token, m.userID, m.login, m.grpcClient, m.crypt, m.nav), nil
		}
		if msg.String() == "enter" && m.fileExists {
			m.loading = true
			return m, m.uploadFile()
		}
		if msg.String() == "tab" {
			if m.pathInput.Focused {
				m.pathInput = m.pathInput.Blur()
				m.metadata, _ = m.metadata.Focus()
			} else {
				m.metadata = m.metadata.Blur()
				m.pathInput, _ = m.pathInput.Focus()
			}
		}
	case tea.WindowSizeMsg:
		m.UpdateSize(msg)
	case BinaryAddedMsg:
		menu := NewBinaryMenuModel(m.token, m.userID, m.login, m.grpcClient, m.crypt, m.nav)
		menu.SetError(fmt.Sprintf("Файл успешно загружен (ID: %d)", msg.ID))
		return menu, nil
	case message.ErrorMsg:
		m.SetError(msg.Error)
		m.loading = false
	}

	var cmd tea.Cmd
	m.pathInput, cmd = m.pathInput.Update(msg)
	cmds = append(cmds, cmd)
	m.metadata, cmd = m.metadata.Update(msg)
	cmds = append(cmds, cmd)

	path := m.pathInput.Model.Value()
	if path != "" {
		info, err := os.Stat(path)
		m.fileExists = err == nil && !info.IsDir()
		if m.fileExists {
			m.fileSize = info.Size()
		}
	} else {
		m.fileExists = false
	}

	return m, tea.Batch(cmds...)
}

func (m AddBinaryModel) View() string {
	title := styles.TitleStyle.Render("Загрузить бинарный файл")

	pathLabel := "Путь к файлу:"
	if m.pathInput.Focused {
		pathLabel = "-> " + pathLabel
	}

	metaLabel := "Метка/Описание:"
	if m.metadata.Focused {
		metaLabel = "-> " + metaLabel
	}

	status := "Файл не выбран"
	if m.fileExists {
		status = fmt.Sprintf("Файл найден (%d байт)", m.fileSize)
	} else if m.pathInput.Model.Value() != "" {
		status = "Файл не найден или это папка"
	}

	form := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().MarginBottom(1).Render(pathLabel),
		m.pathInput.View(),
		"",
		lipgloss.NewStyle().Foreground(styles.SecondaryColor).Render(status),
		"",
		lipgloss.NewStyle().MarginBottom(1).Render(metaLabel),
		m.metadata.View(),
	)
	form = lipgloss.NewStyle().Width(80).Render(form)

	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		form,
		"",
		m.RenderError(),
		m.renderProgress(),
		"",
		styles.HelpStyle.Render("Tab: переключение полей • Enter: загрузить (если файл существует) • Esc: назад"),
	)

	return m.Center(content)
}

func (m AddBinaryModel) renderProgress() string {
	if !m.loading {
		return ""
	}
	return lipgloss.NewStyle().Foreground(styles.PrimaryColor).Render("Загрузка файла...")
}

func (m AddBinaryModel) uploadFile() tea.Cmd {
	if m.pathInput.Value() == "" {
		return func() tea.Msg {
			return message.ErrorMsg{Error: "Файл не выбран"}
		}
	}
	return func() tea.Msg {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in uploadFile: %v", r)
			}
		}()
		log.Printf("uploadFile started for file: %s", m.pathInput.Value())
		file, err := os.Open(m.pathInput.Value())
		if err != nil {
			log.Printf("Open file error: %v", err)
			return message.ErrorMsg{Error: fmt.Sprintf("Не удалось открыть файл: %v", err)}
		}
		defer file.Close()
		metadata := m.metadata.Model.Value()
		if metadata == "" {
			metadata = filepath.Base(m.pathInput.Value())
		}
		log.Printf("Metadata: %s", metadata)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		stream, err := m.grpcClient.UploadBinary(ctx)
		if err != nil {
			log.Printf("UploadBinary stream error: %v", err)
			return message.ErrorMsg{Error: fmt.Sprintf("Ошибка открытия stream: %v", err)}
		}
		log.Printf("Stream opened successfully")
		const chunkSize = 4 * 1024 * 1024
		buf := make([]byte, chunkSize)
		sequence := int32(0)
		first := true
		var sentData bool
		hasMoreData := true
		for hasMoreData {
			n, readErr := file.Read(buf)
			log.Printf("Read %d bytes from file", n)
			if readErr != nil && readErr != io.EOF {
				log.Printf("Read error: %v", readErr)
				return message.ErrorMsg{Error: fmt.Sprintf("Ошибка чтения файла: %v", readErr)}
			}

			var plainChunk []byte
			skipSend := false

			if n == 0 {
				if readErr == io.EOF {
					if sentData {
						skipSend = true
					} else {
						plainChunk = []byte{}
					}
				} else {
					continue
				}
			} else {
				plainChunk = buf[:n]
			}

			if skipSend {
				hasMoreData = false
				continue
			}

			encrypted, nonce, encErr := m.crypt.Encrypt(plainChunk)
			if encErr != nil {
				return message.ErrorMsg{Error: fmt.Sprintf("Ошибка шифрования: %v", encErr)}
			}

			chunkData := append(nonce, encrypted...)

			meta := ""
			if first {
				meta = metadata
				first = false
			}

			isLast := readErr == io.EOF
			chunk := proto.BinaryChunk_builder{
				Data:     chunkData,
				Sequence: &sequence,
				Metadata: &meta,
				IsLast:   &isLast,
			}
			builtChunk := chunk.Build()
			log.Printf("Sending chunk: size=%d (encrypted), sequence=%d, is_last=%v", len(builtChunk.GetData()), sequence, isLast)
			sendErr := stream.Send(builtChunk)
			if sendErr != nil {
				log.Printf("Send error: %v", sendErr)
				return message.ErrorMsg{Error: fmt.Sprintf("Ошибка отправки чанка: %v", sendErr)}
			}
			log.Printf("Chunk sent successfully")
			sequence++

			if n > 0 {
				sentData = true
			}

			if readErr == io.EOF {
				hasMoreData = false
			}
		}
		log.Printf("All chunks sent, closing stream")
		resp, closeErr := stream.CloseAndRecv()
		if closeErr != nil {
			log.Printf("CloseAndRecv error: %v", closeErr)
			return message.ErrorMsg{Error: fmt.Sprintf("Ошибка завершения upload: %v", closeErr)}
		}
		log.Printf("Upload successful, ID=%d", resp.GetId())
		return BinaryAddedMsg{ID: resp.GetId()}
	}
}
