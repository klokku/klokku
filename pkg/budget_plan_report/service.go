package budget_plan_report

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/klokku/klokku/internal/utils"
	"github.com/klokku/klokku/pkg/budget_plan"
	"github.com/klokku/klokku/pkg/calendar"
	"github.com/klokku/klokku/pkg/user"
	"github.com/klokku/klokku/pkg/weekly_plan"
)

type Service interface {
	GetSummaryReport(ctx context.Context, planId int) (SummaryReport, error)
	GetWeeklyReport(ctx context.Context, planId int) ([]WeeklyReportEntry, error)
	GetMonthlyReport(ctx context.Context, planId int) ([]MonthlyReportEntry, error)
}

type budgetPlanReader interface {
	GetPlan(ctx context.Context, planId int) (budget_plan.BudgetPlan, error)
}

type calendarEventsReader interface {
	GetEvents(ctx context.Context, from time.Time, to time.Time) ([]calendar.Event, error)
}

type earliestEventFinder interface {
	GetEarliestEventTimeForBudgetItems(ctx context.Context, budgetItemIds []int) (time.Time, bool, error)
}

type weeklyPlanItemsReader interface {
	GetItemsForWeek(ctx context.Context, date time.Time) ([]weekly_plan.WeeklyPlanItem, error)
}

type ServiceImpl struct {
	budgetPlanReader    budgetPlanReader
	calendarReader      calendarEventsReader
	earliestEventFinder earliestEventFinder
	weeklyPlanReader    weeklyPlanItemsReader
	clock               utils.Clock
}

func NewService(
	budgetPlanReader budgetPlanReader,
	calendarReader calendarEventsReader,
	earliestEventFinder earliestEventFinder,
	weeklyPlanReader weeklyPlanItemsReader,
	clock utils.Clock,
) Service {
	return &ServiceImpl{
		budgetPlanReader:    budgetPlanReader,
		calendarReader:      calendarReader,
		earliestEventFinder: earliestEventFinder,
		weeklyPlanReader:    weeklyPlanReader,
		clock:               clock,
	}
}

func (s *ServiceImpl) GetSummaryReport(ctx context.Context, planId int) (SummaryReport, error) {
	weeklyEntries, bp, err := s.buildWeeklyEntries(ctx, planId)
	if err != nil {
		return SummaryReport{}, err
	}
	if len(weeklyEntries) == 0 {
		return SummaryReport{
			PlanId:   bp.Id,
			PlanName: bp.Name,
		}, nil
	}

	// Aggregate across all weeks
	itemTotals := make(map[int]*ReportItem)
	for _, item := range bp.Items {
		itemTotals[item.Id] = &ReportItem{
			BudgetItemId: item.Id,
			Name:         item.Name,
			Icon:         item.Icon,
			Color:        item.Color,
			Position:     item.Position,
		}
	}

	var totalBudget, totalWeekly, totalActual time.Duration
	for _, week := range weeklyEntries {
		for _, item := range week.Items {
			if ri, ok := itemTotals[item.BudgetItemId]; ok {
				ri.BudgetPlanTime += item.BudgetPlanTime
				ri.WeeklyPlanTime += item.WeeklyPlanTime
				ri.ActualTime += item.ActualTime
			}
		}
		totalBudget += week.TotalBudgetPlanTime
		totalWeekly += week.TotalWeeklyPlanTime
		totalActual += week.TotalActualTime
	}

	items := make([]ReportItem, 0, len(itemTotals))
	for _, ri := range itemTotals {
		items = append(items, *ri)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Position < items[j].Position
	})

	return SummaryReport{
		PlanId:              bp.Id,
		PlanName:            bp.Name,
		StartDate:           weeklyEntries[0].StartDate,
		EndDate:             weeklyEntries[len(weeklyEntries)-1].EndDate,
		WeekCount:           len(weeklyEntries),
		Items:               items,
		TotalBudgetPlanTime: totalBudget,
		TotalWeeklyPlanTime: totalWeekly,
		TotalActualTime:     totalActual,
	}, nil
}

func (s *ServiceImpl) GetWeeklyReport(ctx context.Context, planId int) ([]WeeklyReportEntry, error) {
	weeklyEntries, _, err := s.buildWeeklyEntries(ctx, planId)
	if err != nil {
		return nil, err
	}
	return weeklyEntries, nil
}

func (s *ServiceImpl) GetMonthlyReport(ctx context.Context, planId int) ([]MonthlyReportEntry, error) {
	weeklyEntries, _, err := s.buildWeeklyEntries(ctx, planId)
	if err != nil {
		return nil, err
	}
	if len(weeklyEntries) == 0 {
		return []MonthlyReportEntry{}, nil
	}

	// Group weeks into 4-week periods aligned to plan start
	var monthlyEntries []MonthlyReportEntry
	periodNumber := 1
	for i := 0; i < len(weeklyEntries); i += 4 {
		end := i + 4
		if end > len(weeklyEntries) {
			end = len(weeklyEntries)
		}
		periodWeeks := weeklyEntries[i:end]
		entry := aggregateMonthlyPeriod(periodNumber, periodWeeks)
		monthlyEntries = append(monthlyEntries, entry)
		periodNumber++
	}

	return monthlyEntries, nil
}

// buildWeeklyEntries is the shared core that gathers all data and produces per-week entries.
func (s *ServiceImpl) buildWeeklyEntries(ctx context.Context, planId int) ([]WeeklyReportEntry, budget_plan.BudgetPlan, error) {
	currentUser, err := user.CurrentUser(ctx)
	if err != nil {
		return nil, budget_plan.BudgetPlan{}, fmt.Errorf("failed to get current user: %w", err)
	}

	bp, err := s.budgetPlanReader.GetPlan(ctx, planId)
	if err != nil {
		return nil, budget_plan.BudgetPlan{}, fmt.Errorf("failed to get budget plan: %w", err)
	}

	if len(bp.Items) == 0 {
		return nil, bp, nil
	}

	// Collect budget item IDs
	budgetItemIds := make([]int, len(bp.Items))
	budgetItemMap := make(map[int]budget_plan.BudgetItem, len(bp.Items))
	for i, item := range bp.Items {
		budgetItemIds[i] = item.Id
		budgetItemMap[item.Id] = item
	}

	// Find the earliest event
	earliest, found, err := s.earliestEventFinder.GetEarliestEventTimeForBudgetItems(ctx, budgetItemIds)
	if err != nil {
		return nil, bp, fmt.Errorf("failed to find earliest event: %w", err)
	}
	if !found {
		return nil, bp, nil
	}

	// Load user timezone for correct week boundary calculations
	userTimezone, err := time.LoadLocation(currentUser.Settings.Timezone)
	if err != nil {
		return nil, bp, fmt.Errorf("failed to load user timezone: %w", err)
	}

	// Calculate week boundaries in user timezone
	weekFirstDay := currentUser.Settings.WeekFirstDay
	earliestInTz := earliest.In(userTimezone)
	rangeStart := weekStart(earliestInTz, weekFirstDay)
	_, currentWeekEnd := weekTimeRange(s.clock.Now().In(userTimezone), weekFirstDay)

	// Fetch all calendar events in the range
	allEvents, err := s.calendarReader.GetEvents(ctx, rangeStart, currentWeekEnd)
	if err != nil {
		return nil, bp, fmt.Errorf("failed to get calendar events: %w", err)
	}

	// Filter events: only those whose budget_item_id is in this plan's current items
	budgetItemIdSet := make(map[int]bool, len(budgetItemIds))
	for _, id := range budgetItemIds {
		budgetItemIdSet[id] = true
	}

	// Group filtered events by week start date (in user timezone)
	eventsByWeek := make(map[time.Time][]calendar.Event)
	for _, e := range allEvents {
		if !budgetItemIdSet[e.Metadata.BudgetItemId] {
			continue
		}
		ws := weekStart(e.StartTime.In(userTimezone), weekFirstDay)
		eventsByWeek[ws] = append(eventsByWeek[ws], e)
	}

	// Build weekly entries
	var entries []WeeklyReportEntry
	for ws := rangeStart; !ws.After(currentWeekEnd); ws = ws.AddDate(0, 0, 7) {
		we := ws.AddDate(0, 0, 7).Add(-time.Nanosecond)

		weekNumber := weekly_plan.WeekNumberFromDate(ws, weekFirstDay)

		// Get weekly plan items for this week
		weeklyPlanItems, err := s.weeklyPlanReader.GetItemsForWeek(ctx, ws)
		if err != nil {
			// If no weekly plan exists, we'll use budget plan defaults
			weeklyPlanItems = nil
		}
		weeklyPlanMap := make(map[int]weekly_plan.WeeklyPlanItem, len(weeklyPlanItems))
		for _, wpi := range weeklyPlanItems {
			weeklyPlanMap[wpi.BudgetItemId] = wpi
		}

		// Aggregate event durations for this week per budget item
		eventDurations := make(map[int]time.Duration)
		for _, e := range eventsByWeek[ws] {
			eventDurations[e.Metadata.BudgetItemId] += e.EndTime.Sub(e.StartTime)
		}

		// Build report items for all budget items
		var totalBudget, totalWeekly, totalActual time.Duration
		items := make([]ReportItem, 0, len(bp.Items))
		for _, bi := range bp.Items {
			budgetPlanHours := bi.WeeklyDuration

			weeklyPlanHours := bi.WeeklyDuration // fallback
			if wpi, ok := weeklyPlanMap[bi.Id]; ok {
				weeklyPlanHours = wpi.WeeklyDuration
			}

			actualHours := eventDurations[bi.Id]

			items = append(items, ReportItem{
				BudgetItemId:   bi.Id,
				Name:           bi.Name,
				Icon:           bi.Icon,
				Color:          bi.Color,
				Position:       bi.Position,
				BudgetPlanTime: budgetPlanHours,
				WeeklyPlanTime: weeklyPlanHours,
				ActualTime:     actualHours,
			})

			totalBudget += budgetPlanHours
			totalWeekly += weeklyPlanHours
			totalActual += actualHours
		}

		entries = append(entries, WeeklyReportEntry{
			WeekNumber:          weekNumber.String(),
			StartDate:           ws,
			EndDate:             we,
			Items:               items,
			TotalBudgetPlanTime: totalBudget,
			TotalWeeklyPlanTime: totalWeekly,
			TotalActualTime:     totalActual,
		})
	}

	return entries, bp, nil
}

func aggregateMonthlyPeriod(periodNumber int, weeks []WeeklyReportEntry) MonthlyReportEntry {
	// Aggregate items across the weeks in this period
	itemTotals := make(map[int]*ReportItem)
	for _, week := range weeks {
		for _, item := range week.Items {
			if ri, ok := itemTotals[item.BudgetItemId]; ok {
				ri.BudgetPlanTime += item.BudgetPlanTime
				ri.WeeklyPlanTime += item.WeeklyPlanTime
				ri.ActualTime += item.ActualTime
			} else {
				copy := item
				itemTotals[item.BudgetItemId] = &copy
			}
		}
	}

	items := make([]ReportItem, 0, len(itemTotals))
	for _, ri := range itemTotals {
		items = append(items, *ri)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Position < items[j].Position
	})

	var totalBudget, totalWeekly, totalActual time.Duration
	for _, item := range items {
		totalBudget += item.BudgetPlanTime
		totalWeekly += item.WeeklyPlanTime
		totalActual += item.ActualTime
	}

	return MonthlyReportEntry{
		PeriodNumber:        periodNumber,
		StartDate:           weeks[0].StartDate,
		EndDate:             weeks[len(weeks)-1].EndDate,
		WeekCount:           len(weeks),
		Items:               items,
		TotalBudgetPlanTime: totalBudget,
		TotalWeeklyPlanTime: totalWeekly,
		TotalActualTime:     totalActual,
	}
}

func weekStart(date time.Time, weekStartDay time.Weekday) time.Time {
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	delta := (int(date.Weekday()) - int(weekStartDay) + 7) % 7
	return date.AddDate(0, 0, -delta)
}

func weekTimeRange(date time.Time, weekStartDay time.Weekday) (time.Time, time.Time) {
	ws := weekStart(date, weekStartDay)
	we := ws.AddDate(0, 0, 7).Add(-time.Nanosecond)
	return ws, we
}
