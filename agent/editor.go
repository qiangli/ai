package agent

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

type errMsg error

type model struct {
	textarea textarea.Model
	err      error

	title string

	canceled bool
}

func SimpleEditor(title, content string) (string, bool, error) {
	m := initialModel()
	m.title = title

	//
	if len(content) > 0 {
		m.textarea.SetValue(content)
	}

	p := tea.NewProgram(m, tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		return "", true, err
	}

	if m.canceled {
		return "", true, nil
	}

	// TODO investigate: extra spaces
	v := m.textarea.Value()
	s := strings.Join(strings.Fields(v), " ")
	return s, false, nil
}

func initialModel() *model {
	ti := textarea.New()
	ti.Placeholder = "Enter your message here..."
	ti.MaxHeight = 20
	ti.MaxWidth = 70
	ti.SetHeight(8)
	ti.SetWidth(70)
	ti.Focus()

	return &model{
		textarea: ti,
		err:      nil,
	}
}

func (m *model) Init() tea.Cmd {
	return textarea.Blink
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
		case tea.KeyCtrlC:
			m.canceled = true
			return m, tea.Quit
		case tea.KeyCtrlD:
			m.canceled = false
			return m, tea.Quit
		default:
			if !m.textarea.Focused() {
				cmd = m.textarea.Focus()
				cmds = append(cmds, cmd)
			}
		}
	case tea.MouseMsg:
		//TODO notworking
	case errMsg:
		// We handle errors just like any other message
		m.err = msg
		return m, nil
	}

	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	const layout = `
%s

%s
             
←↑↓→ Move cursor, Ctrl+D send, Ctrl+C cancel

`
	return fmt.Sprintf(layout, m.title, m.textarea.View())
}
