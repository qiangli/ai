package confirm

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/qiangli/ai/internal/bubble/util/stdin"
	"github.com/qiangli/ai/internal/bubble/util/timeout"
)

const Yes = "yes"
const No = "no"

// Run provides a shell script interface for prompting a user to confirm an
// action with an affirmative or negative answer.
func (o *Options) Run() error {
	line, err := stdin.Read(stdin.SingleLine(true))
	if err == nil {
		switch line {
		case "yes", "y":
			o.Result = Yes
			return nil
		default:
			o.Result = No
			return nil
		}
	}

	ctx, cancel := timeout.Context(o.Timeout)
	defer cancel()

	m := model{
		affirmative:      o.Affirmative,
		negative:         o.Negative,
		showOutput:       o.ShowOutput,
		confirmation:     o.Default,
		defaultSelection: o.Default,
		keys:             defaultKeymap(o.Affirmative, o.Negative),
		help:             help.New(),
		showHelp:         o.ShowHelp,
		prompt:           o.Prompt,
		selectedStyle:    o.SelectedStyle.ToLipgloss(),
		unselectedStyle:  o.UnselectedStyle.ToLipgloss(),
		promptStyle:      o.PromptStyle.ToLipgloss(),
	}
	tm, err := tea.NewProgram(
		m,
		tea.WithOutput(os.Stderr),
		tea.WithContext(ctx),
	).Run()
	if err != nil && ctx.Err() != context.DeadlineExceeded {
		return fmt.Errorf("unable to confirm: %w", err)
	}
	m = tm.(model)

	if o.ShowOutput {
		confirmationText := m.negative
		if m.confirmation {
			confirmationText = m.affirmative
		}
		fmt.Println(m.prompt, confirmationText)
	}

	if m.confirmation {
		o.Result = Yes
		return nil
	}

	o.Result = No
	return nil
}
