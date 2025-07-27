package hub

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/qiangli/ai/internal/log"
)

var hubAddress string

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop Hub service",
	Run: func(cmd *cobra.Command, args []string) {
		flags := cmd.Flags()
		addr, _ := flags.GetString("hub-address")

		stopHub(addr)
	},
}

func stopHub(addr string) {
	// Create the shutdown endpoint URL
	shutdownURL := fmt.Sprintf("http://%s/shutdown", addr)

	// Create a new HTTP request
	req, err := http.NewRequest("POST", shutdownURL, nil)
	if err != nil {
		log.Errorf("Failed to create request: %v\n", err)
		return
	}

	// Perform the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Failed to perform request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode == http.StatusOK {
		log.Infoln("Hub shutdown successfully.")
	} else {
		log.Errorf("Failed to shutdown hub: %s\n", resp.Status)
	}
}

func init() {
	flags := stopCmd.Flags()
	flags.String("hub-address", ":58080", "Hub service host:port")

	stopCmd.CompletionOptions.DisableDefaultCmd = true
	HubCmd.AddCommand(stopCmd)
}
