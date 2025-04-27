package internal

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"github.com/qiangli/ai/swarm/api"
)

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

func parseArgs(app *api.AppConfig, args []string) []string {
	newArgs := parseAgentArgs(app, args)
	newArgs = parseSpecialChars(app, newArgs)
	return newArgs
}

// return the agent/command and the rest of the args
func parseAgentArgs(app *api.AppConfig, args []string) []string {
	defaultAgent := viper.GetString("agent")
	if defaultAgent == "" {
		defaultAgent = "ask"
	}
	shellAgent := "shell"

	// first or last arg could be the agent/command
	// the last takes precedence
	var arg string
	isAgent := func(s string) bool {
		return strings.HasPrefix(s, "@")
	}
	isSlash := func(s string) bool {
		return strings.HasPrefix(s, "/")
	}
	switch len(args) {
	case 0:
		// no args, use default agent
	case 1:
		if isSlash(args[0]) || isAgent(args[0]) {
			arg = args[0]
			args = args[1:]
		}
	default:
		if isSlash(args[0]) || isAgent(args[0]) {
			arg = args[0]
			args = args[1:]
		}
		// agent check only
		// slash could file path
		if isAgent(args[len(args)-1]) {
			arg = args[len(args)-1]
			args = args[:len(args)-1]
		}
	}

	var agent string
	if arg != "" {
		if arg[0] == '/' {
			agent = shellAgent + arg
		} else {
			agent = arg[1:]
		}
	}
	if agent == "" {
		agent = defaultAgent
	}

	parts := strings.SplitN(agent, "/", 2)
	app.Agent = parts[0]
	if len(parts) > 1 {
		app.Command = parts[1]
	}

	return args
}

// parse special char sequence for stdin/clipboard
// they can:
// + be at the end of the args or as a suffix to the last one
// + be in any order
// + be multiple instances
func parseSpecialChars(app *api.AppConfig, args []string) []string {
	// special char sequence handling
	var stdin = viper.GetBool("stdin")
	var pbRead = viper.GetBool("pb_read")
	var pbReadWait = viper.GetBool("pb_read_wait")
	var pbWrite = viper.GetBool("pb_write")
	var pbWriteAppend = viper.GetBool("pb_write_append")
	var isStdin, isClipin, isClipWait, isClipout, isClipAppend bool

	newArgs := make([]string, len(args))

	if len(args) > 0 {
		for i := len(args) - 1; i >= 0; i-- {
			lastArg := args[i]

			if lastArg == StdinRedirect {
				isStdin = true
			} else if lastArg == ClipinRedirect {
				isClipin = true
			} else if lastArg == ClipinRedirect2 {
				isClipin = true
				isClipWait = true
			} else if lastArg == ClipoutRedirect {
				isClipout = true
			} else if lastArg == ClipoutRedirect2 {
				isClipout = true
				isClipAppend = true
			} else {
				// check for suffix for cases where the special char is not the last arg
				// but is part of the last arg
				if strings.HasSuffix(lastArg, StdinRedirect) {
					isStdin = true
					args[i] = strings.TrimSuffix(lastArg, StdinRedirect)
				} else if strings.HasSuffix(lastArg, ClipinRedirect) {
					isClipin = true
					args[i] = strings.TrimSuffix(lastArg, ClipinRedirect)
				} else if strings.HasSuffix(lastArg, ClipinRedirect2) {
					isClipin = true
					isClipWait = true
					args[i] = strings.TrimSuffix(lastArg, ClipinRedirect2)
				} else if strings.HasSuffix(lastArg, ClipoutRedirect) {
					isClipout = true
					args[i] = strings.TrimSuffix(lastArg, ClipoutRedirect)
				} else if strings.HasSuffix(lastArg, ClipoutRedirect2) {
					isClipout = true
					isClipAppend = true
					args[i] = strings.TrimSuffix(lastArg, ClipoutRedirect2)
				}
				newArgs = args[:i+1]
				break
			}
		}
	}

	isPiped := func() bool {
		stat, _ := os.Stdin.Stat()
		return (stat.Mode() & os.ModeCharDevice) == 0
	}

	app.IsPiped = isPiped()
	app.Stdin = isStdin || stdin
	app.Clipin = isClipin || pbRead || pbReadWait
	app.ClipWait = isClipWait || pbReadWait
	app.Clipout = isClipout || pbWrite || pbWriteAppend
	app.ClipAppend = isClipAppend || pbWriteAppend

	return newArgs
}

var ConfigFile string

var InputFiles []string
var FormatFlag string
var OutputFlag string
var TemplateFile string

//go:embed ai.yaml
var configFileContent string

func GetDefaultConfig() string {
	return configFileContent
}

// global flags

var DryRun bool
var DryRunContent string

func getCurrentUser() string {
	currentUser, err := user.Current()
	if err != nil {
		return "ME"
	}
	return strings.ToUpper(currentUser.Username)
}

func ParseConfig(args []string) (*api.AppConfig, error) {
	var lc = &api.LLMConfig{}
	var app = &api.AppConfig{}

	app.LLM = lc
	app.Version = Version
	app.Role = viper.GetString("role")
	app.Prompt = viper.GetString("role_prompt")

	app.Me = "👤 " + getCurrentUser()
	app.Files = InputFiles
	app.Format = FormatFlag
	app.Output = OutputFlag
	app.ConfigFile = viper.ConfigFileUsed()

	app.Message = viper.GetString("message")
	// read input file if message is empty
	inputFile := viper.GetString("input")
	if inputFile != "" && app.Message == "" {
		b, err := os.ReadFile(inputFile)
		if err != nil {
			return nil, errors.New("failed to read input file")
		} else {
			app.Message = string(b)
		}
	}
	app.Template = TemplateFile

	//
	home, err := homeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	app.Home = home

	temp, err := tempDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get temp directory: %w", err)
	}
	app.Temp = temp

	ws := viper.GetString("workspace")
	ws, err = resolveWorkspaceDir(ws)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve workspace: %w", err)
	}
	app.Workspace = ws

	repo, err := resolveRepoDir(ws)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve repo directory: %w", err)
	}
	app.Repo = repo

	// LLM config
	lc.ApiKey = viper.GetString("api_key")
	lc.Model = viper.GetString("model")
	lc.BaseUrl = viper.GetString("base_url")

	lc.L1Model = viper.GetString("l1_model")
	lc.L1BaseUrl = viper.GetString("l1_base_url")
	lc.L1ApiKey = viper.GetString("l1_api_key")
	lc.L2Model = viper.GetString("l2_model")
	lc.L2BaseUrl = viper.GetString("l2_base_url")
	lc.L2ApiKey = viper.GetString("l2_api_key")
	lc.L3Model = viper.GetString("l3_model")
	lc.L3BaseUrl = viper.GetString("l3_base_url")
	lc.L3ApiKey = viper.GetString("l3_api_key")
	if lc.L1Model == "" {
		lc.L1Model = lc.Model
	}
	if lc.L2Model == "" {
		lc.L2Model = lc.Model
	}
	if lc.L3Model == "" {
		lc.L3Model = lc.Model
	}
	if lc.L1ApiKey == "" {
		lc.L1ApiKey = lc.ApiKey
	}
	if lc.L2ApiKey == "" {
		lc.L2ApiKey = lc.ApiKey
	}
	if lc.L3ApiKey == "" {
		lc.L3ApiKey = lc.ApiKey
	}
	if lc.L1BaseUrl == "" {
		lc.L1BaseUrl = lc.BaseUrl
	}
	if lc.L2BaseUrl == "" {
		lc.L2BaseUrl = lc.BaseUrl
	}
	if lc.L3BaseUrl == "" {
		lc.L3BaseUrl = lc.BaseUrl
	}
	lc.ImageModel = viper.GetString("image_model")
	lc.ImageBaseUrl = viper.GetString("image_base_url")
	lc.ImageApiKey = viper.GetString("image_api_key")

	//
	app.Log = viper.GetString("log")
	app.Debug = viper.GetBool("verbose")
	app.Quiet = viper.GetBool("quiet")
	app.Internal = viper.GetBool("internal")

	app.Unsafe = viper.GetBool("unsafe")

	app.Editor = viper.GetString("editor")
	app.Interactive = viper.GetBool("interactive")
	app.Watch = viper.GetBool("watch")

	shell := viper.GetString("shell")
	if shell == "" {
		shell = "bash"
	}
	shellBin, _ := exec.LookPath(shell)
	if shellBin == "" {
		shellBin = "/bin/bash"
	}
	app.Shell = shellBin

	app.MaxTurns = viper.GetInt("max_turns")
	app.MaxTime = viper.GetInt("max_time")

	app.Args = parseArgs(app, args)

	// mcp
	app.McpServerUrl = viper.GetString("mcp.server_url")

	// sql db
	dbCfg := &api.DBCred{}
	dbCfg.Host = viper.GetString("sql.db_host")
	dbCfg.Port = viper.GetString("sql.db_port")
	dbCfg.Username = viper.GetString("sql.db_username")
	dbCfg.Password = viper.GetString("sql.db_password")
	dbCfg.DBName = viper.GetString("sql.db_name")
	app.DBCred = dbCfg
	//
	gitConfig := &api.GitConfig{}
	app.Git = gitConfig

	return app, nil
}

// resolveWorkspaceDir returns the workspace directory.
// If the workspace is not provided, it returns the current working directory
// or the directory of the git repository containing the current working directory
// if it is in a git repository.
func resolveWorkspaceDir(ws string) (string, error) {
	if ws != "" {
		return ensureWorkspace(ws)
	}
	ws, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return resolveRepoDir(ws)
}

// resolveRepoDir returns the directory of the current git repository
func resolveRepoDir(ws string) (string, error) {
	if ws == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		ws = wd
	}
	dir, err := detectGitRepo(ws)
	if err != nil {
		return "", fmt.Errorf("failed to detect git repository: %w", err)
	}
	return dir, nil
}

func homeDir() (string, error) {
	return os.UserHomeDir()
}

func tempDir() (string, error) {
	return os.TempDir(), nil
}

// detectGitRepo returns the directory of the git repository
// containing the given path.
// If the path is not in a git repository, it returns the original path.
func detectGitRepo(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}
	original := path
	for {
		if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
			return path, nil
		}
		np := filepath.Dir(path)
		if np == path || np == "/" {
			break
		}
		path = np
	}
	return original, nil
}

func ensureWorkspace(ws string) (string, error) {
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
// If the path exists, it returns its absolute path.
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
