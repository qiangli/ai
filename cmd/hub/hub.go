package hub

import (
	"github.com/spf13/cobra"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
)

var HubCmd = &cobra.Command{
	Use:   "hub",
	Short: "Manage hub service",
}

func init() {
	HubCmd.CompletionOptions.DisableDefaultCmd = true
}

func setLogLevel(app *api.AppConfig) {
	if app.Quiet {
		log.SetLogLevel(log.Quiet)
		return
	}
	if app.Debug {
		log.SetLogLevel(log.Verbose)
	}
}

func setLogOutput(path string) (*log.FileWriter, error) {
	if path != "" {
		f, err := log.NewFileWriter(path)
		if err != nil {
			return nil, err
		}
		log.SetLogOutput(f)
		return f, nil
	}
	return nil, nil
}
