package log

import (
	"context"
	"io"
)

type Logger interface {
	Prompt(string, ...any)
	//
	Print(string, ...any)
	Error(string, ...any)
	Info(string, ...any)
	Debug(string, ...any)

	SetLogLevel(Level)
	SetLogOutput(io.Writer)

	IsQuiet() bool
	IsNormal() bool
	IsVerbose() bool
	IsTrace() bool
}

type LogManager interface {
	GetLogger(ctx context.Context) Logger
}

// default
var manager LogManager = newLogManager()

// set custom manager
func SetLogManager(m LogManager) {
	manager = m
}

type defaultLogManager struct {
	logger Logger
}

func newLogManager() *defaultLogManager {
	return &defaultLogManager{
		logger: newDefaultLogger(),
	}
}

func (r *defaultLogManager) GetLogger(ctx context.Context) Logger {
	return r.logger
}

func GetLogger(ctx context.Context) Logger {
	return manager.GetLogger(ctx)
}
