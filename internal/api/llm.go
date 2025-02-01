package api

import (
	"context"
)

type LLM interface {
	Call(ctx context.Context, req *Request) (*Response, error)
}
