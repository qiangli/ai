package log

import (
	"context"
	"io"
	// "fmt"
	// "sync"
)

type ContextKey string

const (
	UserIDKey    ContextKey = "userID"
	SessionIDKey ContextKey = "sessionID"
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

// func InitDefault() {
// 	manager = newLogManager()
// }

// set custom manager
func SetLogManager(m LogManager) {
	manager = m
}

type defaultLogManager struct {
	// loggers map[string]Logger
	// mu      sync.Mutex
	logger Logger
}

func newLogManager() *defaultLogManager {
	return &defaultLogManager{
		// loggers: make(map[string]Logger),
		logger: newDefaultLogger(),
	}
}

func (r *defaultLogManager) GetLogger(ctx context.Context) Logger {
	// userID := ctx.Value(UserIDKey).(string)
	// sessionID := ctx.Value(SessionIDKey).(string)
	// uniqueKey := fmt.Sprintf("%s/%s", userID, sessionID)

	// r.mu.Lock()
	// defer r.mu.Unlock()

	// if logger, exists := r.loggers[uniqueKey]; exists {
	// 	return logger
	// }

	// r.loggers[uniqueKey] = logger
	return r.logger
}

func GetLogger(ctx context.Context) Logger {
	return manager.GetLogger(ctx)
}
