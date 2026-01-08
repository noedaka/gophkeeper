package models

import (
	"context"
	"gophkeeper/cmd/client/internal/ui/styles"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// BaseModel содержит общую логику для всех моделей
type BaseModel struct {
	Width       int
	Height      int
	Ctx         context.Context
	Cancel      context.CancelFunc
	Error       string
	Loading     bool
	LoadingText string
}

// NewBaseModel создаёт новую базовую модель
func NewBaseModel() BaseModel {
	ctx, cancel := context.WithCancel(context.Background())
	return BaseModel{
		Ctx:    ctx,
		Cancel: cancel,
	}
}

// UpdateSize обновляет размеры окна
func (m *BaseModel) UpdateSize(msg tea.WindowSizeMsg) {
	m.Width = msg.Width
	m.Height = msg.Height
}

// SetError устанавливает ошибку
func (m *BaseModel) SetError(err string) {
	m.Error = err
	m.Loading = false
}

// ClearError очищает ошибку
func (m *BaseModel) ClearError() {
	m.Error = ""
}

// SetLoading устанавливает состояние загрузки
func (m *BaseModel) SetLoading(loading bool, text ...string) {
	m.Loading = loading
	if len(text) > 0 {
		m.LoadingText = text[0]
	} else {
		m.LoadingText = "Загрузка..."
	}
}

// RenderError отображает ошибку, если есть
func (m *BaseModel) RenderError() string {
	if m.Error == "" {
		return ""
	}
	return styles.ErrorStyle.Render("Error: " + m.Error)
}

// RenderLoading отображает индикатор загрузки
func (m *BaseModel) RenderLoading() string {
	if !m.Loading {
		return ""
	}

	// Простой спиннер (анимация)
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := frames[int(time.Now().UnixNano()/int64(time.Millisecond))%len(frames)]

	return lipgloss.NewStyle().
		Foreground(styles.PrimaryColor).
		MarginTop(1).
		Render(frame + " " + m.LoadingText)
}

// Center выравнивает контент по центру
func (m BaseModel) Center(content string) string {
	return lipgloss.Place(
		m.Width,
		m.Height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}
