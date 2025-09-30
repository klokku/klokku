package calendar

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klokku/klokku/pkg/user"
	"github.com/stretchr/testify/assert"
)

// Test setup helper
func setupServiceTest(t *testing.T) (*Service, context.Context) {
	repo := setupTestRepository(t)
	service := NewService(repo)
	ctx := context.WithValue(context.Background(), user.UserIDKey, "1")
	return service, ctx
}

func TestService_AddStickyEvent(t *testing.T) {
	start := time.Date(2026, 1, 1, 10, 0, 0, 0, time.Local)
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
			},
			want: []compareEvent{
				{
					Summary:   "Added event",
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
				},
			},
			eventToAdd: Event{
				Summary:   "Added event",
				StartTime: start.Add(-30 * time.Minute),
				EndTime:   start.Add(time.Hour),
			},
			want: []compareEvent{
				{
					Summary:   "Previous event should be shortened",
					StartTime: start.Add(-time.Hour),
					EndTime:   start.Add(-30 * time.Minute),
				},
				{
					Summary:   "Added event",
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
				},
			},
			eventToAdd: Event{
				Summary:   "Added event",
				StartTime: start,                // 10:00
				EndTime:   start.Add(time.Hour), // 11:00
			},
			want: []compareEvent{
				{
					Summary:   "Added event",
					StartTime: start,                // 10:00
					EndTime:   start.Add(time.Hour), // 11:00
				},
				{
					Summary:   "Following event start time should be shifted",
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
				},
				{
					Summary:   "Even during the newly added event should be removed",
					StartTime: start,                    // 10:00
					EndTime:   start.Add(2 * time.Hour), // 12:00
				},
				{
					Summary:   "Following event end time should be shifted",
					StartTime: start.Add(2 * time.Hour), // 12:00
					EndTime:   start.Add(4 * time.Hour), // 14:00
				},
			},
			eventToAdd: Event{
				UID:       uuid.NullUUID{},
				Summary:   "Added event",
				StartTime: start.Add(-1 * time.Hour), // 09:00
				EndTime:   start.Add(3 * time.Hour),  // 13:00
				Metadata:  EventMetadata{BudgetId: 1},
			},
			want: []compareEvent{
				{
					Summary:   "Previous event should be shortened",
					StartTime: start.Add(-2 * time.Hour),
					EndTime:   start.Add(-1 * time.Hour),
				},
				{
					Summary:   "Added event",
					StartTime: start.Add(-1 * time.Hour),
					EndTime:   start.Add(3 * time.Hour),
				},
				{
					Summary:   "Following event end time should be shifted",
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
				},
			},
			eventToAdd: Event{
				Summary:   "Added event",
				StartTime: start,                    // 10:00
				EndTime:   start.Add(1 * time.Hour), // 11:00
			},
			want: []compareEvent{
				{
					Summary:   "Existing event should be split in two",
					StartTime: start.Add(-1 * time.Hour), // 09:00
					EndTime:   start,                     // 10:00
				},
				{
					Summary:   "Added event",
					StartTime: start,                    // 10:00
					EndTime:   start.Add(1 * time.Hour), // 11:00
				},
				{
					Summary:   "Existing event should be split in two",
					StartTime: start.Add(1 * time.Hour), // 11:00
					EndTime:   start.Add(2 * time.Hour), // 12:00
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s, ctx := setupServiceTest(t)

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
	start := time.Date(2026, 1, 1, 10, 0, 0, 0, time.Local)
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
			},
			want: []compareEvent{
				{
					Summary:   "Modified event",
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
				},
			},
			eventToModify: Event{
				Summary:   "Modified event",
				StartTime: start.Add(-30 * time.Minute),
				EndTime:   start.Add(time.Hour),
			},
			want: []compareEvent{
				{
					Summary:   "Previous event should be shortened",
					StartTime: start.Add(-time.Hour),
					EndTime:   start.Add(-30 * time.Minute),
				},
				{
					Summary:   "Modified event",
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
				},
			},
			eventToModify: Event{
				Summary:   "Modified event",
				StartTime: start,                // 10:00
				EndTime:   start.Add(time.Hour), // 11:00
			},
			want: []compareEvent{
				{
					Summary:   "Modified event",
					StartTime: start,                // 10:00
					EndTime:   start.Add(time.Hour), // 11:00
				},
				{
					Summary:   "Following event start time should be shifted",
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
				},
				{
					Summary:   "Even during the newly added event should be removed",
					StartTime: start,                    // 10:00
					EndTime:   start.Add(2 * time.Hour), // 12:00
				},
				{
					Summary:   "Following event end time should be shifted",
					StartTime: start.Add(2 * time.Hour), // 12:00
					EndTime:   start.Add(4 * time.Hour), // 14:00
				},
			},
			eventToModify: Event{
				UID:       uuid.NullUUID{},
				Summary:   "Modified event",
				StartTime: start.Add(-1 * time.Hour), // 09:00
				EndTime:   start.Add(3 * time.Hour),  // 13:00
				Metadata:  EventMetadata{BudgetId: 1},
			},
			want: []compareEvent{
				{
					Summary:   "Previous event should be shortened",
					StartTime: start.Add(-2 * time.Hour),
					EndTime:   start.Add(-1 * time.Hour),
				},
				{
					Summary:   "Modified event",
					StartTime: start.Add(-1 * time.Hour),
					EndTime:   start.Add(3 * time.Hour),
				},
				{
					Summary:   "Following event end time should be shifted",
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
				},
			},
			eventToModify: Event{
				Summary:   "Modified event",
				StartTime: start,                    // 10:00
				EndTime:   start.Add(1 * time.Hour), // 11:00
			},
			want: []compareEvent{
				{
					Summary:   "Existing event should be split in two",
					StartTime: start.Add(-1 * time.Hour), // 09:00
					EndTime:   start,                     // 10:00
				},
				{
					Summary:   "Modified event",
					StartTime: start,                    // 10:00
					EndTime:   start.Add(1 * time.Hour), // 11:00
				},
				{
					Summary:   "Existing event should be split in two",
					StartTime: start.Add(1 * time.Hour), // 11:00
					EndTime:   start.Add(2 * time.Hour), // 12:00
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s, ctx := setupServiceTest(t)

			// Seed existing overlapping events
			for _, event := range tc.existingEvents {
				_, err := s.AddEvent(ctx, event)
				assert.NoError(t, err)
			}

			// Create the event we will modify (with desired final times), capture UID
			created, err := s.AddEvent(ctx, tc.eventToModify)
			assert.NoError(t, err)
			// Perform sticky modification using the created event (same times, has UID)
			_, err = s.ModifyStickyEvent(ctx, *created)
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
