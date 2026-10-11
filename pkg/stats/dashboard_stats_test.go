package stats

import (
	"context"
	"testing"
	"time"

	"github.com/klokku/klokku/internal/utils"
	"github.com/klokku/klokku/pkg/budget_plan"
	"github.com/klokku/klokku/pkg/calendar"
	"github.com/klokku/klokku/pkg/user"
	"github.com/klokku/klokku/pkg/weekly_plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatsServiceImpl_GetWeeklyStats_DaylightSavingBoundaries(t *testing.T) {
	for _, test := range []struct {
		name     string
		timezone string
		month    time.Month
		day      int
		offset   int
	}{
		{"Warsaw spring", "Europe/Warsaw", time.March, 23, 3600},
		{"Warsaw autumn", "Europe/Warsaw", time.October, 19, 7200},
		{"New York spring", "America/New_York", time.March, 2, -18000},
		{"New York autumn", "America/New_York", time.October, 26, -14400},
	} {
		t.Run(test.name, func(t *testing.T) {
			loc, err := time.LoadLocation(test.timezone)
			require.NoError(t, err)
			start := time.Date(2026, test.month, test.day, 0, 0, 0, 0, loc)
			nextWeek := start.AddDate(0, 0, 7)
			ctx := user.WithUser(context.Background(), user.User{Id: 1, Settings: user.Settings{
				Timezone: test.timezone, WeekFirstDay: time.Monday,
			}})
			plans := newWeeklyPlanItemsReaderStub()
			plans.setItems([]weekly_plan.WeeklyPlanItem{{BudgetPlanId: 1, BudgetItemId: 1, WeeklyDuration: time.Hour}})
			budgets := newBudgetPlanReaderStub()
			budgets.addPlan(budget_plan.BudgetPlan{Id: 1, Items: []budget_plan.BudgetItem{{Id: 1, PlanId: 1, WeeklyDuration: time.Hour}}})
			events := calendar.NewStubCalendar()
			_, err = events.AddEvent(ctx, calendar.Event{
				StartTime: nextWeek.Add(-45 * time.Minute), EndTime: nextWeek.Add(-15 * time.Minute),
				Metadata: calendar.EventMetadata{BudgetItemId: 1},
			})
			require.NoError(t, err)
			_, err = events.AddEvent(ctx, calendar.Event{
				StartTime: nextWeek.Add(15 * time.Minute), EndTime: nextWeek.Add(45 * time.Minute),
				Metadata: calendar.EventMetadata{BudgetItemId: 1},
			})
			require.NoError(t, err)
			svc := &StatsServiceImpl{
				weeklyPlanService: plans, budgetPlanService: budgets, calendar: events,
				currentEventProvider: newCurrentEventProviderStub(),
				clock:                &utils.MockClock{FixedNow: time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC)},
			}
			// RFC3339 offsets do not preserve an IANA location's transition rules.
			requestDate := time.Date(2026, test.month, test.day, 0, 0, 0, 0, time.FixedZone("request", test.offset))
			stats, err := svc.GetWeeklyStats(ctx, requestDate)
			require.NoError(t, err)
			assert.Equal(t, start.UTC(), stats.StartDate.UTC())
			assert.Equal(t, nextWeek.Add(-time.Nanosecond).UTC(), stats.EndDate.UTC())
			assert.Equal(t, 30*time.Minute, stats.TotalTime)
			require.Len(t, stats.PerDay, 7)
			assert.Equal(t, 30*time.Minute, stats.PerDay[6].TotalTime)
			assert.Equal(t, start.AddDate(0, 0, 6).UTC(), stats.PerDay[6].Date.UTC())
		})
	}
}
