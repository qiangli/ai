package agent

import (
	"slices"

	// "fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/qiangli/ai/internal/util"
	"github.com/qiangli/ai/swarm/api"
)

const DefaultEditor = "vi"

const StdinRedirect = "-"

// clipboard redirection
const (
	// read from clipboard
	ClipinRedirect = "{"

	// read from clipboard and wait, allowing multiple copy
	// until Ctrl-D is entered
	ClipinRedirect2 = "{{"

	// write to clipboard, overwriting its content
	ClipoutRedirect = "}"

	// append to clipboard
	ClipoutRedirect2 = "}}"
)

// var Version = "0.0.1" // version of the ai binary

// parse special char sequence for stdin/clipboard
// they can:
// + be at the end of the args
// + be in any order
// + be multiple instances
func ParseSpecialChars(args []string) *api.InputConfig {
	var isStdin, isClipin, isClipWait, isClipout, isClipAppend bool

	if len(args) > 0 {
	loop:
		for i := len(args) - 1; i >= 0; i-- {
			lastArg := args[i]
			switch lastArg {
			case StdinRedirect:
				isStdin = true
			case ClipinRedirect:
				isClipin = true
			case ClipinRedirect2:
				isClipin = true
				isClipWait = true
			case ClipoutRedirect:
				isClipout = true
			case ClipoutRedirect2:
				isClipout = true
				isClipAppend = true
			default:
				// continue until a non special char
				break loop
			}
			args = args[:i+1]
		}
	}

	var cfg api.InputConfig

	cfg.Stdin = isStdin
	cfg.Clipin = isClipin
	cfg.ClipWait = isClipWait
	cfg.Clipout = isClipout
	cfg.ClipAppend = isClipAppend

	cfg.Args = args

	return &cfg
}

var validFormats = []string{"txt", "json", "markdown", "yaml"}

func isValidFormat(format string) bool {
	return slices.Contains(validFormats, format)
}

// func Validate(app *api.AppConfig) error {
// 	if app.Format != "" && !isValidFormat(app.Format) {
// 		return fmt.Errorf("invalid format: %s", app.Format)
// 	}
// 	return nil
// }

func SetupAppConfig(app *api.App) error {
	app.Session = uuid.NewString()
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	app.Base = filepath.Join(home, ".ai")
	return nil
}

func Run(argv []string) error {
	var app = &api.App{}
	err := SetupAppConfig(app)
	if err != nil {
		return err
	}

	//
	var user *api.User
	who, _ := util.WhoAmI()
	app.User = who
	if v, err := loadUser(app.Base); err != nil {
		user = &api.User{
			Display:  who,
			Settings: make(map[string]any),
		}
	} else {
		user = v
		user.Display = who
	}

	if err := RunSwarm(app, user, argv); err != nil {
		return err
	}
	return nil
}
