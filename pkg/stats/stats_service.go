package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/klokku/klokku/internal/utils"
	"github.com/klokku/klokku/pkg/budget_override"
	"github.com/klokku/klokku/pkg/budget_plan"
	"github.com/klokku/klokku/pkg/event"
	"github.com/klokku/klokku/pkg/user"
	log "github.com/sirupsen/logrus"
)

type StatsService interface {
	GetStats(ctx context.Context, from time.Time, to time.Time) (StatsSummary, error)
}

type StatsServiceImpl struct {
	eventService       event.EventService
	eventStatsService  event.EventStatsService
	budgetRepo         budget_plan.Repository
	budgetOverrideRepo budget_override.BudgetOverrideRepo
	clock              utils.Clock
}

func NewStatsServiceImpl(
	eventService event.EventService,
	eventStatsService event.EventStatsService,
	budgetRepo budget_plan.Repository,
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
	budgets, err := s.budgetRepo.GetPlan(ctx, userId, true)
	if err != nil {
		return StatsSummary{}, err
	}
	log.Tracef("Budgets: %v", budgets)

	// filter out inactive budgets
	activeBudgets := make([]budget_plan.BudgetItem, 0, len(budgets))
	for _, b := range budgets {
		if b.IsActiveBetween(from, to) {
			activeBudgets = append(activeBudgets, b)
		}
	}

	overridesByBudgetId, err := s.getAllBudgetOverrides(ctx, from)
	if err != nil {
		return StatsSummary{}, err
	}
	log.Tracef("BudgetItem overrides: %v", overridesByBudgetId)

	totalPlanned := time.Duration(0)
	for _, b := range activeBudgets {
		budgetTime := b.WeeklyDuration
		if overridesByBudgetId[b.Id] != nil {
			budgetTime = overridesByBudgetId[b.Id].WeeklyTime
		}
		log.Debugf("BudgetItem: %v, Time: %v, TimeWithOverride: %v", b.Name, b.WeeklyDuration, budgetTime)
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
			currentEventBudgetId = currentEvent.Budget.Id
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
		budgetsStats := prepareStatsByBudget(activeBudgets, nil, dateBudgetDuration, currentEventBudgetId, todayCurrentEventTime)
		dateTotalTime := time.Duration(0)
		for _, budgetStat := range budgetsStats {
			dateTotalTime += budgetStat.Duration
		}

		dailyStats := DailyStats{date, budgetsStats, dateTotalTime}
		statsByDate = append(statsByDate, dailyStats)
	}

	statsByBudget := prepareStatsByBudget(
		activeBudgets,
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
	budgets []budget_plan.BudgetItem,
	overridesByBudgetId map[int]*budget_override.BudgetOverride,
	durationByBudgetId map[int]time.Duration,
	currentEventBudgetId int,
	currentEventTime time.Duration,
) []BudgetStats {

	statsByBudget := make([]BudgetStats, 0, len(budgets))
	for _, b := range budgets {
		budgetDuration := durationByBudgetId[b.Id]
		budgetOverride := overridesByBudgetId[b.Id]
		budgetCurrentEventTime := time.Duration(0)
		if b.Id == currentEventBudgetId {
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
	budget *budget_plan.BudgetItem,
	override *budget_override.BudgetOverride,
	duration time.Duration,
) time.Duration {
	weeklyTime := budget.WeeklyDuration
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
