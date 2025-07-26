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

	"github.com/google/uuid"
	fangs "github.com/spf13/viper"
	"github.com/tailscale/hujson"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
)

var V *fangs.Viper

func init() {
	V = fangs.New()
}

func ParseArgs(viper *fangs.Viper, app *api.AppConfig, args []string, defaultAgent string) {
	newArgs := ParseAgentArgs(app, args, defaultAgent)
	newArgs = ParseSpecialChars(viper, app, newArgs)
	app.Args = newArgs
}

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
		return "unkown"
	}
	return currentUser.Username
}

func ParseConfig(viper *fangs.Viper, app *api.AppConfig, args []string) error {
	app.ConfigFile = viper.ConfigFileUsed()
	app.Base = filepath.Dir(app.ConfigFile)

	app.Version = Version
	app.Role = viper.GetString("role")
	app.Prompt = viper.GetString("prompt")

	// TODO read from user.json
	user := getCurrentUser()
	app.Me = &api.User{
		Username: user,
		Display:  "👤 " + strings.ToUpper(user),
	}

	app.Files = InputFiles
	app.Format = FormatFlag
	app.Output = OutputFlag

	//
	app.Message = viper.GetString("message")

	app.Template = TemplateFile

	app.Screenshot = viper.GetBool("screenshot")
	app.Voice = viper.GetBool("voice")

	//
	home, err := homeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	app.Home = home

	temp, err := tempDir()
	if err != nil {
		return fmt.Errorf("failed to get temp directory: %w", err)
	}
	app.Temp = temp

	ws := viper.GetString("workspace")
	ws, err = resolveWorkspaceDir(ws)
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}
	app.Workspace = ws

	//
	app.New = viper.GetBool("new")
	app.ChatID = viper.GetString("chat")
	app.MaxHistory = viper.GetInt("max_history")
	app.MaxSpan = viper.GetInt("max_span")

	app.MaxTurns = viper.GetInt("max_turns")
	app.MaxTime = viper.GetInt("max_time")

	//
	if !app.New {
		baseDir := filepath.Join(app.Base, "chat")

		var cid = app.ChatID
		if cid == "" {
			if last, err := api.FindLastChatID(baseDir); err == nil {
				cid = last
			}
		}

		if cid != "" {
			chatDir := filepath.Join(baseDir, cid)
			log.Debugf("Loading old conversation from %s", chatDir)
			messages, err := api.LoadHistory(chatDir, app.MaxHistory, app.MaxSpan)
			if err != nil {
				return fmt.Errorf("error loading history: %v", err)
			}
			app.ChatID = cid
			app.History = messages
		}
	}

	// ensure chat id is assigned
	if app.ChatID == "" {
		app.ChatID = uuid.New().String()
	}

	if err := ParseLLM(viper, app); err != nil {
		return err
	}

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
	hub.LLMProxy = viper.GetBool("hub_llm_proxy")
	hub.LLMProxyAddress = viper.GetString("hub_llm_proxy_address")
	hub.LLMProxySecret = viper.GetString("hub_llm_proxy_secret")
	hub.LLMProxyApiKey = viper.GetString("hub_llm_proxy_api_key")
	//
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

	//
	ParseArgs(viper, app, args, defaultAgent)

	// mcp
	app.McpServerRoot = viper.GetString("mcp.server_root")
	if app.McpServerRoot != "" {
		mcp := NewMcpServersConfig(app.McpServerRoot)
		if err := mcp.LoadAll(); err != nil {
			return err
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

	return nil
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
