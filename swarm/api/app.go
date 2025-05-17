package api

import (
	"fmt"
	"strings"
)

type AppConfig struct {
	Version string

	ConfigFile string

	LLM *LLMConfig

	Git    *GitConfig
	DBCred *DBCred

	Role   string
	Prompt string

	Agent   string
	Command string
	Args    []string

	// --message takes precedence over all other forms of input
	Message string

	Editor string

	Clipin   bool
	ClipWait bool

	Clipout    bool
	ClipAppend bool

	IsPiped bool
	Stdin   bool

	Files []string

	// MCP server url
	McpServerUrl string

	// Output format: raw or markdown
	Format string

	// Save output to file
	Output string

	Me string

	//
	Template string

	Log      string
	Debug    bool
	Quiet    bool
	Internal bool

	Unsafe bool

	//
	Workspace string
	Repo      string
	Home      string
	Temp      string

	Interactive bool
	Editing     bool
	Shell       string
	Watch       bool

	// MetaPrompt  bool

	MaxTime  int
	MaxTurns int

	//
	Stdout string
	Stderr string
}

func (r *AppConfig) IsStdin() bool {
	return r.Stdin || r.IsPiped
}

func (r *AppConfig) IsSpecial() bool {
	return r.IsStdin() || r.Clipin
}

func (r *AppConfig) HasInput() bool {
	return r.Message != "" || len(r.Files) > 0 || len(r.Args) > 0
}

func (r *AppConfig) GetQuery() string {
	if r.Message != "" {
		return r.Message
	}
	return strings.Join(r.Args, " ")
}

type GitConfig struct {
}

type DBCred struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"name"`
}

// DSN returns the data source name for connecting to the database.
func (d *DBCred) DSN() string {
	host := d.Host
	if host == "" {
		host = "localhost"
	}
	port := d.Port
	if port == "" {
		port = "5432"
	}
	dbname := d.DBName
	if dbname == "" {
		dbname = "postgres"
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, d.Username, d.Password, dbname)
}

func (d *DBCred) IsValid() bool {
	return d.Username != "" && d.Password != ""
}

func (d *DBCred) Clone() *DBCred {
	return &DBCred{
		Host:     d.Host,
		Port:     d.Port,
		Username: d.Username,
		Password: d.Password,
		DBName:   d.DBName,
	}
}
