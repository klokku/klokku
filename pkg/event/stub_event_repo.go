package event

import (
	"context"
)

type StubEventRepository struct {
	Events []Event
}

func (s *StubEventRepository) StoreEvent(ctx context.Context, userId int, event Event) (Event, error) {
	s.Events = append(s.Events, event)
	return event, nil
}

func (s *StubEventRepository) DeleteCurrentEvent(ctx context.Context, userId int) error {
	s.Events = s.Events[:len(s.Events)-1]
	return nil
}

func (s *StubEventRepository) FindCurrentEvent(ctx context.Context, userId int) (*Event, error) {
	if len(s.Events) == 0 {
		return nil, nil
	}
	currentEvent := &s.Events[len(s.Events)-1]
	currentEvent.StartTime = currentEvent.StartTime.UTC()

	return currentEvent, nil
}

func (s *StubEventRepository) Cleanup() {
	s.Events = []Event{}
}
