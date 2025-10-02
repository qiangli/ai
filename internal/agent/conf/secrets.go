package conf

import (
	"fmt"
	"os"

	"github.com/qiangli/ai/swarm/api"
)

var getApiKey func(string) string

func init() {
	// save the env before they are cleared during runtime
	var env = make(map[string]string)
	env["openai"] = os.Getenv("OPENAI_API_KEY")
	env["gemini"] = os.Getenv("GEMINI_API_KEY")
	env["anthropic"] = os.Getenv("ANTHROPIC_API_KEY")
	//
	env["google"] = os.Getenv("GOOGLE_SEARCH_ENGINE_ID") + ":" + os.Getenv("GOOGLE_API_KEY")
	env["brave"] = os.Getenv("BRAVE_API_KEY")

	getApiKey = func(provider string) string {
		return env[provider]
	}
}

type secrets struct {
}

var LocalSecrets api.SecretStore = &secrets{}

func (r *secrets) Get(owner, key string) (string, error) {
	ak := getApiKey(key)
	if ak != "" {
		return ak, nil
	}
	return "", fmt.Errorf("api key not found: %s", key)
}
