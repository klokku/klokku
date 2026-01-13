package stats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/klokku/klokku/internal/utils"
	"github.com/klokku/klokku/pkg/budget_plan"
	"github.com/klokku/klokku/pkg/calendar"
	"github.com/klokku/klokku/pkg/current_event"
	"github.com/klokku/klokku/pkg/user"
	"github.com/klokku/klokku/pkg/weekly_plan"
	log "github.com/sirupsen/logrus"
)

var ErrPlanItemNotFound = fmt.Errorf("plan item not found")
var ErrNoStatsFound = fmt.Errorf("no stats found")

type StatsService interface {
	GetWeeklyStats(ctx context.Context, weekTime time.Time) (WeeklyStatsSummary, error)
	GetPlanItemByWeekHistoryStats(
		ctx context.Context,
		from time.Time,
		to time.Time,
		budgetItemId int,
	) (PlanItemHistoryStats, error)
}

type StatsServiceImpl struct {
	currentEventProvider currentEventProvider
	weeklyPlanService    weeklyPlanItemsReader
	budgetPlanService    budgetPlanReader
	calendar             calendarEventsReader
	clock                utils.Clock
}

type currentEventProvider interface {
	FindCurrentEvent(ctx context.Context) (current_event.CurrentEvent, error)
}

type weeklyPlanItemsReader interface {
	GetItemsForWeek(ctx context.Context, date time.Time) ([]weekly_plan.WeeklyPlanItem, error)
}

type budgetPlanReader interface {
	GetPlan(ctx context.Context, planId int) (budget_plan.BudgetPlan, error)
	GetItem(ctx context.Context, id int) (budget_plan.BudgetItem, error)
}

type calendarEventsReader interface {
	GetEvents(ctx context.Context, from time.Time, to time.Time) ([]calendar.Event, error)
}

func NewService(
	currentEventProvider currentEventProvider,
	weeklyPlanService weeklyPlanItemsReader,
	budgetPlanService budgetPlanReader,
	calendar calendarEventsReader,
) StatsService {
	return &StatsServiceImpl{
		currentEventProvider: currentEventProvider,
		weeklyPlanService:    weeklyPlanService,
		budgetPlanService:    budgetPlanService,
		calendar:             calendar,
		clock:                &utils.SystemClock{},
	}
}

func (s *StatsServiceImpl) GetWeeklyStats(ctx context.Context, weekTime time.Time) (WeeklyStatsSummary, error) {
	currentUser, err := user.CurrentUser(ctx)
	if err != nil {
		return WeeklyStatsSummary{}, err
	}

	// Currently supports only weekly stats. `weekTime` is used to find out which week.
	from, to := weekTimeRange(weekTime, currentUser.Settings.WeekFirstDay)

	weeklyItems, err := s.weeklyPlanService.GetItemsForWeek(ctx, from)
	if err != nil {
		if errors.Is(err, weekly_plan.ErrNoCurrentPlan) {
			return WeeklyStatsSummary{}, ErrNoStatsFound
		}
		return WeeklyStatsSummary{}, err
	}
	log.Tracef("Plan items: %v", weeklyItems)

	if len(weeklyItems) == 0 {
		return WeeklyStatsSummary{}, nil
	}

	budgetPlan, err := s.budgetPlanService.GetPlan(ctx, weeklyItems[0].BudgetPlanId)
	if err != nil {
		return WeeklyStatsSummary{}, err
	}

	planItems := make([]PlanItem, 0, len(weeklyItems))
	for _, item := range weeklyItems {
		var budgetItem budget_plan.BudgetItem
		for _, bi := range budgetPlan.Items {
			if item.BudgetItemId == bi.Id {
				budgetItem = bi
				break
			}
		}
		planItems = append(planItems, combinePlanItemData(item, budgetItem))
	}

	totalPlanned := time.Duration(0)
	for _, item := range planItems {
		totalPlanned += item.WeeklyItemDuration
	}

	currentEventBudgetItemId := 0
	currentEventTime := time.Duration(0)
	if s.clock.Now().After(from) && s.clock.Now().Before(to) {
		log.Debugf("Calculating stats for current week. Taking into account current event if any.")
		currentEvent, err := s.currentEventProvider.FindCurrentEvent(ctx)
		if err != nil {
			log.Warnf("Unable to find current event: %v. Stats will not include current event.", err)
		}
		if currentEvent.Id != 0 { // current event exists
			currentEventBudgetItemId = currentEvent.PlanItem.BudgetItemId
			currentEventTime = s.clock.Now().Sub(currentEvent.StartTime)
		}
	}

	calendarEvents, err := s.calendar.GetEvents(ctx, from, to)
	if err != nil {
		return WeeklyStatsSummary{}, err
	}
	userTimezone, err := time.LoadLocation(currentUser.Settings.Timezone)
	if err != nil {
		return WeeklyStatsSummary{}, fmt.Errorf("failed to load user timezone: %w", err)
	}
	eventsDurationPerDay := s.eventsDurationPerDay(calendarEvents, userTimezone)
	eventsDurationPerBudget := s.eventsDurationPerBudget(calendarEvents)

	statsByDate := make([]DailyStats, 0, len(eventsDurationPerDay))
	for date := from; !date.After(to); date = date.AddDate(0, 0, 1) {
		isToday := sameDays(s.clock.Now(), date, userTimezone)
		todayCurrentEventTime := time.Duration(0)
		if isToday {
			todayCurrentEventTime = currentEventTime
		}

		// Normalize loop date to user timezone midnight for map lookup
		lookupDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, userTimezone)
		dateBudgetDuration := eventsDurationPerDay[lookupDate]

		budgetsStats := prepareStatsByBudget(
			planItems,
			dateBudgetDuration,
			currentEventBudgetItemId,
			todayCurrentEventTime,
			date,
			date.AddDate(0, 0, 1),
		)
		dateTotalTime := time.Duration(0)
		for _, budgetStat := range budgetsStats {
			dateTotalTime += budgetStat.Duration
		}

		dailyStats := DailyStats{date, budgetsStats, dateTotalTime}
		statsByDate = append(statsByDate, dailyStats)
	}

	statsByBudget := prepareStatsByBudget(
		planItems,
		eventsDurationPerBudget,
		currentEventBudgetItemId,
		currentEventTime,
		from,
		to,
	)

	totalTime := time.Duration(0)
	for _, budgetDuration := range eventsDurationPerBudget {
		totalTime += budgetDuration
	}
	totalTime += currentEventTime

	return WeeklyStatsSummary{
		StartDate:      from,
		EndDate:        to,
		PerDay:         statsByDate,
		PerPlanItem:    statsByBudget,
		TotalPlanned:   totalPlanned,
		TotalTime:      totalTime,
		TotalRemaining: totalPlanned - totalTime,
	}, nil
}

func (s *StatsServiceImpl) eventsDurationPerDay(events []calendar.Event, userTimezone *time.Location) map[time.Time]map[int]time.Duration {
	eventsByDate := make(map[time.Time]map[int]time.Duration)
	for _, e := range events {
		// Use user timezone midnight for the map key to avoid location pointer mismatches
		t := e.StartTime.In(userTimezone)
		date := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, userTimezone)

		if eventsByDate[date] == nil {
			eventsByDate[date] = make(map[int]time.Duration)
		}
		eventsByDate[date][e.Metadata.BudgetItemId] += duration(e)
	}
	return eventsByDate
}

func (s *StatsServiceImpl) eventsDurationPerBudget(events []calendar.Event) map[int]time.Duration {
	eventsByBudget := make(map[int]time.Duration)
	for _, e := range events {
		eventsByBudget[e.Metadata.BudgetItemId] += duration(e)
	}
	return eventsByBudget
}

func duration(event calendar.Event) time.Duration {
	return event.EndTime.Sub(event.StartTime)
}

func prepareStatsByBudget(
	planItems []PlanItem,
	durationByBudgetItemId map[int]time.Duration,
	currentEventBudgetItemId int,
	currentEventTime time.Duration,
	startDate time.Time,
	endDate time.Time,
) []PlanItemStats {

	statsByBudget := make([]PlanItemStats, 0, len(planItems))
	for _, item := range planItems {
		budgetDuration := durationByBudgetItemId[item.BudgetItemId]
		budgetCurrentEventTime := time.Duration(0)
		if item.BudgetItemId == currentEventBudgetItemId {
			budgetCurrentEventTime = currentEventTime
		}

		budgetStats := PlanItemStats{
			PlanItem:  item,
			Duration:  budgetDuration + budgetCurrentEventTime,
			Remaining: calculateRemainingDuration(&item, budgetDuration) - budgetCurrentEventTime,
			StartDate: startDate,
			EndDate:   endDate,
		}
		statsByBudget = append(statsByBudget, budgetStats)
	}
	return statsByBudget
}

func calculateRemainingDuration(
	planItem *PlanItem,
	duration time.Duration,
) time.Duration {
	return planItem.WeeklyItemDuration - duration
}

func sameDays(date1, date2 time.Time, loc *time.Location) bool {
	year1, month1, day1 := date1.In(loc).Date()
	year2, month2, day2 := date2.In(loc).Date()
	return year1 == year2 && month1 == month2 && day1 == day2
}

func weekTimeRange(date time.Time, weekStartDay time.Weekday) (time.Time, time.Time) {
	if weekStartDay < time.Sunday || weekStartDay > time.Saturday {
		weekStartDay = time.Monday
	}

	// Normalize to midnight in the original location
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	delta := (int(date.Weekday()) - int(weekStartDay) + 7) % 7
	weekStart := date.AddDate(0, 0, -delta)
	// Ensure weekStart is also a clean midnight (redundant but safe)
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())

	weekEnd := weekStart.AddDate(0, 0, 7).Add(-time.Nanosecond)
	return weekStart, weekEnd
}

func (s *StatsServiceImpl) GetPlanItemByWeekHistoryStats(
	ctx context.Context,
	from time.Time,
	to time.Time,
	budgetItemId int,
) (PlanItemHistoryStats, error) {
	currentUser, err := user.CurrentUser(ctx)
	if err != nil {
		return PlanItemHistoryStats{}, err
	}

	// Normalize "from" date to week start
	weekStart, _ := weekTimeRange(from, currentUser.Settings.WeekFirstDay)
	// Normalize "to" date to the end of the week
	_, weekEnd := weekTimeRange(to, currentUser.Settings.WeekFirstDay)

	var historyStats []PlanItemStats
	for startDate := weekStart; !startDate.After(weekEnd); startDate = startDate.AddDate(0, 0, 7) {

		items, err := s.weeklyPlanService.GetItemsForWeek(ctx, startDate)
		if err != nil {
			log.Errorf("Failed to get weekly plan items for date %v: %v", startDate, err)
			return PlanItemHistoryStats{}, err
		}
		var weeklyItem weekly_plan.WeeklyPlanItem
		for _, item := range items {
			if item.BudgetItemId == budgetItemId {
				weeklyItem = item
				break
			}
		}
		if weeklyItem.BudgetItemId == 0 {
			return PlanItemHistoryStats{}, ErrPlanItemNotFound
		}
		budgetItem, err := s.budgetPlanService.GetItem(ctx, weeklyItem.BudgetItemId)
		if err != nil {
			return PlanItemHistoryStats{}, err
		}

		planItem := combinePlanItemData(weeklyItem, budgetItem)

		endDate := startDate.AddDate(0, 0, 7).Add(-time.Nanosecond)
		calendarEvents, err := s.calendar.GetEvents(ctx, startDate, endDate)
		if err != nil {
			return PlanItemHistoryStats{}, err
		}

		eventsDurationPerBudget := s.eventsDurationPerBudget(calendarEvents)

		historyStats = append(historyStats, PlanItemStats{
			PlanItem:  planItem,
			Duration:  eventsDurationPerBudget[budgetItemId],
			Remaining: weeklyItem.WeeklyDuration - eventsDurationPerBudget[budgetItemId],
			StartDate: startDate,
			EndDate:   endDate,
		})
	}

	return PlanItemHistoryStats{
		StartDate:    weekStart,
		EndDate:      weekEnd,
		StatsPerWeek: historyStats,
	}, nil
}

func combinePlanItemData(weeklyItem weekly_plan.WeeklyPlanItem, budgetItem budget_plan.BudgetItem) PlanItem {
	return PlanItem{
		BudgetPlanId:       budgetItem.PlanId,
		BudgetItemId:       budgetItem.Id,
		WeeklyItemId:       weeklyItem.Id,
		Name:               weeklyItem.Name,
		Icon:               weeklyItem.Icon,
		Color:              weeklyItem.Color,
		Position:           weeklyItem.Position,
		WeeklyItemDuration: weeklyItem.WeeklyDuration,
		BudgetItemDuration: budgetItem.WeeklyDuration,
		WeeklyOccurrences:  weeklyItem.WeeklyOccurrences,
		Notes:              weeklyItem.Notes,
	}
}
