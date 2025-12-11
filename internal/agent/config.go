package agent

import (
	"slices"

	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	// "github.com/qiangli/ai/internal/agent"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/log"
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
func ParseSpecialChars(app *api.AppConfig, args []string) []string {
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

	app.Stdin = isStdin
	app.Clipin = isClipin
	app.ClipWait = isClipWait
	app.Clipout = isClipout
	app.ClipAppend = isClipAppend

	return args
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

func setupAppConfig(ctx context.Context, app *api.AppConfig) error {
	app.Format = "markdown"
	app.LogLevel = "quiet"
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

func parseAppConfig(ctx context.Context, app *api.AppConfig, argv []string) error {
	argv = ParseSpecialChars(app, argv)
	argm, err := conf.ParseActionArgs(argv)
	if err != nil {
		return err
	}
	maps.Copy(app.Arguments, argm)

	in, err := GetUserInput(ctx, app, api.ToString(argm["message"]))
	if err != nil {
		return err
	}
	app.Message = in.Message
	return nil
}

func Run(ctx context.Context, argv []string) error {
	var app = &api.AppConfig{}
	app.Arguments = make(map[string]any)
	err := setupAppConfig(ctx, app)
	if err != nil {
		return err
	}

	if conf.IsAction(argv[0]) {
		err := parseAppConfig(ctx, app, argv)
		if err != nil {
			return err
		}
	} else if conf.IsSlash(argv[0]) {
		// call local system command as tool:
		// sh:bash command
		app.Arguments["kit"] = "sh"
		app.Arguments["name"] = "bash"
		app.Arguments["command"] = strings.Join(argv, " ")
	} else {
		app.Arguments["message"] = strings.Join(argv, " ")
	}

	level := api.ToLogLevel(app.Arguments["log_level"])
	log.GetLogger(ctx).SetLogLevel(level)
	log.GetLogger(ctx).Debugf("Config: %+v\n", app)

	if err := RunSwarm(ctx, app); err != nil {
		return err
	}
	return nil
}
