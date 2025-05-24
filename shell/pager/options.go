package pager

import (
	"time"

	"github.com/charmbracelet/gum/style"
)

// Options are the options for the pager.
type Options struct {
	//nolint:staticcheck
	Style               style.Styles  `hidden:"" embed:"" help:"Style the pager" set:"defaultBorder=rounded" set:"defaultPadding=0 1" set:"defaultBorderForeground=236" envprefix:"GUM_PAGER_"`
	Content             string        `hidden:"" arg:"" optional:"" help:"Display content to scroll"`
	ShowLineNumbers     bool          `hidden:"" help:"Show line numbers" default:"true"`
	LineNumberStyle     style.Styles  `hidden:"" embed:"" prefix:"line-number." help:"Style the line numbers" set:"defaultForeground=237" envprefix:"GUM_PAGER_LINE_NUMBER_"`
	SoftWrap            bool          `hidden:"" help:"Soft wrap lines" default:"true" negatable:""`
	MatchStyle          style.Styles  `hidden:"" embed:"" prefix:"match." help:"Style the matched text" set:"defaultForeground=196" set:"defaultBold=true" envprefix:"GUM_PAGER_MATCH_"`                                                      //nolint:staticcheck
	MatchHighlightStyle style.Styles  `hidden:"" embed:"" prefix:"match-highlight." help:"Style the matched highlight text" set:"defaultForeground=196" set:"defaultBackground=241" set:"defaultBold=true" envprefix:"GUM_PAGER_MATCH_HIGH_"` //nolint:staticcheck
	Timeout             time.Duration `hidden:"" help:"Timeout until command exits" default:"0s" env:"GUM_PAGER_TIMEOUT"`

	Scroll bool   `help:"Scroll content" short:"s" default:"false"`
	File   string `help:"File to scroll"`

	// Deprecated: this has no effect anymore.
	HelpStyle style.Styles `hidden:"" embed:"" prefix:"help." help:"Style the help text" set:"defaultForeground=241" envprefix:"GUM_PAGER_HELP_" hidden:""`
}
