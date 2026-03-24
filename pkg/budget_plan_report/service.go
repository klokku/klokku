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
	GetItemReport(ctx context.Context, planId int, itemId int, from *time.Time, to *time.Time) (ItemDetailReport, error)
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
	GetPlanForWeek(ctx context.Context, date time.Time) (weekly_plan.WeeklyPlan, error)
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
	var excludedWeekCount int
	for ws := rangeStart; !ws.After(rangeEnd); ws = ws.AddDate(0, 0, 7) {
		we := ws.AddDate(0, 0, 7).Add(-time.Nanosecond)
		weekNumber := weekly_plan.WeekNumberFromDate(ws, weekFirstDay)

		weeklyPlan, err := s.weeklyPlanReader.GetPlanForWeek(ctx, ws)
		if err == nil && weeklyPlan.IsOffWeek {
			excludedWeekCount++
			continue
		}

		var weeklyPlanItems []weekly_plan.WeeklyPlanItem
		if err == nil {
			weeklyPlanItems = weeklyPlan.Items
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
		ExcludedWeekCount:   excludedWeekCount,
		Weeks:               weeks,
		TotalItems:          totalItems,
		TotalBudgetPlanTime: totalBudget,
		TotalWeeklyPlanTime: totalWeekly,
		TotalActualTime:     totalActual,
	}, nil
}

func (s *ServiceImpl) GetItemReport(ctx context.Context, planId int, itemId int, from *time.Time, to *time.Time) (ItemDetailReport, error) {
	currentUser, err := user.CurrentUser(ctx)
	if err != nil {
		return ItemDetailReport{}, fmt.Errorf("failed to get current user: %w", err)
	}

	bp, err := s.budgetPlanReader.GetPlan(ctx, planId)
	if err != nil {
		return ItemDetailReport{}, fmt.Errorf("failed to get budget plan: %w", err)
	}

	// Find the budget item in the plan
	var budgetItem *budget_plan.BudgetItem
	for i := range bp.Items {
		if bp.Items[i].Id == itemId {
			budgetItem = &bp.Items[i]
			break
		}
	}
	if budgetItem == nil {
		return ItemDetailReport{}, fmt.Errorf("budget item %d not found in plan %d", itemId, planId)
	}

	// Find earliest event for this item
	earliest, found, err := s.earliestEventFinder.GetEarliestEventTimeForBudgetItems(ctx, []int{itemId})
	if err != nil {
		return ItemDetailReport{}, fmt.Errorf("failed to find earliest event: %w", err)
	}
	if !found {
		return ItemDetailReport{
			PlanId:    bp.Id,
			PlanName:  bp.Name,
			ItemId:    budgetItem.Id,
			ItemName:  budgetItem.Name,
			ItemIcon:  budgetItem.Icon,
			ItemColor: budgetItem.Color,
		}, nil
	}

	userTimezone, err := time.LoadLocation(currentUser.Settings.Timezone)
	if err != nil {
		return ItemDetailReport{}, fmt.Errorf("failed to load user timezone: %w", err)
	}
	weekFirstDay := currentUser.Settings.WeekFirstDay

	// Calculate range
	var rangeStart, rangeEnd time.Time
	if from != nil && to != nil {
		rangeStart = weekStart(from.In(userTimezone), weekFirstDay)
		_, rangeEnd = weekTimeRange(to.In(userTimezone), weekFirstDay)
	} else {
		rangeStart = weekStart(earliest.In(userTimezone), weekFirstDay)
		_, rangeEnd = weekTimeRange(s.clock.Now().In(userTimezone), weekFirstDay)
	}

	// Fetch and filter events to this item only
	allEvents, err := s.calendarReader.GetEvents(ctx, rangeStart, rangeEnd)
	if err != nil {
		return ItemDetailReport{}, fmt.Errorf("failed to get calendar events: %w", err)
	}

	var itemEvents []calendar.Event
	for _, e := range allEvents {
		if e.Metadata.BudgetItemId == itemId {
			itemEvents = append(itemEvents, e)
		}
	}

	// Group events by week and by day
	eventsByWeek := make(map[time.Time][]calendar.Event)
	dailyDurations := make(map[time.Time]time.Duration)
	for _, e := range itemEvents {
		eventInTz := e.StartTime.In(userTimezone)
		ws := weekStart(eventInTz, weekFirstDay)
		eventsByWeek[ws] = append(eventsByWeek[ws], e)

		dayKey := time.Date(eventInTz.Year(), eventInTz.Month(), eventInTz.Day(), 0, 0, 0, 0, userTimezone)
		dailyDurations[dayKey] += e.EndTime.Sub(e.StartTime)
	}

	// Build weekly entries (including off-weeks)
	var weeks []ItemWeekEntry
	var excludedWeekCount int
	var totalActual, totalBudget, totalWeekly time.Duration
	var weeklyActualTimes []time.Duration

	for ws := rangeStart; !ws.After(rangeEnd); ws = ws.AddDate(0, 0, 7) {
		we := ws.AddDate(0, 0, 7).Add(-time.Nanosecond)
		weekNumber := weekly_plan.WeekNumberFromDate(ws, weekFirstDay)

		weeklyPlan, wpErr := s.weeklyPlanReader.GetPlanForWeek(ctx, ws)
		isOff := wpErr == nil && weeklyPlan.IsOffWeek

		if isOff {
			excludedWeekCount++
			weeks = append(weeks, ItemWeekEntry{
				WeekNumber: weekNumber.String(),
				StartDate:  ws,
				EndDate:    we,
				IsOffWeek:  true,
			})
			continue
		}

		budgetPlanTime := budgetItem.WeeklyDuration
		weeklyPlanTime := budgetItem.WeeklyDuration
		if wpErr == nil {
			for _, wpi := range weeklyPlan.Items {
				if wpi.BudgetItemId == itemId {
					weeklyPlanTime = wpi.WeeklyDuration
					break
				}
			}
		}

		var actualTime time.Duration
		for _, e := range eventsByWeek[ws] {
			actualTime += e.EndTime.Sub(e.StartTime)
		}

		weeks = append(weeks, ItemWeekEntry{
			WeekNumber:     weekNumber.String(),
			StartDate:      ws,
			EndDate:        we,
			BudgetPlanTime: budgetPlanTime,
			WeeklyPlanTime: weeklyPlanTime,
			ActualTime:     actualTime,
			IsOffWeek:      false,
		})

		totalActual += actualTime
		totalBudget += budgetPlanTime
		totalWeekly += weeklyPlanTime
		weeklyActualTimes = append(weeklyActualTimes, actualTime)
	}

	// Build daily entries
	days := make([]ItemDayEntry, 0, len(dailyDurations))
	// We need to exclude days that fall in off-weeks
	offWeekStarts := make(map[time.Time]bool)
	for _, w := range weeks {
		if w.IsOffWeek {
			offWeekStarts[w.StartDate] = true
		}
	}

	for dayKey, dur := range dailyDurations {
		ws := weekStart(dayKey, weekFirstDay)
		if offWeekStarts[ws] {
			continue // skip days in off-weeks
		}
		days = append(days, ItemDayEntry{
			Date:       dayKey,
			ActualTime: dur,
			DayOfWeek:  dayKey.Weekday(),
		})
	}
	sort.Slice(days, func(i, j int) bool {
		return days[i].Date.Before(days[j].Date)
	})

	// Count total days in the period (excluding off-weeks)
	totalDaysCount := 0
	for _, w := range weeks {
		if !w.IsOffWeek {
			// Count actual days in this week that fall within the range
			for d := w.StartDate; !d.After(w.EndDate) && !d.After(rangeEnd); d = d.AddDate(0, 0, 1) {
				if !d.Before(rangeStart) {
					totalDaysCount++
				}
			}
		}
	}

	activeDaysCount := len(days)
	activeWeekCount := len(weeklyActualTimes)

	// Compute averages
	var avgPerDay, avgPerActiveDay, avgPerWeek time.Duration
	if totalDaysCount > 0 {
		avgPerDay = totalActual / time.Duration(totalDaysCount)
	}
	if activeDaysCount > 0 {
		avgPerActiveDay = totalActual / time.Duration(activeDaysCount)
	}
	if activeWeekCount > 0 {
		avgPerWeek = totalActual / time.Duration(activeWeekCount)
	}

	// Compute medians
	// All daily durations (including zero-days in non-off-weeks)
	allDailyDurations := make([]time.Duration, 0, totalDaysCount)
	activeDailyDurations := make([]time.Duration, 0, activeDaysCount)
	for _, w := range weeks {
		if w.IsOffWeek {
			continue
		}
		for d := w.StartDate; !d.After(w.EndDate) && !d.After(rangeEnd); d = d.AddDate(0, 0, 1) {
			if d.Before(rangeStart) {
				continue
			}
			dur := dailyDurations[d]
			if offWeekStarts[weekStart(d, weekFirstDay)] {
				continue
			}
			allDailyDurations = append(allDailyDurations, dur)
			if dur > 0 {
				activeDailyDurations = append(activeDailyDurations, dur)
			}
		}
	}
	sort.Slice(allDailyDurations, func(i, j int) bool { return allDailyDurations[i] < allDailyDurations[j] })
	sort.Slice(activeDailyDurations, func(i, j int) bool { return activeDailyDurations[i] < activeDailyDurations[j] })
	sort.Slice(weeklyActualTimes, func(i, j int) bool { return weeklyActualTimes[i] < weeklyActualTimes[j] })

	medianPerDay := medianDuration(allDailyDurations)
	medianPerActiveDay := medianDuration(activeDailyDurations)
	medianPerWeek := medianDuration(weeklyActualTimes)

	// Completion and remaining/over-budget
	var completionPercent float64
	if totalBudget > 0 {
		completionPercent = float64(totalActual) / float64(totalBudget) * 100
	}
	var remainingTime, overBudgetTime time.Duration
	if totalActual < totalBudget {
		remainingTime = totalBudget - totalActual
	} else {
		overBudgetTime = totalActual - totalBudget
	}

	// Day-of-week averages
	dayOfWeekAvg := computeDayOfWeekAverages(weeks, dailyDurations, weekFirstDay, rangeStart, rangeEnd)

	startDate := rangeStart
	endDate := rangeEnd
	if len(weeks) > 0 {
		startDate = weeks[0].StartDate
		endDate = weeks[len(weeks)-1].EndDate
	}

	return ItemDetailReport{
		PlanId:              bp.Id,
		PlanName:            bp.Name,
		ItemId:              budgetItem.Id,
		ItemName:            budgetItem.Name,
		ItemIcon:            budgetItem.Icon,
		ItemColor:           budgetItem.Color,
		StartDate:           startDate,
		EndDate:             endDate,
		TotalActualTime:     totalActual,
		TotalBudgetPlanTime: totalBudget,
		TotalWeeklyPlanTime: totalWeekly,
		CompletionPercent:   completionPercent,
		RemainingTime:       remainingTime,
		OverBudgetTime:      overBudgetTime,
		AveragePerDay:       avgPerDay,
		AveragePerActiveDay: avgPerActiveDay,
		AveragePerWeek:      avgPerWeek,
		MedianPerDay:        medianPerDay,
		MedianPerActiveDay:  medianPerActiveDay,
		MedianPerWeek:       medianPerWeek,
		ActiveDaysCount:     activeDaysCount,
		TotalDaysCount:      totalDaysCount,
		WeekCount:           activeWeekCount,
		ExcludedWeekCount:   excludedWeekCount,
		Weeks:               weeks,
		Days:                days,
		DayOfWeekAvg:        dayOfWeekAvg,
	}, nil
}

func computeDayOfWeekAverages(weeks []ItemWeekEntry, dailyDurations map[time.Time]time.Duration, weekFirstDay time.Weekday, rangeStart, rangeEnd time.Time) []DayOfWeekEntry {
	// For each weekday, sum durations and count how many non-off-weeks contain that weekday
	type dayStats struct {
		totalDuration time.Duration
		count         int
	}
	stats := make(map[time.Weekday]*dayStats)
	for wd := time.Sunday; wd <= time.Saturday; wd++ {
		stats[wd] = &dayStats{}
	}

	for _, w := range weeks {
		if w.IsOffWeek {
			continue
		}
		for d := w.StartDate; !d.After(w.EndDate) && !d.After(rangeEnd); d = d.AddDate(0, 0, 1) {
			if d.Before(rangeStart) {
				continue
			}
			wd := d.Weekday()
			stats[wd].count++
			stats[wd].totalDuration += dailyDurations[d]
		}
	}

	result := make([]DayOfWeekEntry, 0, 7)
	// Order starting from weekFirstDay
	for i := 0; i < 7; i++ {
		wd := time.Weekday((int(weekFirstDay) + i) % 7)
		s := stats[wd]
		var avg time.Duration
		if s.count > 0 {
			avg = s.totalDuration / time.Duration(s.count)
		}
		result = append(result, DayOfWeekEntry{
			DayOfWeek:   wd,
			AverageTime: avg,
			TotalTime:   s.totalDuration,
		})
	}
	return result
}

func medianDuration(sorted []time.Duration) time.Duration {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
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
