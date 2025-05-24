package edit

import (
	"os"

	"golang.org/x/term"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type keymap struct {
	textarea.KeyMap
	Submit key.Binding
	Quit   key.Binding
	Abort  key.Binding
}

// FullHelp implements help.KeyMap.
func (k keymap) FullHelp() [][]key.Binding { return nil }

// ShortHelp implements help.KeyMap.
func (k keymap) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(
			key.WithKeys("up", "down", "right", "left"),
			key.WithHelp("←↓↑→", "navigate"),
		),
		k.Submit,
		k.Abort,
	}
}

func defaultKeymap() keymap {
	km := textarea.DefaultKeyMap
	// km.InsertNewline = key.NewBinding(
	// 	key.WithKeys("ctrl+j"),
	// 	key.WithHelp("ctrl+j", "insert newline"),
	// )
	return keymap{
		KeyMap: km,
		Quit: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "quit"),
		),

		Abort: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "cancel"),
		),
		// OpenInEditor: key.NewBinding(
		// 	key.WithKeys("ctrl+e"),
		// 	key.WithHelp("ctrl+e", "open editor"),
		// ),
		Submit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "submit"),
		),
	}
}

type errMsg error

type model struct {
	textarea textarea.Model
	err      error

	header      string
	headerStyle lipgloss.Style

	help   help.Model
	keymap keymap

	canceled bool
}

func initialModel(o *Options) *model {
	textarea := textarea.New()
	textarea.Placeholder = o.Placeholder
	// textarea.MaxHeight = 20
	// textarea.MaxWidth = 80
	if o.Height > 0 {
		textarea.SetHeight(o.Height)
	}
	if o.Width > 0 {
		textarea.SetWidth(o.Width)
	} else {
		if width, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
			w := width - 6
			if w > 0 {
				textarea.SetWidth(w)
			}
		}
	}

	textarea.SetValue(o.Value)
	textarea.Placeholder = o.Placeholder

	textarea.Focus()

	km := defaultKeymap()

	return &model{
		textarea:    textarea,
		header:      o.Header,
		headerStyle: o.HeaderStyle.ToLipgloss(),
		keymap:      km,
		err:         nil,
		help:        help.New(),
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
			if len(m.textarea.Value()) > 0 {
				return m, tea.Quit
			}
		default:
			if !m.textarea.Focused() {
				cmd = m.textarea.Focus()
				cmds = append(cmds, cmd)
			}
		}
	case tea.MouseMsg:
		//TODO
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
	var parts []string
	if m.header != "" {
		parts = append(parts, m.headerStyle.Render(m.header+"\n"))
	}
	parts = append(parts, m.textarea.View())
	parts = append(parts, "", m.help.View(m.keymap))

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}
