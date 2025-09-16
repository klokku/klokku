package budget

import (
	"context"
	"fmt"

	"github.com/klokku/klokku/pkg/user"
	log "github.com/sirupsen/logrus"
)

type BudgetService interface {
	GetAll(ctx context.Context, includeInactive bool) ([]Budget, error)
	Create(ctx context.Context, budget Budget) (Budget, error)
	MoveAfter(ctx context.Context, id, precedingId int64) (bool, error)
	Update(ctx context.Context, budget Budget) (bool, error)
	Delete(ctx context.Context, id int) (bool, error)
}

type BudgetServiceImpl struct {
	repo BudgetRepo
}

func NewBudgetServiceImpl(repo BudgetRepo) *BudgetServiceImpl {
	return &BudgetServiceImpl{repo: repo}
}

func (s *BudgetServiceImpl) GetAll(ctx context.Context, includeInactive bool) ([]Budget, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	return s.repo.GetAll(ctx, userId, includeInactive)
}

func (s *BudgetServiceImpl) Create(ctx context.Context, budget Budget) (Budget, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return Budget{}, fmt.Errorf("failed to get current user: %w", err)
	}
	maxPosition, err := s.repo.FindMaxPosition(ctx, userId)
	if err != nil {
		return Budget{}, err
	}
	budget.Position = maxPosition + 100

	id, err := s.repo.Store(ctx, userId, budget)
	if err != nil {
		return Budget{}, err
	}
	budget.ID = id

	return budget, nil
}

func (s *BudgetServiceImpl) Update(ctx context.Context, budget Budget) (bool, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get current user: %w", err)
	}

	updated, err := s.repo.Update(ctx, userId, budget)
	if err != nil {
		return false, err
	}
	if !updated {
		log.Warnf("budget not updated, probably because it does not exist (%d) or the user (%d) is not the owner", budget.ID, userId)
		return false, fmt.Errorf("budget not updated")
	}
	return true, nil
}

func (s *BudgetServiceImpl) Delete(ctx context.Context, id int) (bool, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get current user: %w", err)
	}

	deleted, err := s.repo.Delete(ctx, userId, id)
	if err != nil {
		return false, err
	}
	if !deleted {
		log.Warnf("budget not deleted, probably because it does not exist (%d) or the user (%d) is not the owner", id, userId)
		return false, fmt.Errorf("budget not deleted")
	}
	return true, nil
}

func (s *BudgetServiceImpl) MoveAfter(ctx context.Context, id int64, precedingId int64) (bool, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get current user: %w", err)
	}
	budgets, err := s.repo.GetAll(ctx, userId, false)
	if err != nil {
		return false, err
	}

	newPos := 0
	prevPos, nextPos := findPreviousAndNextPositions(precedingId, budgets)
	if nextPos == -1 {
		newPos = prevPos + 100
	} else if nextPos-prevPos > 1 {
		newPos = prevPos + ((nextPos - prevPos) / 2)
	} else { // no space between prev and next - reorder all budgets
		prevIdx := findBudget(precedingId, budgets)
		newBudgets := append(budgets[:prevIdx], append([]Budget{budgets[findBudget(id, budgets)]}, budgets[prevIdx+1:]...)...)
		err := s.reorderBudgets(ctx, userId, newBudgets)
		if err != nil {
			return false, err
		}
	}
	budgetToMove := budgets[findBudget(id, budgets)]
	budgetToMove.Position = newPos
	return s.repo.UpdatePosition(ctx, userId, budgetToMove)
}

func (s *BudgetServiceImpl) reorderBudgets(ctx context.Context, userId int, budgets []Budget) error {
	for i, budget := range budgets {
		budget.Position = (i + 1) * 100
		_, err := s.repo.UpdatePosition(ctx, userId, budget)
		if err != nil {
			return err
		}
	}
	return nil
}

func findPreviousAndNextPositions(previousId int64, budgets []Budget) (int, int) {
	previousBudgetIdx := findBudget(previousId, budgets)
	if previousBudgetIdx == -1 {
		return 0, budgets[0].Position
	}
	previousBudgetPos := budgets[previousBudgetIdx].Position
	if previousBudgetIdx == len(budgets)-1 { // it is the last one
		return previousBudgetPos, -1
	}
	nextBudgetPos := budgets[previousBudgetIdx+1].Position
	return previousBudgetPos, nextBudgetPos
}

func findBudget(id int64, budgets []Budget) int {
	for idx, budget := range budgets {
		if budget.ID == int(id) {
			return idx
		}
	}
	return -1
}
