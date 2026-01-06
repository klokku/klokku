package stats

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/klokku/klokku/pkg/budget_plan"
	"github.com/klokku/klokku/pkg/current_event"
	"github.com/klokku/klokku/pkg/weekly_plan"
)

type weeklyPlanItemsReaderStub struct {
	items map[int]weekly_plan.WeeklyPlanItem // id -> item
}

func newWeeklyPlanItemsReaderStub() *weeklyPlanItemsReaderStub {
	return &weeklyPlanItemsReaderStub{
		items: make(map[int]weekly_plan.WeeklyPlanItem),
	}
}

func (s *weeklyPlanItemsReaderStub) setItems(items []weekly_plan.WeeklyPlanItem) {
	s.reset()
	for _, item := range items {
		s.items[item.Id] = item
	}
}

func (s *weeklyPlanItemsReaderStub) GetItemsForWeek(ctx context.Context, date time.Time) ([]weekly_plan.WeeklyPlanItem, error) {
	var items []weekly_plan.WeeklyPlanItem
	for _, item := range s.items {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Position < items[j].Position
	})
	return items, nil
}

func (s *weeklyPlanItemsReaderStub) reset() {
	s.items = make(map[int]weekly_plan.WeeklyPlanItem)
}

type currentEventProviderStub struct {
	event *current_event.CurrentEvent
}

func newCurrentEventProviderStub() *currentEventProviderStub {
	return &currentEventProviderStub{
		event: nil,
	}
}

func (s *currentEventProviderStub) FindCurrentEvent(ctx context.Context) (current_event.CurrentEvent, error) {
	return *s.event, nil
}

func (s *currentEventProviderStub) set(event *current_event.CurrentEvent) {
	s.event = event
}

func (s *currentEventProviderStub) reset() {
	s.event = nil
}

type budgetPlanReaderStub struct {
	plans []budget_plan.BudgetPlan
}

func newBudgetPlanReaderStub() *budgetPlanReaderStub {
	return &budgetPlanReaderStub{
		plans: []budget_plan.BudgetPlan{},
	}
}

func (s *budgetPlanReaderStub) GetPlan(ctx context.Context, planId int) (budget_plan.BudgetPlan, error) {
	for _, plan := range s.plans {
		if plan.Id == planId {
			return plan, nil
		}
	}
	return budget_plan.BudgetPlan{}, errors.New("plan not found")
}

func (s *budgetPlanReaderStub) addPlan(plan budget_plan.BudgetPlan) {
	s.plans = append(s.plans, plan)
}

func (s *budgetPlanReaderStub) reset() {
	s.plans = []budget_plan.BudgetPlan{}
}

func (s *budgetPlanReaderStub) GetItem(ctx context.Context, budgetItemId int) (budget_plan.BudgetItem, error) {
	for _, item := range s.plans {
		for _, budgetItem := range item.Items {
			if budgetItem.Id == budgetItemId {
				return budgetItem, nil
			}
		}
	}
	return budget_plan.BudgetItem{}, errors.New("budget item not found")
}
