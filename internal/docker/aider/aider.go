package aider

import (
	"context"
	"fmt"
	"strings"

	"github.com/qiangli/ai/internal/log"
)

func Run(ctx context.Context, mode ChatMode, query string) error {
	query = strings.TrimSpace(query)
	if len(query) == 0 {
		return fmt.Errorf("query is required")
	}

	log.Infoln("Building aider docker image, please wait...")
	if err := BuildImage(ctx); err != nil {
		return err
	}

	log.Infoln("Running aider...")
	if err := RunContainer(ctx, mode, query); err != nil {
		return err
	}
	return nil
}
