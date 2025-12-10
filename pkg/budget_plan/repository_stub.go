package budget_plan

import (
	"context"
	"fmt"
)

type RepositoryStub struct {
	nextId        int
	plans         map[int]BudgetPlan
	currentPlanId int
}

func (s *RepositoryStub) CreatePlan(ctx context.Context, userId int, plan BudgetPlan) (BudgetPlan, error) {
	s.nextId++
	plan.Id = s.nextId
	if len(s.plans) == 0 {
		s.currentPlanId = plan.Id
		plan.IsCurrent = true
	}
	if s.currentPlanId != plan.Id {
		plan.IsCurrent = false
	}
	s.plans[plan.Id] = plan
	return plan, nil
}

func (s *RepositoryStub) UpdatePlan(ctx context.Context, userId int, plan BudgetPlan) (BudgetPlan, error) {
	if plan.IsCurrent {
		s.currentPlanId = plan.Id
	}
	if s.currentPlanId != plan.Id {
		plan.IsCurrent = false
	}
	s.plans[plan.Id] = plan
	return plan, nil
}

func (s *RepositoryStub) DeletePlan(ctx context.Context, userId int, planId int) (bool, error) {
	if s.plans[planId].IsCurrent {
		return false, fmt.Errorf("cannot delete current plan")
	}
	if _, exists := s.plans[planId]; exists {
		delete(s.plans, planId)
		return true, nil
	}
	return false, nil
}

func (s *RepositoryStub) ListPlans(ctx context.Context, userId int) ([]BudgetPlan, error) {
	plans := make([]BudgetPlan, 0, len(s.plans))
	for _, plan := range s.plans {
		plans = append(plans, plan)
	}
	return plans, nil
}

func NewStubBudgetRepo() *RepositoryStub {
	nextId := 2
	plans := map[int]BudgetPlan{}
	return &RepositoryStub{nextId, plans, 0}
}

func (s *RepositoryStub) StoreItem(ctx context.Context, userId int, item BudgetItem) (int, error) {
	planId := item.PlanId
	if plan, exists := s.plans[planId]; exists {
		s.nextId++
		item.Id = s.nextId
		plan.Items = append(plan.Items, item)
		s.plans[planId] = plan
		return item.Id, nil
	}
	return 0, fmt.Errorf("plan with id %d does not exist", planId)
}

func (s *RepositoryStub) GetPlan(ctx context.Context, userId int, planId int) (BudgetPlan, error) {
	if plan, exists := s.plans[planId]; exists {
		if s.currentPlanId == planId {
			plan.IsCurrent = true
		}
		return plan, nil
	}
	return BudgetPlan{}, ErrPlanNotFound
}

func (s *RepositoryStub) UpdateItem(ctx context.Context, userId int, item BudgetItem) (bool, error) {
	planId := item.PlanId
	if plan, exists := s.plans[planId]; exists {
		for i, it := range plan.Items {
			if it.Id == item.Id {
				plan.Items[i] = item
				s.plans[planId] = plan
				return true, nil
			}
		}
	}
	return false, nil
}

func (s *RepositoryStub) DeleteItem(ctx context.Context, userId int, budgetId int) (bool, error) {
	for _, plan := range s.plans {
		for i, item := range plan.Items {
			if item.Id == budgetId {
				plan.Items = append(plan.Items[:i], plan.Items[i+1:]...)
				s.plans[plan.Id] = plan
				return true, nil
			}
		}
	}
	return false, nil
}

func (s *RepositoryStub) UpdateItemPosition(ctx context.Context, userId int, budget BudgetItem) (bool, error) {
	return s.UpdateItem(ctx, userId, budget)
}

func (s *RepositoryStub) FindMaxPlanItemPosition(ctx context.Context, planId int, userId int) (int, error) {
	maxPosition := 0
	for _, plan := range s.plans {
		for _, item := range plan.Items {
			if item.Position > maxPosition {
				maxPosition = item.Position
			}
		}
	}
	return maxPosition, nil
}

func (s *RepositoryStub) Cleanup() {
	s.plans = map[int]BudgetPlan{}
}
