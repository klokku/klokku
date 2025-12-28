package clickup

import (
	"context"
	"fmt"
	"github.com/klokku/klokku/pkg/user"
)

type Service interface {
	StoreConfiguration(ctx context.Context, config Configuration) error
	GetConfiguration(ctx context.Context) (Configuration, error)
	DisableIntegration(ctx context.Context) error
	GetTasksByBudgetId(ctx context.Context, budgetId int) ([]Task, error)
}

type ServiceImpl struct {
	repo   Repository
	client Client
}

func NewServiceImpl(repo Repository, clickUpClient Client) *ServiceImpl {
	return &ServiceImpl{repo: repo, client: clickUpClient}
}

func (s *ServiceImpl) StoreConfiguration(ctx context.Context, config Configuration) error {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	return s.repo.StoreConfiguration(ctx, userId, config)
}

func (s *ServiceImpl) GetConfiguration(ctx context.Context) (Configuration, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return Configuration{}, fmt.Errorf("failed to get current user: %w", err)
	}
	configuration, err := s.repo.GetConfiguration(ctx, userId)
	if configuration == nil {
		return Configuration{}, err
	}
	return *configuration, nil
}

func (s *ServiceImpl) DisableIntegration(ctx context.Context) error {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	err = s.repo.DeleteConfiguration(ctx, userId)
	if err != nil {
		return err
	}

	err = s.repo.DeleteAuthData(ctx, userId)
	if err != nil {
		return err
	}
	return nil
}

func (s *ServiceImpl) GetTasksByBudgetId(ctx context.Context, budgetId int) ([]Task, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	configuration, err := s.repo.GetConfiguration(ctx, userId)
	if err != nil {
		return nil, err
	}

	var clickUpTagName string
	for _, mapping := range configuration.Mappings {
		if mapping.BudgetItemId == budgetId {
			clickUpTagName = mapping.ClickupTagName
		}
	}
	if clickUpTagName == "" {
		return []Task{}, nil
	}

	var allTasks []Task
	page := 0

	for {
		tasks, err := s.client.GetFilteredTeamTasks(
			ctx,
			configuration.WorkspaceId,
			configuration.SpaceId,
			configuration.FolderId,
			page,
			clickUpTagName,
			configuration.OnlyTasksWithPriority,
		)
		if err != nil {
			return nil, err
		}

		// If no tasks are returned, we've reached the end
		if len(tasks) == 0 {
			break
		}

		// Append the tasks from this page to our result
		allTasks = append(allTasks, tasks...)
		page++
	}

	return allTasks, nil
}
