package rest

import (
	"context"
	"testing"
)

func TestSwagKit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	kit := &SwagKit{}
	ctx := context.Background()

	file := "/Users/qiang.li/10xdev/cloud/backend/cloud-swagger-docs/konfigurator.json"
	// file := "/Users/qiang.li/10xdev/cloud/backend/swagger-docs/unified-outs/cloud.json"
	doc, err := kit.LoadV2(ctx, file)
	if err != nil {
		t.Fatalf("Failed to load V2 doc: %v", err)
	}
	err = kit.Run(ctx, doc, "/pipelines-api/v1/libraries")
	if err != nil {
		t.Fatalf("Failed to run test: %v", err)
	}
}
