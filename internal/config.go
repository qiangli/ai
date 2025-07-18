package internal

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/tailscale/hujson"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/api/model"
)

const DefaultEditor = "ai -i edit"

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

func parseArgs(app *api.AppConfig, args []string, defaultAgent string) []string {
	newArgs := parseAgentArgs(app, args, defaultAgent)
	newArgs = parseSpecialChars(app, newArgs)
	return newArgs
}

// return the agent/command and the rest of the args
func parseAgentArgs(app *api.AppConfig, args []string, defaultAgent string) []string {
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
	var pbReadWait = viper.GetBool("pb_tail")
	var pbWrite = viper.GetBool("pb_write")
	var pbWriteAppend = viper.GetBool("pb_append")
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

//go:embed data/*
var configData embed.FS

func GetConfigData() embed.FS {
	return configData
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
	var app = &api.AppConfig{}

	app.ConfigFile = viper.ConfigFileUsed()
	app.Base = filepath.Dir(app.ConfigFile)

	app.Version = Version
	app.Role = viper.GetString("role")
	app.Prompt = viper.GetString("prompt")

	app.Me = "👤 " + getCurrentUser()
	app.Files = InputFiles
	app.Format = FormatFlag
	app.Output = OutputFlag

	//
	app.Message = viper.GetString("message")

	// app.Content = viper.GetString("content")
	// read input file if message is empty
	// inputFile := viper.GetString("input")
	// if inputFile != "" && app.Message == "" {
	// 	b, err := os.ReadFile(inputFile)
	// 	if err != nil {
	// 		return nil, errors.New("failed to read input file")
	// 	} else {
	// 		app.Message = string(b)
	// 	}
	// }

	app.Template = TemplateFile

	app.Screenshot = viper.GetBool("screenshot")
	app.Voice = viper.GetBool("voice")

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

	//
	app.New = viper.GetBool("new")
	app.MaxHistory = viper.GetInt("max_history")
	app.MaxSpan = viper.GetInt("max_span")

	app.MaxTurns = viper.GetInt("max_turns")
	app.MaxTime = viper.GetInt("max_time")

	//
	if !app.New {
		historyDir := filepath.Join(app.Base, "history")
		messages, err := api.LoadHistory(historyDir, app.MaxHistory, app.MaxSpan)
		if err != nil {
			return nil, fmt.Errorf("error loading history: %v", err)
		}
		app.History = messages
	}

	// LLM config
	var lc = &api.LLMConfig{}
	app.LLM = lc
	// default
	lc.Provider = viper.GetString("provider")

	lc.ApiKey = viper.GetString("api_key")
	lc.Model = viper.GetString("model")
	lc.BaseUrl = viper.GetString("base_url")

	// <provider>/<model>
	modelName := func(n string) string {
		if strings.Contains(n, "/") {
			return n
		}
		if lc.Provider == "" {
			return "openai/" + n
		}
		return lc.Provider + "/" + n
	}

	//
	alias := viper.GetString("models")
	// use same models to continue the conversation
	// if not set
	if alias == "" {
		if len(app.History) > 0 {
			last := app.History[len(app.History)-1]
			alias = last.Models
		}
	}
	app.Models = alias

	//
	modelBase := filepath.Join(app.Base, "models")
	modelCfg, err := model.LoadModels(modelBase)
	if err != nil {
		return nil, err
	}
	if alias != "" {
		if m, ok := modelCfg[alias]; ok {
			app.LLM.Models = m.Models
		}
	}

	// if no models, setup defaults
	if len(app.LLM.Models) == 0 {
		// all levels share same config
		var m model.Model
		switch {
		case lc.ApiKey != "" && lc.Model != "":
			// assume openai compatible
			m = model.Model{
				Name:    modelName(lc.Model),
				BaseUrl: lc.BaseUrl,
				ApiKey:  lc.ApiKey,
			}
		case os.Getenv("OPENAI_API_KEY") != "":
			m = model.Model{
				Name:    "openai/gpt-4.1-mini",
				BaseUrl: "https://api.openai.com/v1/",
				ApiKey:  os.Getenv("OPENAI_API_KEY"),
			}
		case os.Getenv("GEMINI_API_KEY") != "":
			m = model.Model{
				Name:    "gemini/gemini-2.0-flash-lite",
				BaseUrl: "",
				ApiKey:  os.Getenv("GEMINI_API_KEY"),
			}
		case os.Getenv("ANTHROPIC_API_KEY") != "":
			m = model.Model{
				Name:    "anthropic/claude-3-5-haiku-latest",
				BaseUrl: "",
				ApiKey:  os.Getenv("ANTHROPIC_API_KEY"),
			}
		default:
		}

		// TODO improve to allow any alias other than L*
		models := make(map[model.Level]*model.Model)
		models[model.L1] = m.Clone()
		models[model.L2] = m.Clone()
		models[model.L3] = m.Clone()

		app.LLM.Models = models
	}
	// update or add model from command line flags
	for _, l := range model.Levels {
		s := strings.ToLower(string(l))
		k := viper.GetString(s + "_api_key")
		n := viper.GetString(s + "_model")
		u := viper.GetString(s + "_base_url")
		if v, ok := app.LLM.Models[l]; ok {
			if k != "" {
				v.ApiKey = k
			}
			if n != "" {
				v.Name = modelName(n)
			}
			if u != "" {
				v.BaseUrl = u
			}
			app.LLM.Models[l] = v
		} else {
			app.LLM.Models[l] = &model.Model{
				Name:    modelName(n),
				ApiKey:  k,
				BaseUrl: u,
			}
		}
	}
	// model config is required
	if len(app.LLM.Models) == 0 {
		return nil, fmt.Errorf("No LLM configuration found")
	}

	// TODO
	tts := &api.TTSConfig{}
	tts.ApiKey = viper.GetString("tts_api_key")
	tts.Provider = viper.GetString("tts_provider")
	tts.Model = viper.GetString("tts_model")
	tts.BaseUrl = viper.GetString("tts_base_url")
	if tts.ApiKey == "" {
		tts.ApiKey = os.Getenv("OPENAI_API_KEY")
	}
	if tts.Model == "" {
		tts.Model = "gpt-4o-mini-tts"
	}

	app.TTS = tts

	//
	app.Log = viper.GetString("log")
	app.Debug = viper.GetBool("verbose")
	app.Quiet = viper.GetBool("quiet")
	app.Internal = viper.GetBool("internal")

	app.Unsafe = viper.GetBool("unsafe")
	toList := func(s string) []string {
		sa := strings.Split(s, ",")
		var list []string
		for _, v := range sa {
			list = append(list, strings.TrimSpace(v))
		}
		if len(list) > 0 {
			return list
		}
		return nil
	}
	app.DenyList = toList(viper.GetString("deny"))
	app.AllowList = toList(viper.GetString("allow"))

	app.Editor = viper.GetString("editor")
	app.Editing = viper.GetBool("edit")
	app.Interactive = viper.GetBool("interactive")

	app.Watch = viper.GetBool("watch")
	app.ClipWatch = viper.GetBool("pb_watch")

	// Hub services
	hub := &api.HubConfig{}
	hub.Enable = viper.GetBool("hub")
	hub.Address = viper.GetString("hub_address")
	hub.Pg = viper.GetBool("hub_pg")
	hub.PgAddress = viper.GetString("hub_pg_address")
	hub.Mysql = viper.GetBool("hub_mysql")
	hub.MysqlAddress = viper.GetString("hub_mysql_address")
	hub.Redis = viper.GetBool("hub_redis")
	hub.RedisAddress = viper.GetString("hub_redis_address")
	hub.Terminal = viper.GetBool("hub_terminal")
	hub.TerminalAddress = viper.GetString("hub_terminal_address")
	app.Hub = hub

	shell := viper.GetString("shell")
	if shell == "" {
		shell = "bash"
	}
	shellBin, _ := exec.LookPath(shell)
	if shellBin == "" {
		shellBin = "/bin/bash"
	}
	app.Shell = shellBin

	// default agent:
	// --agent, "ask"
	var defaultAgent = viper.GetString("agent")
	if defaultAgent == "" {
		defaultAgent = "ask"
	}
	app.Args = parseArgs(app, args, defaultAgent)

	// mcp
	app.McpServerRoot = viper.GetString("mcp.server_root")
	if app.McpServerRoot != "" {
		mcp := NewMcpServersConfig(app.McpServerRoot)
		if err := mcp.LoadAll(); err != nil {
			return nil, err
		}
		app.McpServers = mcp.ServersConfig
	}

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

type McpServersConfig struct {
	ServersRoot string

	ServersConfig map[string]*api.McpServerConfig `json:"mcpServers"`
}

func NewMcpServersConfig(root string) *McpServersConfig {
	return &McpServersConfig{
		ServersRoot:   root,
		ServersConfig: make(map[string]*api.McpServerConfig),
	}
}

func (r *McpServersConfig) LoadAll() error {
	entries, err := os.ReadDir(r.ServersRoot)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Type().IsRegular() {
			name := entry.Name()
			ext := filepath.Ext(name)
			if ext == ".jsonc" || ext == ".json" {
				configPath := filepath.Join(r.ServersRoot, name)
				if err := r.LoadFile(configPath); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (r *McpServersConfig) LoadFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return r.LoadData(data)
}

func (r *McpServersConfig) LoadData(data []byte) error {
	hu, err := hujson.Standardize(data)
	if err != nil {
		return err
	}
	ex := expandWithDefault(string(hu))
	err = json.Unmarshal([]byte(ex), r)
	if err != nil {
		return fmt.Errorf("unmarshal mcp config: %v", err)
	}

	// set server name for each config
	for k, v := range r.ServersConfig {
		v.Server = k
	}
	return nil
}

func expandWithDefault(input string) string {
	return os.Expand(input, func(key string) string {
		parts := strings.SplitN(key, ":-", 2)
		value := os.Getenv(parts[0])
		if value == "" && len(parts) > 1 {
			return parts[1]
		}
		return value
	})
}
