package webhook

import (
	"context"
	"errors"

	"github.com/klokku/klokku/pkg/budget_plan"
	"github.com/klokku/klokku/pkg/current_event"
	"github.com/klokku/klokku/pkg/user"
)

// EventStarterStub implements EventStarter for testing
type EventStarterStub struct {
	startedEvents []current_event.CurrentEvent
}

func NewEventStarterStub() *EventStarterStub {
	return &EventStarterStub{
		startedEvents: make([]current_event.CurrentEvent, 0),
	}
}

func (s *EventStarterStub) StartNewEvent(ctx context.Context, event current_event.CurrentEvent) (current_event.CurrentEvent, error) {
	s.startedEvents = append(s.startedEvents, event)
	event.Id = len(s.startedEvents)
	return event, nil
}

func (s *EventStarterStub) GetStartedEvents() []current_event.CurrentEvent {
	return s.startedEvents
}

func (s *EventStarterStub) Reset() {
	s.startedEvents = make([]current_event.CurrentEvent, 0)
}

// BudgetProviderStub implements BudgetItemProvider for testing
type BudgetProviderStub struct {
	items map[int]budget_plan.BudgetItem
}

func NewBudgetProviderStub() *BudgetProviderStub {
	return &BudgetProviderStub{
		items: make(map[int]budget_plan.BudgetItem),
	}
}

func (s *BudgetProviderStub) GetItem(ctx context.Context, id int) (budget_plan.BudgetItem, error) {
	item, exists := s.items[id]
	if !exists {
		return budget_plan.BudgetItem{}, errors.New("item not found")
	}
	return item, nil
}

func (s *BudgetProviderStub) SetItem(id int, item budget_plan.BudgetItem) {
	s.items[id] = item
}

func (s *BudgetProviderStub) Reset() {
	s.items = make(map[int]budget_plan.BudgetItem)
}

// UserProviderStub implements UserProvider for testing
type UserProviderStub struct {
	users map[int]user.User
}

func NewUserProviderStub() *UserProviderStub {
	return &UserProviderStub{
		users: make(map[int]user.User),
	}
}

func (s *UserProviderStub) GetUser(ctx context.Context, id int) (user.User, error) {
	u, exists := s.users[id]
	if !exists {
		return user.User{}, user.ErrNoUser
	}
	return u, nil
}

func (s *UserProviderStub) SetUser(id int, u user.User) {
	s.users[id] = u
}

func (s *UserProviderStub) Reset() {
	s.users = make(map[int]user.User)
}
