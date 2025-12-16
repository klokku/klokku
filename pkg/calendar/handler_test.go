package calendar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klokku/klokku/internal/event_bus"
	"github.com/klokku/klokku/pkg/user"
	"github.com/stretchr/testify/assert"
)

func contextWithUser(ctx context.Context, userId int) context.Context {
	return user.WithUser(ctx, user.User{
		Id:          userId,
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
}

// A middleware that sets the user in the context
func withUser(userId int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := contextWithUser(r.Context(), userId)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func setupHandlerTest(t *testing.T) (*Handler, func()) {
	repoStub := NewRepositoryStub()
	eventBus := event_bus.NewEventBus()
	service := NewService(repoStub, eventBus, weeklyItemsProvider)
	handler := NewHandler(service)
	return handler, func() {
		t.Log("Teardown after test")
		repoStub.Reset()
	}
}

// Helper to add test events
func addTestEvents(t *testing.T, handler *Handler, userId int, events []EventDTO) {
	for _, event := range events {
		body, err := json.Marshal(event)
		assert.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/event", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.CreateEvent(w, req.WithContext(contextWithUser(context.Background(), userId)))
		assert.Equal(t, http.StatusCreated, w.Code)
	}
}

func TestGetEvents_InvalidFromDate(t *testing.T) {
	// Setup
	handler, teardown := setupHandlerTest(t)
	defer teardown()

	// Create a request with invalid 'from' parameter
	req := httptest.NewRequest(http.MethodGet, "/event?from=invalid-date&to=2023-01-02T15:04:05Z", nil)
	w := httptest.NewRecorder()

	// Call the handler
	handler.GetEvents(w, req)

	// Verify response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResponse struct {
		Error   string `json:"error"`
		Details string `json:"details"`
	}
	err := json.NewDecoder(w.Body).Decode(&errResponse)
	assert.NoError(t, err)
	assert.Contains(t, errResponse.Error, "Invalid from (date) format")
	assert.Contains(t, errResponse.Details, "RFC3339")
}

func TestGetEvents_InvalidToDate(t *testing.T) {
	// Setup
	handler, teardown := setupHandlerTest(t)
	defer teardown()

	// Create a request with valid 'from' but invalid 'to' parameter
	req := httptest.NewRequest(http.MethodGet, "/event?from=2023-01-01T15:04:05Z&to=invalid-date", nil)
	w := httptest.NewRecorder()

	// Call the handler
	handler.GetEvents(w, req)

	// Verify response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResponse struct {
		Error   string `json:"error"`
		Details string `json:"details"`
	}
	err := json.NewDecoder(w.Body).Decode(&errResponse)
	assert.NoError(t, err)
	assert.Contains(t, errResponse.Error, "Invalid to (date) format")
	assert.Contains(t, errResponse.Details, "RFC3339")
}

func TestGetEvents_UserAuthError(t *testing.T) {
	// Setup
	handler, teardown := setupHandlerTest(t)
	defer teardown()

	// Create a request with valid date parameters but missing user id in context
	req := httptest.NewRequest(http.MethodGet, "/event?from=2023-01-01T00:00:00Z&to=2023-01-02T00:00:00Z", nil)
	w := httptest.NewRecorder()

	// Call the handler directly - no user ID in context
	handler.GetEvents(w, req)

	// Should fail with internal server error (user ID lookup fails)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetEvents_Success(t *testing.T) {
	// Setup
	handler, teardown := setupHandlerTest(t)
	defer teardown()

	userId := 123
	from := time.Date(2025, 1, 1, 4, 0, 0, 0, location)
	to := time.Date(2025, 1, 2, 3, 0, 0, 0, location)

	// Add sample events to the repository
	events := []EventDTO{
		{
			Summary:      "Test BudgetItem 1",
			StartTime:    from,
			EndTime:      from.Add(1 * time.Hour),
			BudgetItemId: 101,
		},
		{
			Summary:      "Test BudgetItem 2",
			StartTime:    to.Add(-2 * time.Hour),
			EndTime:      to,
			BudgetItemId: 102,
		},
		// Event outside the time range (should not be returned)
		{
			Summary:      "Test BudgetItem 3",
			StartTime:    to.Add(1 * time.Hour),
			EndTime:      to.Add(2 * time.Hour),
			BudgetItemId: 103,
		},
	}

	addTestEvents(t, handler, userId, events)

	// Create a request with valid parameters
	values := url.Values{}
	values.Set("from", from.Format(time.RFC3339))
	values.Set("to", to.Format(time.RFC3339))
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/event?%s", values.Encode()), nil)

	// Wrap handler with middleware to add user ID
	w := httptest.NewRecorder()
	userMiddleware := withUser(userId, http.HandlerFunc(handler.GetEvents))
	userMiddleware.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Parse response
	var dtos []EventDTO
	err := json.NewDecoder(w.Body).Decode(&dtos)
	assert.NoError(t, err)

	// Should only return events 1 and 2 (not 3, which is outside the range)
	assert.Len(t, dtos, 2)

	// Check that the events are properly transformed to DTOs
	// Use Unix timestamps to compare times, avoiding timezone issues
	foundEvent1 := false
	foundEvent2 := false

	for _, dto := range dtos {
		switch dto.Summary {
		case "Test BudgetItem 1":
			// Compare Unix timestamps instead of direct time comparison
			assert.Equal(t, from.Unix(), dto.StartTime.Unix(), "Start time for Test BudgetItem 1 should match")
			assert.Equal(t, from.Add(1*time.Hour).Unix(), dto.EndTime.Unix(), "End time for Event 1 should match")
			assert.Equal(t, 101, dto.BudgetItemId)
			foundEvent1 = true
		case "Test BudgetItem 2":
			assert.Equal(t, to.Add(-2*time.Hour).Unix(), dto.StartTime.Unix(), "Start time for Test BudgetItem 2 should match")
			assert.Equal(t, to.Unix(), dto.EndTime.Unix(), "End time for Event 2 should match")
			assert.Equal(t, 102, dto.BudgetItemId)
			foundEvent2 = true
		}
	}

	assert.True(t, foundEvent1, "Test BudgetItem 1 should be in the response")
	assert.True(t, foundEvent2, "Test BudgetItem 2 should be in the response")
}

func TestGetEvents_EmptyResults(t *testing.T) {
	// Setup
	handler, teardown := setupHandlerTest(t)
	defer teardown()

	userId := 123
	from := time.Date(2023, 1, 1, 0, 0, 0, 0, location)
	to := time.Date(2023, 1, 2, 0, 0, 0, 0, location)

	// Create a request with valid parameters
	values := url.Values{}
	values.Set("from", from.Format(time.RFC3339))
	values.Set("to", to.Format(time.RFC3339))
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/event?%s", values.Encode()), nil)

	// Wrap handler with middleware to add user ID
	w := httptest.NewRecorder()
	userMiddleware := withUser(userId, http.HandlerFunc(handler.GetEvents))
	userMiddleware.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Parse response - should be an empty array
	var dtos []EventDTO
	err := json.NewDecoder(w.Body).Decode(&dtos)
	assert.NoError(t, err)
	assert.Empty(t, dtos)
}

func TestEventToDTO(t *testing.T) {
	// Create a sample Event
	uid := uuid.NewString()
	event := Event{
		UID:       uid,
		Summary:   "Test Event",
		StartTime: time.Date(2023, 1, 1, 10, 0, 0, 0, location),
		EndTime:   time.Date(2023, 1, 1, 11, 0, 0, 0, location),
		Metadata: EventMetadata{
			BudgetItemId: 123,
		},
	}

	// Convert to DTO
	dto, err := eventToDTO(event)

	// Verify conversion
	assert.NoError(t, err)
	assert.Equal(t, uid, dto.UID)
	assert.Equal(t, "Test Event", dto.Summary)
	assert.Equal(t, time.Date(2023, 1, 1, 10, 0, 0, 0, location), dto.StartTime)
	assert.Equal(t, time.Date(2023, 1, 1, 11, 0, 0, 0, location), dto.EndTime)
	assert.Equal(t, 123, dto.BudgetItemId)
}

func TestUpdateEvent(t *testing.T) {
	// Setup
	handler, teardown := setupHandlerTest(t)
	defer teardown()
	userId := 123

	// 1. First create an event
	startTime := time.Date(2025, 1, 1, 10, 0, 0, 0, location)
	endTime := time.Date(2025, 1, 1, 11, 0, 0, 0, location)

	originalEvent := EventDTO{
		Summary:      "Original Event",
		StartTime:    startTime,
		EndTime:      endTime,
		BudgetItemId: 101,
	}

	// Create the event
	body, err := json.Marshal(originalEvent)
	assert.NoError(t, err)

	createReq := httptest.NewRequest(http.MethodPost, "/event", bytes.NewBuffer(body))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()

	// Add user to context and call the handler
	createCtx := contextWithUser(createReq.Context(), userId)
	handler.CreateEvent(createW, createReq.WithContext(createCtx))

	// Verify event was created
	assert.Equal(t, http.StatusCreated, createW.Code)

	// Parse the created event to get its UID
	var createdEvents []EventDTO
	err = json.NewDecoder(createW.Body).Decode(&createdEvents)
	assert.NoError(t, err)
	assert.NotEmpty(t, createdEvents, "Should have created at least one event")
	assert.NotEmpty(t, createdEvents, "Created event should have a UID")

	// 2. Now update the event
	updatedSummary := "Test BudgetItem 2"
	updatedStartTime := time.Date(2025, 1, 1, 14, 0, 0, 0, location) // Changed from 10:00 to 14:00
	updatedEndTime := time.Date(2025, 1, 1, 16, 0, 0, 0, location)   // Changed from 11:00 to 16:00
	updatedBudgetItemId := 102                                       // Changed from 1 to 2

	updatedEvent := EventDTO{
		UID:          createdEvents[0].UID, // Keep the same UID
		Summary:      updatedSummary,
		StartTime:    updatedStartTime,
		EndTime:      updatedEndTime,
		BudgetItemId: updatedBudgetItemId,
	}

	// Create the update request
	updateBody, err := json.Marshal(updatedEvent)
	assert.NoError(t, err)

	updateReq := httptest.NewRequest(http.MethodPut, "/event", bytes.NewBuffer(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()

	// Add user ID to context and call the handler
	updateCtx := contextWithUser(updateReq.Context(), userId)
	handler.UpdateEvent(updateW, updateReq.WithContext(updateCtx))

	// Verify update was successful
	assert.Equal(t, http.StatusOK, updateW.Code)

	// Parse the updated event
	var returnedUpdatedEvents []EventDTO
	err = json.NewDecoder(updateW.Body).Decode(&returnedUpdatedEvents)
	assert.NoError(t, err)

	// Verify the updated properties
	assert.Equal(t, createdEvents[0].UID, returnedUpdatedEvents[0].UID, "UID should remain the same")
	assert.Equal(t, updatedSummary, returnedUpdatedEvents[0].Summary, "Summary should be updated")
	assert.Equal(t, updatedStartTime.Unix(), returnedUpdatedEvents[0].StartTime.Unix(), "StartTime should be updated")
	assert.Equal(t, updatedEndTime.Unix(), returnedUpdatedEvents[0].EndTime.Unix(), "EndTime should be updated")
	assert.Equal(t, updatedBudgetItemId, returnedUpdatedEvents[0].BudgetItemId, "BudgetItemId should be updated")

	// 3. Verify the update persisted by getting the event
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, location)
	to := time.Date(2025, 1, 2, 0, 0, 0, 0, location)

	// Create a request to get events in this time range
	values := url.Values{}
	values.Set("from", from.Format(time.RFC3339))
	values.Set("to", to.Format(time.RFC3339))
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/event?%s", values.Encode()), nil)
	getW := httptest.NewRecorder()

	// Add user ID to context and call the handler
	getCtx := contextWithUser(getReq.Context(), userId)
	handler.GetEvents(getW, getReq.WithContext(getCtx))

	// Verify response
	assert.Equal(t, http.StatusOK, getW.Code)

	// Parse response
	var events []EventDTO
	err = json.NewDecoder(getW.Body).Decode(&events)
	assert.NoError(t, err)

	// Should find our updated event
	found := false
	for _, event := range events {
		if event.UID == createdEvents[0].UID {
			found = true
			// Verify it has the updated properties
			assert.Equal(t, updatedSummary, event.Summary)
			assert.Equal(t, updatedStartTime.Unix(), event.StartTime.Unix())
			assert.Equal(t, updatedEndTime.Unix(), event.EndTime.Unix())
			assert.Equal(t, updatedBudgetItemId, event.BudgetItemId)
			break
		}
	}

	assert.True(t, found, "Updated event should be returned when querying events")
}

func TestDeleteEvent(t *testing.T) {
	// Setup
	handler, teardown := setupHandlerTest(t)
	defer teardown()
	userId := 123

	// 1. First create an event
	startTime := time.Date(2025, 1, 1, 10, 0, 0, 0, location)
	endTime := time.Date(2025, 1, 1, 11, 0, 0, 0, location)

	originalEvent := EventDTO{
		Summary:      "Original Event",
		StartTime:    startTime,
		EndTime:      endTime,
		BudgetItemId: 101,
	}

	// Create the event
	body, err := json.Marshal(originalEvent)
	assert.NoError(t, err)

	createReq := httptest.NewRequest(http.MethodPost, "/event", bytes.NewBuffer(body))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()

	// Add user ID to context and call the handler
	createCtx := contextWithUser(createReq.Context(), userId)
	handler.CreateEvent(createW, createReq.WithContext(createCtx))

	// Verify event was created
	assert.Equal(t, http.StatusCreated, createW.Code)

	// Parse the created event to get its UID
	var createdEvents []EventDTO
	err = json.NewDecoder(createW.Body).Decode(&createdEvents)
	assert.NoError(t, err)
	assert.NotEmpty(t, createdEvents, "Should have created at least one event")

	// 2. Now delete the event
	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/event/%s", createdEvents[0].UID), nil)
	deleteW := httptest.NewRecorder()
	deleteReq.Header.Set("Content-Type", "application/json")

	// Set up the route parameters - this is what would normally be done by the router
	// We need to manually add the eventUid parameter since we're calling the handler directly
	req := mux.SetURLVars(deleteReq, map[string]string{
		"eventUid": createdEvents[0].UID,
	})

	// Add user ID to context and call the handler
	updateCtx := contextWithUser(req.Context(), userId)
	handler.DeleteEvent(deleteW, req.WithContext(updateCtx))
	assert.Equal(t, http.StatusNoContent, deleteW.Code)

	// 3. Verify deleting by getting the event
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, location)
	to := time.Date(2025, 1, 2, 0, 0, 0, 0, location)
	values := url.Values{}
	values.Set("from", from.Format(time.RFC3339))
	values.Set("to", to.Format(time.RFC3339))
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/event?%s", values.Encode()), nil)
	getW := httptest.NewRecorder()
	getReq.Header.Set("Content-Type", "application/json")
	getCtx := contextWithUser(getReq.Context(), userId)
	handler.GetEvents(getW, getReq.WithContext(getCtx))
	assert.Equal(t, http.StatusOK, getW.Code)
	var events []EventDTO
	err = json.NewDecoder(getW.Body).Decode(&events)
	assert.NoError(t, err)
	assert.Empty(t, events, "Event should not be returned when querying events")
}
