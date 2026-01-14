package app

import (
	"log"

	grpcclient "gophkeeper/cmd/client/internal/grpc_client"
	"gophkeeper/cmd/client/internal/ui/models"
	"gophkeeper/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

// App управляет состоянием приложения
type App struct {
	grpcClient *grpcclient.GophKeeperClient
	model      tea.Model
}

// NewApp создаёт новое приложение
func NewApp() *App {
    cfg, err := config.Init()
    if err != nil {
        log.Printf("Ошибка парсинга переменных: %v", err)
    }

    // Подключаемся к серверу
    grpcClient, err := grpcclient.NewGophKeeperClient(cfg.GRPCServerAddress)
    if err != nil {
        log.Printf("Ошибка подключения к серверу: %v", err)
        log.Println("Запуск в offline режиме...")
        // Можно продолжить с nil клиентом для тестирования UI
    }

    // Начинаем с экрана аутентификации
    authModel := models.NewAuthModel(grpcClient)
    return &App{
        grpcClient: grpcClient,
        model: authModel,
    }
}

// Start запускает приложение
func (a *App) Start() error {
	defer func() {
		if a.grpcClient != nil {
			a.grpcClient.Close()
		}
	}()

	p := tea.NewProgram(a.model,
		tea.WithAltScreen(),       // Полноэкранный режим
		tea.WithMouseCellMotion(), // Поддержка мыши
		tea.WithFPS(60),           // Плавная анимация
	)

	// Запускаем TUI
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

// VersionInfo возвращает информацию о версии
func VersionInfo() string {
	return "GophKeeper v1.0.0\nBuild: 2024.01.15"
}
