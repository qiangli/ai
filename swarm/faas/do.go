package faas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/qiangli/ai/swarm/api"
)

type DoClient struct {
	// digital ocean fn adapter
	baseUrl string
	token   string
}

type DoResult struct {
	StatusCode int `json:"statusCode"`
	Body       any `json:"body"`
}

func NewDoClient(baseUrl, token string) *DoClient {
	return &DoClient{
		baseUrl: baseUrl,
		token:   token,
	}
}

func (r *DoClient) Call(ctx context.Context, vars *api.Vars, tf *api.ToolFunc, args map[string]any) (*api.Result, error) {
	if tf.Body == nil || (tf.Body.Url == "" && tf.Body.Code == "") {
		return nil, fmt.Errorf("no function body: %s", tf.ID())
	}
	requestBody := map[string]any{
		"url":  tf.Body.Url,
		"code": tf.Body.Code,
		"args": args,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+r.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result DoResult

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.StatusCode >= 400 {
		return nil, fmt.Errorf("function call failed: %v %s", result.StatusCode, result.Body)
	}
	return &api.Result{
		Value: fmt.Sprintf("%v", result.Body),
	}, nil
}
