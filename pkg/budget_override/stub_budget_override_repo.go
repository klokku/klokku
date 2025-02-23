package budget_override

import (
	"context"
	"time"
)

type StubBudgetOverrideRepo struct {
	nextId int
	data   map[int]BudgetOverride
}

func NewStubBudgetOverrideRepo() *StubBudgetOverrideRepo {
	nextId := int(2)
	data := map[int]BudgetOverride{}
	return &StubBudgetOverrideRepo{nextId, data}
}

func (s *StubBudgetOverrideRepo) Store(ctx context.Context, userId int, override BudgetOverride) (int, error) {
	s.nextId++
	override.ID = s.nextId
	s.data[override.ID] = override
	return override.ID, nil
}
func (s *StubBudgetOverrideRepo) GetAllForWeek(ctx context.Context, userId int, weekStartDate time.Time) ([]BudgetOverride, error) {
	overrides := make([]BudgetOverride, 0, 10)
	for _, override := range s.data {
		if override.StartDate == weekStartDate {
			overrides = append(overrides, override)
		}
	}
	return overrides, nil
}

func (s *StubBudgetOverrideRepo) Cleanup() {
	s.data = map[int]BudgetOverride{}
}

func (s *StubBudgetOverrideRepo) Delete(ctx context.Context, userId int, id int) error {
	delete(s.data, id)
	return nil
}

func (s *StubBudgetOverrideRepo) Update(ctx context.Context, userId int, override BudgetOverride) error {
	s.data[override.ID] = override
	return nil
}
