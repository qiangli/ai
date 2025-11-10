package llm

import (
	"context"

	"github.com/qiangli/ai/swarm/api"
)

type LLMAdapter func(context.Context, *api.Request) (*api.Response, error)

type AdapterRegistry interface {
	Get(key string) (LLMAdapter, error)
}

type Request = api.Request

type Response = api.Response
