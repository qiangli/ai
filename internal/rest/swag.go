package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

type SwagKit struct {
}

func (r *SwagKit) LoadV2(ctx context.Context, file string) (*openapi3.T, error) {
	input, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var doc2 openapi2.T
	if err = json.Unmarshal(input, &doc2); err != nil {
		return nil, err
	}
	doc3, err := openapi2conv.ToV3(&doc2)
	if err != nil {
		return nil, err
	}
	if err := doc3.Validate(ctx); err != nil {
		return nil, err
	}
	return doc3, nil
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

func (r *SwagKit) Run(ctx context.Context, doc *openapi3.T, endpoint string) error {
	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	route, pathParams, err := router.FindRoute(httpReq)
	if err != nil {
		return err
	}

	requestValidationInput := &openapi3filter.RequestValidationInput{
		Request:    httpReq,
		PathParams: pathParams,
		Route:      route,
	}
	if err := openapi3filter.ValidateRequest(ctx, requestValidationInput); err != nil {
		return err
	}

	// Handle that request
	// --> YOUR CODE GOES HERE <--
	responseHeaders := http.Header{"Content-Type": []string{"application/json"}}
	responseCode := 200
	// responseBody := []byte(`{}`)

	// Validate response
	responseValidationInput := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: requestValidationInput,
		Status:                 responseCode,
		Header:                 responseHeaders,
	}
	// responseValidationInput.SetBodyBytes(responseBody)
	if err := openapi3filter.ValidateResponse(ctx, responseValidationInput); err != nil {
		return err
	}

	return nil
}
