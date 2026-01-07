package current_event

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klokku/klokku/internal/utils"
	"github.com/klokku/klokku/pkg/calendar"
	"github.com/klokku/klokku/pkg/user"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

var clock *utils.MockClock
var location, _ = time.LoadLocation("Europe/Warsaw")
var calendarStub *calendar.StubCalendar

func setupServiceTest(t *testing.T) (Service, context.Context, func()) {
	repoStub := newStubEventRepository()
	calendarStub = calendar.NewStubCalendar()
	clock = &utils.MockClock{FixedNow: time.Date(2025, time.December, 20, 14, 0, 0, 0, location)}
	service := &EventServiceImpl{
		repo:     repoStub,
		calendar: calendarStub,
		clock:    clock,
	}
	ctx := user.WithUser(context.Background(), user.User{
		Id:          1,
		Uid:         uuid.NewString(),
		Username:    "test-user-1",
		DisplayName: "Test User 1",
		PhotoUrl:    "",
		Settings: user.Settings{
			Timezone:          location.String(),
			WeekFirstDay:      time.Monday,
			EventCalendarType: user.KlokkuCalendar,
			GoogleCalendar:    user.GoogleCalendarSettings{},
		},
	})

	return service, ctx, func() {
		t.Log("Teardown after test")
		repoStub.reset()
		calendarStub.Cleanup()
		clock = &utils.MockClock{FixedNow: time.Date(2025, time.December, 20, 14, 0, 0, 0, location)}
	}
}

func TestStartNewEvent(t *testing.T) {

	t.Run("No existing event, successfully starts new event", func(t *testing.T) {
		service, ctx, teardown := setupServiceTest(t)
		defer teardown()
		newEvent := CurrentEvent{
			PlanItem: PlanItem{
				BudgetItemId:   10,
				Name:           "Test 10",
				WeeklyDuration: time.Duration(120) * time.Minute,
			},
			StartTime: clock.Now(),
		}

		result, err := service.StartNewEvent(ctx, newEvent)
		assert.NoError(t, err)
		currentEvent, err := service.FindCurrentEvent(ctx)

		assert.NoError(t, err)
		assert.Equal(t, newEvent.StartTime, result.StartTime)
		assert.Equal(t, newEvent.PlanItem, result.PlanItem)
		assert.Equal(t, newEvent.StartTime, currentEvent.StartTime)
		assert.Equal(t, newEvent.PlanItem, currentEvent.PlanItem)
	})

	t.Run("Existing event present, finishes it and starts new event", func(t *testing.T) {
		service, ctx, teardown := setupServiceTest(t)
		defer teardown()
		existingEvent := CurrentEvent{
			StartTime: time.Now().Add(-1 * time.Hour),
			PlanItem: PlanItem{
				BudgetItemId:   9,
				Name:           "Test 9",
				WeeklyDuration: time.Duration(30) * time.Minute,
			},
		}

		newEvent := CurrentEvent{
			PlanItem: PlanItem{
				BudgetItemId:   10,
				Name:           "Test 10",
				WeeklyDuration: time.Duration(120) * time.Minute,
			},
			StartTime: clock.Now(),
		}

		clock.SetNow(clock.Now().Add(-1 * time.Hour))
		service.StartNewEvent(ctx, existingEvent)
		clock.SetNow(clock.Now().Add(1 * time.Hour))
		result, err := service.StartNewEvent(ctx, newEvent)
		assert.NoError(t, err)
		currentEvent, err := service.FindCurrentEvent(ctx)

		assert.NoError(t, err)
		assert.Equal(t, newEvent.StartTime, result.StartTime)
		assert.Equal(t, newEvent.PlanItem, result.PlanItem)
		assert.Equal(t, newEvent.StartTime, currentEvent.StartTime)
		assert.Equal(t, newEvent.PlanItem, currentEvent.PlanItem)
	})

	t.Run("should store event in calendar when replacing with new current event", func(t *testing.T) {
		service, ctx, teardown := setupServiceTest(t)
		defer teardown()

		// given
		existingEvent := CurrentEvent{
			StartTime: clock.Now().Add(-234 * time.Minute),
			PlanItem: PlanItem{
				BudgetItemId:   123,
				Name:           "To be stored",
				WeeklyDuration: time.Duration(30) * time.Minute,
			},
		}

		newEvent := CurrentEvent{
			StartTime: clock.Now(),
			PlanItem: PlanItem{
				BudgetItemId:   345,
				Name:           "Latest current event",
				WeeklyDuration: time.Duration(120) * time.Minute,
			},
		}

		// when
		clock.SetNow(clock.Now().Add(-234 * time.Minute))
		service.StartNewEvent(ctx, existingEvent)
		clock.SetNow(newEvent.StartTime)
		_, err := service.StartNewEvent(ctx, newEvent)
		require.NoError(t, err)

		// then
		calendarEvents, err := calendarStub.GetLastEvents(ctx, 1)
		require.NoError(t, err)
		assert.Equal(t, "To be stored", calendarEvents[0].Summary)
		assert.Equal(t, existingEvent.StartTime, calendarEvents[0].StartTime)
		assert.Equal(t, existingEvent.PlanItem.BudgetItemId, calendarEvents[0].Metadata.BudgetItemId)
	})
}

func TestModifyCurrentEventStartTime(t *testing.T) {

	t.Run("should modify start time of current event", func(t *testing.T) {
		service, ctx, teardown := setupServiceTest(t)
		defer teardown()

		// given
		existingEvent := CurrentEvent{
			StartTime: clock.Now().Add(-90 * time.Minute),
			PlanItem: PlanItem{
				BudgetItemId:   123,
				Name:           "To be modified",
				WeeklyDuration: time.Duration(30) * time.Hour,
			},
		}

		// when
		service.StartNewEvent(ctx, existingEvent)
		service.ModifyCurrentEventStartTime(ctx, clock.Now().Add(-120*time.Minute))

		// then
		afterModification, err := service.FindCurrentEvent(ctx)
		require.NoError(t, err)
		assert.Equal(t, clock.Now().Add(-120*time.Minute), afterModification.StartTime)
		assert.Equal(t, existingEvent.PlanItem.BudgetItemId, afterModification.PlanItem.BudgetItemId)
	})

	t.Run("should modify end time of previous calendar event when current event start time is changed to more in the past", func(t *testing.T) {
		service, ctx, teardown := setupServiceTest(t)
		defer teardown()

		// given
		startTime := clock.Now().Add(-180 * time.Minute)
		clock.SetNow(startTime)
		previousEvent := CurrentEvent{
			StartTime: clock.Now(),
			PlanItem: PlanItem{
				BudgetItemId:   1,
				Name:           "Previous event",
				WeeklyDuration: time.Duration(30) * time.Hour,
			},
		}
		service.StartNewEvent(ctx, previousEvent)
		clock.SetNow(startTime.Add(120 * time.Minute))
		currentEvent := CurrentEvent{
			StartTime: clock.Now(),
			PlanItem: PlanItem{
				BudgetItemId:   2,
				Name:           "Current event",
				WeeklyDuration: time.Duration(30) * time.Hour,
			},
		}
		service.StartNewEvent(ctx, currentEvent)
		clock.SetNow(clock.Now().Add(60 * time.Minute))

		// when
		service.ModifyCurrentEventStartTime(ctx, clock.Now().Add(-120*time.Minute))

		// then
		calendarEvents, err := calendarStub.GetLastEvents(ctx, 1)
		require.NoError(t, err)
		calendarEvent := calendarEvents[0]
		assert.Equal(t, "Previous event", calendarEvent.Summary)
		assert.Equal(t, clock.Now().Add(-180*time.Minute), calendarEvent.StartTime)
		assert.Equal(t, clock.Now().Add(-120*time.Minute), calendarEvent.EndTime)
	})

	t.Run("should modify end time of previous calendar event when current event start time is changed to less in the past", func(t *testing.T) {
		service, ctx, teardown := setupServiceTest(t)
		defer teardown()

		// given
		clock.SetNow(time.Date(2025, time.December, 20, 10, 0, 0, 0, location))
		previousEvent := CurrentEvent{
			StartTime: clock.Now(), // 10:00
			PlanItem: PlanItem{
				BudgetItemId:   1,
				Name:           "Previous event",
				WeeklyDuration: time.Duration(30) * time.Hour,
			},
		}
		service.StartNewEvent(ctx, previousEvent)
		clock.SetNow(clock.Now().Add(120 * time.Minute)) // 12:00
		currentEvent := CurrentEvent{
			StartTime: clock.Now(),
			PlanItem: PlanItem{
				BudgetItemId:   2,
				Name:           "Current event",
				WeeklyDuration: time.Duration(30) * time.Hour,
			},
		}
		service.StartNewEvent(ctx, currentEvent)
		clock.SetNow(clock.Now().Add(60 * time.Minute)) // 13:00

		// when
		service.ModifyCurrentEventStartTime(ctx, clock.Now().Add(-30*time.Minute)) // 12:30

		// then
		calendarEvents, err := calendarStub.GetLastEvents(ctx, 1)
		require.NoError(t, err)
		calendarEvent := calendarEvents[0]
		assert.Equal(t, "Previous event", calendarEvent.Summary)
		assert.Equal(t, clock.Now().Add(-180*time.Minute), calendarEvent.StartTime)
		assert.Equal(t, clock.Now().Add(-30*time.Minute), calendarEvent.EndTime)
	})

	t.Run("should not modify start time when given start time is in the future", func(t *testing.T) {
		service, ctx, teardown := setupServiceTest(t)
		defer teardown()

		// given
		clock.SetNow(time.Date(2025, time.December, 20, 10, 0, 0, 0, location))
		previousEvent := CurrentEvent{
			StartTime: clock.Now(), // 10:00
			PlanItem: PlanItem{
				BudgetItemId:   1,
				Name:           "Previous event",
				WeeklyDuration: time.Duration(30) * time.Hour,
			},
		}
		service.StartNewEvent(ctx, previousEvent)
		clock.SetNow(clock.Now().Add(120 * time.Minute)) // 12:00
		currentEvent := CurrentEvent{
			StartTime: clock.Now(),
			PlanItem: PlanItem{
				BudgetItemId:   2,
				Name:           "Current event",
				WeeklyDuration: time.Duration(30) * time.Hour,
			},
		}
		service.StartNewEvent(ctx, currentEvent)
		clock.SetNow(clock.Now().Add(60 * time.Minute)) // 13:00

		// when
		_, err := service.ModifyCurrentEventStartTime(ctx, clock.Now().Add(2*time.Minute)) // 13:02

		// then
		assert.EqualError(t, err, "new start time cannot be in the future")
	})

	t.Run("should remove previous event end modify end time of the earlier one when current event start time is changed to more in the past", func(t *testing.T) {
		service, ctx, teardown := setupServiceTest(t)
		defer teardown()

		// given
		clock.SetNow(time.Date(2025, time.December, 20, 13, 0, 0, 0, location))
		startTime := clock.Now().Add(-180 * time.Minute)
		clock.SetNow(startTime)
		earlierEvent := CurrentEvent{
			StartTime: clock.Now(), // 10:00
			PlanItem: PlanItem{
				BudgetItemId:   1,
				Name:           "Earlier event",
				WeeklyDuration: time.Duration(30) * time.Hour,
			},
		}
		service.StartNewEvent(ctx, earlierEvent)
		clock.SetNow(startTime.Add(60 * time.Minute))
		previousEvent := CurrentEvent{
			StartTime: clock.Now(), // 11:00
			PlanItem: PlanItem{
				BudgetItemId:   2,
				Name:           "Previous event",
				WeeklyDuration: time.Duration(30) * time.Hour,
			},
		}
		service.StartNewEvent(ctx, previousEvent)
		clock.SetNow(clock.Now().Add(60 * time.Minute))
		currentEvent := CurrentEvent{
			StartTime: clock.Now(), // 12:00
			PlanItem: PlanItem{
				BudgetItemId:   3,
				Name:           "Current event",
				WeeklyDuration: time.Duration(30) * time.Hour,
			},
		}
		service.StartNewEvent(ctx, currentEvent)
		clock.SetNow(clock.Now().Add(60 * time.Minute)) // 13:00

		// when
		service.ModifyCurrentEventStartTime(ctx, clock.Now().Add(-130*time.Minute)) // 10:50

		// then
		calendarEvents, err := calendarStub.GetLastEvents(ctx, 3)
		require.NoError(t, err)
		assert.Len(t, calendarEvents, 1)
		calendarEvent := calendarEvents[0]
		assert.Equal(t, "Earlier event", calendarEvent.Summary)
		assert.Equal(t, clock.Now().Add(-180*time.Minute), calendarEvent.StartTime)
		assert.Equal(t, clock.Now().Add(-130*time.Minute), calendarEvent.EndTime)
	})
}
