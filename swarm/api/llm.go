package api

import (
	"context"
)

const MaxTurnsLimit = 100
const MaxTimeLimit = 900 // 15 min

const DefaultMaxTurns = 50
const DefaultMaxTime = 600 // 10 min

const DefaultMaxSpan = 1440 // 24 hours
const DefaultMaxHistory = 5

type LLMAdapter interface {
	Call(context.Context, *Request) (*Response, error)
}

type AdapterRegistry interface {
	Get(key string) (LLMAdapter, error)
}
