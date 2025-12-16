package calendar

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klokku/klokku/internal/event_bus"
	"github.com/klokku/klokku/pkg/user"
	"github.com/klokku/klokku/pkg/weekly_plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var eventBus = event_bus.NewEventBus()

var location, _ = time.LoadLocation("Europe/Warsaw")
var weeklyItemsProvider = func(ctx context.Context, date time.Time) ([]weekly_plan.WeeklyPlanItem, error) {
	return []weekly_plan.WeeklyPlanItem{
		{
			Id:           1,
			BudgetItemId: 101,
			Name:         "Test BudgetItem 1",
		},
		{
			Id:           2,
			BudgetItemId: 102,
			Name:         "Test BudgetItem 2",
		},
		{
			Id:           3,
			BudgetItemId: 103,
			Name:         "Test BudgetItem 3",
		},
	}, nil
}

// Test setup helper
func setupServiceTest(t *testing.T) (*Service, context.Context, func()) {
	repoStub := NewRepositoryStub()
	service := NewService(repoStub, eventBus, weeklyItemsProvider)
	ctx := user.WithUser(context.Background(), user.User{
		Id:          1,
		Uid:         uuid.NewString(),
		Username:    "test-user-1",
		DisplayName: "Test User 1",
		PhotoUrl:    "",
		Settings: user.Settings{
			Timezone:          "Europe/Warsaw",
			WeekFirstDay:      time.Monday,
			EventCalendarType: user.KlokkuCalendar,
			GoogleCalendar:    user.GoogleCalendarSettings{},
		},
	})

	return service, ctx, func() {
		t.Log("Teardown after test")
		repoStub.Reset()
	}
}

func TestService_AddStickyEvent(t *testing.T) {
	start := time.Date(2026, 1, 1, 10, 0, 0, 0, location)
	type compareEvent struct {
		Summary   string
		StartTime time.Time
		EndTime   time.Time
	}
	testCases := []struct {
		name           string
		existingEvents []Event
		eventToAdd     Event
		want           []compareEvent
	}{
		{
			name:           "No other events",
			existingEvents: []Event{},
			eventToAdd: Event{
				Summary:   "Added event",
				StartTime: start,
				EndTime:   start.Add(time.Hour),
				Metadata:  EventMetadata{BudgetItemId: 101},
			},
			want: []compareEvent{
				{
					Summary:   "Test BudgetItem 1",
					StartTime: start,
					EndTime:   start.Add(time.Hour),
				},
			},
		},
		{
			name: "Previous event overlaps",
			existingEvents: []Event{
				{
					UID:       uuid.NewString(),
					Summary:   "Previous event should be shortened",
					StartTime: start.Add(-time.Hour),
					EndTime:   start,
					Metadata:  EventMetadata{BudgetItemId: 101},
				},
			},
			eventToAdd: Event{
				Summary:   "Added event",
				StartTime: start.Add(-30 * time.Minute),
				EndTime:   start.Add(time.Hour),
				Metadata:  EventMetadata{BudgetItemId: 102},
			},
			want: []compareEvent{
				{
					Summary:   "Test BudgetItem 1",
					StartTime: start.Add(-time.Hour),
					EndTime:   start.Add(-30 * time.Minute),
				},
				{
					Summary:   "Test BudgetItem 2",
					StartTime: start.Add(-30 * time.Minute),
					EndTime:   start.Add(time.Hour),
				},
			},
		},
		{
			name: "Following event overlaps",
			existingEvents: []Event{
				{
					Summary:   "Following event start time should be shifted",
					StartTime: start.Add(30 * time.Minute),                // 10:30
					EndTime:   start.Add(time.Hour).Add(30 * time.Minute), // 11:30
					Metadata:  EventMetadata{BudgetItemId: 101},
				},
			},
			eventToAdd: Event{
				Summary:   "Added event",
				StartTime: start,                // 10:00
				EndTime:   start.Add(time.Hour), // 11:00
				Metadata:  EventMetadata{BudgetItemId: 102},
			},
			want: []compareEvent{
				{
					Summary:   "Test BudgetItem 2",  // Added event
					StartTime: start,                // 10:00
					EndTime:   start.Add(time.Hour), // 11:00
				},
				{
					Summary:   "Test BudgetItem 1",                        // Following event start time should be shifted
					StartTime: start.Add(time.Hour),                       // 11:00
					EndTime:   start.Add(time.Hour).Add(30 * time.Minute), // 11:30
				},
			},
		},
		{
			name: "Multiple events overlap",
			existingEvents: []Event{
				{
					Summary:   "Previous event should be shortened",
					StartTime: start.Add(-2 * time.Hour), // 08:00
					EndTime:   start,                     // 10:00
					Metadata:  EventMetadata{BudgetItemId: 101},
				},
				{
					Summary:   "Even during the newly added event should be removed",
					StartTime: start,                    // 10:00
					EndTime:   start.Add(2 * time.Hour), // 12:00
					Metadata:  EventMetadata{BudgetItemId: 102},
				},
				{
					Summary:   "Following event end time should be shifted",
					StartTime: start.Add(2 * time.Hour), // 12:00
					EndTime:   start.Add(4 * time.Hour), // 14:00
					Metadata:  EventMetadata{BudgetItemId: 103},
				},
			},
			eventToAdd: Event{
				Summary:   "Added event",
				StartTime: start.Add(-1 * time.Hour), // 09:00
				EndTime:   start.Add(3 * time.Hour),  // 13:00
				Metadata:  EventMetadata{BudgetItemId: 101},
			},
			want: []compareEvent{
				{
					Summary:   "Test BudgetItem 1", // Previous event should be shortened
					StartTime: start.Add(-2 * time.Hour),
					EndTime:   start.Add(-1 * time.Hour),
				},
				{
					Summary:   "Test BudgetItem 1", // Added event
					StartTime: start.Add(-1 * time.Hour),
					EndTime:   start.Add(3 * time.Hour),
				},
				{
					Summary:   "Test BudgetItem 3", // Following event end time should be shifted
					StartTime: start.Add(3 * time.Hour),
					EndTime:   start.Add(4 * time.Hour),
				},
			},
		},
		{
			name: "Add event inside existing event",
			existingEvents: []Event{
				{
					Summary:   "Existing event should be split in two",
					StartTime: start.Add(-1 * time.Hour), // 09:00
					EndTime:   start.Add(2 * time.Hour),  // 12:00
					Metadata:  EventMetadata{BudgetItemId: 101},
				},
			},
			eventToAdd: Event{
				Summary:   "Added event",
				StartTime: start,                    // 10:00
				EndTime:   start.Add(1 * time.Hour), // 11:00
				Metadata:  EventMetadata{BudgetItemId: 102},
			},
			want: []compareEvent{
				{
					Summary:   "Test BudgetItem 1",       // Existing event should be split in two
					StartTime: start.Add(-1 * time.Hour), // 09:00
					EndTime:   start,                     // 10:00
				},
				{
					Summary:   "Test BudgetItem 2",      // Added event
					StartTime: start,                    // 10:00
					EndTime:   start.Add(1 * time.Hour), // 11:00
				},
				{
					Summary:   "Test BudgetItem 1",      // Existing event should be split in two
					StartTime: start.Add(1 * time.Hour), // 11:00
					EndTime:   start.Add(2 * time.Hour), // 12:00
				},
			},
		},
		{
			name:           "Add multi-day event",
			existingEvents: []Event{},
			eventToAdd: Event{
				Summary:   "Added event",
				StartTime: start,                     // 10:00
				EndTime:   start.Add(24 * time.Hour), // 10:00 the next day
				Metadata:  EventMetadata{BudgetItemId: 103},
			},
			want: []compareEvent{
				{
					Summary:   "Test BudgetItem 3",
					StartTime: start,                                                                                        // 10:00
					EndTime:   time.Date(start.Year(), start.Month(), start.Day(), 23, 59, 59, 999999999, start.Location()), // 23:59
				},
				{
					Summary:   "Test BudgetItem 3",
					StartTime: start.Add(14 * time.Hour), // 00:00
					EndTime:   start.Add(24 * time.Hour), // 10:00 the next day
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s, ctx, teardown := setupServiceTest(t)
			defer teardown()

			for _, event := range tc.existingEvents {
				_, err := s.AddEvent(ctx, event)
				assert.NoError(t, err)
			}

			_, err := s.AddStickyEvent(ctx, tc.eventToAdd)
			assert.NoError(t, err)
			got, err := s.GetEvents(ctx, tc.eventToAdd.StartTime, tc.eventToAdd.EndTime)
			assert.NoError(t, err)
			assert.Len(t, got, len(tc.want))

			gotEventsToCompare := make([]compareEvent, len(got))
			for i, event := range got {
				gotEventsToCompare[i] = compareEvent{
					Summary:   event.Summary,
					StartTime: event.StartTime,
					EndTime:   event.EndTime,
				}
			}
			assert.Equalf(t, tc.want, gotEventsToCompare, "Got events: %v", gotEventsToCompare)

		})
	}
}

func TestService_ModifyStickyEvent(t *testing.T) {
	start := time.Date(2026, 1, 1, 10, 0, 0, 0, location)
	type compareEvent struct {
		Summary   string
		StartTime time.Time
		EndTime   time.Time
	}
	testCases := []struct {
		name           string
		existingEvents []Event
		eventToModify  Event
		want           []compareEvent
	}{
		{
			name:           "No other events",
			existingEvents: []Event{},
			eventToModify: Event{
				Summary:   "Modified event",
				StartTime: start,
				EndTime:   start.Add(time.Hour),
				Metadata:  EventMetadata{BudgetItemId: 101},
			},
			want: []compareEvent{
				{
					Summary:   "Test BudgetItem 1", // Modified event
					StartTime: start,
					EndTime:   start.Add(time.Hour),
				},
			},
		},
		{
			name: "Previous event overlaps",
			existingEvents: []Event{
				{
					Summary:   "Previous event should be shortened",
					StartTime: start.Add(-time.Hour),
					EndTime:   start,
					Metadata:  EventMetadata{BudgetItemId: 101},
				},
			},
			eventToModify: Event{
				Summary:   "Modified event",
				StartTime: start.Add(-30 * time.Minute),
				EndTime:   start.Add(time.Hour),
				Metadata:  EventMetadata{BudgetItemId: 102},
			},
			want: []compareEvent{
				{
					Summary:   "Test BudgetItem 1", // Previous event should be shortened
					StartTime: start.Add(-time.Hour),
					EndTime:   start.Add(-30 * time.Minute),
				},
				{
					Summary:   "Test BudgetItem 2", // Modified event
					StartTime: start.Add(-30 * time.Minute),
					EndTime:   start.Add(time.Hour),
				},
			},
		},
		{
			name: "Following event overlaps",
			existingEvents: []Event{
				{
					Summary:   "Following event start time should be shifted",
					StartTime: start.Add(30 * time.Minute),                // 10:30
					EndTime:   start.Add(time.Hour).Add(30 * time.Minute), // 11:30
					Metadata:  EventMetadata{BudgetItemId: 103},
				},
			},
			eventToModify: Event{
				Summary:   "Modified event",
				StartTime: start,                // 10:00
				EndTime:   start.Add(time.Hour), // 11:00
				Metadata:  EventMetadata{BudgetItemId: 102},
			},
			want: []compareEvent{
				{
					Summary:   "Test BudgetItem 2",  // Modified event
					StartTime: start,                // 10:00
					EndTime:   start.Add(time.Hour), // 11:00
				},
				{
					Summary:   "Test BudgetItem 3",                        // Following event start time should be shifted
					StartTime: start.Add(time.Hour),                       // 11:00
					EndTime:   start.Add(time.Hour).Add(30 * time.Minute), // 11:30
				},
			},
		},
		{
			name: "Multiple events overlap",
			existingEvents: []Event{
				{
					Summary:   "Previous event should be shortened",
					StartTime: start.Add(-2 * time.Hour), // 08:00
					EndTime:   start,                     // 10:00
					Metadata:  EventMetadata{BudgetItemId: 101},
				},
				{
					Summary:   "Even during the newly added event should be removed",
					StartTime: start,                    // 10:00
					EndTime:   start.Add(2 * time.Hour), // 12:00
					Metadata:  EventMetadata{BudgetItemId: 102},
				},
				{
					Summary:   "Following event end time should be shifted",
					StartTime: start.Add(2 * time.Hour), // 12:00
					EndTime:   start.Add(4 * time.Hour), // 14:00
					Metadata:  EventMetadata{BudgetItemId: 103},
				},
			},
			eventToModify: Event{
				Summary:   "Modified event",
				StartTime: start.Add(-1 * time.Hour), // 09:00
				EndTime:   start.Add(3 * time.Hour),  // 13:00
				Metadata:  EventMetadata{BudgetItemId: 101},
			},
			want: []compareEvent{
				{
					Summary:   "Test BudgetItem 1", // Previous event should be shortened
					StartTime: start.Add(-2 * time.Hour),
					EndTime:   start.Add(-1 * time.Hour),
				},
				{
					Summary:   "Test BudgetItem 1", // Modified event
					StartTime: start.Add(-1 * time.Hour),
					EndTime:   start.Add(3 * time.Hour),
				},
				{
					Summary:   "Test BudgetItem 3", // Following event end time should be shifted
					StartTime: start.Add(3 * time.Hour),
					EndTime:   start.Add(4 * time.Hour),
				},
			},
		},
		{
			name: "Modify event inside existing event",
			existingEvents: []Event{
				{
					Summary:   "Existing event should be split in two",
					StartTime: start.Add(-1 * time.Hour), // 09:00
					EndTime:   start.Add(2 * time.Hour),  // 12:00
					Metadata:  EventMetadata{BudgetItemId: 101},
				},
			},
			eventToModify: Event{
				Summary:   "Modified event",
				StartTime: start,                    // 10:00
				EndTime:   start.Add(1 * time.Hour), // 11:00
				Metadata:  EventMetadata{BudgetItemId: 102},
			},
			want: []compareEvent{
				{
					Summary:   "Test BudgetItem 1",       // Existing event should be split in two
					StartTime: start.Add(-1 * time.Hour), // 09:00
					EndTime:   start,                     // 10:00
				},
				{
					Summary:   "Test BudgetItem 2",      // Modified event
					StartTime: start,                    // 10:00
					EndTime:   start.Add(1 * time.Hour), // 11:00
				},
				{
					Summary:   "Test BudgetItem 1",      // Existing event should be split in two
					StartTime: start.Add(1 * time.Hour), // 11:00
					EndTime:   start.Add(2 * time.Hour), // 12:00
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s, ctx, teardown := setupServiceTest(t)
			defer teardown()

			// Seed existing overlapping events
			for _, event := range tc.existingEvents {
				_, err := s.AddEvent(ctx, event)
				assert.NoError(t, err)
			}

			// Create the event we will modify (with desired final times), capture UID
			created, err := s.AddEvent(ctx, tc.eventToModify)
			assert.NoError(t, err)
			// Perform sticky modification using the created event (same times, has UID)
			_, err = s.ModifyStickyEvent(ctx, created[0])
			assert.NoError(t, err)

			got, err := s.GetEvents(ctx, tc.eventToModify.StartTime, tc.eventToModify.EndTime)
			assert.NoError(t, err)
			assert.Len(t, got, len(tc.want))

			gotEventsToCompare := make([]compareEvent, len(got))
			for i, event := range got {
				gotEventsToCompare[i] = compareEvent{
					Summary:   event.Summary,
					StartTime: event.StartTime,
					EndTime:   event.EndTime,
				}
			}
			assert.Equalf(t, tc.want, gotEventsToCompare, "Got events: %v", gotEventsToCompare)

		})
	}
}

func TestService_ModifyStickyEvent_MultiDay(t *testing.T) {
	s, ctx, teardown := setupServiceTest(t)
	defer teardown()

	// given
	start := time.Date(2026, 1, 1, 22, 0, 0, 0, location)
	event := Event{
		Summary:   "Modified event",
		StartTime: start,                    // 22:00
		EndTime:   start.Add(1 * time.Hour), // 23:00
		Metadata:  EventMetadata{BudgetItemId: 101},
	}
	addedEvent, err := s.AddStickyEvent(ctx, event)
	require.NoError(t, err)

	// when
	addedEvent[0].EndTime = start.Add(4 * time.Hour) // 02:00 the next day
	modifiedEvents, err := s.ModifyStickyEvent(ctx, addedEvent[0])

	// then
	require.NoError(t, err)
	assert.Len(t, modifiedEvents, 2)
	assert.Equal(t, start, modifiedEvents[0].StartTime)
	assert.Equal(t, time.Date(start.Year(), start.Month(), start.Day(), 23, 59, 59, 999999999, location), modifiedEvents[0].EndTime)
	assert.Equal(t, start.Add(2*time.Hour), modifiedEvents[1].StartTime)
	assert.Equal(t, start.Add(4*time.Hour), modifiedEvents[1].EndTime)
}

func TestService_AddEvent(t *testing.T) {
	t.Run("publishes event to event bus", func(t *testing.T) {
		s, ctx, teardown := setupServiceTest(t)
		defer teardown()

		// given
		event := Event{
			Summary:   "Test BudgetItem 1",
			StartTime: time.Date(2023, 1, 1, 10, 0, 0, 0, location),
			EndTime:   time.Date(2023, 1, 1, 11, 0, 0, 0, location),
			Metadata:  EventMetadata{BudgetItemId: 101},
		}
		var publishedEvent event_bus.EventT[event_bus.CalendarEventCreated]
		event_bus.SubscribeTyped[event_bus.CalendarEventCreated](
			eventBus,
			"calendar.event.created",
			func(e event_bus.EventT[event_bus.CalendarEventCreated]) error {
				publishedEvent = e
				return nil
			},
		)

		// when
		addedEvents, err := s.AddEvent(ctx, event)
		assert.NoError(t, err)
		assert.Equal(t, event_bus.EventType("calendar.event.created"), publishedEvent.Type)
		assert.Equal(t, addedEvents[0].UID, publishedEvent.Data.UID)
		assert.Equal(t, event.Summary, publishedEvent.Data.Summary)
		assert.Equal(t, event.StartTime, publishedEvent.Data.StartTime)
		assert.Equal(t, event.EndTime, publishedEvent.Data.EndTime)
		assert.Equal(t, event.Metadata.BudgetItemId, publishedEvent.Data.BudgetItemId)
	})
}

func TestService_AddEvent_Validation(t *testing.T) {
	tests := []struct {
		name  string
		event Event
		want  error
	}{
		{
			name: "End time before start time",
			event: Event{
				Summary:   "Modified event",
				StartTime: time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				EndTime:   time.Date(2026, 1, 1, 9, 59, 0, 0, location),
				Metadata:  EventMetadata{BudgetItemId: 101},
			},
			want: errors.New("end time must be after start time"),
		},
		{
			name: "End time same as start time",
			event: Event{
				Summary:   "Modified event",
				StartTime: time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				EndTime:   time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				Metadata:  EventMetadata{BudgetItemId: 101},
			},
			want: errors.New("end time must be after start time"),
		},
		{
			name: "Start time is not set",
			event: Event{
				Summary:  "Modified event",
				EndTime:  time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				Metadata: EventMetadata{BudgetItemId: 101},
			},
			want: errors.New("start time cannot be zero"),
		},
		{
			name: "End time is not set",
			event: Event{
				StartTime: time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				Metadata:  EventMetadata{BudgetItemId: 101},
			},
			want: errors.New("end time cannot be zero"),
		},
		{
			name: "Budget item id is not set",
			event: Event{
				StartTime: time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				EndTime:   time.Date(2026, 1, 1, 11, 0, 0, 0, location),
			},
			want: errors.New("budget item id cannot be zero"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, ctx, teardown := setupServiceTest(t)
			defer teardown()
			_, err := s.AddEvent(ctx, test.event)
			assert.Equal(t, test.want, err)
		})
	}
}

func TestService_AddStickyEvent_Validation(t *testing.T) {
	tests := []struct {
		name  string
		event Event
		want  error
	}{
		{
			name: "End time before start time",
			event: Event{
				StartTime: time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				EndTime:   time.Date(2026, 1, 1, 9, 59, 0, 0, location),
				Metadata:  EventMetadata{BudgetItemId: 101},
			},
			want: errors.New("end time must be after start time"),
		},
		{
			name: "End time same as start time",
			event: Event{
				StartTime: time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				EndTime:   time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				Metadata:  EventMetadata{BudgetItemId: 101},
			},
			want: errors.New("end time must be after start time"),
		},
		{
			name: "Start time is not set",
			event: Event{
				EndTime:  time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				Metadata: EventMetadata{BudgetItemId: 101},
			},
			want: errors.New("start time cannot be zero"),
		},
		{
			name: "End time is not set",
			event: Event{
				StartTime: time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				Metadata:  EventMetadata{BudgetItemId: 101},
			},
			want: errors.New("end time cannot be zero"),
		},
		{
			name: "Budget item id is not set",
			event: Event{
				StartTime: time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				EndTime:   time.Date(2026, 1, 1, 11, 0, 0, 0, location),
			},
			want: errors.New("budget item id cannot be zero"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, ctx, teardown := setupServiceTest(t)
			defer teardown()
			_, err := s.AddStickyEvent(ctx, test.event)
			assert.Equal(t, test.want, err)
		})
	}
}

func TestService_ModifyEvent_Validation(t *testing.T) {
	tests := []struct {
		name  string
		event Event
		want  error
	}{
		{
			name: "End time before start time",
			event: Event{
				StartTime: time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				EndTime:   time.Date(2026, 1, 1, 9, 59, 0, 0, location),
				Metadata:  EventMetadata{BudgetItemId: 101},
			},
			want: errors.New("end time must be after start time"),
		},
		{
			name: "End time same as start time",
			event: Event{
				StartTime: time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				EndTime:   time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				Metadata:  EventMetadata{BudgetItemId: 101},
			},
			want: errors.New("end time must be after start time"),
		},
		{
			name: "Start time is not set",
			event: Event{
				EndTime:  time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				Metadata: EventMetadata{BudgetItemId: 101},
			},
			want: errors.New("start time cannot be zero"),
		},
		{
			name: "End time is not set",
			event: Event{
				StartTime: time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				Metadata:  EventMetadata{BudgetItemId: 101},
			},
			want: errors.New("end time cannot be zero"),
		},
		{
			name: "Budget Item id is not set",
			event: Event{
				StartTime: time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				EndTime:   time.Date(2026, 1, 1, 11, 0, 0, 0, location),
			},
			want: errors.New("budget item id cannot be zero"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, ctx, teardown := setupServiceTest(t)
			defer teardown()
			_, err := s.ModifyEvent(ctx, test.event)
			assert.Equal(t, test.want, err)
		})
	}
}

func TestService_ModifyStickyEvent_Validation(t *testing.T) {
	tests := []struct {
		name  string
		event Event
		want  error
	}{
		{
			name: "End time before start time",
			event: Event{
				StartTime: time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				EndTime:   time.Date(2026, 1, 1, 9, 59, 0, 0, location),
				Metadata:  EventMetadata{BudgetItemId: 101},
			},
			want: errors.New("end time must be after start time"),
		},
		{
			name: "End time same as start time",
			event: Event{
				StartTime: time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				EndTime:   time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				Metadata:  EventMetadata{BudgetItemId: 101},
			},
			want: errors.New("end time must be after start time"),
		},
		{
			name: "Start time is not set",
			event: Event{
				EndTime:  time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				Metadata: EventMetadata{BudgetItemId: 101},
			},
			want: errors.New("start time cannot be zero"),
		},
		{
			name: "End time is not set",
			event: Event{
				StartTime: time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				Metadata:  EventMetadata{BudgetItemId: 101},
			},
			want: errors.New("end time cannot be zero"),
		},
		{
			name: "Budget item id is not set",
			event: Event{
				StartTime: time.Date(2026, 1, 1, 10, 0, 0, 0, location),
				EndTime:   time.Date(2026, 1, 1, 11, 0, 0, 0, location),
			},
			want: errors.New("budget item id cannot be zero"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, ctx, teardown := setupServiceTest(t)
			defer teardown()
			_, err := s.ModifyStickyEvent(ctx, test.event)
			assert.Equal(t, test.want, err)
		})
	}
}
