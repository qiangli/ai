package api

import (
	"strings"
	"time"
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

func ToLogLevel(level any) LogLevel {

	switch strings.ToLower(ToString(level)) {
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

type CallLogEntry struct {
	// tool call
	Kit       string         `json:"kit"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`

	// active agent
	Agent string `json:"agent"`

	// failure
	Error error `json:"error"`

	// success
	Result *Result `json:"result"`

	Started time.Time `json:"started"`
	Ended   time.Time `json:"ended"`
}

type CallLogger interface {
	Base() string
	Save(*CallLogEntry)
}
