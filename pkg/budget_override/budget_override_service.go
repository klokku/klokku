package budget_override

import (
	"context"
	"fmt"
	"github.com/klokku/klokku/pkg/user"
	"time"
)

type BudgetOverrideService interface {
	Store(ctx context.Context, override BudgetOverride) (BudgetOverride, error)
	GetAllForWeek(ctx context.Context, weekStartDate time.Time) ([]BudgetOverride, error)
	Delete(ctx context.Context, id int) error
	Update(ctx context.Context, override BudgetOverride) error
}

type BudgetOverrideServiceImpl struct {
	repo BudgetOverrideRepo
}

func NewBudgetOverrideService(repo BudgetOverrideRepo) *BudgetOverrideServiceImpl {
	return &BudgetOverrideServiceImpl{repo}
}

func (s *BudgetOverrideServiceImpl) Store(ctx context.Context, override BudgetOverride) (BudgetOverride, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return BudgetOverride{}, fmt.Errorf("failed to get current user: %w", err)
	}
	storedId, err := s.repo.Store(ctx, userId, override)
	if err != nil {
		return BudgetOverride{}, fmt.Errorf("failed to store budget override: %w", err)
	}
	override.ID = storedId
	return override, nil
}

func (s *BudgetOverrideServiceImpl) GetAllForWeek(ctx context.Context, weekStartDate time.Time) ([]BudgetOverride, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	return s.repo.GetAllForWeek(ctx, userId, weekStartDate)
}

func (s *BudgetOverrideServiceImpl) Delete(ctx context.Context, id int) error {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}
	return s.repo.Delete(ctx, userId, id)
}

func (s *BudgetOverrideServiceImpl) Update(ctx context.Context, override BudgetOverride) error {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}
	return s.repo.Update(ctx, userId, override)
}
