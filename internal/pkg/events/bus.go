package events

import (
	"sync"

	"github.com/rs/zerolog/log"
)

// Handler is a callback invoked when an event of a subscribed type fires.
type Handler func(event Event)

// subscription represents a single event subscription with an ID for unsubscribing.
type subscription struct {
	id      int
	handler Handler
}

// Bus is a thread-safe, in-process event bus supporting subscribe, unsubscribe,
// and publish operations. Handlers are invoked asynchronously.
type Bus struct {
	mu            sync.RWMutex
	subscriptions map[string][]subscription
	nextID        int
}

// NewBus creates a new event bus.
func NewBus() *Bus {
	return &Bus{
		subscriptions: make(map[string][]subscription),
	}
}

// Subscribe registers a handler for the given event type and returns a
// subscription ID that can be used to unsubscribe later.
func (b *Bus) Subscribe(eventType string, handler Handler) int {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	sub := subscription{
		id:      b.nextID,
		handler: handler,
	}
	b.subscriptions[eventType] = append(b.subscriptions[eventType], sub)
	return b.nextID
}

// Unsubscribe removes a subscription by its ID. Returns true if a subscription
// was found and removed, false otherwise.
func (b *Bus) Unsubscribe(eventType string, subID int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs, exists := b.subscriptions[eventType]
	if !exists {
		return false
	}

	for i, sub := range subs {
		if sub.id == subID {
			b.subscriptions[eventType] = append(subs[:i], subs[i+1:]...)
			return true
		}
	}
	return false
}

// Publish fires an event, invoking all registered handlers for the event type
// and any wildcard ("*") subscribers. Each handler runs in its own goroutine.
func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	subs := make([]subscription, 0, len(b.subscriptions[event.Type])+len(b.subscriptions["*"]))
	subs = append(subs, b.subscriptions[event.Type]...)
	subs = append(subs, b.subscriptions["*"]...)
	b.mu.RUnlock()

	for _, sub := range subs {
		go func(h Handler) {
			defer func() {
				if r := recover(); r != nil {
					log.Error().
						Interface("panic", r).
						Str("event_type", event.Type).
						Msg("event handler panicked")
				}
			}()
			h(event)
		}(sub.handler)
	}
}

// PublishSync fires an event and invokes all registered handlers synchronously
// in the caller's goroutine. Useful for testing or when ordering matters.
func (b *Bus) PublishSync(event Event) {
	b.mu.RLock()
	subs := make([]subscription, len(b.subscriptions[event.Type]))
	copy(subs, b.subscriptions[event.Type])
	b.mu.RUnlock()

	for _, sub := range subs {
		sub.handler(event)
	}
}

// SubscribeAll registers a handler that is invoked for every event type.
// Uses "*" as a wildcard event type.
func (b *Bus) SubscribeAll(handler Handler) int {
	return b.Subscribe("*", handler)
}

// SubscriberCount returns the number of subscribers for a given event type.
func (b *Bus) SubscriberCount(eventType string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscriptions[eventType])
}
