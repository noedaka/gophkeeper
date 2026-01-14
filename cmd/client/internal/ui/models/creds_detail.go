// creds_detail.go (новый файл, копия CardDetailModel)

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

type CredsDetailModel struct {
    BaseModel
    token      string
    userID     string
    login      string
    grpcClient *grpcclient.GophKeeperClient
    credID     int32
    cred       *proto.Credentials
    loading    bool
    errorMsg   string
    menuItems  []string
    cursor     int
}

type (
    CredLoadedMsg struct {
        Cred *proto.Credentials
    }
    CredLoadErrorMsg struct {
        Error string
    }
    CredDeleteConfirmMsg struct{}
)

func NewCredsDetailModel(token, userID, login string, grpcClient *grpcclient.GophKeeperClient, credID int32) CredsDetailModel {
    m := CredsDetailModel{
        BaseModel:  NewBaseModel(),
        token:      token,
        userID:     userID,
        login:      login,
        grpcClient: grpcClient,
        credID:     credID,
        cred:       nil,
        loading:    true,
        menuItems:  []string{"Удалить запись", "Назад"},
        cursor:     0,
    }
    return m
}

func (m CredsDetailModel) Init() tea.Cmd {
    return m.loadCred
}

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
                credsListModel := NewCredsListModel(m.token, m.userID, m.login, m.grpcClient)
                return credsListModel, credsListModel.Init()
            }
        case "esc":
            credsListModel := NewCredsListModel(m.token, m.userID, m.login, m.grpcClient)
            return credsListModel, credsListModel.Init()
        }
    case tea.WindowSizeMsg:
        m.UpdateSize(msg)
    case CredLoadedMsg:
        m.cred = msg.Cred
        m.loading = false
    case CredLoadErrorMsg:
        m.errorMsg = msg.Error
        m.loading = false
    case CredDeleteConfirmMsg:
        credsListModel := NewCredsListModel(m.token, m.userID, m.login, m.grpcClient)
        credsListModel.SetError("Запись успешно удалена")
        return credsListModel, credsListModel.Init()
    }
    return m, nil
}

func (m CredsDetailModel) View() string {
    if m.Width == 0 || m.Height == 0 {
        return "Загрузка..."
    }

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
            m.renderCredField("Логин:", m.cred.GetUsername()),
            m.renderCredField("Пароль:", m.cred.GetPassword()),
            m.renderCredField("Сервис:", m.cred.GetServiceName()),
            m.renderCredField("Метка:", m.cred.GetMetadata()),
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

func (m CredsDetailModel) loadCred() tea.Msg {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    cred, err := m.grpcClient.GetCredentials(ctx, m.credID)
    if err != nil {
        return CredLoadErrorMsg{Error: fmt.Sprintf("Ошибка загрузки: %v", err)}
    }
    return CredLoadedMsg{Cred: cred}
}

func (m CredsDetailModel) deleteCred() tea.Msg {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    err := m.grpcClient.DeleteCredentials(ctx, m.credID)
    if err != nil {
        return CredLoadErrorMsg{Error: fmt.Sprintf("Ошибка удаления: %v", err)}
    }
    return CredDeleteConfirmMsg{}
}