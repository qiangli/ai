package agent

import (
	"context"
	"os"
	"testing"

	"github.com/qiangli/ai/swarm/api"
)

func TestProcessImageContent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	var filename = "c78c48fa-622c-437e-b0b1-44c31c7c1f94"

	ctx := context.TODO()
	cfg := &api.AppConfig{}
	msg := &api.Output{}

	b, err := os.ReadFile("../../" + filename)
	if err != nil {
		t.FailNow()
	}
	msg.Content = string(b)
	cfg.Output = "/tmp/" + filename + ".png"
	processImageContent(ctx, cfg, msg)
}
