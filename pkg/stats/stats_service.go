package stats

import (
	"context"
	"time"

	"github.com/klokku/klokku/internal/utils"
	"github.com/klokku/klokku/pkg/calendar"
	"github.com/klokku/klokku/pkg/current_event"
	"github.com/klokku/klokku/pkg/user"
	"github.com/klokku/klokku/pkg/weekly_plan"
	log "github.com/sirupsen/logrus"
)

type StatsService interface {
	GetStats(ctx context.Context, weekTime time.Time) (StatsSummary, error)
}

type StatsServiceImpl struct {
	currentEventProvider currentEventProvider
	weeklyPlanService    weeklyPlanItemsReader
	calendar             calendarEventsReader
	clock                utils.Clock
}

type currentEventProvider interface {
	FindCurrentEvent(ctx context.Context) (current_event.CurrentEvent, error)
}

type weeklyPlanItemsReader interface {
	GetItemsForWeek(ctx context.Context, date time.Time) ([]weekly_plan.WeeklyPlanItem, error)
}

type calendarEventsReader interface {
	GetEvents(ctx context.Context, from time.Time, to time.Time) ([]calendar.Event, error)
}

func NewService(
	currentEventProvider currentEventProvider,
	weeklyPlanService weeklyPlanItemsReader,
	calendar calendarEventsReader,
) StatsService {
	return &StatsServiceImpl{
		currentEventProvider: currentEventProvider,
		weeklyPlanService:    weeklyPlanService,
		calendar:             calendar,
		clock:                &utils.SystemClock{},
	}
}

func (s *StatsServiceImpl) GetStats(ctx context.Context, weekTime time.Time) (StatsSummary, error) {
	currentUser, err := user.CurrentUser(ctx)
	if err != nil {
		return StatsSummary{}, err
	}

	// Currently supports only weekly stats. `weekTime` is used to find out which week.
	from, to := weekTimeRange(weekTime, currentUser.Settings.WeekFirstDay)

	planItems, err := s.weeklyPlanService.GetItemsForWeek(ctx, from)
	if err != nil {
		return StatsSummary{}, err
	}
	log.Tracef("Plan items: %v", planItems)

	totalPlanned := time.Duration(0)
	for _, item := range planItems {
		totalPlanned += item.WeeklyDuration
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
		return StatsSummary{}, err
	}
	eventsDurationPerDay := s.eventsDurationPerDay(calendarEvents)
	eventsDurationPerBudget := s.eventsDurationPerBudget(calendarEvents)

	statsByDate := make([]DailyStats, 0, len(eventsDurationPerDay))
	for date := from; !date.After(to); date = date.AddDate(0, 0, 1) {
		isToday := sameDays(s.clock.Now(), date, date.Location())
		todayCurrentEventTime := time.Duration(0)
		if isToday {
			todayCurrentEventTime = currentEventTime
		}

		dateBudgetDuration := eventsDurationPerDay[date]
		budgetsStats := prepareStatsByBudget(planItems, dateBudgetDuration, currentEventBudgetItemId, todayCurrentEventTime)
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
	)

	totalTime := time.Duration(0)
	for _, budgetDuration := range eventsDurationPerBudget {
		totalTime += budgetDuration
	}
	totalTime += currentEventTime

	return StatsSummary{
		StartDate:      from,
		EndDate:        to,
		PerDay:         statsByDate,
		PerPlanItem:    statsByBudget,
		TotalPlanned:   totalPlanned,
		TotalTime:      totalTime,
		TotalRemaining: totalPlanned - totalTime,
	}, nil
}

func (s *StatsServiceImpl) eventsDurationPerDay(events []calendar.Event) map[time.Time]map[int]time.Duration {
	eventsByDate := make(map[time.Time]map[int]time.Duration)
	for _, e := range events {
		loc := e.StartTime.Location()
		year, month, day := e.StartTime.In(loc).Date()
		date := time.Date(year, month, day, 0, 0, 0, 0, loc)
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
	planItems []weekly_plan.WeeklyPlanItem,
	durationByBudgetId map[int]time.Duration,
	currentEventBudgetItemId int,
	currentEventTime time.Duration,
) []PlanItemStats {

	statsByBudget := make([]PlanItemStats, 0, len(planItems))
	for _, item := range planItems {
		budgetDuration := durationByBudgetId[item.BudgetItemId]
		budgetCurrentEventTime := time.Duration(0)
		if item.BudgetItemId == currentEventBudgetItemId {
			budgetCurrentEventTime = currentEventTime
		}

		budgetStats := PlanItemStats{
			PlanItem:  item,
			Duration:  budgetDuration + budgetCurrentEventTime,
			Remaining: calculateRemainingDuration(&item, budgetDuration) - budgetCurrentEventTime,
		}
		statsByBudget = append(statsByBudget, budgetStats)
	}
	return statsByBudget
}

func calculateRemainingDuration(
	planItem *weekly_plan.WeeklyPlanItem,
	duration time.Duration,
) time.Duration {
	return planItem.WeeklyDuration - duration
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

	delta := (int(date.Weekday()) - int(weekStartDay) + 7) % 7
	weekStart := date.AddDate(0, 0, -delta)
	weekEnd := weekStart.AddDate(0, 0, 7).Add(-time.Nanosecond)
	return weekStart, weekEnd
}
