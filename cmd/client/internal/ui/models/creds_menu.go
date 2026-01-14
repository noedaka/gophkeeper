// creds_menu.go (новый файл, копия card_menu.go с небольшими изменениями)

package models

import (
    "fmt"
    grpcclient "gophkeeper/cmd/client/internal/grpc_client"
    "gophkeeper/cmd/client/internal/ui/styles"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

// CredsMenuModel управляет меню работы с логинами/паролями
type CredsMenuModel struct {
    BaseModel
    token      string
    userID     string
    login      string
    menuItems  []string
    cursor     int
    grpcClient *grpcclient.GophKeeperClient
}

// NewCredsMenuModel создаёт новую модель меню логинов/паролей
func NewCredsMenuModel(token, userID, login string, grpcClient *grpcclient.GophKeeperClient) CredsMenuModel {
    return CredsMenuModel{
        BaseModel: NewBaseModel(),
        token:     token,
        userID:    userID,
        login:     login,
        menuItems: []string{
            "Добавить логин/пароль",
            "Список записей",
            "Назад",
        },
        cursor:     0,
        grpcClient: grpcClient,
    }
}

// Init инициализирует модель
func (m CredsMenuModel) Init() tea.Cmd {
    return nil
}

// Update обрабатывает сообщения
func (m CredsMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
            if m.cursor < len(m.menuItems)-1 {
                m.cursor++
            }
        case "enter", " ":
            switch m.cursor {
            case 0: // Добавить
                addCredsModel := NewAddCredsModel(m.token, m.userID, m.login, m.grpcClient)
                return addCredsModel, addCredsModel.Init()
            case 1: // Список
                credsListModel := NewCredsListModel(m.token, m.userID, m.login, m.grpcClient)
                return credsListModel, credsListModel.Init()
            case 2: // Назад
                mainModel := NewMainModel(m.token, m.userID, m.login, m.grpcClient)
                return mainModel, mainModel.Init()
            }
        case "esc":
            mainModel := NewMainModel(m.token, m.userID, m.login, m.grpcClient)
            return mainModel, mainModel.Init()
        }
    case tea.WindowSizeMsg:
        m.UpdateSize(msg)
    }
    return m, nil
}

// View отображает интерфейс
func (m CredsMenuModel) View() string {
    if m.Width == 0 || m.Height == 0 {
        return "Загрузка..."
    }

    title := styles.TitleStyle.Render("Логины и пароли")
    subtitle := lipgloss.NewStyle().
        Foreground(styles.SecondaryColor).
        Render(fmt.Sprintf("Пользователь: %s", m.login))

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

    content := lipgloss.JoinVertical(lipgloss.Center,
        title,
        subtitle,
        "",
        menu,
        "",
        m.RenderError(),
        "",
        styles.HelpStyle.Render("↑/↓: навигация • Enter: выбрать • Esc: назад"),
    )

    return m.Center(content)
}