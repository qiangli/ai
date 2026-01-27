package log

import (
	"context"
)

type Logger interface {
	Promptf(string, ...any)
	//
	Printf(string, ...any)
	Errorf(string, ...any)
	Infof(string, ...any)
	Debugf(string, ...any)

	SetLogLevel(Level)
	// tee/file specific
	SetTeeFile(string) error
	SetTeeLogLevel(Level)

	IsQuiet() bool
	IsInformative() bool
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
