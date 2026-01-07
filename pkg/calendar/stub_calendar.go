package calendar

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/klokku/klokku/pkg/user"
)

type StubCalendar struct {
	data map[string]Event
}

func NewStubCalendar() *StubCalendar {
	data := map[string]Event{}
	return &StubCalendar{data}
}

func (c *StubCalendar) AddEvent(ctx context.Context, event Event) ([]Event, error) {
	uid := uuid.NewString()
	event.UID = uid

	currentUser, err := user.CurrentUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	events, err := splitEventIfNeeded(&event, currentUser.Settings.Timezone)
	if err != nil {
		return nil, err
	}
	for _, e := range events {
		c.data[e.UID] = e
	}
	return events, nil

}

func (c *StubCalendar) GetEvents(ctx context.Context, from time.Time, to time.Time) ([]Event, error) {
	var events []Event
	for _, event := range c.data {
		if event.StartTime.Before(to) && event.EndTime.After(from) {
			events = append(events, event)
		}
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].StartTime.Before(events[j].StartTime)
	})

	return events, nil
}

func (c *StubCalendar) ModifyEvent(ctx context.Context, event Event) ([]Event, error) {
	if event.UID == "" {
		return nil, errors.New("event.UID is required")
	}

	currentUser, err := user.CurrentUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	events, err := splitEventIfNeeded(&event, currentUser.Settings.Timezone)
	if err != nil {
		return nil, err
	}
	eventToUpdate := events[0]
	eventsToAdd := events[1:]

	foundEvent, ok := c.data[eventToUpdate.UID]
	if !ok {
		return nil, errors.New("event with given UID not found")
	}
	foundEvent.Summary = eventToUpdate.Summary
	foundEvent.Metadata = eventToUpdate.Metadata
	foundEvent.StartTime = eventToUpdate.StartTime
	foundEvent.EndTime = eventToUpdate.EndTime
	c.data[foundEvent.UID] = foundEvent

	for _, e := range eventsToAdd {
		c.data[e.UID] = e
	}

	return events, nil
}

func (c *StubCalendar) Cleanup() {
	c.data = map[string]Event{}
}

func (c *StubCalendar) GetLastEvents(ctx context.Context, limit int) ([]Event, error) {
	events := make([]Event, 0, len(c.data))
	for _, event := range c.data {
		events = append(events, event)
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].StartTime.Before(events[j].StartTime)
	})

	if len(events) > limit {
		events = events[:limit]
	}

	return events, nil
}

func (c *StubCalendar) DeleteEvent(ctx context.Context, eventUid string) error {
	_, ok := c.data[eventUid]
	if !ok {
		return errors.New("event with given UID not found")
	}
	delete(c.data, eventUid)
	return nil
}
