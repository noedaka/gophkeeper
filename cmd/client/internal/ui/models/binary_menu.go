// models/binary_menu.go
package models

import (
    grpcclient "gophkeeper/cmd/client/internal/grpc_client"
    "gophkeeper/cmd/client/internal/ui/styles"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type BinaryMenuModel struct {
    BaseModel
    token      string
    userID     string
    login      string
    grpcClient *grpcclient.GophKeeperClient
    menuItems  []string
    cursor     int
}

func NewBinaryMenuModel(token, userID, login string, grpcClient *grpcclient.GophKeeperClient) BinaryMenuModel {
    return BinaryMenuModel{
        BaseModel:  NewBaseModel(),
        token:      token,
        userID:     userID,
        login:      login,
        grpcClient: grpcClient,
        menuItems: []string{
            "Загрузить файл",
            "Список файлов",
            "Назад",
        },
        cursor: 0,
    }
}

func (m BinaryMenuModel) Init() tea.Cmd {
    return nil
}

func (m BinaryMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
        case "enter":
            switch m.cursor {
            case 0:
                return NewAddBinaryModel(m.token, m.userID, m.login, m.grpcClient), nil
            case 1:
                return NewBinaryListModel(m.token, m.userID, m.login, m.grpcClient), nil
            case 2:
                return NewMainModel(m.token, m.userID, m.login, m.grpcClient), nil
            }
        case "esc":
            return NewMainModel(m.token, m.userID, m.login, m.grpcClient), nil
        }
    case tea.WindowSizeMsg:
        m.UpdateSize(msg)
    }
    return m, nil
}

func (m BinaryMenuModel) View() string {
    title := styles.TitleStyle.Render("Бинарные файлы")

    var items []string
    for i, item := range m.menuItems {
        if i == m.cursor {
            items = append(items, styles.ButtonActiveStyle.Width(40).Render("> "+item))
        } else {
            items = append(items, lipgloss.NewStyle().PaddingLeft(2).Width(40).Render("  "+item))
        }
    }

    menu := lipgloss.JoinVertical(lipgloss.Left, items...)
    menu = lipgloss.NewStyle().MarginTop(2).Render(menu)

    content := lipgloss.JoinVertical(lipgloss.Center,
        title,
        "",
        menu,
        m.RenderError(),
        "",
        styles.HelpStyle.Render("↑/↓: навигация • Enter: выбрать • Esc: назад"),
    )

    return m.Center(content)
}