package conf

import (
	"bytes"
	"os"
	"testing"
)

// TestLoadModelsData_File loads the repository model.yaml (relative path) and ensures
// loadModelsData can parse the multi-document YAML and returns the expected L1/L2/L3 keys.
func TestLoadModelsData_File(t *testing.T) {
	// relative to this package (swarm/atm/conf)
	p := "../../resource/standard/agents/swe/model.yaml"
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("failed to read model.yaml at %s: %v", p, err)
	}

	// split YAML documents by the standard '---' separator. Keep non-empty parts.
	parts := bytes.Split(b, []byte("\n---\n"))
	var docs [][]byte
	for _, part := range parts {
		part = bytes.TrimSpace(part)
		if len(part) == 0 {
			continue
		}
		docs = append(docs, part)
	}

	mc, err := loadModelsData(docs)
	if err != nil {
		t.Fatalf("loadModelsData returned error: %v", err)
	}
	if mc == nil {
		t.Fatalf("expected non-nil AppConfig")
	}

	if len(mc.Models) == 0 {
		t.Fatalf("expected at least one model in AppConfig.Models")
	}

	// ensure L1/L2/L3 keys exist
	for _, k := range []string{"L1", "L2", "L3"} {
		if _, ok := mc.Models[k]; !ok {
			t.Fatalf("missing model level %s in merged config", k)
		}
	}

	// basic sanity: first doc in the repo model.yaml should be the OpenAI set
	if mc.Models["L1"].Provider != "openai" {
		t.Logf("warning: expected L1 provider 'openai', got '%s'", mc.Models["L1"].Provider)
	}
}

// TestLoadModelsData_MissingProvider ensures loadModelsData validates that each model
// entry has a provider and returns an error when provider is missing.
func TestLoadModelsData_MissingProvider(t *testing.T) {
	yaml := []byte(`set: "test"
models:
  L1:
    model: "m1"
`)

	_, err := loadModelsData([][]byte{yaml})
	if err == nil {
		t.Fatalf("expected error due to missing provider, got nil")
	}
}
