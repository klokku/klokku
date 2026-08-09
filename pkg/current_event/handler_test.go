package current_event

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klokku/klokku/internal/utils"
	"github.com/klokku/klokku/pkg/calendar"
	"github.com/klokku/klokku/pkg/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func currentEventHandlerTestContext() context.Context {
	return user.WithUser(context.Background(), user.User{
		Id:       1,
		Uid:      uuid.NewString(),
		Username: "handler-test",
		Settings: user.Settings{Timezone: "UTC"},
	})
}

func TestEventHandler_ModifyCurrentEventNotes(t *testing.T) {
	note := "Existing note"
	tests := []struct {
		name         string
		body         string
		seedEvent    bool
		wantStatus   int
		wantResponse *string
	}{
		{name: "sets notes", body: `{"notes":"Updated note"}`, seedEvent: true, wantStatus: http.StatusOK, wantResponse: stringPointer("Updated note")},
		{name: "clears notes", body: `{"notes":""}`, seedEvent: true, wantStatus: http.StatusOK, wantResponse: stringPointer("")},
		{name: "rejects missing notes field", body: `{}`, seedEvent: true, wantStatus: http.StatusBadRequest},
		{name: "rejects malformed body", body: `{`, seedEvent: true, wantStatus: http.StatusBadRequest},
		{name: "rejects notes over maximum length", body: `{"notes":"` + strings.Repeat("a", calendar.MaxNotesLength+1) + `"}`, seedEvent: true, wantStatus: http.StatusBadRequest},
		{name: "returns not found without current event", body: `{"notes":"Updated note"}`, wantStatus: http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := newStubEventRepository()
			service := &EventServiceImpl{
				repo:     repo,
				calendar: calendar.NewStubCalendar(),
				clock:    &utils.MockClock{FixedNow: time.Now()},
			}
			ctx := currentEventHandlerTestContext()
			if test.seedEvent {
				_, err := repo.ReplaceCurrentEvent(ctx, 1, CurrentEvent{
					StartTime: time.Now().Add(-time.Hour),
					Notes:     note,
					PlanItem:  PlanItem{BudgetItemId: 1, Name: "Test", WeeklyDuration: time.Hour},
				})
				require.NoError(t, err)
			}

			handler := NewEventHandler(service)
			req := httptest.NewRequest(http.MethodPatch, "/api/event/current/notes", bytes.NewBufferString(test.body))
			w := httptest.NewRecorder()
			handler.ModifyCurrentEventNotes(w, req.WithContext(ctx))

			assert.Equal(t, test.wantStatus, w.Code)
			if test.wantResponse != nil {
				var response CurrentEventDTO
				require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
				assert.Equal(t, *test.wantResponse, response.Notes)
			}
		})
	}
}

func stringPointer(value string) *string {
	return &value
}
