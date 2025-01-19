package agent

import (
	"net"
	"strings"
)

func isLoopback(hostport string) bool {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}

	ip := net.ParseIP(host)

	if ip != nil && ip.IsLoopback() {
		return true
	}

	if host == "localhost" {
		return true
	}

	return false
}

// clipText truncates the input text to no more than the specified maximum length.
func clipText(text string, maxLen int) string {
	if len(text) > maxLen {
		return strings.TrimSpace(text[:maxLen]) + "\n[more...]"
	}
	return text
}
