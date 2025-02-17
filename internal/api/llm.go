package api

type Result struct {
	// The result value as a string
	Value string

	// The agent name to transfer to for StateTransfer
	NextAgent string

	State State
}

type State int

const (
	StateUnknown State = iota

	StateExit
	StateTransfer
	StateInputWait
)
