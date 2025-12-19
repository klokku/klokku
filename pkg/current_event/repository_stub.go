package current_event

import (
	"context"
)

type stubEventRepository struct {
	events map[int]CurrentEvent // userId -> event
	nextId int
}

func newStubEventRepository() *stubEventRepository {
	return &stubEventRepository{
		events: map[int]CurrentEvent{},
		nextId: 1,
	}
}

func (s *stubEventRepository) ReplaceCurrentEvent(ctx context.Context, userId int, event CurrentEvent) (CurrentEvent, error) {
	event.Id = s.nextId
	s.events[userId] = event
	s.nextId++

	return event, nil
}

func (s *stubEventRepository) DeleteCurrentEvent(ctx context.Context, userId int) error {
	delete(s.events, userId)
	return nil
}

func (s *stubEventRepository) FindCurrentEvent(ctx context.Context, userId int) (CurrentEvent, error) {
	if len(s.events) == 0 {
		return CurrentEvent{}, nil
	}
	currentEvent, ok := s.events[userId]
	if !ok {
		return CurrentEvent{}, nil
	}

	return currentEvent, nil
}

func (s *stubEventRepository) reset() {
	s.events = map[int]CurrentEvent{}
}
