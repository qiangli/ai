package swarm

import (
	"fmt"
	"reflect"

	"github.com/google/uuid"
)

// User interface
// Input
// Output
// type Input = UserInput
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
	// Run(action Action) Result
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
