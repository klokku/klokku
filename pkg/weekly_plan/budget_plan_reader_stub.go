package weekly_plan

import (
	"context"
	"sync"

	"github.com/klokku/klokku/pkg/budget_plan"
)

// BudgetPlanReaderStub is a test stub implementation of BudgetPlanReader
type BudgetPlanReaderStub struct {
	mu            sync.RWMutex
	plans         map[int]budget_plan.BudgetPlan // planId -> plan
	items         map[int]budget_plan.BudgetItem // itemId -> item
	currentPlan   *budget_plan.BudgetPlan
	getPlanErr    error
	getCurrentErr error
	getItemErr    error
}

func NewBudgetPlanReaderStub() *BudgetPlanReaderStub {
	return &BudgetPlanReaderStub{
		plans: make(map[int]budget_plan.BudgetPlan),
		items: make(map[int]budget_plan.BudgetItem),
	}
}

func (s *BudgetPlanReaderStub) GetCurrentPlan(ctx context.Context) (budget_plan.BudgetPlan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.getCurrentErr != nil {
		return budget_plan.BudgetPlan{}, s.getCurrentErr
	}

	if s.currentPlan == nil {
		return budget_plan.BudgetPlan{}, budget_plan.ErrPlanNotFound
	}

	return *s.currentPlan, nil
}

func (s *BudgetPlanReaderStub) GetPlan(ctx context.Context, planId int) (budget_plan.BudgetPlan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.getPlanErr != nil {
		return budget_plan.BudgetPlan{}, s.getPlanErr
	}

	plan, exists := s.plans[planId]
	if !exists {
		return budget_plan.BudgetPlan{}, budget_plan.ErrPlanNotFound
	}

	return plan, nil
}

func (s *BudgetPlanReaderStub) GetItem(ctx context.Context, id int) (budget_plan.BudgetItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.getItemErr != nil {
		return budget_plan.BudgetItem{}, s.getItemErr
	}

	item, exists := s.items[id]
	if !exists {
		return budget_plan.BudgetItem{}, budget_plan.ErrPlanNotFound
	}

	return item, nil
}

// Helper methods for test setup

func (s *BudgetPlanReaderStub) SetCurrentPlan(plan budget_plan.BudgetPlan) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentPlan = &plan
	s.plans[plan.Id] = plan
	for _, item := range plan.Items {
		s.items[item.Id] = item
	}
}

func (s *BudgetPlanReaderStub) SetPlan(plan budget_plan.BudgetPlan) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plans[plan.Id] = plan
	for _, item := range plan.Items {
		s.items[item.Id] = item
	}
}

func (s *BudgetPlanReaderStub) SetItem(item budget_plan.BudgetItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[item.Id] = item
}

func (s *BudgetPlanReaderStub) SetGetPlanError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getPlanErr = err
}

func (s *BudgetPlanReaderStub) SetGetCurrentError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getCurrentErr = err
}

func (s *BudgetPlanReaderStub) SetGetItemError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getItemErr = err
}

func (s *BudgetPlanReaderStub) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plans = make(map[int]budget_plan.BudgetPlan)
	s.items = make(map[int]budget_plan.BudgetItem)
	s.currentPlan = nil
	s.getPlanErr = nil
	s.getCurrentErr = nil
	s.getItemErr = nil
}
