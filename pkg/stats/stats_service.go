package stats

import (
	"context"
	"fmt"
	"github.com/klokku/klokku/internal/utils"
	"github.com/klokku/klokku/pkg/budget"
	"github.com/klokku/klokku/pkg/budget_override"
	"github.com/klokku/klokku/pkg/event"
	"github.com/klokku/klokku/pkg/user"
	log "github.com/sirupsen/logrus"
	"time"
)

type StatsService interface {
	GetStats(ctx context.Context, from time.Time, to time.Time) (StatsSummary, error)
}

type StatsServiceImpl struct {
	eventService       event.EventService
	eventStatsService  event.EventStatsService
	budgetRepo         budget.BudgetRepo
	budgetOverrideRepo budget_override.BudgetOverrideRepo
	clock              utils.Clock
}

func NewStatsServiceImpl(
	eventService event.EventService,
	eventStatsService event.EventStatsService,
	budgetRepo budget.BudgetRepo,
	budgetOverrideRepo budget_override.BudgetOverrideRepo,
) *StatsServiceImpl {
	return &StatsServiceImpl{
		eventService:       eventService,
		eventStatsService:  eventStatsService,
		budgetRepo:         budgetRepo,
		budgetOverrideRepo: budgetOverrideRepo,
		clock:              &utils.SystemClock{},
	}
}

func (s *StatsServiceImpl) GetStats(ctx context.Context, from time.Time, to time.Time) (StatsSummary, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return StatsSummary{}, fmt.Errorf("failed to get current user: %w", err)
	}
	budgets, err := s.budgetRepo.GetAll(ctx, userId, true)
	if err != nil {
		return StatsSummary{}, err
	}
	log.Tracef("Budgets: %v", budgets)

	overridesByBudgetId, err := s.getAllBudgetOverrides(ctx, from)
	if err != nil {
		return StatsSummary{}, err
	}
	log.Tracef("Budget overrides: %v", overridesByBudgetId)

	totalPlanned := time.Duration(0)
	for _, b := range budgets {
		if b.Status != budget.BudgetStatusActive {
			continue
		}
		budgetTime := b.WeeklyTime
		if overridesByBudgetId[b.ID] != nil {
			budgetTime = overridesByBudgetId[b.ID].WeeklyTime
		}
		log.Debugf("Budget: %v, Time: %v, TimeWithOverride: %v", b.Name, b.WeeklyTime, budgetTime)
		totalPlanned += budgetTime
	}

	currentEventBudgetId := 0
	currentEventTime := time.Duration(0)
	if s.clock.Now().After(from) && s.clock.Now().Before(to) {
		log.Debugf("Calculating stats for current week. Taking into account current event if any.")
		currentEvent, err := s.eventService.FindCurrentEvent(ctx)
		if err != nil {
			log.Warnf("Unable to find current event: %v. Stats will not include current event.", err)
		}
		if currentEvent != nil {
			currentEventBudgetId = currentEvent.Budget.ID
			currentEventTime = s.clock.Now().Sub(currentEvent.StartTime)
		}
	}

	eventStats, err := s.eventStatsService.GetStats(ctx, from, to)
	if err != nil {
		return StatsSummary{}, err
	}

	statsByDate := make([]DailyStats, 0, len(eventStats.ByDate))
	for date := from; !date.After(to); date = date.AddDate(0, 0, 1) {
		isToday := sameDays(s.clock.Now(), date, date.Location())
		todayCurrentEventTime := time.Duration(0)
		if isToday {
			todayCurrentEventTime = currentEventTime
		}

		dateBudgetDuration := eventStats.ByDate[date]
		budgetsStats := prepareStatsByBudget(budgets, nil, dateBudgetDuration, currentEventBudgetId, todayCurrentEventTime)
		dateTotalTime := time.Duration(0)
		for _, budgetStat := range budgetsStats {
			dateTotalTime += budgetStat.Duration
		}

		dailyStats := DailyStats{date, budgetsStats, dateTotalTime}
		statsByDate = append(statsByDate, dailyStats)
	}

	statsByBudget := prepareStatsByBudget(
		budgets,
		overridesByBudgetId,
		eventStats.ByBudget,
		currentEventBudgetId,
		currentEventTime,
	)

	totalTime := time.Duration(0)
	for _, budgetDuration := range eventStats.ByBudget {
		totalTime += budgetDuration
	}
	totalTime += currentEventTime

	return StatsSummary{
		StartDate:      from,
		EndDate:        to,
		Days:           statsByDate,
		Budgets:        statsByBudget,
		TotalPlanned:   totalPlanned,
		TotalTime:      totalTime,
		TotalRemaining: totalPlanned - totalTime,
	}, nil
}

func prepareStatsByBudget(
	budgets []budget.Budget,
	overridesByBudgetId map[int]*budget_override.BudgetOverride,
	durationByBudgetId map[int]time.Duration,
	currentEventBudgetId int,
	currentEventTime time.Duration,
) []BudgetStats {

	statsByBudget := make([]BudgetStats, 0, len(budgets))
	for _, b := range budgets {
		budgetDuration := durationByBudgetId[b.ID]
		if b.Status != budget.BudgetStatusActive && budgetDuration == 0 {
			// non-active budget without events in this period
			continue
		}
		budgetOverride := overridesByBudgetId[b.ID]
		budgetCurrentEventTime := time.Duration(0)
		if b.ID == currentEventBudgetId {
			budgetCurrentEventTime = currentEventTime
		}

		budgetStats := BudgetStats{
			Budget:         b,
			Duration:       budgetDuration + budgetCurrentEventTime,
			Remaining:      calculateRemainingDuration(&b, budgetOverride, budgetDuration) - budgetCurrentEventTime,
			BudgetOverride: budgetOverride,
		}
		statsByBudget = append(statsByBudget, budgetStats)
	}
	return statsByBudget
}

func calculateRemainingDuration(
	budget *budget.Budget,
	override *budget_override.BudgetOverride,
	duration time.Duration,
) time.Duration {
	weeklyTime := budget.WeeklyTime
	if override != nil {
		weeklyTime = override.WeeklyTime
	}

	return weeklyTime - duration
}

func (s *StatsServiceImpl) getAllBudgetOverrides(ctx context.Context, startDate time.Time) (map[int]*budget_override.BudgetOverride, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	overrides, err := s.budgetOverrideRepo.GetAllForWeek(ctx, userId, startDate)
	if err != nil {
		return nil, err
	}
	overridesMap := map[int]*budget_override.BudgetOverride{}
	for _, override := range overrides {
		overridesMap[override.BudgetID] = &override
	}
	return overridesMap, nil
}

func sameDays(date1, date2 time.Time, loc *time.Location) bool {
	year1, month1, day1 := date1.In(loc).Date()
	year2, month2, day2 := date2.In(loc).Date()
	return year1 == year2 && month1 == month2 && day1 == day2
}
