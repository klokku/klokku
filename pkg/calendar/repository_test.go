package calendar

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klokku/klokku/internal/test_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRepository creates a test repository with a fresh database
func setupTestRepository(t *testing.T) *RepositoryImpl {
	db := test_utils.SetupTestDB(t)
	repository := NewRepository(db)
	return repository
}

func setupRepositoryTest(t *testing.T) (*RepositoryImpl, context.Context, int) {
	repository := setupTestRepository(t)
	ctx := context.Background()
	userId := 1
	return repository, ctx, userId
}

// createTestEvent creates an event with the given parameters
func createTestEvent(summary string, start, end time.Time, budgetId int) Event {
	return Event{
		Summary:   summary,
		StartTime: start,
		EndTime:   end,
		Metadata:  EventMetadata{BudgetId: budgetId},
	}
}

// assertEventEqual verifies that the fetched event matches the expected event
func assertEventEqual(t *testing.T, expected Event, actual Event, ignoreUID bool) {
	assert.Equal(t, expected.Summary, actual.Summary)
	assert.Equal(t, expected.StartTime, actual.StartTime)
	assert.Equal(t, expected.EndTime, actual.EndTime)
	assert.Equal(t, expected.Metadata.BudgetId, actual.Metadata.BudgetId)

	if !ignoreUID && expected.UID.Valid {
		assert.Equal(t, expected.UID, actual.UID)
	}
}

func TestRepositoryImpl_StoreEvent(t *testing.T) {
	// Setup
	repository, ctx, userId := setupRepositoryTest(t)

	// Given
	baseTime := time.Now().Truncate(time.Millisecond)
	testEvent := createTestEvent("Test Event", baseTime, baseTime.Add(time.Hour), 654)

	// When
	storedEventUid, err := repository.StoreEvent(ctx, userId, testEvent)

	// Then
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, storedEventUid)

	// Verify event was stored correctly
	storedEvents, err := repository.GetEvents(ctx, userId, baseTime, baseTime.Add(time.Hour))
	require.NoError(t, err)
	require.Len(t, storedEvents, 1)

	// The stored event should match our original event plus have the generated UID
	expectedEvent := testEvent
	expectedEvent.UID = uuid.NullUUID{UUID: storedEventUid, Valid: true}

	assertEventEqual(t, expectedEvent, storedEvents[0], false)
}

func TestRepositoryImpl_GetEventsEmptyResult(t *testing.T) {
	// Setup
	repository, ctx, userId := setupRepositoryTest(t)

	// When
	queryStart := time.Now().Truncate(time.Millisecond)
	queryEnd := queryStart.Add(time.Hour)
	events, err := repository.GetEvents(ctx, userId, queryStart, queryEnd)

	// Then
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestRepositoryImpl_GetEvents(t *testing.T) {
	// Setup test cases for different overlap scenarios
	testCases := []struct {
		name          string
		eventStart    time.Duration // relative to baseTime
		eventEnd      time.Duration // relative to baseTime
		queryStart    time.Duration // relative to baseTime
		queryEnd      time.Duration // relative to baseTime
		shouldBeFound bool
	}{
		{
			name:          "Event fully inside query period",
			eventStart:    30 * time.Minute,
			eventEnd:      45 * time.Minute,
			queryStart:    0,
			queryEnd:      time.Hour,
			shouldBeFound: true,
		},
		{
			name:          "Event fully contains query period",
			eventStart:    -30 * time.Minute,
			eventEnd:      2 * time.Hour,
			queryStart:    0,
			queryEnd:      time.Hour,
			shouldBeFound: true,
		},
		{
			name:          "Event starts before and ends during query period",
			eventStart:    -30 * time.Minute,
			eventEnd:      30 * time.Minute,
			queryStart:    0,
			queryEnd:      time.Hour,
			shouldBeFound: true,
		},
		{
			name:          "Event starts during and ends after query period",
			eventStart:    30 * time.Minute,
			eventEnd:      90 * time.Minute,
			queryStart:    0,
			queryEnd:      time.Hour,
			shouldBeFound: true,
		},
		{
			name:          "Event ends exactly at query start (edge case)",
			eventStart:    -30 * time.Minute,
			eventEnd:      0,
			queryStart:    0,
			queryEnd:      time.Hour,
			shouldBeFound: true, // Should be found if inclusive of boundary
		},
		{
			name:          "Event starts exactly at query end (edge case)",
			eventStart:    time.Hour,
			eventEnd:      90 * time.Minute,
			queryStart:    0,
			queryEnd:      time.Hour,
			shouldBeFound: true, // Should be found if inclusive of boundary
		},
		{
			name:          "Event entirely before query period",
			eventStart:    -2 * time.Hour,
			eventEnd:      -time.Hour,
			queryStart:    0,
			queryEnd:      time.Hour,
			shouldBeFound: false,
		},
		{
			name:          "Event entirely after query period",
			eventStart:    2 * time.Hour,
			eventEnd:      3 * time.Hour,
			queryStart:    0,
			queryEnd:      time.Hour,
			shouldBeFound: false,
		},
	}

	// Run each test case
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			repository, ctx, userId := setupRepositoryTest(t)

			// Given
			baseTime := time.Now().Truncate(time.Millisecond)
			event := createTestEvent(
				tc.name,
				baseTime.Add(tc.eventStart),
				baseTime.Add(tc.eventEnd),
				1,
			)

			eventUID, err := repository.StoreEvent(ctx, userId, event)
			require.NoError(t, err)

			// When
			fetchedEvents, err := repository.GetEvents(
				ctx,
				userId,
				baseTime.Add(tc.queryStart),
				baseTime.Add(tc.queryEnd),
			)

			// Then
			require.NoError(t, err)

			if tc.shouldBeFound {
				require.Len(t, fetchedEvents, 1)

				// Check if the returned event matches the stored one
				expected := event
				expected.UID = uuid.NullUUID{UUID: eventUID, Valid: true}

				assertEventEqual(t, expected, fetchedEvents[0], false)
			} else {
				assert.Empty(t, fetchedEvents, "Expected no events to be returned")
			}
		})
	}
}

func TestRepositoryImpl_GetEventsMultipleEvents(t *testing.T) {
	// Setup
	repository, ctx, userId := setupRepositoryTest(t)

	// Given
	baseTime := time.Now().Truncate(time.Millisecond)
	events := []Event{
		createTestEvent("Event 1", baseTime, baseTime.Add(30*time.Minute), 1),
		createTestEvent("Event 2", baseTime.Add(time.Hour), baseTime.Add(90*time.Minute), 2),
		createTestEvent("Event 3", baseTime.Add(-time.Hour), baseTime.Add(-30*time.Minute), 3), // Before query
		createTestEvent("Event 4", baseTime.Add(2*time.Hour), baseTime.Add(3*time.Hour), 4),    // After query
	}

	// Store all events
	for i, event := range events {
		uid, err := repository.StoreEvent(ctx, userId, event)
		require.NoError(t, err)

		// Update the UID in our reference events
		events[i].UID = uuid.NullUUID{UUID: uid, Valid: true}
	}

	// When - query period contains only events 1 and 2
	fetchStart := baseTime
	fetchEnd := baseTime.Add(90 * time.Minute)
	fetchedEvents, err := repository.GetEvents(ctx, userId, fetchStart, fetchEnd)

	// Then
	require.NoError(t, err)
	assert.Len(t, fetchedEvents, 2)

	// Check if events match
	summaries := make([]string, len(fetchedEvents))
	for i, event := range fetchedEvents {
		summaries[i] = event.Summary
	}

	assert.Contains(t, summaries, "Event 1")
	assert.Contains(t, summaries, "Event 2")
	assert.NotContains(t, summaries, "Event 3")
	assert.NotContains(t, summaries, "Event 4")
}

func TestRepositoryImpl_GetLastEvents(t *testing.T) {
	// Setup
	repository, ctx, userId := setupRepositoryTest(t)

	// Given - Create events with different end times
	now := time.Now().Truncate(time.Millisecond)

	// Define events with end times in the past
	pastEvents := []Event{
		createTestEvent("Event 1", now.Add(-3*time.Hour), now.Add(-2*time.Hour), 1),   // Ended 2 hours ago
		createTestEvent("Event 2", now.Add(-5*time.Hour), now.Add(-4*time.Hour), 2),   // Ended 4 hours ago
		createTestEvent("Event 3", now.Add(-7*time.Hour), now.Add(-6*time.Hour), 3),   // Ended 6 hours ago
		createTestEvent("Event 4", now.Add(-9*time.Hour), now.Add(-8*time.Hour), 4),   // Ended 8 hours ago
		createTestEvent("Event 5", now.Add(-11*time.Hour), now.Add(-10*time.Hour), 5), // Ended 10 hours ago
	}

	// Also define a future event and an ongoing event that shouldn't be included
	futureEvents := []Event{
		createTestEvent("Future Event", now.Add(1*time.Hour), now.Add(2*time.Hour), 6),   // Starts in the future
		createTestEvent("Ongoing Event", now.Add(-1*time.Hour), now.Add(1*time.Hour), 7), // Ongoing (hasn't ended)
	}

	// Store all events
	var allEvents []Event
	allEvents = append(allEvents, pastEvents...)
	allEvents = append(allEvents, futureEvents...)

	for i, event := range allEvents {
		uid, err := repository.StoreEvent(ctx, userId, event)
		require.NoError(t, err)

		// Update the UID in our reference events
		allEvents[i].UID = uuid.NullUUID{UUID: uid, Valid: true}
	}

	// Test cases with different limits
	testCases := []struct {
		name          string
		limit         int
		expectedCount int
		expectedFirst string // Summary of the first (most recent) event
	}{
		{
			name:          "Get last 3 events",
			limit:         3,
			expectedCount: 3,
			expectedFirst: "Event 1", // Most recent past event
		},
		{
			name:          "Get all past events",
			limit:         10,
			expectedCount: 5, // Only 5 past events exist
			expectedFirst: "Event 1",
		},
		{
			name:          "Get single event",
			limit:         1,
			expectedCount: 1,
			expectedFirst: "Event 1",
		},
		{
			name:          "Zero limit returns empty list",
			limit:         0,
			expectedCount: 0,
			expectedFirst: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// When
			lastEvents, err := repository.GetLastEvents(ctx, userId, tc.limit)

			// Then
			require.NoError(t, err)
			assert.Len(t, lastEvents, tc.expectedCount)

			if tc.expectedCount > 0 {
				// Verify that events are returned in the correct order (most recent first)
				assert.Equal(t, tc.expectedFirst, lastEvents[0].Summary)

				// Check that all returned events have end times in the past
				for _, event := range lastEvents {
					assert.True(t, event.EndTime.Before(now),
						"Expected event to have ended in the past: %s at %v",
						event.Summary, event.EndTime)
				}

				// Verify correct ordering by end time (descending)
				for i := 0; i < len(lastEvents)-1; i++ {
					assert.True(t, !lastEvents[i].EndTime.Before(lastEvents[i+1].EndTime),
						"Events not properly ordered by end time: %v should be after %v",
						lastEvents[i].EndTime, lastEvents[i+1].EndTime)
				}

				// Verify no future or ongoing events are included
				for _, event := range lastEvents {
					assert.NotEqual(t, "Future Event", event.Summary)
					assert.NotEqual(t, "Ongoing Event", event.Summary)
				}
			}
		})
	}
}

func TestRepositoryImpl_UpdateEvent(t *testing.T) {
	// Setup
	repository, ctx, userId := setupRepositoryTest(t)

	// Given - Create and store an initial event
	baseTime := time.Now().Truncate(time.Millisecond)
	initialEvent := createTestEvent(
		"Initial Summary",
		baseTime,
		baseTime.Add(time.Hour),
		123,
	)

	// Store the event
	eventUID, err := repository.StoreEvent(ctx, userId, initialEvent)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, eventUID)

	// Verify the event was stored correctly
	storedEvents, err := repository.GetEvents(ctx, userId, baseTime, baseTime.Add(time.Hour))
	require.NoError(t, err)
	require.Len(t, storedEvents, 1)

	// Prepare the update - modify all fields
	updatedEvent := Event{
		Summary:   "Updated Summary",
		StartTime: baseTime.Add(15 * time.Minute),
		EndTime:   baseTime.Add(45 * time.Minute),
		Metadata: EventMetadata{
			BudgetId: 456,
		},
		UID: uuid.NullUUID{UUID: eventUID, Valid: true},
	}

	// When - Update the event
	err = repository.UpdateEvent(ctx, userId, updatedEvent)

	// Then
	require.NoError(t, err)

	// Verify the event was updated correctly
	updatedEvents, err := repository.GetEvents(ctx, userId, baseTime, baseTime.Add(time.Hour))
	require.NoError(t, err)
	require.Len(t, updatedEvents, 1)

	// Check that all fields were updated correctly
	assertEventEqual(t, updatedEvent, updatedEvents[0], false)
}

func TestRepositoryImpl_DeleteEvent(t *testing.T) {
	// Setup
	repository, ctx, userId := setupRepositoryTest(t)

	// Given - Create and store an initial event
	baseTime := time.Now().Truncate(time.Millisecond)
	event1 := createTestEvent(
		"Event 1",
		baseTime,
		baseTime.Add(time.Hour),
		123,
	)
	event2 := createTestEvent(
		"Event 2",
		baseTime,
		baseTime.Add(time.Hour),
		123,
	)

	// Store the events
	allEvents := []Event{event1, event2}
	for i, event := range allEvents {
		eventUID, err := repository.StoreEvent(ctx, userId, event)
		require.NoError(t, err)

		// Update the UID in our reference events
		allEvents[i].UID = uuid.NullUUID{UUID: eventUID, Valid: true}
	}

	// When - Delete the event
	err := repository.DeleteEvent(ctx, userId, allEvents[1].UID.UUID)

	// Then
	require.NoError(t, err)

	// Verify the event was deleted correctly
	finalEvents, err := repository.GetEvents(ctx, userId, baseTime, baseTime.Add(time.Hour))
	require.NoError(t, err)
	assert.Len(t, finalEvents, 1)
	assert.Equal(t, allEvents[0].UID.UUID, finalEvents[0].UID.UUID)
}
