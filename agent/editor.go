package agent

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/qiangli/ai/swarm/api"
)

type errMsg error

type model struct {
	textarea textarea.Model
	err      error

	cfg *api.AppConfig

	canceled bool
}

func SimpleEditor(cfg *api.AppConfig, text string) (string, bool, error) {
	m := initialModel()
	m.cfg = cfg

	if len(text) > 0 {
		m.textarea.SetValue(text)
	}

	p := tea.NewProgram(m)
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
			if m.textarea.Focused() {
				m.textarea.Blur()
			}
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
	// We handle errors just like any other message
	case errMsg:
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
	return fmt.Sprintf(layout, m.cfg.Me, m.textarea.View())
}
