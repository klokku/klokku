package budget_plan_report

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/klokku/klokku/pkg/budget_plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_GetReport_FirstTrackedDateOutsideRequestedPeriod(t *testing.T) {
	for _, test := range []struct {
		name  string
		found bool
	}{
		{"established but inactive plan", true},
		{"plan never tracked", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			firstTracked := time.Date(2025, time.January, 6, 12, 0, 0, 0, time.UTC)
			from := time.Date(2026, time.September, 7, 0, 0, 0, 0, warsawTz)
			to := from.AddDate(0, 0, 28).Add(-time.Second)
			svc := NewService(
				&budgetPlanReaderStub{plans: map[int]budget_plan.BudgetPlan{1: testBudgetPlan()}},
				&calendarEventsReaderStub{},
				&earliestEventFinderStub{earliest: firstTracked, found: test.found},
				&weeklyPlanItemsReaderStub{}, mockClock(to),
			)
			query := url.Values{"from": {from.Format(time.RFC3339)}, "to": {to.Format(time.RFC3339)}}
			request := httptest.NewRequest(http.MethodGet, "/api/budgetplan/1/report?"+query.Encode(), nil).WithContext(testContext())
			request = mux.SetURLVars(request, map[string]string{"planId": "1"})
			response := httptest.NewRecorder()
			NewHandler(svc).GetReport(response, request)
			require.Equal(t, http.StatusOK, response.Code)
			var body struct {
				FirstTrackedAt *time.Time `json:"firstTrackedAt"`
				WeekCount      int        `json:"weekCount"`
			}
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
			if test.found {
				require.NotNil(t, body.FirstTrackedAt)
				assert.True(t, firstTracked.Equal(*body.FirstTrackedAt))
				assert.Equal(t, 4, body.WeekCount)
			} else {
				assert.Nil(t, body.FirstTrackedAt)
				assert.Equal(t, 0, body.WeekCount)
			}
		})
	}
}
