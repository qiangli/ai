package agent

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/kaptinlin/jsonrepair"
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

// baseCommand returns the last part of the string separated by /.
func baseCommand(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "/")
	sa := strings.Split(s, "/")
	return sa[len(sa)-1]
}

// tryUnmarshal tries to unmarshal the data into the v.
// If it fails, it will try to repair the data and unmarshal it again.
func tryUnmarshal(data string, v any) error {
	err := json.Unmarshal([]byte(data), v)
	if err == nil {
		return nil
	}

	repaired, err := jsonrepair.JSONRepair(data)
	if err != nil {
		return fmt.Errorf("failed to repair JSON: %v", err)
	}
	return json.Unmarshal([]byte(repaired), v)
}
