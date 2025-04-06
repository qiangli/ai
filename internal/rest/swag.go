package rest

import (
	"context"
	"encoding/json"
	"os"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
)

type SwagKit struct {
}

func (r *SwagKit) LoadV2(ctx context.Context, file string) (*openapi2.T, error) {
	input, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var doc2 openapi2.T
	if err = json.Unmarshal(input, &doc2); err != nil {
		return nil, err
	}

	return &doc2, nil
}

func (r *SwagKit) LoadV3(ctx context.Context, file string) (*openapi3.T, error) {
	loader := &openapi3.Loader{Context: ctx, IsExternalRefsAllowed: true}
	doc, err := loader.LoadFromFile(file)
	if err != nil {
		return nil, err
	}
	if err := doc.Validate(ctx); err != nil {
		return nil, err
	}
	return doc, nil
}
