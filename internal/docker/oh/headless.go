package oh

import (
	"context"
	"fmt"
	"strings"

	"github.com/qiangli/ai/swarm/log"
)

func Run(ctx context.Context, query string) error {
	query = strings.TrimSpace(query)
	if len(query) == 0 {
		return fmt.Errorf("query is required")
	}

	log.Infof("Building oh docker image, please wait...\n")
	if err := BuildImage(ctx); err != nil {
		return err
	}

	log.Infof("Running...\n")
	if err := RunContainer(ctx, query); err != nil {
		return err
	}
	return nil
}
