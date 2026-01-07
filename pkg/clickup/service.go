package clickup

import (
	"context"
	"fmt"

	"github.com/klokku/klokku/pkg/user"
	log "github.com/sirupsen/logrus"
)

type Service interface {
	StoreConfiguration(ctx context.Context, budgetPlanId int, config Configuration) error
	GetConfiguration(ctx context.Context, budgetPlanId int) (Configuration, error)
	GetTasksByBudgetItemId(ctx context.Context, budgetItemId int) ([]Task, error)
	DisableIntegration(ctx context.Context) error
	DeleteBudgetPlanConfiguration(ctx context.Context, budgetPlanId int) error
}

type ServiceImpl struct {
	repo   Repository
	client Client
}

func NewServiceImpl(repo Repository, clickUpClient Client) *ServiceImpl {
	return &ServiceImpl{repo: repo, client: clickUpClient}
}

func (s *ServiceImpl) StoreConfiguration(ctx context.Context, budgetPlanId int, config Configuration) error {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	return s.repo.StoreConfiguration(ctx, userId, budgetPlanId, config)
}

func (s *ServiceImpl) GetConfiguration(ctx context.Context, budgetPlanId int) (Configuration, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return Configuration{}, fmt.Errorf("failed to get current user: %w", err)
	}
	configuration, err := s.repo.GetConfiguration(ctx, userId, budgetPlanId)
	if configuration == nil {
		return Configuration{}, err
	}
	return *configuration, nil
}

func (s *ServiceImpl) GetTasksByBudgetItemId(ctx context.Context, budgetItemId int) ([]Task, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	configuration, err := s.repo.GetConfigurationWithMappingByBudgetItemId(ctx, userId, budgetItemId)
	if err != nil {
		return nil, err
	}
	if configuration == nil || len(configuration.Mappings) == 0 {
		log.Debugf("No ClickUp mappings found for budget item ID %d", budgetItemId)
		return []Task{}, nil
	}
	allTasks := make([]Task, 0)
	page := 0

	for {
		tasks, err := s.client.GetFilteredTeamTasks(
			ctx,
			configuration.WorkspaceId,
			configuration.SpaceId,
			configuration.FolderId,
			page,
			configuration.Mappings[0].ClickupTagName,
			configuration.OnlyTasksWithPriority,
		)
		if err != nil {
			return nil, err
		}

		// If no tasks are returned, we've reached the end
		if len(tasks) == 0 {
			break
		}

		if page > 100 {
			log.Infof("Reached maximum page limit of 100 for budget item ID %d", budgetItemId)
			break
		}

		// Append the tasks from this page to our result
		allTasks = append(allTasks, tasks...)
		page++
	}

	return allTasks, nil
}

func (s *ServiceImpl) DisableIntegration(ctx context.Context) error {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	err = s.repo.DeleteAllConfigurations(ctx, userId)
	if err != nil {
		return err
	}

	err = s.repo.DeleteAuthData(ctx, userId)
	if err != nil {
		return err
	}
	return nil
}

func (s *ServiceImpl) DeleteBudgetPlanConfiguration(ctx context.Context, budgetPlanId int) error {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	if budgetPlanId <= 0 {
		return fmt.Errorf("invalid budget plan ID: %d", budgetPlanId)
	}

	err = s.repo.DeleteBudgetPlanConfiguration(ctx, userId, budgetPlanId)
	if err != nil {
		return fmt.Errorf("failed to delete configuration for budget plan ID %d: %w", budgetPlanId, err)
	}

	return nil
}
