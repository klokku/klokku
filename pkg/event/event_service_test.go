package event

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klokku/klokku/internal/test_utils"
	"github.com/klokku/klokku/internal/utils"
	"github.com/klokku/klokku/pkg/calendar"
	"github.com/klokku/klokku/pkg/user"

	"github.com/stretchr/testify/assert"
)

func TestStartNewEvent(t *testing.T) {
	ctx := context.WithValue(context.Background(), user.UserIDKey, 1)
	location, _ := time.LoadLocation("Europe/Warsaw")

	t.Run("No existing event, successfully starts new event", func(t *testing.T) {
		repo := &StubEventRepository{}
		cal := calendar.NewStubCalendar()
		clock := &utils.MockClock{FixedNow: time.Now().Truncate(time.Second)}
		service := &EventServiceImpl{repo: repo, calendar: cal, clock: clock}

		uid := uuid.NewString()
		newEvent := Event{UID: uid, StartTime: clock.Now()}

		result, err := service.StartNewEvent(ctx, newEvent)
		assert.NoError(t, err)
		currentEvent, err := repo.FindCurrentEvent(ctx, 1)
		currentEvent.StartTime = currentEvent.StartTime.In(time.Local)

		assert.NoError(t, err)
		assert.Equal(t, newEvent, result)
		assert.Equal(t, &newEvent, currentEvent)
	})

	t.Run("Existing event present, finishes it and starts new event", func(t *testing.T) {
		existingEvent := Event{UID: uuid.NewString(), StartTime: time.Now().Add(-1 * time.Hour)}
		repo := &StubEventRepository{}
		cal := calendar.NewStubCalendar()
		service := &EventServiceImpl{repo: repo, calendar: cal, clock: &utils.MockClock{FixedNow: time.Now()}, userProvider: test_utils.TestUserProvider{}}
		newEvent := Event{UID: uuid.NewString(), StartTime: time.Now()}

		service.StartNewEvent(ctx, existingEvent)
		result, err := service.StartNewEvent(ctx, newEvent)
		assert.NoError(t, err)
		currentEvent, err := repo.FindCurrentEvent(ctx, 1)

		assert.NoError(t, err)
		assert.Equal(t, newEvent, result)
		assert.Equal(t, newEvent.UID, currentEvent.UID)
	})

	t.Run("Existing event present between days, split previous event", func(t *testing.T) {
		now := time.Date(2024, time.December, 24, 1, 0, 0, 0, location)
		beforeMidnight := time.Date(now.Year(), now.Month(), now.Day()-1, 23, 0, 0, 0, location)
		endOfADay := time.Date(now.Year(), now.Month(), now.Day()-1, 23, 59, 59, 999999999, location)
		midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
		repo := &StubEventRepository{}
		cal := calendar.NewStubCalendar()
		service := &EventServiceImpl{repo: repo, calendar: cal, clock: &utils.MockClock{FixedNow: now}, userProvider: test_utils.TestUserProvider{}}

		previousEvent := Event{UID: uuid.NewString(), StartTime: beforeMidnight}
		service.StartNewEvent(ctx, previousEvent)
		newEvent := Event{UID: uuid.NewString(), StartTime: now}
		service.StartNewEvent(ctx, newEvent)

		calEvents, err := cal.GetEvents(ctx, beforeMidnight.Add(-1*time.Hour), now.Add(time.Hour))
		assert.NoError(t, err)
		assert.Len(t, calEvents, 2)
		assert.Equal(t, calEvents[0].StartTime.Unix(), beforeMidnight.Unix())
		assert.Equal(t, calEvents[0].EndTime, endOfADay)
		assert.Equal(t, calEvents[1].StartTime, midnight)
		assert.Equal(t, calEvents[1].EndTime, now)
	})

	t.Run("Existing event present early next day, do not split previous event", func(t *testing.T) {
		now := time.Date(2024, time.December, 24, 1, 0, 0, 0, location)
		firstEventStart := time.Date(2024, time.December, 24, 0, 5, 0, 0, location)
		repo := &StubEventRepository{}
		cal := calendar.NewStubCalendar()
		service := &EventServiceImpl{repo: repo, calendar: cal, clock: &utils.MockClock{FixedNow: now}, userProvider: test_utils.TestUserProvider{}}

		previousEvent := Event{UID: uuid.NewString(), StartTime: firstEventStart}
		service.StartNewEvent(ctx, previousEvent)
		newEvent := Event{UID: uuid.NewString(), StartTime: now}
		service.StartNewEvent(ctx, newEvent)

		calEvents, err := cal.GetEvents(ctx, firstEventStart.Add(-1*time.Hour), now.Add(time.Hour))
		assert.True(t, calEvents[0].StartTime.Before(calEvents[0].EndTime))
		assert.NoError(t, err)
		assert.Len(t, calEvents, 1)
	})
}
