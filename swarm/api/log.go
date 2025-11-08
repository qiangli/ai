package api

import (
	"strings"
)

type LogLevel int

func (r LogLevel) String() string {
	return LogLevelToString(r)
}

const (
	Quiet LogLevel = iota + 1
	Informative
	Verbose
	Tracing
)

func LogLevelToString(level LogLevel) string {
	switch level {
	case Quiet:
		return "Quiet"
	case Informative:
		return "Informative"
	case Verbose:
		return "Verbose"
	case Tracing:
		return "Tracing"
	default:
		return ""
	}
}

func ToLogLevel(level string) LogLevel {

	switch strings.ToLower(level) {
	case "quiet":
		return Quiet
	case "info", "informative":
		return Informative
	case "debug", "verbose":
		return Verbose
	case "trace", "tracing":
		return Tracing
	default:
		return 0
	}
}
