package weekly_plan

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/klokku/klokku/internal/event_bus"
	"github.com/klokku/klokku/pkg/budget_plan"
	"github.com/klokku/klokku/pkg/user"
	log "github.com/sirupsen/logrus"
)

var ErrNoCurrentPlan = fmt.Errorf("no current plan")
var ErrBudgetItemNotFound = fmt.Errorf("budget item not found")
var ErrWeeklyItemAlreadyExists = fmt.Errorf("weekly items already exist for week")
var ErrWeeklyItemNotFound = fmt.Errorf("weekly item not found")

type Service interface {
	GetItemsForWeek(ctx context.Context, date time.Time) ([]WeeklyPlanItem, error)
	UpdateItem(ctx context.Context, weekDate time.Time, id int, budgetItemId int, weeklyDuration time.Duration, notes string) (WeeklyPlanItem, error)
	// ResetWeekItemToBudgetPlanItem resets the specified weekly plan item to the value of the budget plan item it was created from.
	ResetWeekItemToBudgetPlanItem(ctx context.Context, id int) (WeeklyPlanItem, error)
	ResetWeekItemsToBudgetPlan(ctx context.Context, weekDate time.Time) ([]WeeklyPlanItem, error)
}

type BudgetPlanReader interface {
	GetCurrentPlan(ctx context.Context) (budget_plan.BudgetPlan, error)
	GetPlan(ctx context.Context, planId int) (budget_plan.BudgetPlan, error)
	GetItem(ctx context.Context, id int) (budget_plan.BudgetItem, error)
}

type ServiceImpl struct {
	repo     Repository
	bpReader BudgetPlanReader
	eventBus *event_bus.EventBus
}

func NewService(repo Repository, bpReader BudgetPlanReader, eventBus *event_bus.EventBus) Service {
	service := &ServiceImpl{repo, bpReader, eventBus}
	event_bus.SubscribeTyped[event_bus.BudgetPlanItemUpdated](
		eventBus,
		"budget_plan.item.updated",
		func(e event_bus.EventT[event_bus.BudgetPlanItemUpdated]) error {
			log.Debugf("received budget plan item updated event: %v", e)
			countUpdated, err := service.handleBudgetPlanItemUpdated(e.Context(), e.Data)
			if err != nil {
				log.Errorf("failed to handle budget plan item update: %v", err)
				return err
			}
			log.Debugf("updated weekly plan items: %d", countUpdated)

			return nil
		},
	)
	event_bus.SubscribeTyped[event_bus.CalendarEventCreated](
		eventBus,
		"calendar.event.updated",
		func(e event_bus.EventT[event_bus.CalendarEventCreated]) error {
			log.Debugf("received calendar event updated event: %v", e)
			err := service.handleCalendarEventChanged(e.Context(), e.Data)
			if err != nil {
				log.Errorf("failed to handle calendar event change: %v", err)
				return err
			}

			return nil
		},
	)
	return service
}

func (s *ServiceImpl) GetItemsForWeek(ctx context.Context, date time.Time) ([]WeeklyPlanItem, error) {
	currentUser, err := user.CurrentUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	weekNumber := WeekNumberFromDate(date, currentUser.Settings.WeekFirstDay)
	items, err := s.repo.GetItemsForWeek(ctx, currentUser.Id, weekNumber)
	if err != nil {
		log.Errorf("failed to get weekly plan items for week %s: %v", weekNumber, err)
		return nil, fmt.Errorf("failed to get weekly plan items: %w", err)
	}
	if items != nil && len(items) > 0 {
		return items, nil
	}

	currentPlan, err := s.bpReader.GetCurrentPlan(ctx)
	if err != nil {
		if errors.Is(err, budget_plan.ErrPlanNotFound) {
			return nil, ErrNoCurrentPlan
		}
		return nil, err
	}
	for _, bpItem := range currentPlan.Items {
		items = append(items, budgetPlanItemToWeekPlanItem(bpItem, weekNumber))
	}

	return items, nil
}

func (s *ServiceImpl) UpdateItem(
	ctx context.Context,
	weekDate time.Time,
	id int,
	budgetItemId int,
	weeklyDuration time.Duration,
	notes string,
) (WeeklyPlanItem, error) {
	currentUser, err := user.CurrentUser(ctx)
	if err != nil {
		return WeeklyPlanItem{}, fmt.Errorf("failed to get current user: %w", err)
	}
	// Update existing item (weekly item already exists)
	if id != 0 {
		return s.repo.UpdateItem(ctx, currentUser.Id, id, weeklyDuration, notes)
	}

	week := WeekNumberFromDate(weekDate, currentUser.Settings.WeekFirstDay)

	// Weekly items do not exist yet, create them
	budgetItem, err := s.bpReader.GetItem(ctx, budgetItemId)
	if err != nil {
		return WeeklyPlanItem{}, ErrBudgetItemNotFound
	}
	weeklyPlanItems, err := s.repo.GetItemsForWeek(ctx, currentUser.Id, week)
	if err != nil {
		return WeeklyPlanItem{}, err
	}
	if len(weeklyPlanItems) > 0 {
		// User sent id=0, but weekly items already exist for the given week
		// This may be secured with transaction in the future, but for now it's secured by the unique index on the DB
		return WeeklyPlanItem{}, ErrWeeklyItemAlreadyExists
	}

	var updatedItem WeeklyPlanItem
	err = s.repo.WithTransaction(ctx, func(repo Repository) error {
		transactionalService := ServiceImpl{repo, s.bpReader, nil}
		items, err := transactionalService.createItemsFromBudgetPlan(ctx, budgetItem.PlanId, week)
		if err != nil {
			return err
		}
		for _, item := range items {
			if item.BudgetItemId == budgetItemId {
				updatedItem, err = repo.UpdateItem(ctx, currentUser.Id, item.Id, weeklyDuration, notes)
				if err != nil {
					return err
				}
				break
			}
		}
		return nil
	})
	if err != nil {
		return WeeklyPlanItem{}, err
	}
	return updatedItem, nil
}

// createItemsFromBudgetPlan generates weekly plan items from the specified budget plan for a given week and persists them in the repository.
// This is done in two cases:
// 1. When a user updates any weekly item for the week that did not have the WeeklyItems yet
// 2. When a first calendar event is created for the given week
func (s *ServiceImpl) createItemsFromBudgetPlan(ctx context.Context, budgetPlanId int, week WeekNumber) ([]WeeklyPlanItem, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	plan, err := s.bpReader.GetPlan(ctx, budgetPlanId)
	if err != nil {
		return nil, fmt.Errorf("failed to get budget plan: %w", err)
	}
	var items []WeeklyPlanItem
	for _, bpItem := range plan.Items {
		items = append(items, budgetPlanItemToWeekPlanItem(bpItem, week))
	}
	createdItems, err := s.repo.createItems(ctx, userId, items)
	if err != nil {
		return nil, fmt.Errorf("failed to create weekly plan items: %w", err)
	}
	return createdItems, nil

}

func (s *ServiceImpl) ResetWeekItemToBudgetPlanItem(ctx context.Context, id int) (WeeklyPlanItem, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return WeeklyPlanItem{}, fmt.Errorf("failed to get current user: %w", err)
	}

	item, err := s.repo.GetItem(ctx, userId, id)
	if err != nil {
		log.Errorf("failed to get weekly plan item: %v", err)
		return WeeklyPlanItem{}, ErrWeeklyItemNotFound
	}

	budgetItem, err := s.bpReader.GetItem(ctx, item.BudgetItemId)
	if err != nil {
		log.Errorf("failed to get budget plan item: %v", err)
		return WeeklyPlanItem{}, ErrBudgetItemNotFound
	}

	updatedItem, err := s.repo.UpdateItem(ctx, userId, item.Id, budgetItem.WeeklyDuration, "")
	if err != nil {
		if errors.Is(err, ErrWeeklyItemNotFound) {
			return WeeklyPlanItem{}, ErrWeeklyItemNotFound
		}
		log.Errorf("failed to reset weekly plan item: %v", err)
		return WeeklyPlanItem{}, err
	}

	return updatedItem, nil
}

func (s *ServiceImpl) ResetWeekItemsToBudgetPlan(ctx context.Context, weekDate time.Time) ([]WeeklyPlanItem, error) {
	currentUser, err := user.CurrentUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	week := WeekNumberFromDate(weekDate, currentUser.Settings.WeekFirstDay)
	currentWeek := WeekNumberFromDate(time.Now(), currentUser.Settings.WeekFirstDay)
	// For future weeks simply delete all weekly plan items
	if week.After(currentWeek) {
		_, err := s.repo.DeleteWeekItems(ctx, currentUser.Id, week)
		if err != nil {
			return nil, fmt.Errorf("failed to reset weekly plan items: %w", err)
		}
		items, err := s.GetItemsForWeek(ctx, weekDate)
		if err != nil {
			return nil, fmt.Errorf("failed to get weekly plan items: %w", err)
		}
		return items, nil
	}

	// For past and current weeks only restore the items' WeeklyDuration and remove notes
	items, err := s.GetItemsForWeek(ctx, weekDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get weekly plan items before reset: %w", err)
	}
	var resetItems []WeeklyPlanItem
	err = s.repo.WithTransaction(ctx, func(repo Repository) error {
		for _, item := range items {
			budgetItem, err := s.bpReader.GetItem(ctx, item.BudgetItemId)
			if err != nil {
				log.Errorf("failed to get budget plan item: %v", err)
				return err
			}
			updatedItem, err := repo.UpdateItem(ctx, currentUser.Id, item.Id, budgetItem.WeeklyDuration, "")
			if err != nil {
				return err
			}
			resetItems = append(resetItems, updatedItem)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to reset weekly plan items: %w", err)
	}

	return resetItems, nil
}

func (s *ServiceImpl) handleBudgetPlanItemUpdated(ctx context.Context, budgetItem event_bus.BudgetPlanItemUpdated) (int, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get current user: %w", err)
	}
	return s.repo.UpdateAllItemsByBudgetItemId(ctx, userId, budgetItem.Id, budgetItem.Name, budgetItem.Icon, budgetItem.Color)
}

func budgetPlanItemToWeekPlanItem(bpItem budget_plan.BudgetItem, weekNumber WeekNumber) WeeklyPlanItem {
	return WeeklyPlanItem{
		BudgetItemId:      bpItem.Id,
		BudgetPlanId:      bpItem.PlanId,
		WeekNumber:        weekNumber,
		Name:              bpItem.Name,
		WeeklyDuration:    bpItem.WeeklyDuration,
		WeeklyOccurrences: bpItem.WeeklyOccurrences,
		Icon:              bpItem.Icon,
		Color:             bpItem.Color,
		Notes:             "",
		Position:          bpItem.Position,
	}
}

func (s *ServiceImpl) handleCalendarEventChanged(ctx context.Context, event event_bus.CalendarEventCreated) error {
	currentUser, err := user.CurrentUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	week := WeekNumberFromDate(event.StartTime, currentUser.Settings.WeekFirstDay)
	err = s.repo.WithTransaction(ctx, func(repo Repository) error {
		transactionalService := ServiceImpl{repo, s.bpReader, s.eventBus}
		weeklyPlanItems, err := repo.GetItemsForWeek(ctx, currentUser.Id, week)
		if err != nil {
			return err
		}
		if len(weeklyPlanItems) > 0 {
			// items already exist for the given week, nothing to do
			return nil
		}
		item, err := s.bpReader.GetItem(ctx, event.BudgetItemId)
		if err != nil {
			return err
		}

		_, err = transactionalService.createItemsFromBudgetPlan(ctx, item.PlanId, week)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}
	return nil
}
