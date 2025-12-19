package stats

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klokku/klokku/internal/utils"
	"github.com/klokku/klokku/pkg/calendar"
	"github.com/klokku/klokku/pkg/current_event"
	"github.com/klokku/klokku/pkg/user"
	"github.com/klokku/klokku/pkg/weekly_plan"
	"github.com/stretchr/testify/assert"
)

var location, _ = time.LoadLocation("Europe/Warsaw")
var calendarStub = calendar.NewStubCalendar()
var clock = &utils.MockClock{FixedNow: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)}
var weeklyPlanService = newWeeklyPlanItemsReaderStub()
var currentEventStub = newCurrentEventProviderStub()

func setup(t *testing.T) (StatsService, context.Context, func()) {
	service := &StatsServiceImpl{
		currentEventProvider: currentEventStub,
		weeklyPlanService:    weeklyPlanService,
		calendar:             calendarStub,
		clock:                clock,
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
		weeklyPlanService.reset()
		currentEventStub.reset()
		calendarStub.Cleanup()
	}
}

func TestStatsServiceImpl_GetStats(t *testing.T) {
	statsService, ctx, teardown := setup(t)
	defer teardown()

	// given
	startTime := time.Date(2023, time.January, 2, 0, 0, 0, 0, location)
	endTime := time.Date(2023, time.January, 9, 0, 0, 0, 0, location).Add(-1 * time.Nanosecond)
	planItem1 := weekly_plan.WeeklyPlanItem{
		Id:             101,
		BudgetItemId:   1,
		Name:           "BudgetItem 1",
		WeeklyDuration: time.Duration(120) * time.Minute,
	}
	planItem2 := weekly_plan.WeeklyPlanItem{
		Id:             102,
		BudgetItemId:   2,
		Name:           "BudgetItem 2",
		WeeklyDuration: time.Duration(120) * time.Minute,
	}
	weeklyPlanService.setItems([]weekly_plan.WeeklyPlanItem{planItem1, planItem2})
	calendarStub.AddEvent(ctx, calendar.Event{ // 60 minutes
		Summary:   "BudgetItem 1",
		StartTime: startTime,
		EndTime:   startTime.Add(time.Hour),
		Metadata:  calendar.EventMetadata{BudgetItemId: planItem1.BudgetItemId},
	})
	calendarStub.AddEvent(ctx, calendar.Event{ // 30 minutes
		Summary:   "BudgetItem 2",
		StartTime: startTime.Add(time.Hour),
		EndTime:   startTime.Add(time.Hour).Add(30 * time.Minute),
		Metadata:  calendar.EventMetadata{BudgetItemId: planItem2.BudgetItemId},
	})
	calendarStub.AddEvent(ctx, calendar.Event{ // 90 minutes
		Summary:   "BudgetItem 1",
		StartTime: startTime.Add(90 * time.Minute),
		EndTime:   startTime.Add(90 * time.Minute).Add(90 * time.Minute),
		Metadata:  calendar.EventMetadata{BudgetItemId: planItem1.BudgetItemId},
	})
	secondDay := startTime.Add(24 * time.Hour)
	calendarStub.AddEvent(ctx, calendar.Event{ // 75 minutes
		Summary:   "BudgetItem 2",
		StartTime: secondDay.Add(2 * time.Hour),
		EndTime:   secondDay.Add(2 * time.Hour).Add(75 * time.Minute),
		Metadata:  calendar.EventMetadata{BudgetItemId: planItem2.BudgetItemId},
	})

	// when
	stats, _ := statsService.GetStats(ctx, startTime)

	// then
	assert.NotNil(t, stats)
	assert.Equal(t, startTime, stats.StartDate)
	assert.Equal(t, endTime, stats.EndDate)
	assert.Equal(t, 7, len(stats.PerDay))
	assert.Equal(t, 2, len(stats.PerPlanItem))
	assert.Equal(t, time.Duration(255)*time.Minute, stats.TotalTime)

	// Check on Day 1
	foundBudget1 := findBudgetByName(stats.PerDay[0].StatsPerPlanItem, "BudgetItem 1")
	assert.Equal(t, time.Duration(150)*time.Minute, foundBudget1.Duration)
	foundBudget2 := findBudgetByName(stats.PerDay[0].StatsPerPlanItem, "BudgetItem 2")
	assert.Equal(t, time.Duration(30)*time.Minute, foundBudget2.Duration)
	// Check on Day 2
	foundBudget2 = findBudgetByName(stats.PerDay[1].StatsPerPlanItem, "BudgetItem 2")
	assert.Equal(t, time.Duration(75)*time.Minute, foundBudget2.Duration)

	// Check by PerPlanItem
	b1 := findBudgetByName(stats.PerPlanItem, "BudgetItem 1")
	assert.Equal(t, time.Duration(150)*time.Minute, b1.Duration)
	b2 := findBudgetByName(stats.PerPlanItem, "BudgetItem 2")
	assert.Equal(t, time.Duration(105)*time.Minute, b2.Duration)
}

func TestStatsServiceImpl_GetStats_WithCurrentEvent(t *testing.T) {
	statsService, ctx, teardown := setup(t)
	defer teardown()

	// given
	startTime := time.Date(2023, time.January, 2, 0, 0, 0, 0, location)
	planItem1 := weekly_plan.WeeklyPlanItem{Id: 101, BudgetItemId: 1, Name: "BudgetItem 1", WeeklyDuration: time.Duration(120) * time.Minute}
	weeklyPlanService.setItems([]weekly_plan.WeeklyPlanItem{planItem1})

	calendarStub.AddEvent(ctx, calendar.Event{ // 60 minutes
		Summary:   "BudgetItem 1",
		StartTime: startTime,
		EndTime:   startTime.Add(time.Hour),
		Metadata:  calendar.EventMetadata{BudgetItemId: planItem1.BudgetItemId},
	})
	clock.SetNow(startTime.Add(90 * time.Minute))
	currentEventStub.set(&current_event.CurrentEvent{ // started 30 minutes ago
		PlanItem: current_event.PlanItem{
			BudgetItemId:   planItem1.BudgetItemId,
			Name:           "BudgetItem 1",
			WeeklyDuration: time.Duration(120) * time.Minute,
		},
		StartTime: startTime.Add(time.Hour),
	})

	// when
	stats, _ := statsService.GetStats(ctx, startTime)

	// then
	// check on a budget list
	budget1 := findBudgetByName(stats.PerPlanItem, "BudgetItem 1")
	assert.Equal(t, time.Duration(90)*time.Minute, budget1.Duration)
	assert.Equal(t, time.Duration(30)*time.Minute, budget1.Remaining)
	// check in day data
	budget1 = findBudgetByName(stats.PerDay[0].StatsPerPlanItem, "BudgetItem 1")
	assert.Equal(t, time.Duration(90)*time.Minute, budget1.Duration)
	assert.Equal(t, time.Duration(90)*time.Minute, stats.PerDay[0].TotalTime)
	// check summary
	assert.Equal(t, time.Duration(90)*time.Minute, stats.TotalTime)
	assert.Equal(t, time.Duration(30)*time.Minute, stats.TotalRemaining)
}

func findBudgetByName(budgets []PlanItemStats, budgetName string) *PlanItemStats {
	for _, b := range budgets {
		if b.PlanItem.Name == budgetName {
			return &b
		}
	}
	return nil
}

func Test_weekTimeRange(t *testing.T) {
	type args struct {
		date         time.Time
		weekStartDay time.Weekday
	}
	tests := []struct {
		name  string
		args  args
		want  time.Time
		want1 time.Time
	}{
		{
			name:  "should return full week range when start day is Monday",
			args:  args{date: time.Date(2023, 10, 16, 0, 0, 0, 0, time.UTC), weekStartDay: time.Monday},
			want:  time.Date(2023, 10, 16, 0, 0, 0, 0, time.UTC),
			want1: time.Date(2023, 10, 22, 23, 59, 59, 999999999, time.UTC),
		},
		{
			name:  "should return full week range when start day is Sunday",
			args:  args{date: time.Date(2023, 10, 16, 0, 0, 0, 0, time.UTC), weekStartDay: time.Sunday},
			want:  time.Date(2023, 10, 15, 0, 0, 0, 0, time.UTC),
			want1: time.Date(2023, 10, 21, 23, 59, 59, 999999999, time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := weekTimeRange(tt.args.date, tt.args.weekStartDay)
			assert.Equalf(t, tt.want, got, "weekTimeRange(%v, %v)", tt.args.date, tt.args.weekStartDay)
			assert.Equalf(t, tt.want1, got1, "weekTimeRange(%v, %v)", tt.args.date, tt.args.weekStartDay)
		})
	}
}
