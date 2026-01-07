package event_bus

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// EventType is an identifier for events.
type EventType string

// Event is the generic envelope used by the bus. Data is kept as any to allow
// different payload types on the same bus.
type Event struct {
	ctx       context.Context
	Type      EventType
	Timestamp time.Time
	Data      any
}

// NewEvent creates a new Event with the given context, type, and data.
// The timestamp is set to the current time automatically.
func NewEvent(ctx context.Context, eventType EventType, data any) Event {
	return Event{
		ctx:       ctx,
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}
}

// Context returns the context associated with this event.
// Handlers should use this context for any operations that need cancellation,
// deadlines, or access to context values (like user ID, trace ID, etc.)
func (e Event) Context() context.Context {
	if e.ctx == nil {
		return context.Background()
	}
	return e.ctx
}

// EventT is a typed envelope used by typed handlers.
type EventT[T any] struct {
	ctx       context.Context
	Type      EventType
	Timestamp time.Time
	Data      T
}

// Context returns the context associated with this typed event.
func (e EventT[T]) Context() context.Context {
	if e.ctx == nil {
		return context.Background()
	}
	return e.ctx
}

// handler is the internal shape for subscribers: a function that accepts the generic Event.
type handler func(Event) error

// EventBus is a concurrency-safe synchronous event dispatcher.
// All handlers are executed sequentially and synchronously during Publish.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[EventType]map[uint64]handler
	nextID      uint64
}

// NewEventBus creates an empty EventBus.
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[EventType]map[uint64]handler),
	}
}

// Subscribe registers a generic handler for the given eventType. It returns an
// unsubscribe function that removes the handler when called.
// Handlers should return an error if they fail to process the event.
// Handlers receive the event with its context and should respect cancellation.
func (eb *EventBus) Subscribe(eventType EventType, h func(Event) error) (unsubscribe func()) {
	eb.mu.Lock()
	eb.nextID++
	id := eb.nextID

	if eb.subscribers[eventType] == nil {
		eb.subscribers[eventType] = make(map[uint64]handler)
	}
	eb.subscribers[eventType][id] = handler(h)
	eb.mu.Unlock()

	return func() {
		eb.mu.Lock()
		defer eb.mu.Unlock()

		if handlers := eb.subscribers[eventType]; handlers != nil {
			delete(handlers, id)
			// Clean up empty map to free memory
			if len(handlers) == 0 {
				delete(eb.subscribers, eventType)
			}
		}
	}
}

// SubscribeTyped registers a handler that expects a specific payload type T.
// It is implemented as a generic free function (not a method) because Go does
// not allow adding type parameters to a method on a non-generic receiver type.
// The wrapper performs a type assertion and only invokes the typed handler when
// the assertion succeeds.
//
// Example:
//
//	unsub := event_bus.SubscribeTyped[budget_plan.BudgetItem](bus, "budget.created",
//	    func(e event_bus.EventT[budget_plan.BudgetItem]) error {
//	        // Access context for user ID, tracing, etc.
//	        userId, _ := user.CurrentId(e.Context())
//	        log.Infof("User %d created budget: %s", userId, e.Data.Name)
//	        return nil
//	    })
func SubscribeTyped[T any](eb *EventBus, eventType EventType, h func(EventT[T]) error) (unsubscribe func()) {
	// wrapper converts generic Event to EventT[T] after type assertion
	wrapper := func(e Event) error {
		// Handle nil data
		if e.Data == nil {
			log.Debugf("EventBus: nil data for event type %s, skipping typed handler", eventType)
			return nil
		}

		payload, ok := e.Data.(T)
		if !ok {
			// Log type mismatch for debugging
			log.Debugf("EventBus: type mismatch for event %s: expected %T, got %T",
				eventType, *new(T), e.Data)
			return nil
		}

		typed := EventT[T]{
			ctx:       e.ctx,
			Type:      e.Type,
			Timestamp: e.Timestamp,
			Data:      payload,
		}
		return h(typed)
	}
	return eb.Subscribe(eventType, wrapper)
}

// Publish sends the event to all handlers registered for event.Type synchronously.
// All handlers are executed in the order they were registered.
// If any handler returns an error, execution continues but all errors are collected
// and returned as a single error. Panics in handlers are recovered and treated as errors.
//
// If the event's context is cancelled before or during handler execution, remaining
// handlers are skipped and a context error is returned.
func (eb *EventBus) Publish(e Event) error {
	// Check if context is already cancelled
	if err := e.Context().Err(); err != nil {
		return fmt.Errorf("event %s: context cancelled before publish: %w", e.Type, err)
	}

	eb.mu.RLock()
	// Copy handler map to avoid holding lock during invocation
	handlers := make([]struct {
		id uint64
		h  handler
	}, 0, len(eb.subscribers[e.Type]))
	for id, h := range eb.subscribers[e.Type] {
		handlers = append(handlers, struct {
			id uint64
			h  handler
		}{id, h})
	}
	eb.mu.RUnlock()

	var errors []error
	for _, handler := range handlers {
		// Check context cancellation before each handler
		if err := e.Context().Err(); err != nil {
			errors = append(errors, fmt.Errorf("context cancelled during event processing: %w", err))
			break
		}

		// Recover from panics and treat them as errors
		err := func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("handler panic (ID %d) for event %s: %v",
						handler.id, e.Type, r)
					log.Error(err)
				}
			}()
			return handler.h(e)
		}()

		if err != nil {
			log.Errorf("EventBus: handler error (ID %d) for event %s: %v",
				handler.id, e.Type, err)
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("event %s: %d handler(s) failed: %v", e.Type, len(errors), errors)
	}

	return nil
}
