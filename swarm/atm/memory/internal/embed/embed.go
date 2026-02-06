package embed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"
)

const dim = 768

type Provider interface {
	Embed(texts []string) ([][]float64, error)
	Name() string
	ModelName() string
}

type LocalRandomEmbedder struct{}

func (l *LocalRandomEmbedder) Embed(texts []string) ([][]float64, error) {
	rand.Seed(time.Now().UnixNano())
	res := make([][]float64, len(texts))
	for i := range texts {
		res[i] = make([]float64, dim)
		for j := range res[i] {
			res[i][j] = rand.Float64()*2 - 1
		}
	}
	return res, nil
}

func (l *LocalRandomEmbedder) Name() string      { return "local" }
func (l *LocalRandomEmbedder) ModelName() string { return "random" }

type OpenAIEmbedder struct {
	model  string
	apiKey string
}

func NewOpenAIEmbedder(model, apiKey string) *OpenAIEmbedder {
	return &OpenAIEmbedder{model, apiKey}
}

type OpenAIReq struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type OpenAIEmbedding struct {
	Embedding []float64 `json:"embedding"`
}

type OpenAIResp struct {
	Data []OpenAIEmbedding `json:"data"`
}

func (o *OpenAIEmbedder) Embed(texts []string) ([][]float64, error) {
	for attempt := 0; attempt < 3; attempt++ {
		reqBody, _ := json.Marshal(OpenAIReq{Model: o.model, Input: texts})
		httpReq, _ := http.NewRequest("POST", "https://api.openai.com/v1/embeddings", bytes.NewBuffer(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(httpReq)
		if err != nil {
			time.Sleep(time.Duration(1<<attempt) * time.Second)
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			time.Sleep(time.Duration(1<<attempt) * time.Second)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		var ores OpenAIResp
		if err := json.Unmarshal(body, &ores); err != nil {
			time.Sleep(time.Duration(1<<attempt) * time.Second)
			continue
		}
		res := make([][]float64, len(texts))
		for i := range ores.Data {
			res[i] = ores.Data[i].Embedding
		}
		return res, nil
	}
	return nil, fmt.Errorf("OpenAI embed failed after retries")
}

func (o *OpenAIEmbedder) Name() string      { return "openai" }
func (o *OpenAIEmbedder) ModelName() string { return o.model }

type GeminiEmbedder struct{}

func (g *GeminiEmbedder) Embed([]string) ([][]float64, error) {
	return nil, fmt.Errorf("not implemented")
}

func (g *GeminiEmbedder) Name() string      { return "gemini" }
func (g *GeminiEmbedder) ModelName() string { return "" }

func NewProvider(name, model string) Provider {
	switch name {
	case "local":
		return &LocalRandomEmbedder{}
	case "openai":
		key := os.Getenv("OPENAI_API_KEY")
		return NewOpenAIEmbedder(model, key)
	case "gemini":
		return &GeminiEmbedder{}
	default:
		return &LocalRandomEmbedder{}
	}
}
