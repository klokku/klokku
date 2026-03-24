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
	GetReport(ctx context.Context, planId int, from *time.Time, to *time.Time) (Report, error)
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

func (s *ServiceImpl) GetReport(ctx context.Context, planId int, from *time.Time, to *time.Time) (Report, error) {
	currentUser, err := user.CurrentUser(ctx)
	if err != nil {
		return Report{}, fmt.Errorf("failed to get current user: %w", err)
	}

	bp, err := s.budgetPlanReader.GetPlan(ctx, planId)
	if err != nil {
		return Report{}, fmt.Errorf("failed to get budget plan: %w", err)
	}

	if len(bp.Items) == 0 {
		return Report{PlanId: bp.Id, PlanName: bp.Name}, nil
	}

	// Collect budget item IDs
	budgetItemIds := make([]int, len(bp.Items))
	for i, item := range bp.Items {
		budgetItemIds[i] = item.Id
	}

	// Find the earliest event
	earliest, found, err := s.earliestEventFinder.GetEarliestEventTimeForBudgetItems(ctx, budgetItemIds)
	if err != nil {
		return Report{}, fmt.Errorf("failed to find earliest event: %w", err)
	}
	if !found {
		return Report{PlanId: bp.Id, PlanName: bp.Name}, nil
	}

	// Load user timezone
	userTimezone, err := time.LoadLocation(currentUser.Settings.Timezone)
	if err != nil {
		return Report{}, fmt.Errorf("failed to load user timezone: %w", err)
	}

	weekFirstDay := currentUser.Settings.WeekFirstDay

	// Calculate range boundaries
	var rangeStart, rangeEnd time.Time
	if from != nil && to != nil {
		rangeStart = weekStart(from.In(userTimezone), weekFirstDay)
		_, rangeEnd = weekTimeRange(to.In(userTimezone), weekFirstDay)
	} else {
		rangeStart = weekStart(earliest.In(userTimezone), weekFirstDay)
		_, rangeEnd = weekTimeRange(s.clock.Now().In(userTimezone), weekFirstDay)
	}

	// Fetch all calendar events in the range
	allEvents, err := s.calendarReader.GetEvents(ctx, rangeStart, rangeEnd)
	if err != nil {
		return Report{}, fmt.Errorf("failed to get calendar events: %w", err)
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
	var weeks []WeeklyReportEntry
	for ws := rangeStart; !ws.After(rangeEnd); ws = ws.AddDate(0, 0, 7) {
		we := ws.AddDate(0, 0, 7).Add(-time.Nanosecond)
		weekNumber := weekly_plan.WeekNumberFromDate(ws, weekFirstDay)

		weeklyPlanItems, err := s.weeklyPlanReader.GetItemsForWeek(ctx, ws)
		if err != nil {
			weeklyPlanItems = nil
		}
		weeklyPlanMap := make(map[int]weekly_plan.WeeklyPlanItem, len(weeklyPlanItems))
		for _, wpi := range weeklyPlanItems {
			weeklyPlanMap[wpi.BudgetItemId] = wpi
		}

		eventDurations := make(map[int]time.Duration)
		for _, e := range eventsByWeek[ws] {
			eventDurations[e.Metadata.BudgetItemId] += e.EndTime.Sub(e.StartTime)
		}

		var totalBudget, totalWeekly, totalActual time.Duration
		items := make([]ReportItem, 0, len(bp.Items))
		for _, bi := range bp.Items {
			budgetPlanTime := bi.WeeklyDuration
			weeklyPlanTime := bi.WeeklyDuration
			if wpi, ok := weeklyPlanMap[bi.Id]; ok {
				weeklyPlanTime = wpi.WeeklyDuration
			}
			actualTime := eventDurations[bi.Id]

			items = append(items, ReportItem{
				BudgetItemId:   bi.Id,
				Name:           bi.Name,
				Icon:           bi.Icon,
				Color:          bi.Color,
				Position:       bi.Position,
				BudgetPlanTime: budgetPlanTime,
				WeeklyPlanTime: weeklyPlanTime,
				ActualTime:     actualTime,
			})
			totalBudget += budgetPlanTime
			totalWeekly += weeklyPlanTime
			totalActual += actualTime
		}

		weeks = append(weeks, WeeklyReportEntry{
			WeekNumber:          weekNumber.String(),
			StartDate:           ws,
			EndDate:             we,
			Items:               items,
			TotalBudgetPlanTime: totalBudget,
			TotalWeeklyPlanTime: totalWeekly,
			TotalActualTime:     totalActual,
		})
	}

	// Aggregate totals
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
	for _, week := range weeks {
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

	occurrencesById := make(map[int]int, len(bp.Items))
	for _, bi := range bp.Items {
		occurrencesById[bi.Id] = bi.WeeklyOccurrences
	}

	weekCount := len(weeks)
	totalItems := make([]ReportItem, 0, len(itemTotals))
	for _, ri := range itemTotals {
		if weekCount > 0 {
			ri.AveragePerWeek = ri.ActualTime / time.Duration(weekCount)
			occ := occurrencesById[ri.BudgetItemId]
			if occ == 0 {
				occ = 7
			}
			ri.AveragePerDay = ri.ActualTime / time.Duration(weekCount*occ)
		}
		totalItems = append(totalItems, *ri)
	}
	sort.Slice(totalItems, func(i, j int) bool {
		return totalItems[i].Position < totalItems[j].Position
	})

	startDate := rangeStart
	endDate := rangeEnd
	if len(weeks) > 0 {
		startDate = weeks[0].StartDate
		endDate = weeks[len(weeks)-1].EndDate
	}

	return Report{
		PlanId:              bp.Id,
		PlanName:            bp.Name,
		StartDate:           startDate,
		EndDate:             endDate,
		WeekCount:           len(weeks),
		Weeks:               weeks,
		TotalItems:          totalItems,
		TotalBudgetPlanTime: totalBudget,
		TotalWeeklyPlanTime: totalWeekly,
		TotalActualTime:     totalActual,
	}, nil
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
