package stats

import (
	"context"
	"testing"
	"time"

	"github.com/klokku/klokku/internal/utils"
	"github.com/klokku/klokku/pkg/budget"
	"github.com/klokku/klokku/pkg/budget_override"
	"github.com/klokku/klokku/pkg/calendar"
	"github.com/klokku/klokku/pkg/event"
	"github.com/klokku/klokku/pkg/user"
)

var ctx = context.WithValue(context.Background(), user.UserIDKey, 1)
var calendarStub = calendar.NewStubCalendar()
var budgetRepoStub = budget.NewStubBudgetRepo()
var budgetRepoService = budget.NewBudgetServiceImpl(budgetRepoStub)
var budgetOverrideRepoStub = budget_override.NewStubBudgetOverrideRepo()
var eventRepoStub = &event.StubEventRepository{}
var userRepoStub = user.NewStubUserRepository()
var userService = user.NewUserService(userRepoStub)
var eventService = event.NewEventService(eventRepoStub, calendarStub, userService)
var clock = &utils.MockClock{FixedNow: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)}
var eventsStatsService = event.NewEventStatsServiceImpl(calendarStub, clock)
var statsService = StatsServiceImpl{
	eventService:       eventService,
	eventStatsService:  eventsStatsService,
	budgetRepo:         budgetRepoStub,
	budgetOverrideRepo: budgetOverrideRepoStub,
	clock:              clock,
}

func setup(t *testing.T) func() {
	return func() {
		t.Log("Teardown after test")
		budgetRepoStub.Cleanup()
		budgetOverrideRepoStub.Cleanup()
		calendarStub.Cleanup()
		eventRepoStub.Cleanup()
	}
}

func TestStatsServiceImpl_GetStats(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	// given
	startTime := time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2023, time.January, 7, 0, 0, 0, 0, time.UTC)
	budget1, _ := budgetRepoService.Create(ctx, budget.Budget{
		Name:       "Budget 1",
		WeeklyTime: time.Duration(120) * time.Minute,
	})
	budget2, _ := budgetRepoService.Create(ctx, budget.Budget{
		Name:       "Budget 2",
		WeeklyTime: time.Duration(120) * time.Minute,
	})
	calendarStub.AddEvent(ctx, calendar.Event{ // 60 minutes
		Summary:   "Budget 1",
		StartTime: startTime,
		EndTime:   startTime.Add(time.Hour),
		Metadata:  calendar.EventMetadata{BudgetId: budget1.ID},
	})
	calendarStub.AddEvent(ctx, calendar.Event{ // 30 minutes
		Summary:   "Budget 2",
		StartTime: startTime.Add(time.Hour),
		EndTime:   startTime.Add(time.Hour).Add(30 * time.Minute),
		Metadata:  calendar.EventMetadata{BudgetId: budget2.ID},
	})
	calendarStub.AddEvent(ctx, calendar.Event{ // 90 minutes
		Summary:   "Budget 1",
		StartTime: startTime.Add(90 * time.Minute),
		EndTime:   startTime.Add(90 * time.Minute).Add(90 * time.Minute),
		Metadata:  calendar.EventMetadata{BudgetId: budget1.ID},
	})
	secondDay := startTime.Add(24 * time.Hour)
	calendarStub.AddEvent(ctx, calendar.Event{ // 75 minutes
		Summary:   "Budget 2",
		StartTime: secondDay.Add(2 * time.Hour),
		EndTime:   secondDay.Add(2 * time.Hour).Add(75 * time.Minute),
		Metadata:  calendar.EventMetadata{BudgetId: budget2.ID},
	})

	// when
	stats, _ := statsService.GetStats(ctx, startTime, endTime)

	// then
	if stats.StartDate != startTime {
		t.Errorf("stats.StartDate = %v, want %v", stats.StartDate, startTime)
	}
	if stats.EndDate != endTime {
		t.Errorf("stats.EndDate = %v, want %v", stats.EndDate, endTime)
	}
	if len(stats.Days) != 7 {
		t.Errorf("len(stats.Days) = %v, want %v", len(stats.Days), 7)
	}
	if len(stats.Budgets) != 2 {
		t.Errorf("len(stats.Budgets) = %v, want %v", len(stats.Budgets), 2)
	}
	if stats.TotalTime != time.Duration(255)*time.Minute {
		t.Errorf("stats.TotalTime = %v, want %v", stats.TotalTime, time.Duration(255)*time.Minute)
	}

	// Check on Day 1
	foundBudget1 := findBudgetByName(stats.Days[0].Budgets, "Budget 1")
	if foundBudget1.Duration != time.Duration(150)*time.Minute {
		t.Errorf("foundBudget1.Duration = %v, want %v", foundBudget1.Duration, time.Duration(150)*time.Minute)
	}
	foundBudget2 := findBudgetByName(stats.Days[0].Budgets, "Budget 2")
	if foundBudget2.Duration != time.Duration(30)*time.Minute {
		t.Errorf("foundBudget2.Duration = %v, want %v", foundBudget2.Duration, time.Duration(30)*time.Minute)
	}
	// Check on Day 2
	foundBudget2 = findBudgetByName(stats.Days[1].Budgets, "Budget 2")
	if foundBudget2.Duration != time.Duration(75)*time.Minute {
		t.Errorf("foundBudget2.Duration = %v, want %v", foundBudget2.Duration, time.Duration(75)*time.Minute)
	}

	// Check by Budgets
	b1 := findBudgetByName(stats.Budgets, "Budget 1")
	if b1.Duration != time.Duration(150)*time.Minute {
		t.Errorf("b1.Duration = %v, want %v", b1.Duration, time.Duration(150)*time.Minute)
	}
	b2 := findBudgetByName(stats.Budgets, "Budget 2")
	if b2.Duration != time.Duration(105)*time.Minute {
		t.Errorf("b2.Duration = %v, want %v", b2.Duration, time.Duration(105)*time.Minute)
	}
}

func TestStatsServiceImpl_GetStats_WithBudgetOverrides(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	// given
	startTime := time.Date(2023, time.January, 2, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2023, time.January, 8, 0, 0, 0, 0, time.UTC)
	budget1Id, _ := budgetRepoStub.Store(ctx, 1, budget.Budget{
		Name:       "Budget 1",
		WeeklyTime: time.Duration(120) * time.Minute,
	})
	budget2Id, _ := budgetRepoStub.Store(ctx, 1, budget.Budget{
		Name:       "Budget 2",
		WeeklyTime: time.Duration(30) * time.Minute,
	})
	budgetOverrideRepoStub.Store(ctx, 1, budget_override.BudgetOverride{ // 120 -> 100
		BudgetID:   budget1Id,
		StartDate:  startTime,
		WeeklyTime: time.Duration(100) * time.Minute,
	})
	budgetOverrideRepoStub.Store(ctx, 1, budget_override.BudgetOverride{ // 30 -> 60
		BudgetID:   budget2Id,
		StartDate:  startTime,
		WeeklyTime: time.Duration(60) * time.Minute,
	})
	calendarStub.AddEvent(ctx, calendar.Event{ // 60 minutes
		Summary:   "Budget 1",
		StartTime: startTime,
		EndTime:   startTime.Add(time.Hour),
		Metadata:  calendar.EventMetadata{BudgetId: budget1Id},
	})
	calendarStub.AddEvent(ctx, calendar.Event{ // 30 minutes
		Summary:   "Budget 2",
		StartTime: startTime.Add(time.Hour),
		EndTime:   startTime.Add(time.Hour).Add(30 * time.Minute),
		Metadata:  calendar.EventMetadata{BudgetId: budget2Id},
	})

	// when
	stats, _ := statsService.GetStats(ctx, startTime, endTime)

	// then
	if stats.TotalPlanned != time.Duration(160)*time.Minute {
		t.Errorf("stats.TotalPlanned = %v, want %v", stats.TotalPlanned, time.Duration(160)*time.Minute)
	}
	if stats.TotalRemaining != time.Duration(70)*time.Minute {
		t.Errorf("stats.TotalRemaining = %v, want %v", stats.TotalRemaining, time.Duration(70)*time.Minute)
	}
	if stats.TotalTime != time.Duration(90)*time.Minute {
		t.Errorf("stats.TotalTime = %v, want %v", stats.TotalRemaining, time.Duration(90)*time.Minute)
	}

	// Check budgets
	budget1 := findBudgetByName(stats.Budgets, "Budget 1")
	if budget1.BudgetOverride.WeeklyTime != time.Duration(100)*time.Minute {
		t.Errorf("budget1.BudgetOverride.WeeklyTime = %v, want %v", budget1.BudgetOverride.WeeklyTime, time.Duration(100)*time.Minute)
	}
	budget2 := findBudgetByName(stats.Budgets, "Budget 2")
	if budget2.BudgetOverride.WeeklyTime != time.Duration(60)*time.Minute {
		t.Errorf("budget2.BudgetOverride.WeeklyTime = %v, want %v", budget2.BudgetOverride.WeeklyTime, time.Duration(60)*time.Minute)
	}

	// Check budgets remaining are from override
	budget1 = findBudgetByName(stats.Budgets, "Budget 1") // 100 - 60
	if budget1.Remaining != time.Duration(40)*time.Minute {
		t.Errorf("budget1.Remaining = %v, want %v", budget1.Remaining, time.Duration(40)*time.Minute)
	}
	budget2 = findBudgetByName(stats.Budgets, "Budget 2") // 60 - 30
	if budget2.Remaining != time.Duration(30)*time.Minute {
		t.Errorf("budget2.Remaining = %v, want %v", budget2.Remaining, time.Duration(30)*time.Minute)
	}
}

func TestStatsServiceImpl_GetStats_WithCurrentEvent(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	// given
	startTime := time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2023, time.January, 7, 0, 0, 0, 0, time.UTC)
	budget1Id, _ := budgetRepoStub.Store(ctx, 1, budget.Budget{
		Name:       "Budget 1",
		WeeklyTime: time.Duration(120) * time.Minute,
	})
	calendarStub.AddEvent(ctx, calendar.Event{ // 60 minutes
		Summary:   "Budget 1",
		StartTime: startTime,
		EndTime:   startTime.Add(time.Hour),
		Metadata:  calendar.EventMetadata{BudgetId: budget1Id},
	})
	clock.SetNow(startTime.Add(90 * time.Minute))
	eventService.StartNewEvent(ctx, event.Event{ // started 30 minutes ago
		Budget: budget.Budget{
			ID:         budget1Id,
			Name:       "Budget 1",
			WeeklyTime: time.Duration(120) * time.Minute,
		},
		StartTime: startTime.Add(time.Hour),
	})

	// when
	stats, _ := statsService.GetStats(ctx, startTime, endTime)

	// then
	// check on a budget list
	budget1 := findBudgetByName(stats.Budgets, "Budget 1")
	if budget1.Duration != time.Duration(90)*time.Minute {
		t.Errorf("budget1.Duration = %v, want %v", budget1.Duration, time.Duration(90)*time.Minute)
	}
	if budget1.Remaining != time.Duration(30)*time.Minute {
		t.Errorf("budget1.Remaining = %v, want %v", budget1.Remaining, time.Duration(30)*time.Minute)
	}
	// check in day data
	budget1 = findBudgetByName(stats.Days[0].Budgets, "Budget 1")
	if budget1.Duration != time.Duration(90)*time.Minute {
		t.Errorf("budget1.Duration = %v, want %v", budget1.Duration, time.Duration(90)*time.Minute)
	}
	if stats.Days[0].TotalTime != time.Duration(90)*time.Minute {
		t.Errorf("stats.Days[0].TotalTime = %v, want %v", stats.Days[0].TotalTime, time.Duration(90)*time.Minute)
	}
	// check summary
	if stats.TotalTime != time.Duration(90)*time.Minute {
		t.Errorf("stats.TotalTime = %v, want %v", stats.TotalTime, time.Duration(90)*time.Minute)
	}
	if stats.TotalRemaining != time.Duration(30)*time.Minute {
		t.Errorf("stats.TotalRemaining = %v, want %v", stats.TotalRemaining, time.Duration(30)*time.Minute)
	}

}

func TestStatsServiceImpl_GetStats_WithNonActiveBudget(t *testing.T) {
	teardown := setup(t)
	defer teardown()

	// given
	startTime := time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2023, time.January, 7, 0, 0, 0, 0, time.UTC)
	budgetRepoStub.Store(ctx, 1, budget.Budget{
		Name:       "Budget with end date in the past",
		WeeklyTime: time.Duration(30) * time.Minute,
		// EndDate before the stats period
		EndDate: time.Date(2022, time.December, 31, 0, 0, 0, 0, time.UTC),
	})
	budgetRepoStub.Store(ctx, 1, budget.Budget{
		Name:       "Budget with start date in the future",
		WeeklyTime: time.Duration(40) * time.Minute,
		// StartDate after the stats period
		StartDate: time.Date(2023, time.January, 8, 0, 0, 0, 0, time.UTC),
	})
	budgetRepoStub.Store(ctx, 1, budget.Budget{
		Name:       "Active Budget",
		WeeklyTime: time.Duration(50) * time.Minute,
		StartDate:  startTime,
	})
	// when
	stats, _ := statsService.GetStats(ctx, startTime, endTime)

	// then
	// check the summary
	if stats.TotalPlanned != time.Duration(50)*time.Minute {
		t.Errorf("stats.TotalTime = %v, want %v", stats.TotalPlanned, time.Duration(50)*time.Minute)
	}
	if stats.TotalRemaining != time.Duration(50)*time.Minute {
		t.Errorf("stats.TotalRemaining = %v, want %v", stats.TotalRemaining, time.Duration(50)*time.Minute)
	}
}

func findBudgetByName(budgets []BudgetStats, budgetName string) *BudgetStats {
	for _, b := range budgets {
		if b.Budget.Name == budgetName {
			return &b
		}
	}
	return nil
}
