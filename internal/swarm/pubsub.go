// https://eli.thegreenplace.net/2020/pubsub-using-channels-in-go/
// https://gist.github.com/VMois/b50f71114b4086c724b1aec4b7b916a3
package swarm

import (
	"sync"
)

type Topic interface {
	String() string
}

type Pubsub[T any] struct {
	subs   map[Topic][]chan T
	closed bool

	mu sync.RWMutex
}

func NewPubsub[T any]() *Pubsub[T] {
	ps := &Pubsub[T]{}
	ps.subs = make(map[Topic][]chan T)
	return ps
}

func (ps *Pubsub[T]) Subscribe(topic Topic) <-chan T {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ch := make(chan T, 1)

	ps.subs[topic] = append(ps.subs[topic], ch)
	return ch
}

func (ps *Pubsub[T]) Unsubscribe(topic Topic, c <-chan T) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	indexToRemove := -1
	for index, ch := range ps.subs[topic] {
		if c == ch {
			indexToRemove = index
			close(ch)
			break
		}
	}
	if indexToRemove >= 0 {
		ps.subs[topic] = append(ps.subs[topic][:indexToRemove], ps.subs[topic][indexToRemove+1:]...)
	}
}

func (ps *Pubsub[T]) Topics() []Topic {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	topics := make([]Topic, 0, len(ps.subs))
	for topic := range ps.subs {
		topics = append(topics, topic)
	}
	return topics
}

func (ps *Pubsub[T]) Publish(topic Topic, message T) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if ps.closed {
		return
	}

	for _, ch := range ps.subs[topic] {
		select {
		case ch <- message:
		default:
		}
	}
}

func (ps *Pubsub[T]) Close() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if !ps.closed {
		ps.closed = true
		for _, subs := range ps.subs {
			for _, ch := range subs {
				close(ch)
			}
		}
	}
}
