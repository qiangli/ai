package log

import (
	"context"
	"fmt"
	"sync"
)

type ContextKey string

const (
	UserIDKey    ContextKey = "userID"
	SessionIDKey ContextKey = "sessionID"
)

type Logger interface {
	Error(string, ...any)
	Info(string, ...any)
	Debug(string, ...any)
	Trace(string, ...any)
}

type LogManager interface {
	GetLogger(ctx context.Context) Logger
}

var manager LogManager

func Init() {
	manager = newLogManager()
}

// set custom manager
func SetLogManager(m LogManager) {
	manager = m
}

type defaultLogManager struct {
	loggers map[string]Logger
	mu      sync.Mutex
}

func newLogManager() *defaultLogManager {
	return &defaultLogManager{
		loggers: make(map[string]Logger),
	}
}

func (r *defaultLogManager) GetLogger(ctx context.Context) Logger {
	userID := ctx.Value(UserIDKey).(string)
	sessionID := ctx.Value(SessionIDKey).(string)
	uniqueKey := fmt.Sprintf("%s/%s", userID, sessionID)

	r.mu.Lock()
	defer r.mu.Unlock()

	if logger, exists := r.loggers[uniqueKey]; exists {
		return logger
	}

	logger := &defaultLogger{}
	r.loggers[uniqueKey] = logger
	return logger
}

func GetLogger(ctx context.Context) Logger {
	return manager.GetLogger(ctx)
}

type defaultLogger struct {
}

func (r *defaultLogger) Error(string, ...any) {

}

func (r *defaultLogger) Info(string, ...any) {

}

func (r *defaultLogger) Debug(string, ...any) {

}

func (r *defaultLogger) Trace(string, ...any) {

}
