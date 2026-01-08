package components

import (
	"gophkeeper/cmd/client/internal/ui/styles"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// InputField - кастомный компонент ввода
type InputField struct {
	textinput.Model
	Placeholder string
	IsPassword  bool
	Width       int
	Focused     bool
}

// NewInput создаёт новое поле ввода
func NewInput(placeholder string, isPassword bool) InputField {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(styles.SecondaryColor)
	ti.CharLimit = 100
	ti.Width = 30
	
	if isPassword {
		ti.EchoMode = textinput.EchoPassword
		ti.EchoCharacter = '•'
	}
	
	return InputField{
		Model:       ti,
		Placeholder: placeholder,
		IsPassword:  isPassword,
		Width:       30,
	}
}

// Update обновляет состояние поля ввода
func (i InputField) Update(msg tea.Msg) (InputField, tea.Cmd) {
	var cmd tea.Cmd
	i.Model, cmd = i.Model.Update(msg)
	return i, cmd
}

// Focus переводит фокус на поле
func (i InputField) Focus() (InputField, tea.Cmd) {
	i.Model.Focus()
	i.Focused = true
	return i, i.Model.Cursor.BlinkCmd()
}

// Blur снимает фокус с поля
func (i InputField) Blur() InputField {
	i.Model.Blur()
	i.Focused = false
	return i
}

// View отображает поле ввода
func (i InputField) View() string {
	style := styles.InputStyle
	if i.Focused {
		style = styles.InputFocusedStyle
	}
	
	// Маскируем пароль в preview
	value := i.Model.Value()
	if i.IsPassword && i.Focused {
		value = strings.Repeat("•", len(value))
	}
	
	return style.Render(value)
}

// Reset сбрасывает значение поля
func (i InputField) Reset() InputField {
	i.Model.SetValue("")
	return i
}

// SetValue устанавливает значение поля
func (i InputField) SetValue(value string) InputField {
	i.Model.SetValue(value)
	return i
}