package gptr

import (
	"context"
	"fmt"
	"strings"

	"github.com/qiangli/ai/internal/log"
)

func GenerateReport(query string, out string) error {
	query = strings.TrimSpace(query)
	if len(query) == 0 {
		return fmt.Errorf("query is required")
	}

	ctx := context.Background()

	log.Infoln("Building gptr docker image, please wait...")
	if err := BuildGPTRImage(ctx); err != nil {
		return err
	}

	log.Infoln("Generating report...")
	if err := RunGPTRContainer(ctx, query, out); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}
	return nil
}
