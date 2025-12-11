package agent

import (
	"slices"

	"fmt"
	"os"
	"path/filepath"
	// "strings"

	"github.com/google/uuid"

	"github.com/qiangli/ai/internal/util"
	"github.com/qiangli/ai/swarm/api"
	// "github.com/qiangli/ai/swarm/atm/conf"
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

var Version = "0.0.1" // version of the ai binary

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

var validFormats = []string{"raw", "text", "json", "markdown", "tts"}

func isValidFormat(format string) bool {
	return slices.Contains(validFormats, format)
}

func Validate(app *api.AppConfig) error {
	if app.Format != "" && !isValidFormat(app.Format) {
		return fmt.Errorf("invalid format: %s", app.Format)
	}
	return nil
}

func EnsureWorkspace(ws string) (string, error) {
	workspace, err := validatePath(ws)
	if err != nil {
		return "", err
	}

	// ensure the workspace directory exists
	if err := os.MkdirAll(workspace, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	return workspace, nil
}

// ValidatePath returns the absolute path of the given path.
// If the path is empty, it returns an error.
// If the path is not an absolute path, it converts it to an absolute path.
// If the path does not exist, it returns its absolute path.
func validatePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}

	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path: %w", err)
		}
		path = absPath
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return path, nil
		}
		return "", fmt.Errorf("failed to stat path: %w", err)
	}

	return path, nil
}

func SetupAppConfig(app *api.App) error {
	// app.Format = "markdown"
	// app.LogLevel = "quiet"
	app.Session = uuid.NewString()

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	app.Base = filepath.Join(home, ".ai")

	ws := filepath.Join(app.Base, "workspace")
	if v, err := EnsureWorkspace(ws); err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	} else {
		app.Workspace = v
	}

	return nil
}

func getSpecialInput(argv []string) (*api.InputConfig, error) {
	cfg := ParseSpecialChars(argv)
	// argm, err := conf.ParseActionArgs(argv)
	// if err != nil {
	// 	return err
	// }
	// maps.Copy(app.Arguments, argm)

	// in, err := GetUserInput(cfg, api.ToString(argm["message"]))
	// if err != nil {
	// 	return err
	// }
	// app.Message = in.Message

	if cfg.Stdin {
		content, err := ReadStdin()
		if err != nil {
			return nil, err
		}
		cfg.Message = content
	}
	return cfg, nil
}

func Run(argv []string) error {
	// shebang
	// TODO load args from first line of
	var app = &api.App{}
	// app.Arguments = make(map[string]any)
	err := SetupAppConfig(app)
	if err != nil {
		return err
	}
	// read args[0] file and parse first line for args

	// if conf.IsAction(argv[0]) {
	// 	cfg, err := getSpecialInput(argv)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if cfg.Message != "" {
	// 		app.Arguments["message"] = cfg.Message
	// 		argv = cfg.Args
	// 	}
	// } else if conf.IsSlash(argv[0]) {
	// 	// call local system command as tool:
	// 	// sh:bash command
	// 	app.Arguments["kit"] = "sh"
	// 	app.Arguments["name"] = "bash"
	// 	app.Arguments["command"] = strings.Join(argv, " ")
	// } else {
	// 	app.Arguments["message"] = strings.Join(argv, " ")
	// }

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
