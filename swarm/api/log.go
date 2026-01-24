package api

import (
	"fmt"
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

func FormRequestLine(req *Request, adapter string, maxTurns, tries int) string {
	var emoji string
	switch req.Model.Provider {
	case "anthropic":
		emoji = "Ⓐ"
	case "gemini":
		emoji = "Ⓖ"
	case "openai":
		emoji = "Ⓞ"
	case "xai":
		emoji = "Ⓧ"
	default:
		emoji = "???"
	}
	return fmt.Sprintf("%s @%s/%s %s [%v/%v] %s %s/%s\n", req.Agent.Display, req.Agent.Pack, req.Agent.Name, emoji, tries, maxTurns, adapter, req.Model.Provider, req.Model.Model)
}

const maxInfoTextLen = 12

func FormatArgMap(args map[string]any) string {
	var sb strings.Builder
	for k, v := range args {
		switch vt := v.(type) {
		case string:
			if len(vt) > maxInfoTextLen {
				sb.WriteString(fmt.Sprintf("%s:%q[%v], ", k, Abbreviate(vt, maxInfoTextLen), len(vt)))
			} else {
				sb.WriteString(fmt.Sprintf("%s:%q, ", k, vt))
			}
		case bool, int8, int, int32, int64, float32, float64:
			sb.WriteString(fmt.Sprintf("%s:%v, ", k, vt))
		case *Result:
			sb.WriteString(fmt.Sprintf("%s:%q(%T), ", k, Abbreviate(vt.Value, maxInfoTextLen), v))
		case *Agent:
			sb.WriteString(fmt.Sprintf("%s:%s/%s(%T), ", k, vt.Pack, vt.Name, v))
		default:
			sb.WriteString(fmt.Sprintf("%s:(%T), ", k, v))
		}
	}
	s := sb.String()
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ",")
	return fmt.Sprintf("map[%s]", s)
}
