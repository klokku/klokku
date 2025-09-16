package budget

import (
	"context"
	"time"
)

type StubBudgetRepo struct {
	nextId int
	data   map[int]Budget
}

func NewStubBudgetRepo() *StubBudgetRepo {
	nextId := 2
	data := map[int]Budget{}
	return &StubBudgetRepo{nextId, data}
}

func (s *StubBudgetRepo) Store(ctx context.Context, userId int, budget Budget) (int, error) {
	s.nextId++
	budget.ID = s.nextId
	s.data[budget.ID] = budget
	return budget.ID, nil
}
func (s *StubBudgetRepo) GetAll(ctx context.Context, userId int, includeInactive bool) ([]Budget, error) {
	budgets := make([]Budget, 0, len(s.data))
	for _, budget := range s.data {
		if budget.IsActiveOn(time.Now()) || includeInactive {
			budgets = append(budgets, budget)
		}
	}
	return budgets, nil
}

func (s *StubBudgetRepo) Update(ctx context.Context, userId int, budget Budget) (bool, error) {
	s.data[budget.ID] = budget
	return true, nil
}

func (s *StubBudgetRepo) Delete(ctx context.Context, userId int, budgetId int) (bool, error) {
	delete(s.data, budgetId)
	return true, nil
}

func (s *StubBudgetRepo) UpdatePosition(ctx context.Context, userId int, budget Budget) (bool, error) {
	return s.Update(ctx, userId, budget)
}

func (s *StubBudgetRepo) FindMaxPosition(ctx context.Context, userId int) (int, error) {
	maxPosition := 0
	for _, budget := range s.data {
		if budget.Position > maxPosition {
			maxPosition = budget.Position
		}
	}
	return maxPosition, nil
}

func (s *StubBudgetRepo) Cleanup() {
	s.data = map[int]Budget{}
}
