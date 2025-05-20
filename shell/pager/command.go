package pager

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/qiangli/ai/shell/stdin"
)

// Run provides a shell script interface for the viewport bubble.
// https://github.com/charmbracelet/bubbles/viewport
func (o *Options) Run() error {
	if o.File != "" {
		data, err := os.ReadFile(o.File)
		if err != nil {
			return err
		}
		o.Content = string(data)
	} else {
		data, err := stdin.GetContent()
		if err != nil {
			return err
		}
		o.Content = data
	}
	if !o.Scroll {
		return Pager(o.Content)
	}
	return o.Pager()
}

func (o *Options) Pager() error {
	vp := viewport.New(o.Style.Width, o.Style.Height)
	vp.Style = o.Style.ToLipgloss()

	if o.Content == "" {
		return nil
		// d, err := stdin.Read()
		// if err != nil {
		// 	return fmt.Errorf("unable to read stdin")
		// }
		// if d != "" {
		// 	// Sanitize the input from stdin by removing backspace sequences.
		// 	backspace := regexp.MustCompile(".\x08")
		// 	o.Content = backspace.ReplaceAllString(d, "")
		// } else {
		// 	return fmt.Errorf("provide some content to display")
		// }
	}

	m := model{
		viewport:            vp,
		help:                help.New(),
		content:             o.Content,
		origContent:         o.Content,
		showLineNumbers:     o.ShowLineNumbers,
		lineNumberStyle:     o.LineNumberStyle.ToLipgloss(),
		softWrap:            o.SoftWrap,
		matchStyle:          o.MatchStyle.ToLipgloss(),
		matchHighlightStyle: o.MatchHighlightStyle.ToLipgloss(),
		keymap:              defaultKeymap(),
	}

	ctx, cancel := timeoutContext(o.Timeout)
	defer cancel()

	_, err := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithReportFocus(),
		tea.WithContext(ctx),
	).Run()
	if err != nil {
		return fmt.Errorf("unable to start program: %w", err)
	}

	return nil
}
