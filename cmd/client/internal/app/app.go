package app

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    grpcclient "gophkeeper/cmd/client/internal/grpc_client"
    "gophkeeper/cmd/client/internal/ui/models"
    "gophkeeper/internal/config"

    tea "github.com/charmbracelet/bubbletea"
)

// App управляет состоянием приложения
type App struct {
    grpcClient *grpcclient.GophKeeperClient
    model      tea.Model
    program    *tea.Program
}

// NewApp создаёт новое приложение
func NewApp() *App {
    cfg, err := config.Init()
    if err != nil {
        log.Printf("Ошибка парсинга переменных: %v", err)
    }

    grpcClient, err := grpcclient.NewGophKeeperClient(cfg.GRPCServerAddress, cfg.CACrtFile)
    if err != nil {
        log.Printf("Ошибка подключения к серверу: %v", err)
    }

    authModel := models.NewAuthModel(grpcClient)
    return &App{
        grpcClient: grpcClient,
        model:      authModel,
    }
}

// Start запускает приложение с graceful shutdown
func (a *App) Start() error {
    ctx, stop := signal.NotifyContext(context.Background(),
        os.Interrupt,
        syscall.SIGTERM,
        syscall.SIGINT,
        syscall.SIGQUIT)
    defer stop()

    a.program = tea.NewProgram(a.model,
        tea.WithAltScreen(),
        tea.WithMouseCellMotion(),
        tea.WithFPS(60),
    )

    done := make(chan error, 1)

    go func() {
        _, err := a.program.Run()
        done <- err
    }()

    select {
    case <-ctx.Done():
        log.Println("Получен сигнал завершения, закрываем соединения...")
        
        gracefulCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        
        if a.program != nil {
            a.program.Quit()
        }
        
        if a.grpcClient != nil {
            if err := a.grpcClient.Close(); err != nil {
                log.Printf("Ошибка при закрытии gRPC соединения: %v", err)
            } else {
                log.Println("gRPC соединение закрыто")
            }
        }
        
        select {
        case err := <-done:
            if err != nil {
                return err
            }
        case <-gracefulCtx.Done():
            log.Println("Таймаут graceful shutdown")
        }
        
        return nil
        
    case err := <-done:
        if err != nil {
            return err
        }
        
        if a.grpcClient != nil {
            if err := a.grpcClient.Close(); err != nil {
                log.Printf("Ошибка при закрытии gRPC соединения: %v", err)
            }
        }
        
        return nil
    }
}

// Shutdown явно завершает приложение
func (a *App) Shutdown() {
    if a.program != nil {
        a.program.Quit()
    }
    if a.grpcClient != nil {
        a.grpcClient.Close()
    }
}

