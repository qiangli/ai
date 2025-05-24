package shell

import (
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/mattn/go-shellwords"

	"github.com/qiangli/ai/shell/pager"
)

// https://github.com/alecthomas/kong
type Sub struct {
	Page pager.Options `cmd:"" help:"Scroll through a file"`
}

func runSub(cmdline string) error {
	args, err := shellwords.Parse(cmdline)
	if err != nil {
		return err
	}
	sub := &Sub{}
	ctx, err := parseKong(sub, args)
	if err != nil {
		return err
	}
	return ctx.Run()
}

func parseKong(sub any, args []string) (*kong.Context, error) {
	parse := func(cli any, options ...kong.Option) (*kong.Context, error) {
		parser, err := kong.New(cli, options...)
		if err != nil {
			return nil, err
		}
		ctx, err := parser.Parse(args)
		parser.FatalIfErrorf(err)
		return ctx, err
	}

	ctx, err := parse(
		sub,
		// kong.Description(fmt.Sprintf("A tool for %s shell scripts.", bubbleGumPink.Render("glamorous"))),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact:             true,
			Summary:             false,
			NoExpandSubcommands: true,
		}),
		kong.Vars{
			// "version":                 version,
			// "versionNumber":           Version,
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
		return nil, fmt.Errorf("failed to parse %v", err)
	}
	return ctx, nil
}

func Pager(content string) error {
	return pager.Pager(content)
	// sub := &Sub{}
	// args := []string{"page"}
	// ctx, err := parseKong(sub, args)
	// if err != nil {
	// 	return err
	// }
	// sub.Page.Content = content
	// return ctx.Run()
}
