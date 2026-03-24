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
	itemsByWeek  map[string][]weekly_plan.WeeklyPlanItem
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
				Id:                10,
				PlanId:            1,
				Name:              "Exercise",
				WeeklyDuration:    5 * time.Hour,
				WeeklyOccurrences: 5,
				Icon:              "run",
				Color:             "#ff0000",
				Position:          100,
			},
			{
				Id:                20,
				PlanId:            1,
				Name:              "Reading",
				WeeklyDuration:    3 * time.Hour,
				WeeklyOccurrences: 3,
				Icon:              "book",
				Color:             "#00ff00",
				Position:          200,
			},
		},
	}
}

func mockClock(t time.Time) *utils.MockClock {
	return &utils.MockClock{FixedNow: t}
}

func findReportItem(items []ReportItem, budgetItemId int) *ReportItem {
	for _, item := range items {
		if item.BudgetItemId == budgetItemId {
			return &item
		}
	}
	return nil
}

// --- tests ---

func TestGetReport_NoEvents(t *testing.T) {
	ctx := testContext()
	bp := testBudgetPlan()

	svc := NewService(
		&budgetPlanReaderStub{plans: map[int]budget_plan.BudgetPlan{1: bp}},
		&calendarEventsReaderStub{},
		&earliestEventFinderStub{found: false},
		&weeklyPlanItemsReaderStub{},
		&utils.MockClock{FixedNow: time.Date(2025, 3, 10, 12, 0, 0, 0, time.UTC)},
	)

	report, err := svc.GetReport(ctx, 1, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, report.PlanId)
	assert.Equal(t, "Test Plan", report.PlanName)
	assert.Equal(t, 0, report.WeekCount)
	assert.Empty(t, report.TotalItems)
}

func TestGetReport_SingleWeek(t *testing.T) {
	ctx := testContext()
	bp := testBudgetPlan()

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
		&weeklyPlanItemsReaderStub{},
		mockClock(time.Date(2025, 3, 7, 12, 0, 0, 0, time.UTC)),
	)

	report, err := svc.GetReport(ctx, 1, nil, nil)
	require.NoError(t, err)
	require.Len(t, report.Weeks, 1)

	week := report.Weeks[0]
	assert.Equal(t, "2025-W10", week.WeekNumber)
	assert.Equal(t, time.Date(2025, 3, 3, 0, 0, 0, 0, warsawTz), week.StartDate)
	require.Len(t, week.Items, 2)

	exercise := week.Items[0]
	assert.Equal(t, 10, exercise.BudgetItemId)
	assert.Equal(t, 3*time.Hour, exercise.ActualTime)

	reading := week.Items[1]
	assert.Equal(t, 20, reading.BudgetItemId)
	assert.Equal(t, 1*time.Hour, reading.ActualTime)

	// Totals
	assert.Equal(t, 1, report.WeekCount)
	assert.Equal(t, 8*time.Hour, report.TotalBudgetPlanTime)
	assert.Equal(t, 4*time.Hour, report.TotalActualTime)
}

func TestGetReport_MultipleWeeks_WithGap(t *testing.T) {
	ctx := testContext()
	bp := testBudgetPlan()

	week1Monday := time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC)
	week3Monday := time.Date(2025, 3, 17, 0, 0, 0, 0, time.UTC)
	clockTime := time.Date(2025, 3, 21, 12, 0, 0, 0, time.UTC)

	events := []calendar.Event{
		makeEvent(10, week1Monday.Add(8*time.Hour), week1Monday.Add(10*time.Hour)),
		makeEvent(10, week3Monday.Add(8*time.Hour), week3Monday.Add(9*time.Hour)),
	}

	svc := NewService(
		&budgetPlanReaderStub{plans: map[int]budget_plan.BudgetPlan{1: bp}},
		&calendarEventsReaderStub{events: events},
		&earliestEventFinderStub{earliest: week1Monday.Add(8 * time.Hour), found: true},
		&weeklyPlanItemsReaderStub{},
		mockClock(clockTime),
	)

	report, err := svc.GetReport(ctx, 1, nil, nil)
	require.NoError(t, err)
	require.Len(t, report.Weeks, 3)

	assert.Equal(t, "2025-W10", report.Weeks[0].WeekNumber)
	assert.Equal(t, 2*time.Hour, findReportItem(report.Weeks[0].Items, 10).ActualTime)

	assert.Equal(t, "2025-W11", report.Weeks[1].WeekNumber)
	assert.Equal(t, time.Duration(0), report.Weeks[1].TotalActualTime)

	assert.Equal(t, "2025-W12", report.Weeks[2].WeekNumber)
	assert.Equal(t, 1*time.Hour, findReportItem(report.Weeks[2].Items, 10).ActualTime)
}

func TestGetReport_WeeklyPlanOverride(t *testing.T) {
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
				{Id: 101, BudgetItemId: 10, BudgetPlanId: 1, Name: "Exercise", WeeklyDuration: 7 * time.Hour, Icon: "run", Color: "#ff0000", Position: 100},
				{Id: 102, BudgetItemId: 20, BudgetPlanId: 1, Name: "Reading", WeeklyDuration: 2 * time.Hour, Icon: "book", Color: "#00ff00", Position: 200},
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

	report, err := svc.GetReport(ctx, 1, nil, nil)
	require.NoError(t, err)
	require.Len(t, report.Weeks, 1)

	exercise := findReportItem(report.Weeks[0].Items, 10)
	assert.Equal(t, 5*time.Hour, exercise.BudgetPlanTime)
	assert.Equal(t, 7*time.Hour, exercise.WeeklyPlanTime)

	reading := findReportItem(report.Weeks[0].Items, 20)
	assert.Equal(t, 3*time.Hour, reading.BudgetPlanTime)
	assert.Equal(t, 2*time.Hour, reading.WeeklyPlanTime)
}

func TestGetReport_DeletedItemsExcluded(t *testing.T) {
	ctx := testContext()
	bp := testBudgetPlan()

	weekMonday := time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC)
	clockTime := time.Date(2025, 3, 7, 12, 0, 0, 0, time.UTC)

	events := []calendar.Event{
		makeEvent(10, weekMonday.Add(8*time.Hour), weekMonday.Add(10*time.Hour)),
		makeEvent(99, weekMonday.Add(10*time.Hour), weekMonday.Add(12*time.Hour)),
	}

	svc := NewService(
		&budgetPlanReaderStub{plans: map[int]budget_plan.BudgetPlan{1: bp}},
		&calendarEventsReaderStub{events: events},
		&earliestEventFinderStub{earliest: weekMonday.Add(8 * time.Hour), found: true},
		&weeklyPlanItemsReaderStub{},
		mockClock(clockTime),
	)

	report, err := svc.GetReport(ctx, 1, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 2*time.Hour, report.Weeks[0].TotalActualTime)
}

func TestGetReport_AggregatesAcrossAllWeeks(t *testing.T) {
	ctx := testContext()
	bp := testBudgetPlan()

	week1Monday := time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC)
	week2Monday := time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC)
	clockTime := time.Date(2025, 3, 14, 12, 0, 0, 0, time.UTC)

	events := []calendar.Event{
		makeEvent(10, week1Monday.Add(8*time.Hour), week1Monday.Add(11*time.Hour)),
		makeEvent(20, week1Monday.Add(11*time.Hour), week1Monday.Add(12*time.Hour)),
		makeEvent(10, week2Monday.Add(8*time.Hour), week2Monday.Add(10*time.Hour)),
	}

	svc := NewService(
		&budgetPlanReaderStub{plans: map[int]budget_plan.BudgetPlan{1: bp}},
		&calendarEventsReaderStub{events: events},
		&earliestEventFinderStub{earliest: week1Monday.Add(8 * time.Hour), found: true},
		&weeklyPlanItemsReaderStub{},
		mockClock(clockTime),
	)

	report, err := svc.GetReport(ctx, 1, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, 1, report.PlanId)
	assert.Equal(t, 2, report.WeekCount)

	exercise := findReportItem(report.TotalItems, 10)
	assert.Equal(t, 5*time.Hour, exercise.ActualTime)
	assert.Equal(t, 10*time.Hour, exercise.BudgetPlanTime)
	// AveragePerWeek: 5h / 2 weeks = 2.5h
	assert.Equal(t, 2*time.Hour+30*time.Minute, exercise.AveragePerWeek)
	// AveragePerDay: 5h / (2 weeks * 5 occurrences) = 5h / 10 = 30min
	assert.Equal(t, 30*time.Minute, exercise.AveragePerDay)

	reading := findReportItem(report.TotalItems, 20)
	assert.Equal(t, 1*time.Hour, reading.ActualTime)
	assert.Equal(t, 6*time.Hour, reading.BudgetPlanTime)
	// AveragePerWeek: 1h / 2 weeks = 30min
	assert.Equal(t, 30*time.Minute, reading.AveragePerWeek)
	// AveragePerDay: 1h / (2 weeks * 3 occurrences) = 1h / 6 = 10min
	assert.Equal(t, 10*time.Minute, reading.AveragePerDay)

	assert.Equal(t, 16*time.Hour, report.TotalBudgetPlanTime)
	assert.Equal(t, 6*time.Hour, report.TotalActualTime)
}

func TestGetReport_AveragePerDay_DefaultsTo7WhenOccurrencesNotSet(t *testing.T) {
	ctx := testContext()
	bp := budget_plan.BudgetPlan{
		Id:   1,
		Name: "Test Plan",
		Items: []budget_plan.BudgetItem{
			{
				Id:                10,
				PlanId:            1,
				Name:              "Exercise",
				WeeklyDuration:    5 * time.Hour,
				WeeklyOccurrences: 0, // zero occurrences
				Icon:              "run",
				Color:             "#ff0000",
				Position:          100,
			},
		},
	}

	weekMonday := time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC)
	events := []calendar.Event{
		makeEvent(10, weekMonday.Add(8*time.Hour), weekMonday.Add(10*time.Hour)),
	}

	svc := NewService(
		&budgetPlanReaderStub{plans: map[int]budget_plan.BudgetPlan{1: bp}},
		&calendarEventsReaderStub{events: events},
		&earliestEventFinderStub{earliest: weekMonday.Add(8 * time.Hour), found: true},
		&weeklyPlanItemsReaderStub{},
		mockClock(time.Date(2025, 3, 7, 12, 0, 0, 0, time.UTC)),
	)

	report, err := svc.GetReport(ctx, 1, nil, nil)
	require.NoError(t, err)

	exercise := findReportItem(report.TotalItems, 10)
	assert.Equal(t, 2*time.Hour, exercise.AveragePerWeek)
	// zero occurrences → defaults to 7: 2h / (1 week * 7) = ~17min
	assert.Equal(t, 2*time.Hour/7, exercise.AveragePerDay)
}

func TestGetReport_WithDateRange(t *testing.T) {
	ctx := testContext()
	bp := testBudgetPlan()

	week1Monday := time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC)
	week2Monday := time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC)
	week3Monday := time.Date(2025, 3, 17, 0, 0, 0, 0, time.UTC)
	clockTime := time.Date(2025, 3, 21, 12, 0, 0, 0, time.UTC)

	events := []calendar.Event{
		makeEvent(10, week1Monday.Add(8*time.Hour), week1Monday.Add(10*time.Hour)), // 2h week 1
		makeEvent(10, week2Monday.Add(8*time.Hour), week2Monday.Add(11*time.Hour)), // 3h week 2
		makeEvent(10, week3Monday.Add(8*time.Hour), week3Monday.Add(9*time.Hour)),  // 1h week 3
	}

	svc := NewService(
		&budgetPlanReaderStub{plans: map[int]budget_plan.BudgetPlan{1: bp}},
		&calendarEventsReaderStub{events: events},
		&earliestEventFinderStub{earliest: week1Monday.Add(8 * time.Hour), found: true},
		&weeklyPlanItemsReaderStub{},
		mockClock(clockTime),
	)

	// Request only weeks 2 and 3
	from := week2Monday
	to := week3Monday.AddDate(0, 0, 6) // Sunday of week 3

	report, err := svc.GetReport(ctx, 1, &from, &to)
	require.NoError(t, err)
	require.Len(t, report.Weeks, 2)

	assert.Equal(t, "2025-W11", report.Weeks[0].WeekNumber)
	assert.Equal(t, 3*time.Hour, findReportItem(report.Weeks[0].Items, 10).ActualTime)

	assert.Equal(t, "2025-W12", report.Weeks[1].WeekNumber)
	assert.Equal(t, 1*time.Hour, findReportItem(report.Weeks[1].Items, 10).ActualTime)

	// Totals should only cover weeks 2-3
	assert.Equal(t, 2, report.WeekCount)
	assert.Equal(t, 4*time.Hour, findReportItem(report.TotalItems, 10).ActualTime) // 3h + 1h
	assert.Equal(t, 16*time.Hour, report.TotalBudgetPlanTime)                      // (5+3) × 2
}

func TestGetReport_EmptyPlan(t *testing.T) {
	ctx := testContext()
	emptyPlan := budget_plan.BudgetPlan{Id: 1, Name: "Empty", Items: nil}

	svc := NewService(
		&budgetPlanReaderStub{plans: map[int]budget_plan.BudgetPlan{1: emptyPlan}},
		&calendarEventsReaderStub{},
		&earliestEventFinderStub{},
		&weeklyPlanItemsReaderStub{},
		mockClock(time.Date(2025, 3, 10, 12, 0, 0, 0, time.UTC)),
	)

	report, err := svc.GetReport(ctx, 1, nil, nil)
	require.NoError(t, err)
	assert.Empty(t, report.Weeks)
}
