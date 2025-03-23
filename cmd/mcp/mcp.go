package mcp

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/qiangli/ai/internal/log"
)

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
}

func setLogLevel() {
	debug := viper.GetBool("verbose")
	if debug {
		log.SetLogLevel(log.Verbose)
	}

	// trace
	log.Trace = viper.GetBool("trace")
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
