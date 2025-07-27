package hub

import (
	"github.com/spf13/cobra"
)

var HubCmd = &cobra.Command{
	Use:   "hub",
	Short: "Manage hub service",
}

func init() {
	HubCmd.CompletionOptions.DisableDefaultCmd = true
}
