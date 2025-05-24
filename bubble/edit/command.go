package edit

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Run provides a shell script interface for the text area bubble.
// https://github.com/charmbracelet/bubbles/textarea
func (o *Options) Run() error {
	m := initialModel(o)

	//
	p := tea.NewProgram(m, tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		return err
	}

	if m.canceled {
		return nil
	}

	o.Result = m.textarea.Value()

	return nil
}
