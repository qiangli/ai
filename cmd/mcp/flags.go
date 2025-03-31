package mcp

import (
	"fmt"
)

// Transport
type transportValue string

func newTransportValue(val string, p *string) *transportValue {
	*p = val
	return (*transportValue)(p)
}

func (s *transportValue) Set(val string) error {
	for _, v := range []string{"stdio", "sse"} {
		if val == v {
			*s = transportValue(val)
			return nil
		}
	}
	return fmt.Errorf("invalid transport: %v. supported: stdio, sse", val)
}
func (s *transportValue) Type() string {
	return "string"
}
func (s *transportValue) String() string { return string(*s) }
