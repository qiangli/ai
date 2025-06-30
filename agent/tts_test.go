package agent

import (
	"os"
	"testing"

	"github.com/qiangli/ai/swarm/api"
)

func TestSpeak(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	var models = []string{
		"gpt-4o-mini-tts",
		// "tts-1",
		// "tts-1-hd",
	}
	var txt = "Why did the chicken cross the road? To get to the other side."

	for _, v := range models {
		cfg := &api.AppConfig{
			TTS: &api.TTSConfig{
				Provider: "openai",
				Model:    v,
				ApiKey:   os.Getenv("OPENAI_API_KEY"),
			},
		}
		err := speak(cfg, txt)
		t.Fatal(err)
	}
}
