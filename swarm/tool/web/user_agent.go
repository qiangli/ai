package web

import (
	_ "embed"
	"encoding/json"
	"log"
	"math/rand"
)

//go:embed resource/desktop.json
var desktopUserAgentJSON []byte

//go:embed resource/mobile.json
var mobileUserAgentJSON []byte

var usersAgents []string

var defaultDesktopUserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36",
}

var defaultMobileUserAgents = []string{
	"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 18_3_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.3.1 Mobile/15E148 Safari/604.1",
}

func init() {
	var desktop []string
	if err := json.Unmarshal(desktopUserAgentJSON, &desktop); err != nil {
		log.Printf("failed to unmarshal desktop user agents: %v", err)
		usersAgents = append(usersAgents, defaultDesktopUserAgents...)
	}
	var mobile []string
	if err := json.Unmarshal(mobileUserAgentJSON, &mobile); err != nil {
		log.Printf("failed to unmarshal mobile user agents: %v", err)
		usersAgents = append(usersAgents, defaultMobileUserAgents...)
	}
	usersAgents = append(usersAgents, desktop...)
	usersAgents = append(usersAgents, mobile...)
}

func UserAgent() string {
	randomIndex := rand.Intn(len(usersAgents))
	return usersAgents[randomIndex]
}
