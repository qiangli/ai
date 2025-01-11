package gptr

import (
	"context"
	"fmt"
	"strings"

	"github.com/qiangli/ai/internal/log"
)

func GenerateReport(ctx context.Context, query string, out string) error {
	query = strings.TrimSpace(query)
	if len(query) == 0 {
		return fmt.Errorf("query is required")
	}

	log.Infoln("Building gptr docker image, please wait...")
	if err := BuildImage(ctx); err != nil {
		return err
	}

	log.Infoln("Generating report...")
	if err := RunContainer(ctx, query, out); err != nil {
		return err
	}
	return nil
}
