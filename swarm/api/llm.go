package api

import (
	"context"
)

type LLMAdapter interface {
	Call(context.Context, *Request) (*Response, error)
}

// type LLMAdapter func(context.Context, *api.Request) (*api.Response, error)

type AdapterRegistry interface {
	Get(key string) (LLMAdapter, error)
}

// type Request = api.Request

// type Response = api.Response
