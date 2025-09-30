package calendar

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type StubCalendar struct {
	data map[uuid.UUID]Event
}

func NewStubCalendar() *StubCalendar {
	data := map[uuid.UUID]Event{}
	return &StubCalendar{data}
}

func (c *StubCalendar) AddEvent(ctx context.Context, event Event) (*Event, error) {
	uid := uuid.New()
	event.UID = uuid.NullUUID{UUID: uid, Valid: true}
	c.data[uid] = event
	return &event, nil
}

func (c *StubCalendar) GetEvents(ctx context.Context, from time.Time, to time.Time) ([]Event, error) {
	var events []Event
	for _, event := range c.data {
		if event.StartTime.Before(to) && event.EndTime.After(from) {
			events = append(events, event)
		}
	}
	return events, nil
}

func (c *StubCalendar) ModifyEvent(ctx context.Context, event Event) (*Event, error) {
	if !event.UID.Valid {
		return nil, errors.New("event.UID is required")
	}
	foundEvent, ok := c.data[event.UID.UUID]
	if !ok {
		return nil, errors.New("event with given UID not found")
	}
	foundEvent.Summary = event.Summary
	foundEvent.Metadata = event.Metadata
	foundEvent.StartTime = event.StartTime
	foundEvent.EndTime = event.EndTime
	return &foundEvent, nil
}

func (c *StubCalendar) Cleanup() {
	c.data = map[uuid.UUID]Event{}
}

func (c *StubCalendar) GetLastEvents(ctx context.Context, limit int) ([]Event, error) {
	oldestEventDate := time.Now()
	oldestEventIndex := -1
	events := make([]Event, 0, limit)
	for _, event := range c.data {
		if (len(events)) >= limit {
			break
		}
		if (len(events)) < limit {
			events = append(events, event)
			if event.StartTime.Before(oldestEventDate) {
				oldestEventDate = event.StartTime
				oldestEventIndex = len(events) - 1
			}
		} else if event.StartTime.After(oldestEventDate) {
			events[oldestEventIndex] = event
			oldestEventDate = findOldestEvent(events).StartTime
		}
	}
	return events, nil
}

func findOldestEvent(events []Event) *Event {
	oldestEventDate := time.Now()
	oldestEventIndex := -1
	for i, event := range events {
		if event.StartTime.Before(oldestEventDate) {
			oldestEventDate = event.StartTime
			oldestEventIndex = i
		}
	}
	return &events[oldestEventIndex]
}
