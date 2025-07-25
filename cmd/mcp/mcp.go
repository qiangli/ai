package mcp

import (
	"github.com/spf13/cobra"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/log"
)

var viper = internal.V

// var updated during build
var ServerName = "Stargate"
var ServerVersion = "0.0.1"

type ServerConfig struct {
	Port         int
	Host         string
	McpServerUrl string
	Transport    string
	Debug        bool

	//
	ServersRoot string

	// stdio
	Server string
	Args   []string
}

var config = &ServerConfig{}

var McpCmd = &cobra.Command{
	Use:                   "mcp",
	Short:                 "Manage MCP server",
	DisableFlagsInUseLine: true,
	DisableSuggestions:    true,
}

func init() {
	McpCmd.SetUsageTemplate(commandUsageTemplate)
	McpCmd.Flags().SortFlags = true
	McpCmd.CompletionOptions.DisableDefaultCmd = true

	//
	flags := McpCmd.PersistentFlags()
	flags.String("log", "", "Log all debugging information to a file")
	flags.Bool("verbose", false, "Show debugging information")

	flags.MarkHidden("log")

	viper.BindPFlag("log", flags.Lookup("log"))
	viper.BindPFlag("verbose", flags.Lookup("verbose"))

}

func setLogLevel() {
	debug := viper.GetBool("verbose")
	if debug {
		log.SetLogLevel(log.Verbose)
	}
}

func setLogOutput() (*log.FileWriter, error) {
	pathname := viper.GetString("log")
	if pathname != "" {
		f, err := log.NewFileWriter(pathname)
		if err != nil {
			return nil, err
		}
		log.SetLogOutput(f)
		return f, nil
	}
	return nil, nil
}
