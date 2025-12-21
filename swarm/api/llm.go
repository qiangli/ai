package api

import (
	"context"
)

type LLMAdapter interface {
	Call(context.Context, *Request) (*Response, error)
}

type AdapterRegistry interface {
	Get(key string) (LLMAdapter, error)
}
