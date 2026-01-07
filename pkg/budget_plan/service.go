package budget_plan

import (
	"context"
	"fmt"

	"github.com/klokku/klokku/internal/event_bus"
	"github.com/klokku/klokku/pkg/user"
	log "github.com/sirupsen/logrus"
)

type Service interface {
	GetPlan(ctx context.Context, planId int) (BudgetPlan, error)
	GetCurrentPlan(ctx context.Context) (BudgetPlan, error)
	ListPlans(ctx context.Context) ([]BudgetPlan, error)
	CreatePlan(ctx context.Context, plan BudgetPlan) (BudgetPlan, error)
	UpdatePlan(ctx context.Context, plan BudgetPlan) (BudgetPlan, error)
	DeletePlan(ctx context.Context, planId int) (bool, error)
	GetItem(ctx context.Context, id int) (BudgetItem, error)
	CreateItem(ctx context.Context, item BudgetItem) (BudgetItem, error)
	MoveItemAfter(ctx context.Context, planId, itemId, precedingId int) (bool, error)
	UpdateItem(ctx context.Context, budget BudgetItem) (BudgetItem, error)
	DeleteItem(ctx context.Context, id int) (bool, error)
}

type ServiceImpl struct {
	repo     Repository
	eventBus *event_bus.EventBus
}

func NewBudgetPlanService(repo Repository, eventBus *event_bus.EventBus) Service {
	return &ServiceImpl{repo: repo, eventBus: eventBus}
}

func (s *ServiceImpl) GetPlan(ctx context.Context, planId int) (BudgetPlan, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return BudgetPlan{}, fmt.Errorf("failed to get current user: %w", err)
	}
	return s.repo.GetPlan(ctx, userId, planId)
}

func (s *ServiceImpl) GetCurrentPlan(ctx context.Context) (BudgetPlan, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return BudgetPlan{}, fmt.Errorf("failed to get current user: %w", err)
	}
	return s.repo.GetCurrentPlan(ctx, userId)
}

func (s *ServiceImpl) ListPlans(ctx context.Context) ([]BudgetPlan, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	return s.repo.ListPlans(ctx, userId)
}

func (s *ServiceImpl) CreatePlan(ctx context.Context, plan BudgetPlan) (BudgetPlan, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return BudgetPlan{}, fmt.Errorf("failed to get current user: %w", err)
	}
	return s.repo.CreatePlan(ctx, userId, plan)
}

func (s *ServiceImpl) UpdatePlan(ctx context.Context, plan BudgetPlan) (BudgetPlan, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return BudgetPlan{}, fmt.Errorf("failed to get current user: %w", err)
	}
	updatedPlan, err := s.repo.UpdatePlan(ctx, userId, plan)
	if err != nil {
		return BudgetPlan{}, err
	}
	return updatedPlan, nil
}

func (s *ServiceImpl) DeletePlan(ctx context.Context, planId int) (bool, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get current user: %w", err)
	}
	return s.repo.DeletePlan(ctx, userId, planId)
}

func (s *ServiceImpl) GetItem(ctx context.Context, id int) (BudgetItem, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return BudgetItem{}, fmt.Errorf("failed to get current user: %w", err)
	}
	return s.repo.GetItem(ctx, userId, id)
}

func (s *ServiceImpl) CreateItem(ctx context.Context, item BudgetItem) (BudgetItem, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return BudgetItem{}, fmt.Errorf("failed to get current user: %w", err)
	}

	id, position, err := s.repo.StoreItem(ctx, userId, item)
	if err != nil {
		return BudgetItem{}, err
	}
	item.Id = id
	item.Position = position
	return item, nil
}

func (s *ServiceImpl) UpdateItem(ctx context.Context, budget BudgetItem) (BudgetItem, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return BudgetItem{}, fmt.Errorf("failed to get current user: %w", err)
	}

	updatedItem, err := s.repo.UpdateItem(ctx, userId, budget)
	if err != nil {
		return BudgetItem{}, err
	}

	// This may fail, and because the transaction is already closed, the budget item is changed and this
	// event may not be properly processed by the subscribers.
	// This is a conscious decision done because of the current architecture of the application.
	// A proper solution would be to implement inbox and outbox patterns, but this is out of scope for now.
	// Application is running as a single process, with a single database, so the risk of the event not being processed is low.
	// Additionally, there is an easy workaround in case data are stale. It is enough to update the budget item again.
	err = s.eventBus.Publish(event_bus.NewEvent(
		ctx,
		"budget_plan.item.updated",
		event_bus.BudgetPlanItemUpdated{
			Id:                updatedItem.Id,
			PlanId:            updatedItem.PlanId,
			Name:              updatedItem.Name,
			WeeklyDuration:    updatedItem.WeeklyDuration,
			WeeklyOccurrences: updatedItem.WeeklyOccurrences,
			Icon:              updatedItem.Icon,
			Color:             updatedItem.Color,
			Position:          updatedItem.Position,
		},
	))
	if err != nil {
		log.Errorf("failed to publish budget item update event: %v", err)
		return BudgetItem{}, err
	}

	return updatedItem, nil
}

func (s *ServiceImpl) DeleteItem(ctx context.Context, id int) (bool, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get current user: %w", err)
	}

	deleted, err := s.repo.DeleteItem(ctx, userId, id)
	if err != nil {
		return false, err
	}
	if !deleted {
		log.Warnf("item not deleted, probably because it does not exist (%d) or the user (%d) is not the owner", id, userId)
		return false, fmt.Errorf("item not deleted")
	}
	return true, nil
}

func (s *ServiceImpl) MoveItemAfter(ctx context.Context, planId int, itemId int, precedingId int) (bool, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get current user: %w", err)
	}
	plan, err := s.repo.GetPlan(ctx, userId, planId)
	if err != nil {
		return false, err
	}

	items := plan.Items

	// Validate that the item to move exists
	itemIdx := findItem(itemId, items)
	if itemIdx == -1 {
		return false, fmt.Errorf("item not found")
	}

	// Validate that the preceding item exists (if not -1 or 0)
	if precedingId > 0 {
		prevIdx := findItem(precedingId, items)
		if prevIdx == -1 {
			return false, fmt.Errorf("item not found")
		}
	}

	newPos := 0
	prevPos, nextPos := findPreviousAndNextPositions(precedingId, items)
	if nextPos == -1 {
		newPos = prevPos + 100
	} else if nextPos-prevPos > 1 {
		newPos = prevPos + ((nextPos - prevPos) / 2)
	} else { // no space between prev and next - reorder all items
		prevIdx := findItem(precedingId, items)
		newItems := append(items[:prevIdx], append([]BudgetItem{items[itemIdx]}, items[prevIdx+1:]...)...)
		err := s.reorderItems(ctx, userId, newItems)
		if err != nil {
			return false, err
		}
	}
	budgetToMove := items[itemIdx]
	budgetToMove.Position = newPos
	return s.repo.UpdateItemPosition(ctx, userId, budgetToMove)
}

func (s *ServiceImpl) reorderItems(ctx context.Context, userId int, items []BudgetItem) error {
	for i, item := range items {
		item.Position = (i + 1) * 100
		_, err := s.repo.UpdateItemPosition(ctx, userId, item)
		if err != nil {
			return err
		}
	}
	return nil
}

func findPreviousAndNextPositions(previousId int, items []BudgetItem) (int, int) {
	previousItemIdx := findItem(previousId, items)
	if previousItemIdx == -1 {
		return 0, items[0].Position
	}
	previousItemPos := items[previousItemIdx].Position
	if previousItemIdx == len(items)-1 { // it is the last one
		return previousItemPos, -1
	}
	nextItemPos := items[previousItemIdx+1].Position
	return previousItemPos, nextItemPos
}

func findItem(id int, items []BudgetItem) int {
	for idx, item := range items {
		if item.Id == id {
			return idx
		}
	}
	return -1
}
