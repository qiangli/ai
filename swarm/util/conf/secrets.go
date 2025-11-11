package conf

import (
	"fmt"
	"os"
	"strings"

	"github.com/qiangli/ai/swarm/api"
)

var getApiKey func(string) string

func init() {
	// save the env before they are cleared during runtime
	var env = make(map[string]string)
	// env["openai"] = os.Getenv("OPENAI_API_KEY")
	// env["gemini"] = os.Getenv("GEMINI_API_KEY")
	// env["anthropic"] = os.Getenv("ANTHROPIC_API_KEY")
	//
	// env["google"] = os.Getenv("GOOGLE_SEARCH_ENGINE_ID") + ":" + os.Getenv("GOOGLE_API_KEY")
	// env["brave"] = os.Getenv("BRAVE_API_KEY")
	// env["dhnt"] = os.Getenv("DHNT_API_KEY")

	// store all keys ending in _API_KEY
	// and make it available as api key without the suffix
	for _, kv := range os.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			if strings.HasSuffix(parts[0], "_API_KEY") {
				k := strings.ToLower(strings.TrimSuffix(parts[0], "_API_KEY"))
				env[k] = parts[1]
			}
		}
	}

	// google is special.
	env["google"] = os.Getenv("GOOGLE_SEARCH_ENGINE_ID") + ":" + os.Getenv("GOOGLE_API_KEY")

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
