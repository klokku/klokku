package stats

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klokku/klokku/internal/utils"
	"github.com/klokku/klokku/pkg/budget_plan"
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
var budgetPlanService = newBudgetPlanReaderStub()
var currentEventStub = newCurrentEventProviderStub()

func setup(t *testing.T) (StatsService, context.Context, func()) {
	service := &StatsServiceImpl{
		currentEventProvider: currentEventStub,
		weeklyPlanService:    weeklyPlanService,
		budgetPlanService:    budgetPlanService,
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
		budgetPlanService.reset()
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
		BudgetPlanId:   1,
		Id:             101,
		BudgetItemId:   1,
		Name:           "BudgetItem 1",
		WeeklyDuration: time.Duration(120) * time.Minute,
	}
	planItem2 := weekly_plan.WeeklyPlanItem{
		BudgetPlanId:   1,
		Id:             102,
		BudgetItemId:   2,
		Name:           "BudgetItem 2",
		WeeklyDuration: time.Duration(120) * time.Minute,
	}
	weeklyPlanService.setItems([]weekly_plan.WeeklyPlanItem{planItem1, planItem2})
	budgetPlan := budget_plan.BudgetPlan{
		Id: 1,
		Items: []budget_plan.BudgetItem{
			{
				Id:             1,
				PlanId:         1,
				Name:           "BudgetItem 1",
				WeeklyDuration: time.Duration(120) * time.Minute,
			},
			{
				Id:             2,
				PlanId:         1,
				Name:           "BudgetItem 2",
				WeeklyDuration: time.Duration(120) * time.Minute,
			},
		},
	}
	budgetPlanService.addPlan(budgetPlan)
	calendarStub.AddEvent(ctx, calendar.Event{ // 60 minutes
		Summary:   "BudgetItem 1",
		StartTime: startTime.UTC(),
		EndTime:   startTime.Add(time.Hour).UTC(),
		Metadata:  calendar.EventMetadata{BudgetItemId: planItem1.BudgetItemId},
	})
	calendarStub.AddEvent(ctx, calendar.Event{ // 30 minutes
		Summary:   "BudgetItem 2",
		StartTime: startTime.Add(time.Hour).UTC(),
		EndTime:   startTime.Add(time.Hour).Add(30 * time.Minute).UTC(),
		Metadata:  calendar.EventMetadata{BudgetItemId: planItem2.BudgetItemId},
	})
	calendarStub.AddEvent(ctx, calendar.Event{ // 90 minutes
		Summary:   "BudgetItem 1",
		StartTime: startTime.Add(90 * time.Minute).UTC(),
		EndTime:   startTime.Add(90 * time.Minute).Add(90 * time.Minute).UTC(),
		Metadata:  calendar.EventMetadata{BudgetItemId: planItem1.BudgetItemId},
	})
	secondDay := startTime.Add(24 * time.Hour)
	calendarStub.AddEvent(ctx, calendar.Event{ // 75 minutes
		Summary:   "BudgetItem 2",
		StartTime: secondDay.Add(2 * time.Hour).UTC(),
		EndTime:   secondDay.Add(2 * time.Hour).Add(75 * time.Minute).UTC(),
		Metadata:  calendar.EventMetadata{BudgetItemId: planItem2.BudgetItemId},
	})

	// when
	stats, _ := statsService.GetWeeklyStats(ctx, startTime)

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
	planItem1 := weekly_plan.WeeklyPlanItem{
		Id:             101,
		BudgetPlanId:   1,
		BudgetItemId:   1,
		Name:           "BudgetItem 1",
		WeeklyDuration: time.Duration(120) * time.Minute,
	}
	weeklyPlanService.setItems([]weekly_plan.WeeklyPlanItem{planItem1})
	budgetPlan := budget_plan.BudgetPlan{
		Id:        1,
		Name:      "Some Plan",
		IsCurrent: true,
		Items: []budget_plan.BudgetItem{
			{
				Id:             1,
				PlanId:         1,
				Name:           "BudgetItem 1",
				WeeklyDuration: time.Duration(120) * time.Minute,
			},
		},
	}
	budgetPlanService.addPlan(budgetPlan)

	calendarStub.AddEvent(ctx, calendar.Event{ // 60 minutes
		Summary:   "BudgetItem 1",
		StartTime: startTime,
		EndTime:   startTime.Add(time.Hour),
		Metadata:  calendar.EventMetadata{BudgetItemId: planItem1.BudgetItemId},
	})
	clock.SetNow(startTime.Add(90 * time.Minute))     // 01:30
	currentEventStub.set(&current_event.CurrentEvent{ // started 30 minutes ago
		Id: 7897,
		PlanItem: current_event.PlanItem{
			BudgetItemId:   planItem1.BudgetItemId,
			Name:           "BudgetItem 1",
			WeeklyDuration: time.Duration(120) * time.Minute,
		},
		StartTime: startTime.Add(time.Hour), // 01:00
	})

	// when
	stats, _ := statsService.GetWeeklyStats(ctx, startTime)

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
		{
			name:  "should return fill week range when given date is the last nanosecond of the week",
			args:  args{date: time.Date(2025, 9, 14, 23, 59, 59, 999999999, time.UTC), weekStartDay: time.Monday},
			want:  time.Date(2025, 9, 8, 0, 0, 0, 0, time.UTC),
			want1: time.Date(2025, 9, 14, 23, 59, 59, 999999999, time.UTC),
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

func Test_GetPlanItemHistoryStats(t *testing.T) {
	t.Run("should calculate history stats correctly", func(t *testing.T) {
		statsService, ctx, teardown := setup(t)
		defer teardown()

		// given
		planItem1 := weekly_plan.WeeklyPlanItem{
			Id:             101,
			BudgetPlanId:   1,
			BudgetItemId:   154,
			Name:           "BudgetItem 1",
			WeeklyDuration: time.Duration(300) * time.Minute, // 5 hours
		}
		weeklyPlanService.setItems([]weekly_plan.WeeklyPlanItem{planItem1})
		budgetPlan := budget_plan.BudgetPlan{
			Id:        1,
			Name:      "Some plan",
			IsCurrent: false,
			Items: []budget_plan.BudgetItem{
				{
					Id:             154,
					PlanId:         1,
					Name:           "BudgetItem 1",
					WeeklyDuration: time.Duration(300) * time.Minute,
				},
			},
		}
		budgetPlanService.addPlan(budgetPlan)

		addCalendarEvents(ctx,
			10,
			time.Date(2025, 9, 1, 0, 0, 0, 0, location),
			time.Date(2025, 9, 7, 23, 59, 59, 999999999, location),
			154,
			330*time.Minute, // 5.5 hours
		)
		addCalendarEvents(ctx,
			7,
			time.Date(2025, 9, 8, 0, 0, 0, 0, location),
			time.Date(2025, 9, 14, 23, 59, 59, 999999999, location),
			154,
			210*time.Minute, // 3.5 hours
		)
		addCalendarEvents(ctx,
			11,
			time.Date(2025, 9, 15, 0, 0, 0, 0, location),
			time.Date(2025, 9, 21, 23, 59, 59, 999999999, location),
			154,
			360*time.Minute, // 6 hours
		)
		addCalendarEvents(ctx,
			8,
			time.Date(2025, 9, 22, 0, 0, 0, 0, location),
			time.Date(2025, 9, 28, 23, 59, 59, 999999999, location),
			154,
			390*time.Minute, // 6.5 hours
		)
		statsStartDate := time.Date(2025, 9, 1, 0, 0, 0, 0, location)
		statsEndDate := time.Date(2025, 9, 28, 23, 59, 59, 999999999, location)

		// when
		stats, _ := statsService.GetPlanItemByWeekHistoryStats(ctx, statsStartDate, statsEndDate, 154)

		// then
		assert.Equal(t, statsStartDate, stats.StartDate)
		assert.Equal(t, statsEndDate, stats.EndDate)
		assert.Equal(t, 4, len(stats.StatsPerWeek))
		assert.Equal(t, time.Date(2025, 9, 1, 0, 0, 0, 0, location), stats.StatsPerWeek[0].StartDate)
		assert.Equal(t, time.Date(2025, 9, 7, 23, 59, 59, 999999999, location), stats.StatsPerWeek[0].EndDate)
		assert.Equal(t, 330*time.Minute, stats.StatsPerWeek[0].Duration)
		assert.Equal(t, -30*time.Minute, stats.StatsPerWeek[0].Remaining)
		assert.Equal(t, time.Date(2025, 9, 8, 0, 0, 0, 0, location), stats.StatsPerWeek[1].StartDate)
		assert.Equal(t, time.Date(2025, 9, 14, 23, 59, 59, 999999999, location), stats.StatsPerWeek[1].EndDate)
		assert.Equal(t, 210*time.Minute, stats.StatsPerWeek[1].Duration)
		assert.Equal(t, 90*time.Minute, stats.StatsPerWeek[1].Remaining)
		assert.Equal(t, time.Date(2025, 9, 15, 0, 0, 0, 0, location), stats.StatsPerWeek[2].StartDate)
		assert.Equal(t, time.Date(2025, 9, 21, 23, 59, 59, 999999999, location), stats.StatsPerWeek[2].EndDate)
		assert.Equal(t, 360*time.Minute, stats.StatsPerWeek[2].Duration)
		assert.Equal(t, -60*time.Minute, stats.StatsPerWeek[2].Remaining)
		assert.Equal(t, time.Date(2025, 9, 22, 0, 0, 0, 0, location), stats.StatsPerWeek[3].StartDate)
		assert.Equal(t, time.Date(2025, 9, 28, 23, 59, 59, 999999999, location), stats.StatsPerWeek[3].EndDate)
		assert.Equal(t, 390*time.Minute, stats.StatsPerWeek[3].Duration)
		assert.Equal(t, -90*time.Minute, stats.StatsPerWeek[3].Remaining)

	})
}

func addCalendarEvents(ctx context.Context, numberOfEvents int, startTime time.Time, endTime time.Time, budgetItemId int, totalDuration time.Duration) {
	// Generate random durations that sum to totalDuration
	durations := make([]time.Duration, numberOfEvents)
	remaining := totalDuration
	minDuration := 5 * time.Minute

	for i := 0; i < numberOfEvents-1; i++ {
		// Calculate how much duration is needed for remaining events
		remainingEvents := numberOfEvents - i - 1
		minNeeded := minDuration * time.Duration(remainingEvents)

		// Available range for this event
		maxForThisEvent := remaining - minNeeded

		if maxForThisEvent <= minDuration {
			durations[i] = minDuration
		} else {
			// Random duration between minDuration and maxForThisEvent
			rangeSize := maxForThisEvent - minDuration
			randomDuration := time.Duration(rand.Int63n(int64(rangeSize)+1)) + minDuration
			durations[i] = randomDuration
		}
		remaining -= durations[i]
	}
	durations[numberOfEvents-1] = remaining

	// Create events sequentially without gaps to ensure they all fit
	currentTime := startTime

	for i := 0; i < numberOfEvents; i++ {
		eventEnd := currentTime.Add(durations[i])

		// Ensure we don't exceed endTime
		if eventEnd.After(endTime) {
			eventEnd = endTime
		}

		calendarStub.AddEvent(ctx, calendar.Event{
			Summary:   fmt.Sprintf("Event %d", i),
			StartTime: currentTime,
			EndTime:   eventEnd,
			Metadata:  calendar.EventMetadata{BudgetItemId: budgetItemId},
		})

		currentTime = eventEnd

		// Stop if we've reached the end time
		if currentTime.Equal(endTime) || currentTime.After(endTime) {
			break
		}
	}
}
