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

	file := "~/10xdev/cloud/backend/swagger-docs/unified-outs/cloud.json"

	file, err := expandPath(file)
	if err != nil {
		t.Fatalf("Failed to expand path: %v", err)
	}
	doc, err := kit.LoadV2(ctx, file)
	if err != nil {
		t.Fatalf("Failed to load V2 doc: %v", err)
	}
	t.Logf("Loaded V2 doc: %+v", doc)
}
