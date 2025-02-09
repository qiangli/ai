package swarm

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/google/uuid"
)

// User interface
// Input
// Output

type Input struct {
	// user requested agent and subcommand
	Agent      string `json:"agent"`
	Subcommand string `json:"subcommand"`

	// input collected from command line
	Message string `json:"message"`

	// input collected from stdin, or editor
	Content string `json:"content"`

	// File paths whose content to be included in the input
	Files []string `json:"files"`

	Extra map[string]any `json:"extra"`
}

// IsEmpty returns true if the message, content, and files are all empty.
func (r *Input) IsEmpty() bool {
	return r.Message == "" && r.Content == "" && len(r.Files) == 0
}

// Input returns a single string concatenating all user inputs.
func (r *Input) Input() string {
	fc, _ := r.FileContent()
	return r.Query() + "\n" + fc
}

// Query returns a single string combining both the message and content.
func (r *Input) Query() string {
	switch {
	case r.Message == "" && r.Content == "":
		return ""
	case r.Message == "":
		return r.Content
	case r.Content == "":
		return r.Message
	default:
		return fmt.Sprintf("###\n%s\n###\n%s", r.Message, r.Content)
	}
}

// FileContent returns the content of the files in the input.
func (r *Input) FileContent() (string, error) {
	var b strings.Builder
	if len(r.Files) > 0 {
		for _, f := range r.Files {
			b.WriteString("\n### " + f + " ###\n")
			c, err := os.ReadFile(f)
			if err != nil {
				return "", err

			}
			b.WriteString(string(c))
		}
	}
	return b.String(), nil
}

// Intent returns a clipped version of the query.
// This is intended for "smart" agents to make decisions based on user inputs.
func (r *Input) Intent() string {
	return r.clipText(r.Message, 500)
}

// clipText truncates the input text to no more than the specified maximum length.
func (r *Input) clipText(text string, maxLen int) string {
	if len(text) > maxLen {
		return strings.TrimSpace(text[:maxLen]) + "\n[more...]"
	}
	return text
}

func (r *Input) String() string {
	return r.Query()
}

type Output struct {
	// The last agent that processed the output content.
	Agent string `json:"agent"`

	Content string `json:"content"`
}

func (r *Output) IsEmpty() bool {
	return r.Content == ""
}

func (r *Output) String() string {
	return r.Content
}

// Render returns formatted markdown.
func (r *Output) Render() string {
	return r.Content
}

// LLM
// Request
// Response

// Sandbox/Runtime
// Action
// Result

type Action struct {
	Agent string `json:"agent"`
}

type Sandbox interface {
	// Run executes the action in the sandbox and returns the result.
	Run(action Action) Result
}

// built-in agents
// ShellAgent
// DockerAgent
// WebAgent
// FileAgent
// SQLAgent

// https://github.com/All-Hands-AI/OpenHands/tree/main/openhands

type EventSource int

const (
	EventUser EventSource = iota
	EventAgent
	EventRuntime
)

type Event struct {
	ID        uuid.UUID
	Source    EventSource
	Message   string
	Data      any
	Timestamp int64
}

func (r Event) String() string {
	return fmt.Sprintf("%v", r.Source)
}

type EventHub struct {
	queue *Pubsub[Event]
}

func NewEventHub() *EventHub {
	return &EventHub{
		queue: NewPubsub[Event](),
	}
}

func (eh *EventHub) Publish(event Event) {
	eh.queue.Publish(event, event)
}

func (eh *EventHub) Close() {
	eh.queue.Close()
}

// while True:
//
//	prompt = agent.generate_prompt(state)
//	response = llm.completion(prompt)
//	action = agent.parse_response(response)
//	observation = runtime.run(action)
//	state = state.update(action, observation)
func (eh *EventHub) Dispatch() {
	topics := eh.queue.Topics()
	var cases []reflect.SelectCase

	for _, topic := range topics {
		ch := eh.queue.Subscribe(topic)
		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ch),
		})
	}

	for {

		if len(cases) > 0 {
			chosen, value, ok := reflect.Select(cases)
			if ok {
				fmt.Println(value.Interface())
			} else {
				fmt.Printf("Received a closed channel for topic index %d\n", chosen)
			}
		}
	}
}
