package calendar

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type RepositoryStub struct {
	mu             sync.RWMutex
	items          map[string]Event // uid -> item
	userIds        map[string]int   // uid -> userId
	nextId         int
	inTransaction  bool
	transactionErr error
}

func NewRepositoryStub() *RepositoryStub {
	return &RepositoryStub{
		items:   make(map[string]Event),
		userIds: make(map[string]int),
		nextId:  1,
	}
}

func (r *RepositoryStub) WithTransaction(ctx context.Context, fn func(repo Repository) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Create a copy of the current state for rollback
	originalItems := make(map[string]Event, len(r.items))
	for k, v := range r.items {
		originalItems[k] = v
	}
	originalUserIds := make(map[string]int, len(r.userIds))
	for k, v := range r.userIds {
		originalUserIds[k] = v
	}
	originalNextId := r.nextId

	// Mark as in transaction
	r.inTransaction = true
	r.transactionErr = nil
	r.mu.Unlock()

	// Execute the function
	err := fn(r)

	r.mu.Lock()
	r.inTransaction = false

	// Rollback on error
	if err != nil || r.transactionErr != nil {
		r.items = originalItems
		r.userIds = originalUserIds
		r.nextId = originalNextId
		if err != nil {
			return err
		}
		return r.transactionErr
	}

	return nil
}

func (r *RepositoryStub) StoreEvent(ctx context.Context, userId int, event Event) (Event, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if event.UID == "" {
		event.UID = fmt.Sprintf("event-%d", r.nextId)
	}

	r.items[event.UID] = event
	r.userIds[event.UID] = userId
	r.nextId++

	return event, nil
}

func (r *RepositoryStub) GetEvents(ctx context.Context, userId int, from, to time.Time) ([]Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Event
	for uid, event := range r.items {
		if r.userIds[uid] == userId && !event.StartTime.After(to) && !event.EndTime.Before(from) {
			result = append(result, event)
		}
	}

	// Sort by start time (simple bubble sort for small slices)
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].StartTime.After(result[j].StartTime) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result, nil
}

func (r *RepositoryStub) GetLastEvents(ctx context.Context, userId int, limit int) ([]Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now()
	var result []Event
	for uid, event := range r.items {
		if r.userIds[uid] == userId && !event.EndTime.After(now) {
			result = append(result, event)
		}
	}

	// Sort by end time descending (simple bubble sort for small slices)
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].EndTime.Before(result[j].EndTime) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	// Apply limit
	if len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

func (r *RepositoryStub) UpdateEvent(ctx context.Context, userId int, event Event) (Event, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, exists := r.items[event.UID]
	if !exists || r.userIds[event.UID] != userId {
		return Event{}, fmt.Errorf("event not found")
	}

	r.items[event.UID] = event

	return event, nil
}

func (r *RepositoryStub) DeleteEvent(ctx context.Context, userId int, eventId string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, exists := r.items[eventId]
	if !exists || r.userIds[eventId] != userId {
		return fmt.Errorf("no event found with uid %s for user %d", eventId, userId)
	}

	delete(r.items, eventId)
	delete(r.userIds, eventId)

	return nil
}

// Helper method to set transaction error (for testing transaction rollback)
func (r *RepositoryStub) SetTransactionError(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.transactionErr = err
}

// Helper method to get all events (useful for test assertions)
func (r *RepositoryStub) GetAllEvents() []Event {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Event, 0, len(r.items))
	for _, event := range r.items {
		result = append(result, event)
	}
	return result
}

// Helper method to reset the stub (useful between tests)
func (r *RepositoryStub) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.items = make(map[string]Event)
	r.userIds = make(map[string]int)
	r.nextId = 1
	r.inTransaction = false
	r.transactionErr = nil
}
