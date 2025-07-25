package bubble

import (
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/qiangli/ai/internal/bubble/choose"
	"github.com/qiangli/ai/internal/bubble/confirm"
	"github.com/qiangli/ai/internal/bubble/edit"
	"github.com/qiangli/ai/internal/bubble/file"
	"github.com/qiangli/ai/internal/bubble/write"
)

// color index
// red: 196
// green: 46
// blue: 21
// black: 16
// white: 231
// gray 232, 241, 255
// pink 212

// Gum is the command-line interface for Gum.
type Gum struct {
	// Choose provides an interface to choose one option from a given list of
	// options. The options can be provided as (new-line separated) stdin or a
	// list of arguments.
	// Let's pick from a list of gum flavors:
	//
	// $ gum choose "Strawberry" "Banana" "Cherry"
	//
	Choose choose.Options `cmd:"" help:"Choose an option from a list of choices"`

	// Confirm provides an interface to ask a user to confirm an action.
	// The user is provided with an interface to choose an affirmative or
	// negative answer, which is then reflected in the exit code for use in
	// scripting.
	//
	// If the user selects the affirmative answer, the program exits with 0.
	// If the user selects the negative answer, the program exits with 1.
	Confirm confirm.Options `cmd:"" help:"Ask a user to confirm an action"`

	// File provides an interface to pick a file from a folder (tree).
	// The user is provided a file manager-like interface to navigate, to
	// select a file.
	//
	// Let's pick a file from the current directory:
	//
	// $ gum file
	// $ gum file .
	//
	// Let's pick a file from the home directory:
	//
	// $ gum file $HOME
	File file.Options `cmd:"" help:"Pick a file from a folder"`

	// Write provides a shell script interface for the text area bubble.
	// https://github.com/charmbracelet/bubbles/tree/master/textarea
	//
	// It can be used to ask the user to write some long form of text
	// (multi-line) input. The text the user entered will be sent to stdout.
	//
	// $ gum write > output.text
	//
	Write write.Options `cmd:"" help:"Prompt for long-form text"`

	Edit edit.Options `cmd:"" help:"Prompt for long-form text"`
}

func BubbleGum(args []string) (*Gum, error) {
	lipgloss.SetColorProfile(termenv.NewOutput(os.Stderr).Profile)

	parse := func(cli any, options ...kong.Option) (*kong.Context, error) {
		parser, err := kong.New(cli, options...)
		if err != nil {
			return nil, err
		}
		ctx, err := parser.Parse(args)
		parser.FatalIfErrorf(err)
		return ctx, nil
	}

	gum := &Gum{}
	ctx, err := parse(
		gum,
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact:             true,
			Summary:             false,
			NoExpandSubcommands: true,
		}),
		kong.Vars{
			"defaultHeight":           "0",
			"defaultWidth":            "0",
			"defaultAlign":            "left",
			"defaultBorder":           "none",
			"defaultBorderForeground": "",
			"defaultBorderBackground": "",
			"defaultBackground":       "",
			"defaultForeground":       "",
			"defaultMargin":           "0 0",
			"defaultPadding":          "0 0",
			"defaultUnderline":        "false",
			"defaultBold":             "false",
			"defaultFaint":            "false",
			"defaultItalic":           "false",
			"defaultStrikethrough":    "false",
		},
	)
	if err != nil {
		return nil, err
	}
	if err := ctx.Run(); err != nil {
		return nil, err
	}
	return gum, nil
}

func Choose(prompt string, options []string, multiple bool) (string, error) {
	args := []string{"choose", "--header", prompt}
	if multiple {
		args = append(args, "--no-limit")
	}
	args = append(args, options...)
	g, err := BubbleGum(args)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(g.Choose.Result), nil
}

func Confirm(prompt string) (string, error) {
	g, err := BubbleGum([]string{"confirm", prompt})
	if err != nil {
		return "", err
	}
	return g.Confirm.Result, nil
}

func PickFile(prompt string, path string) (string, error) {
	g, err := BubbleGum([]string{"file", "--all=false", "--file", "--directory=false", "--header", prompt, path})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(g.File.Result), nil
}

func Write(prompt, placeholder, value string) (string, error) {
	g, err := BubbleGum([]string{"write", "--header", prompt, "--placeholder", placeholder, "--value", value})
	if err != nil {
		return "", err
	}
	return g.Write.Result, nil
}

func Edit(prompt, placeholder, value string) (string, error) {
	g, err := BubbleGum([]string{"edit", "--header", prompt, "--placeholder", placeholder, "--value", value})
	if err != nil {
		return "", err
	}
	return g.Edit.Result, nil
}

func Help() {
	BubbleGum([]string{"--help"})
}
