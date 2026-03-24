package budget_plan_report

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klokku/klokku/internal/utils"
	"github.com/klokku/klokku/pkg/budget_plan"
	"github.com/klokku/klokku/pkg/calendar"
	"github.com/klokku/klokku/pkg/user"
	"github.com/klokku/klokku/pkg/weekly_plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- test stubs ---

type budgetPlanReaderStub struct {
	plans map[int]budget_plan.BudgetPlan
}

func (s *budgetPlanReaderStub) GetPlan(_ context.Context, planId int) (budget_plan.BudgetPlan, error) {
	if p, ok := s.plans[planId]; ok {
		return p, nil
	}
	return budget_plan.BudgetPlan{}, budget_plan.ErrPlanNotFound
}

type calendarEventsReaderStub struct {
	events []calendar.Event
}

func (s *calendarEventsReaderStub) GetEvents(_ context.Context, from, to time.Time) ([]calendar.Event, error) {
	var result []calendar.Event
	for _, e := range s.events {
		if !e.StartTime.After(to) && !e.EndTime.Before(from) {
			result = append(result, e)
		}
	}
	return result, nil
}

type earliestEventFinderStub struct {
	earliest time.Time
	found    bool
}

func (s *earliestEventFinderStub) GetEarliestEventTimeForBudgetItems(_ context.Context, _ []int) (time.Time, bool, error) {
	return s.earliest, s.found, nil
}

type weeklyPlanItemsReaderStub struct {
	// weekStart string -> items
	itemsByWeek map[string][]weekly_plan.WeeklyPlanItem
	// fallback items returned for any week not explicitly configured
	defaultItems []weekly_plan.WeeklyPlanItem
}

func (s *weeklyPlanItemsReaderStub) GetItemsForWeek(_ context.Context, date time.Time) ([]weekly_plan.WeeklyPlanItem, error) {
	wn := weekly_plan.WeekNumberFromDate(date, time.Monday)
	key := wn.String()
	if items, ok := s.itemsByWeek[key]; ok {
		return items, nil
	}
	return s.defaultItems, nil
}

// --- test helpers ---

func makeEvent(budgetItemId int, start, end time.Time) calendar.Event {
	return calendar.Event{
		UID:       uuid.NewString(),
		Summary:   "test",
		StartTime: start,
		EndTime:   end,
		Metadata:  calendar.EventMetadata{BudgetItemId: budgetItemId},
	}
}

var warsawTz, _ = time.LoadLocation("Europe/Warsaw")

func testContext() context.Context {
	return user.WithUser(context.Background(), user.User{
		Id:          1,
		Uid:         uuid.NewString(),
		Username:    "test-user",
		DisplayName: "Test User",
		Settings: user.Settings{
			Timezone:          "Europe/Warsaw",
			WeekFirstDay:      time.Monday,
			EventCalendarType: user.KlokkuCalendar,
		},
	})
}

func testBudgetPlan() budget_plan.BudgetPlan {
	return budget_plan.BudgetPlan{
		Id:   1,
		Name: "Test Plan",
		Items: []budget_plan.BudgetItem{
			{
				Id:             10,
				PlanId:         1,
				Name:           "Exercise",
				WeeklyDuration: 5 * time.Hour,
				Icon:           "run",
				Color:          "#ff0000",
				Position:       100,
			},
			{
				Id:             20,
				PlanId:         1,
				Name:           "Reading",
				WeeklyDuration: 3 * time.Hour,
				Icon:           "book",
				Color:          "#00ff00",
				Position:       200,
			},
		},
	}
}

// --- tests ---

func TestGetSummaryReport_NoEvents(t *testing.T) {
	ctx := testContext()
	bp := testBudgetPlan()

	svc := NewService(
		&budgetPlanReaderStub{plans: map[int]budget_plan.BudgetPlan{1: bp}},
		&calendarEventsReaderStub{},
		&earliestEventFinderStub{found: false},
		&weeklyPlanItemsReaderStub{},
		&utils.MockClock{FixedNow: time.Date(2025, 3, 10, 12, 0, 0, 0, time.UTC)},
	)

	report, err := svc.GetSummaryReport(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, report.PlanId)
	assert.Equal(t, "Test Plan", report.PlanName)
	assert.Equal(t, 0, report.WeekCount)
	assert.Empty(t, report.Items)
}

func TestGetWeeklyReport_SingleWeek(t *testing.T) {
	ctx := testContext()
	bp := testBudgetPlan()

	// Monday 2025-03-03 week
	weekMonday := time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC)
	events := []calendar.Event{
		makeEvent(10, weekMonday.Add(8*time.Hour), weekMonday.Add(10*time.Hour)),  // 2h Exercise
		makeEvent(20, weekMonday.Add(10*time.Hour), weekMonday.Add(11*time.Hour)), // 1h Reading
		makeEvent(10, weekMonday.Add(24*time.Hour), weekMonday.Add(25*time.Hour)), // 1h Exercise (Tuesday)
	}

	svc := NewService(
		&budgetPlanReaderStub{plans: map[int]budget_plan.BudgetPlan{1: bp}},
		&calendarEventsReaderStub{events: events},
		&earliestEventFinderStub{earliest: weekMonday.Add(8 * time.Hour), found: true},
		&weeklyPlanItemsReaderStub{}, // no weekly plan overrides, uses budget plan defaults
		mockClock(time.Date(2025, 3, 7, 12, 0, 0, 0, time.UTC)),
	)

	entries, err := svc.GetWeeklyReport(ctx, 1)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	week := entries[0]
	assert.Equal(t, "2025-W10", week.WeekNumber)
	assert.Equal(t, time.Date(2025, 3, 3, 0, 0, 0, 0, warsawTz), week.StartDate)
	require.Len(t, week.Items, 2)

	// Items should be ordered by position
	exercise := week.Items[0]
	assert.Equal(t, 10, exercise.BudgetItemId)
	assert.Equal(t, "Exercise", exercise.Name)
	assert.Equal(t, 5*time.Hour, exercise.BudgetPlanTime)
	assert.Equal(t, 5*time.Hour, exercise.WeeklyPlanTime) // no override
	assert.Equal(t, 3*time.Hour, exercise.ActualTime)

	reading := week.Items[1]
	assert.Equal(t, 20, reading.BudgetItemId)
	assert.Equal(t, 3*time.Hour, reading.BudgetPlanTime)
	assert.Equal(t, 3*time.Hour, reading.WeeklyPlanTime)
	assert.Equal(t, 1*time.Hour, reading.ActualTime)

	// Totals
	assert.Equal(t, 8*time.Hour, week.TotalBudgetPlanTime)
	assert.Equal(t, 8*time.Hour, week.TotalWeeklyPlanTime)
	assert.Equal(t, 4*time.Hour, week.TotalActualTime)
}

func mockClock(t time.Time) *utils.MockClock {
	return &utils.MockClock{FixedNow: t}
}

func TestGetWeeklyReport_MultipleWeeks_WithGap(t *testing.T) {
	ctx := testContext()
	bp := testBudgetPlan()

	week1Monday := time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC)
	week3Monday := time.Date(2025, 3, 17, 0, 0, 0, 0, time.UTC)
	clockTime := time.Date(2025, 3, 21, 12, 0, 0, 0, time.UTC) // Friday of week 3

	events := []calendar.Event{
		makeEvent(10, week1Monday.Add(8*time.Hour), week1Monday.Add(10*time.Hour)), // 2h week 1
		makeEvent(10, week3Monday.Add(8*time.Hour), week3Monday.Add(9*time.Hour)),  // 1h week 3
	}

	svc := NewService(
		&budgetPlanReaderStub{plans: map[int]budget_plan.BudgetPlan{1: bp}},
		&calendarEventsReaderStub{events: events},
		&earliestEventFinderStub{earliest: week1Monday.Add(8 * time.Hour), found: true},
		&weeklyPlanItemsReaderStub{},
		mockClock(clockTime),
	)

	entries, err := svc.GetWeeklyReport(ctx, 1)
	require.NoError(t, err)
	require.Len(t, entries, 3) // week 1, week 2 (gap), week 3

	// Week 1: has events
	assert.Equal(t, "2025-W10", entries[0].WeekNumber)
	assert.Equal(t, 2*time.Hour, findReportItem(entries[0].Items, 10).ActualTime)

	// Week 2: gap week, zero actual
	assert.Equal(t, "2025-W11", entries[1].WeekNumber)
	assert.Equal(t, time.Duration(0), entries[1].TotalActualTime)
	assert.Equal(t, 8*time.Hour, entries[1].TotalBudgetPlanTime) // planned hours still present

	// Week 3: has events
	assert.Equal(t, "2025-W12", entries[2].WeekNumber)
	assert.Equal(t, 1*time.Hour, findReportItem(entries[2].Items, 10).ActualTime)
}

func TestGetWeeklyReport_WeeklyPlanOverride(t *testing.T) {
	ctx := testContext()
	bp := testBudgetPlan()

	weekMonday := time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC)
	clockTime := time.Date(2025, 3, 7, 12, 0, 0, 0, time.UTC)

	events := []calendar.Event{
		makeEvent(10, weekMonday.Add(8*time.Hour), weekMonday.Add(10*time.Hour)),
	}

	weeklyPlanStub := &weeklyPlanItemsReaderStub{
		itemsByWeek: map[string][]weekly_plan.WeeklyPlanItem{
			"2025-W10": {
				{
					Id:             101,
					BudgetItemId:   10,
					BudgetPlanId:   1,
					Name:           "Exercise",
					WeeklyDuration: 7 * time.Hour, // override: 7h instead of 5h
					Icon:           "run",
					Color:          "#ff0000",
					Position:       100,
				},
				{
					Id:             102,
					BudgetItemId:   20,
					BudgetPlanId:   1,
					Name:           "Reading",
					WeeklyDuration: 2 * time.Hour, // override: 2h instead of 3h
					Icon:           "book",
					Color:          "#00ff00",
					Position:       200,
				},
			},
		},
	}

	svc := NewService(
		&budgetPlanReaderStub{plans: map[int]budget_plan.BudgetPlan{1: bp}},
		&calendarEventsReaderStub{events: events},
		&earliestEventFinderStub{earliest: weekMonday.Add(8 * time.Hour), found: true},
		weeklyPlanStub,
		mockClock(clockTime),
	)

	entries, err := svc.GetWeeklyReport(ctx, 1)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	exercise := findReportItem(entries[0].Items, 10)
	assert.Equal(t, 5*time.Hour, exercise.BudgetPlanTime) // original budget plan
	assert.Equal(t, 7*time.Hour, exercise.WeeklyPlanTime) // weekly plan override

	reading := findReportItem(entries[0].Items, 20)
	assert.Equal(t, 3*time.Hour, reading.BudgetPlanTime)
	assert.Equal(t, 2*time.Hour, reading.WeeklyPlanTime)
}

func TestGetWeeklyReport_DeletedItemsExcluded(t *testing.T) {
	ctx := testContext()
	bp := testBudgetPlan() // has items 10 and 20

	weekMonday := time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC)
	clockTime := time.Date(2025, 3, 7, 12, 0, 0, 0, time.UTC)

	events := []calendar.Event{
		makeEvent(10, weekMonday.Add(8*time.Hour), weekMonday.Add(10*time.Hour)),  // 2h for item 10
		makeEvent(99, weekMonday.Add(10*time.Hour), weekMonday.Add(12*time.Hour)), // 2h for deleted item 99
	}

	svc := NewService(
		&budgetPlanReaderStub{plans: map[int]budget_plan.BudgetPlan{1: bp}},
		&calendarEventsReaderStub{events: events},
		&earliestEventFinderStub{earliest: weekMonday.Add(8 * time.Hour), found: true},
		&weeklyPlanItemsReaderStub{},
		mockClock(clockTime),
	)

	entries, err := svc.GetWeeklyReport(ctx, 1)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	// Only 4h total would include the deleted item, but we should see only 2h
	assert.Equal(t, 2*time.Hour, entries[0].TotalActualTime)
	exercise := findReportItem(entries[0].Items, 10)
	assert.Equal(t, 2*time.Hour, exercise.ActualTime)
}

func TestGetMonthlyReport_GroupsInto4WeekPeriods(t *testing.T) {
	ctx := testContext()
	bp := testBudgetPlan()

	week1Monday := time.Date(2025, 1, 6, 0, 0, 0, 0, time.UTC)
	clockTime := time.Date(2025, 2, 21, 12, 0, 0, 0, time.UTC) // 7 weeks later

	// Create events across 7 weeks
	var events []calendar.Event
	for w := 0; w < 7; w++ {
		weekStart := week1Monday.AddDate(0, 0, w*7)
		events = append(events, makeEvent(10, weekStart.Add(8*time.Hour), weekStart.Add(9*time.Hour))) // 1h per week
	}

	svc := NewService(
		&budgetPlanReaderStub{plans: map[int]budget_plan.BudgetPlan{1: bp}},
		&calendarEventsReaderStub{events: events},
		&earliestEventFinderStub{earliest: week1Monday.Add(8 * time.Hour), found: true},
		&weeklyPlanItemsReaderStub{},
		mockClock(clockTime),
	)

	entries, err := svc.GetMonthlyReport(ctx, 1)
	require.NoError(t, err)
	require.Len(t, entries, 2) // 4 weeks + 3 weeks

	// Period 1: 4 weeks
	assert.Equal(t, 1, entries[0].PeriodNumber)
	assert.Equal(t, 4, entries[0].WeekCount)
	assert.Equal(t, 4*time.Hour, findReportItem(entries[0].Items, 10).ActualTime)      // 1h × 4
	assert.Equal(t, 20*time.Hour, findReportItem(entries[0].Items, 10).BudgetPlanTime) // 5h × 4

	// Period 2: 3 weeks (incomplete)
	assert.Equal(t, 2, entries[1].PeriodNumber)
	assert.Equal(t, 3, entries[1].WeekCount)
	assert.Equal(t, 3*time.Hour, findReportItem(entries[1].Items, 10).ActualTime)      // 1h × 3
	assert.Equal(t, 15*time.Hour, findReportItem(entries[1].Items, 10).BudgetPlanTime) // 5h × 3
}

func TestGetSummaryReport_AggregatesAcrossAllWeeks(t *testing.T) {
	ctx := testContext()
	bp := testBudgetPlan()

	week1Monday := time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC)
	week2Monday := time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC)
	clockTime := time.Date(2025, 3, 14, 12, 0, 0, 0, time.UTC)

	events := []calendar.Event{
		makeEvent(10, week1Monday.Add(8*time.Hour), week1Monday.Add(11*time.Hour)),  // 3h
		makeEvent(20, week1Monday.Add(11*time.Hour), week1Monday.Add(12*time.Hour)), // 1h
		makeEvent(10, week2Monday.Add(8*time.Hour), week2Monday.Add(10*time.Hour)),  // 2h
	}

	svc := NewService(
		&budgetPlanReaderStub{plans: map[int]budget_plan.BudgetPlan{1: bp}},
		&calendarEventsReaderStub{events: events},
		&earliestEventFinderStub{earliest: week1Monday.Add(8 * time.Hour), found: true},
		&weeklyPlanItemsReaderStub{},
		mockClock(clockTime),
	)

	report, err := svc.GetSummaryReport(ctx, 1)
	require.NoError(t, err)

	assert.Equal(t, 1, report.PlanId)
	assert.Equal(t, "Test Plan", report.PlanName)
	assert.Equal(t, 2, report.WeekCount)

	exercise := findSummaryItem(report.Items, 10)
	assert.Equal(t, 5*time.Hour, exercise.ActualTime)      // 3h + 2h
	assert.Equal(t, 10*time.Hour, exercise.BudgetPlanTime) // 5h × 2 weeks

	reading := findSummaryItem(report.Items, 20)
	assert.Equal(t, 1*time.Hour, reading.ActualTime)
	assert.Equal(t, 6*time.Hour, reading.BudgetPlanTime) // 3h × 2 weeks

	assert.Equal(t, 16*time.Hour, report.TotalBudgetPlanTime) // (5+3) × 2
	assert.Equal(t, 6*time.Hour, report.TotalActualTime)
}

func TestGetWeeklyReport_EmptyPlan(t *testing.T) {
	ctx := testContext()
	emptyPlan := budget_plan.BudgetPlan{Id: 1, Name: "Empty", Items: nil}

	svc := NewService(
		&budgetPlanReaderStub{plans: map[int]budget_plan.BudgetPlan{1: emptyPlan}},
		&calendarEventsReaderStub{},
		&earliestEventFinderStub{},
		&weeklyPlanItemsReaderStub{},
		mockClock(time.Date(2025, 3, 10, 12, 0, 0, 0, time.UTC)),
	)

	entries, err := svc.GetWeeklyReport(ctx, 1)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

// --- helpers ---

func findReportItem(items []ReportItem, budgetItemId int) *ReportItem {
	for _, item := range items {
		if item.BudgetItemId == budgetItemId {
			return &item
		}
	}
	return nil
}

func findSummaryItem(items []ReportItem, budgetItemId int) *ReportItem {
	return findReportItem(items, budgetItemId)
}
