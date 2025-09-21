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

	log.Infoln("Building oh docker image, please wait...")
	if err := BuildImage(ctx); err != nil {
		return err
	}

	log.Infoln("Running...")
	if err := RunContainer(ctx, query); err != nil {
		return err
	}
	return nil
}
