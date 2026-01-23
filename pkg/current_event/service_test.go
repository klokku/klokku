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

	t.Run("should not store short event when IgnoreShortEvents is true and use previous event start time", func(t *testing.T) {
		service, ctx, teardown := setupServiceTest(t)
		defer teardown()

		// given - enable IgnoreShortEvents setting
		currentUser, _ := user.CurrentUser(ctx)
		currentUser.Settings.IgnoreShortEvents = true
		ctx = user.WithUser(context.Background(), currentUser)

		shortEventStartTime := clock.Now().Add(-45 * time.Second)
		shortEvent := CurrentEvent{
			StartTime: shortEventStartTime,
			PlanItem: PlanItem{
				BudgetItemId:   100,
				Name:           "Short event (should not be stored)",
				WeeklyDuration: time.Duration(30) * time.Minute,
			},
		}

		newEvent := CurrentEvent{
			StartTime: clock.Now(),
			PlanItem: PlanItem{
				BudgetItemId:   200,
				Name:           "New event",
				WeeklyDuration: time.Duration(120) * time.Minute,
			},
		}

		// when
		clock.SetNow(shortEventStartTime)
		service.StartNewEvent(ctx, shortEvent)
		clock.SetNow(shortEventStartTime.Add(45 * time.Second)) // 45 seconds later (< 1 minute)
		result, err := service.StartNewEvent(ctx, newEvent)
		require.NoError(t, err)

		// then - no events should be stored in calendar
		calendarEvents, err := calendarStub.GetLastEvents(ctx, 10)
		require.NoError(t, err)
		assert.Len(t, calendarEvents, 0, "Short event should not be stored in calendar")

		// and - new event should have the start time of the previous short event
		assert.Equal(t, shortEventStartTime, result.StartTime)
		currentEvent, err := service.FindCurrentEvent(ctx)
		require.NoError(t, err)
		assert.Equal(t, shortEventStartTime, currentEvent.StartTime)
		assert.Equal(t, newEvent.PlanItem, currentEvent.PlanItem)
	})

	t.Run("should store short event when IgnoreShortEvents is false", func(t *testing.T) {
		service, ctx, teardown := setupServiceTest(t)
		defer teardown()

		// given - IgnoreShortEvents is false by default
		shortEventStartTime := clock.Now().Add(-45 * time.Second)
		shortEvent := CurrentEvent{
			StartTime: shortEventStartTime,
			PlanItem: PlanItem{
				BudgetItemId:   100,
				Name:           "Short event (should be stored)",
				WeeklyDuration: time.Duration(30) * time.Minute,
			},
		}

		newEvent := CurrentEvent{
			StartTime: clock.Now(),
			PlanItem: PlanItem{
				BudgetItemId:   200,
				Name:           "New event",
				WeeklyDuration: time.Duration(120) * time.Minute,
			},
		}

		// when
		clock.SetNow(shortEventStartTime)
		service.StartNewEvent(ctx, shortEvent)
		clock.SetNow(shortEventStartTime.Add(45 * time.Second)) // 45 seconds later (< 1 minute)
		result, err := service.StartNewEvent(ctx, newEvent)
		require.NoError(t, err)

		// then - short event should be stored in calendar
		calendarEvents, err := calendarStub.GetLastEvents(ctx, 1)
		require.NoError(t, err)
		assert.Len(t, calendarEvents, 1)
		assert.Equal(t, "Short event (should be stored)", calendarEvents[0].Summary)
		assert.Equal(t, shortEventStartTime, calendarEvents[0].StartTime)

		// and - new event should have its own start time
		assert.Equal(t, shortEventStartTime.Add(45*time.Second), result.StartTime)
	})

	t.Run("should store event when IgnoreShortEvents is true but event is longer than one minute", func(t *testing.T) {
		service, ctx, teardown := setupServiceTest(t)
		defer teardown()

		// given - enable IgnoreShortEvents setting
		currentUser, _ := user.CurrentUser(ctx)
		currentUser.Settings.IgnoreShortEvents = true
		ctx = user.WithUser(context.Background(), currentUser)

		longEventStartTime := clock.Now().Add(-90 * time.Second)
		longEvent := CurrentEvent{
			StartTime: longEventStartTime,
			PlanItem: PlanItem{
				BudgetItemId:   100,
				Name:           "Long event (should be stored)",
				WeeklyDuration: time.Duration(30) * time.Minute,
			},
		}

		newEvent := CurrentEvent{
			StartTime: clock.Now(),
			PlanItem: PlanItem{
				BudgetItemId:   200,
				Name:           "New event",
				WeeklyDuration: time.Duration(120) * time.Minute,
			},
		}

		// when
		clock.SetNow(longEventStartTime)
		service.StartNewEvent(ctx, longEvent)
		clock.SetNow(longEventStartTime.Add(90 * time.Second)) // 90 seconds later (>= 1 minute)
		result, err := service.StartNewEvent(ctx, newEvent)
		require.NoError(t, err)

		// then - long event should be stored in calendar
		calendarEvents, err := calendarStub.GetLastEvents(ctx, 1)
		require.NoError(t, err)
		assert.Len(t, calendarEvents, 1)
		assert.Equal(t, "Long event (should be stored)", calendarEvents[0].Summary)
		assert.Equal(t, longEventStartTime, calendarEvents[0].StartTime)

		// and - new event should have its own start time
		assert.Equal(t, longEventStartTime.Add(90*time.Second), result.StartTime)
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

	t.Run("should only modify the last event when there are multiple previous events but start time only change to the last one", func(t *testing.T) {
		service, ctx, teardown := setupServiceTest(t)
		defer teardown()

		// given
		clock.SetNow(time.Date(2026, time.January, 7, 0, 0, 0, 0, location))
		startTime := clock.Now()
		clock.SetNow(startTime)
		prevDayEvent1 := CurrentEvent{
			StartTime: clock.Now(), // 07.01 00:00
			PlanItem: PlanItem{
				BudgetItemId:   1,
				Name:           "Prev day event 1",
				WeeklyDuration: time.Duration(52) * time.Hour,
			},
		}
		service.StartNewEvent(ctx, prevDayEvent1)
		clock.SetNow(startTime.Add(8 * time.Hour))
		prevDayEvent2 := CurrentEvent{
			StartTime: clock.Now(), // 07.01 8:00
			PlanItem: PlanItem{
				BudgetItemId:   2,
				Name:           "Prev day event 2",
				WeeklyDuration: time.Duration(30) * time.Hour,
			},
		}
		service.StartNewEvent(ctx, prevDayEvent2)
		clock.SetNow(clock.Now().Add(8 * time.Hour))
		betweenDayEvent1 := CurrentEvent{
			StartTime: clock.Now(), // 07.01 16:00
			PlanItem: PlanItem{
				BudgetItemId:   3,
				Name:           "Between days event 1 (3)",
				WeeklyDuration: time.Duration(30) * time.Hour,
			},
		}
		service.StartNewEvent(ctx, betweenDayEvent1)
		clock.SetNow(clock.Now().Add(9 * time.Hour))
		currentDayEvent1 := CurrentEvent{
			StartTime: clock.Now(), // 08.01 01:00
			PlanItem: PlanItem{
				BudgetItemId:   3,
				Name:           "Current day event 1 (4)",
				WeeklyDuration: time.Duration(30) * time.Hour,
			},
		}
		service.StartNewEvent(ctx, currentDayEvent1)
		clock.SetNow(clock.Now().Add(7 * time.Hour))
		currentEvent := CurrentEvent{
			StartTime: clock.Now(), // 08.01 08:00
			PlanItem: PlanItem{
				BudgetItemId:   3,
				Name:           "Current day event 2 (5)",
				WeeklyDuration: time.Duration(30) * time.Hour,
			},
		}
		service.StartNewEvent(ctx, currentEvent)
		clock.SetNow(clock.Now().Add(1 * time.Hour)) // 08.01 09:00

		// when
		service.ModifyCurrentEventStartTime(ctx, clock.Now().Add(-2*time.Hour)) // 08.01 07:00

		// then
		calendarEvents, err := calendarStub.GetLastEvents(ctx, 10)
		require.NoError(t, err)
		assert.Len(t, calendarEvents, 5)
		prevDayCalEvent1 := calendarEvents[0]
		assert.Equal(t, prevDayEvent1.PlanItem.Name, prevDayCalEvent1.Summary)
		assert.Equal(t, prevDayEvent1.StartTime, prevDayCalEvent1.StartTime)
		assert.Equal(t, prevDayEvent1.StartTime.Add(time.Duration(8)*time.Hour), prevDayCalEvent1.EndTime)
		prevDayCalEvent2 := calendarEvents[1]
		assert.Equal(t, prevDayEvent2.PlanItem.Name, prevDayCalEvent2.Summary)
		assert.Equal(t, prevDayEvent2.StartTime, prevDayCalEvent2.StartTime)
		assert.Equal(t, prevDayEvent2.StartTime.Add(time.Duration(8)*time.Hour), prevDayCalEvent2.EndTime)
		betweenDayCalEvent3 := calendarEvents[2]
		assert.Equal(t, betweenDayEvent1.PlanItem.Name, betweenDayCalEvent3.Summary)
		assert.Equal(t, betweenDayEvent1.StartTime, betweenDayCalEvent3.StartTime)
		assert.Equal(t, betweenDayEvent1.StartTime.Add(time.Duration(8)*time.Hour).Add(-time.Duration(1)*time.Nanosecond),
			betweenDayCalEvent3.EndTime)
		betweenDayCalEvent4 := calendarEvents[3]
		assert.Equal(t, betweenDayEvent1.PlanItem.Name, betweenDayCalEvent3.Summary)
		assert.Equal(t, betweenDayEvent1.StartTime.Add(time.Duration(8)*time.Hour), betweenDayCalEvent4.StartTime)
		assert.Equal(t, betweenDayEvent1.StartTime.Add(time.Duration(9)*time.Hour), betweenDayCalEvent4.EndTime)
		currentDayCalEvent1 := calendarEvents[4]
		assert.Equal(t, currentDayEvent1.PlanItem.Name, currentDayCalEvent1.Summary)
		assert.Equal(t, currentDayEvent1.StartTime, currentDayCalEvent1.StartTime)
		assert.Equal(t, currentDayEvent1.StartTime.Add(time.Duration(7-1)*time.Hour), currentDayCalEvent1.EndTime)
	})
}
