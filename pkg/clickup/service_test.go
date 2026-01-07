package clickup

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klokku/klokku/pkg/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testUserId = 123

func ctxWithUserId(userId int) context.Context {
	return context.WithValue(context.Background(), user.UserKey, user.User{
		Id:          userId,
		Uid:         uuid.NewString(),
		Username:    "test-user-1",
		DisplayName: "Test User 1",
		PhotoUrl:    "",
		Settings: user.Settings{
			Timezone:          "Europe/Warsaw",
			WeekFirstDay:      time.Monday,
			EventCalendarType: user.KlokkuCalendar,
			GoogleCalendar:    user.GoogleCalendarSettings{},
		},
	})
}

func setupServiceTest(t *testing.T) (*ServiceImpl, *RepositoryStub, *ClientStub, context.Context) {
	repo := NewRepositoryStub()
	client := NewClientStub()
	service := NewServiceImpl(repo, client)
	ctx := ctxWithUserId(testUserId)
	t.Cleanup(func() {
		repo.Reset()
		client.Reset()
	})
	return service, repo, client, ctx
}

func TestServiceImpl_StoreConfiguration(t *testing.T) {
	t.Run("should store configuration with mappings", func(t *testing.T) {
		// given
		service, repo, _, ctx := setupServiceTest(t)
		budgetPlanId := 10
		config := Configuration{
			WorkspaceId:           "100",
			SpaceId:               "200",
			FolderId:              "300",
			OnlyTasksWithPriority: true,
			Mappings: []BudgetItemMapping{
				{
					ClickupSpaceId: "200",
					ClickupTagName: "urgent",
					BudgetItemId:   1,
					Position:       0,
				},
			},
		}

		// when
		err := service.StoreConfiguration(ctx, budgetPlanId, config)

		// then
		require.NoError(t, err)
		storedConfig, err := repo.GetConfiguration(ctx, testUserId, budgetPlanId)
		require.NoError(t, err)
		require.NotNil(t, storedConfig)
		assert.Equal(t, config.WorkspaceId, storedConfig.WorkspaceId)
		assert.Equal(t, config.SpaceId, storedConfig.SpaceId)
		assert.Equal(t, config.FolderId, storedConfig.FolderId)
		assert.Equal(t, config.OnlyTasksWithPriority, storedConfig.OnlyTasksWithPriority)
		assert.Len(t, storedConfig.Mappings, 1)
		assert.Equal(t, "urgent", storedConfig.Mappings[0].ClickupTagName)
	})

	t.Run("should update existing configuration", func(t *testing.T) {
		// given
		service, repo, _, ctx := setupServiceTest(t)
		budgetPlanId := 10
		initialConfig := Configuration{
			WorkspaceId:           "100",
			SpaceId:               "200",
			FolderId:              "300",
			OnlyTasksWithPriority: false,
			Mappings: []BudgetItemMapping{
				{
					ClickupSpaceId: "200",
					ClickupTagName: "old-tag",
					BudgetItemId:   1,
					Position:       0,
				},
			},
		}
		err := service.StoreConfiguration(ctx, budgetPlanId, initialConfig)
		require.NoError(t, err)

		updatedConfig := Configuration{
			WorkspaceId:           "100",
			SpaceId:               "201",
			FolderId:              "301",
			OnlyTasksWithPriority: true,
			Mappings: []BudgetItemMapping{
				{
					ClickupSpaceId: "201",
					ClickupTagName: "new-tag",
					BudgetItemId:   2,
					Position:       0,
				},
			},
		}

		// when
		err = service.StoreConfiguration(ctx, budgetPlanId, updatedConfig)

		// then
		require.NoError(t, err)
		storedConfig, err := repo.GetConfiguration(ctx, testUserId, budgetPlanId)
		require.NoError(t, err)
		require.NotNil(t, storedConfig)
		assert.Equal(t, "201", storedConfig.SpaceId)
		assert.Equal(t, "301", storedConfig.FolderId)
		assert.True(t, storedConfig.OnlyTasksWithPriority)
		assert.Len(t, storedConfig.Mappings, 1)
		assert.Equal(t, "new-tag", storedConfig.Mappings[0].ClickupTagName)
		assert.Equal(t, 2, storedConfig.Mappings[0].BudgetItemId)
	})

	t.Run("should fail when user ID is missing", func(t *testing.T) {
		// given
		service, _, _, _ := setupServiceTest(t)
		config := Configuration{
			WorkspaceId: "100",
			SpaceId:     "200",
			FolderId:    "300",
		}

		// when
		err := service.StoreConfiguration(context.Background(), 10, config)

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get current user")
	})

	t.Run("should fail when repository returns error", func(t *testing.T) {
		// given
		service, repo, _, ctx := setupServiceTest(t)
		repoWithError := NewRepositoryStubWithError(repo)
		repoWithError.SetStoreConfigurationError(ErrRepositoryTestError)
		service.repo = repoWithError

		config := Configuration{
			WorkspaceId: "100",
			SpaceId:     "200",
			FolderId:    "300",
		}

		// when
		err := service.StoreConfiguration(ctx, 10, config)

		// then
		assert.Error(t, err)
		assert.Equal(t, ErrRepositoryTestError, err)
	})
}

func TestServiceImpl_GetConfiguration(t *testing.T) {
	t.Run("should return configuration with mappings", func(t *testing.T) {
		// given
		service, repo, _, ctx := setupServiceTest(t)
		budgetPlanId := 15
		config := Configuration{
			WorkspaceId:           "100",
			SpaceId:               "200",
			FolderId:              "300",
			OnlyTasksWithPriority: false,
			Mappings: []BudgetItemMapping{
				{
					ClickupSpaceId: "200",
					ClickupTagName: "backend",
					BudgetItemId:   5,
					Position:       0,
				},
			},
		}
		err := repo.StoreConfiguration(ctx, testUserId, budgetPlanId, config)
		require.NoError(t, err)

		// when
		retrievedConfig, err := service.GetConfiguration(ctx, budgetPlanId)

		// then
		require.NoError(t, err)
		assert.Equal(t, "100", retrievedConfig.WorkspaceId)
		assert.Equal(t, "200", retrievedConfig.SpaceId)
		assert.Equal(t, "300", retrievedConfig.FolderId)
		assert.False(t, retrievedConfig.OnlyTasksWithPriority)
		assert.Len(t, retrievedConfig.Mappings, 1)
		assert.Equal(t, "backend", retrievedConfig.Mappings[0].ClickupTagName)
	})

	t.Run("should return empty configuration when not found", func(t *testing.T) {
		// given
		service, _, _, ctx := setupServiceTest(t)

		// when
		retrievedConfig, err := service.GetConfiguration(ctx, 999)

		// then
		require.NoError(t, err)
		assert.Equal(t, Configuration{}, retrievedConfig)
	})

	t.Run("should fail when user ID is missing", func(t *testing.T) {
		// given
		service, _, _, _ := setupServiceTest(t)

		// when
		_, err := service.GetConfiguration(context.Background(), 10)

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get current user")
	})

	t.Run("should fail when repository returns error", func(t *testing.T) {
		// given
		service, repo, _, ctx := setupServiceTest(t)
		repoWithError := NewRepositoryStubWithError(repo)
		repoWithError.SetGetConfigurationError(ErrRepositoryTestError)
		service.repo = repoWithError

		// when
		_, err := service.GetConfiguration(ctx, 10)

		// then
		assert.Error(t, err)
		assert.Equal(t, ErrRepositoryTestError, err)
	})
}

func TestServiceImpl_GetTasksByBudgetItemId(t *testing.T) {
	t.Run("should return tasks with pagination", func(t *testing.T) {
		// given
		service, repo, client, ctx := setupServiceTest(t)
		budgetItemId := 42
		config := Configuration{
			WorkspaceId:           "100",
			SpaceId:               "200",
			FolderId:              "300",
			OnlyTasksWithPriority: false,
			Mappings: []BudgetItemMapping{
				{
					ClickupSpaceId: "200",
					ClickupTagName: "feature",
					BudgetItemId:   budgetItemId,
					Position:       0,
				},
			},
		}
		err := repo.StoreConfiguration(ctx, testUserId, 20, config)
		require.NoError(t, err)

		tasksPage0 := []Task{
			{Id: "task1", Name: "Task 1", TimeEstimateMs: 3600000},
			{Id: "task2", Name: "Task 2", TimeEstimateMs: 200000},
		}
		tasksPage1 := []Task{
			{Id: "task3", Name: "Task 3", TimeEstimateMs: 1800000},
		}
		client.SetTasks("100", "200", "300", 0, "feature", false, tasksPage0)
		client.SetTasks("100", "200", "300", 1, "feature", false, tasksPage1)

		// when
		tasks, err := service.GetTasksByBudgetItemId(ctx, budgetItemId)

		// then
		require.NoError(t, err)
		assert.Len(t, tasks, 3)
		assert.Equal(t, "task1", tasks[0].Id)
		assert.Equal(t, "task2", tasks[1].Id)
		assert.Equal(t, "task3", tasks[2].Id)
	})

	t.Run("should filter tasks by priority when configured", func(t *testing.T) {
		// given
		service, repo, client, ctx := setupServiceTest(t)
		budgetItemId := 50
		config := Configuration{
			WorkspaceId:           "100",
			SpaceId:               "200",
			FolderId:              "300",
			OnlyTasksWithPriority: true,
			Mappings: []BudgetItemMapping{
				{
					ClickupSpaceId: "200",
					ClickupTagName: "urgent",
					BudgetItemId:   budgetItemId,
					Position:       0,
				},
			},
		}
		err := repo.StoreConfiguration(ctx, testUserId, 25, config)
		require.NoError(t, err)

		tasks := []Task{
			{Id: "task1", Name: "Urgent Task", TimeEstimateMs: 3600000},
		}
		client.SetTasks("100", "200", "300", 0, "urgent", true, tasks)

		// when
		retrievedTasks, err := service.GetTasksByBudgetItemId(ctx, budgetItemId)

		// then
		require.NoError(t, err)
		assert.Len(t, retrievedTasks, 1)
		assert.Equal(t, "task1", retrievedTasks[0].Id)
	})

	t.Run("should return empty slice when configuration not found", func(t *testing.T) {
		// given
		service, _, _, ctx := setupServiceTest(t)

		// when
		tasks, err := service.GetTasksByBudgetItemId(ctx, 999)

		// then
		require.NoError(t, err)
		assert.Empty(t, tasks)
		assert.NotNil(t, tasks)
	})

	t.Run("should return empty slice when no mappings exist", func(t *testing.T) {
		// given
		service, repo, _, ctx := setupServiceTest(t)
		config := Configuration{
			WorkspaceId:           "100",
			SpaceId:               "200",
			FolderId:              "300",
			OnlyTasksWithPriority: false,
			Mappings:              []BudgetItemMapping{},
		}
		err := repo.StoreConfiguration(ctx, testUserId, 30, config)
		require.NoError(t, err)

		// when
		tasks, err := service.GetTasksByBudgetItemId(ctx, 999)

		// then
		require.NoError(t, err)
		assert.Empty(t, tasks)
	})

	t.Run("should return empty slice when no tasks found", func(t *testing.T) {
		// given
		service, repo, client, ctx := setupServiceTest(t)
		budgetItemId := 42
		config := Configuration{
			WorkspaceId:           "100",
			SpaceId:               "200",
			FolderId:              "300",
			OnlyTasksWithPriority: false,
			Mappings: []BudgetItemMapping{
				{
					ClickupSpaceId: "200",
					ClickupTagName: "feature",
					BudgetItemId:   budgetItemId,
					Position:       0,
				},
			},
		}
		err := repo.StoreConfiguration(ctx, testUserId, 20, config)
		require.NoError(t, err)
		client.SetTasks("100", "200", "300", 0, "feature", false, []Task{})

		// when
		tasks, err := service.GetTasksByBudgetItemId(ctx, budgetItemId)

		// then
		require.NoError(t, err)
		assert.Empty(t, tasks)
		assert.NotNil(t, tasks)
	})

	t.Run("should stop pagination at limit of 100 pages", func(t *testing.T) {
		// given
		service, repo, client, ctx := setupServiceTest(t)
		budgetItemId := 42
		config := Configuration{
			WorkspaceId:           "100",
			SpaceId:               "200",
			FolderId:              "300",
			OnlyTasksWithPriority: false,
			Mappings: []BudgetItemMapping{
				{
					ClickupSpaceId: "200",
					ClickupTagName: "feature",
					BudgetItemId:   budgetItemId,
					Position:       0,
				},
			},
		}
		err := repo.StoreConfiguration(ctx, testUserId, 20, config)
		require.NoError(t, err)

		for i := 0; i <= 101; i++ {
			tasks := []Task{
				{Id: "task", Name: "Task", TimeEstimateMs: 3600000},
			}
			client.SetTasks("100", "200", "300", i, "feature", false, tasks)
		}

		// when
		tasks, err := service.GetTasksByBudgetItemId(ctx, budgetItemId)

		// then
		require.NoError(t, err)
		assert.Equal(t, 101, len(tasks))
	})

	t.Run("should fail when user ID is missing", func(t *testing.T) {
		// given
		service, _, _, _ := setupServiceTest(t)

		// when
		_, err := service.GetTasksByBudgetItemId(context.Background(), 42)

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get current user")
	})

	t.Run("should fail when repository returns error", func(t *testing.T) {
		// given
		service, repo, _, ctx := setupServiceTest(t)
		repoWithError := NewRepositoryStubWithError(repo)
		repoWithError.SetGetConfigurationWithMappingByBudgetItemIdError(ErrRepositoryTestError)
		service.repo = repoWithError

		// when
		_, err := service.GetTasksByBudgetItemId(ctx, 42)

		// then
		assert.Error(t, err)
		assert.Equal(t, ErrRepositoryTestError, err)
	})

	t.Run("should fail when client returns error", func(t *testing.T) {
		// given
		service, repo, client, ctx := setupServiceTest(t)
		budgetItemId := 42
		config := Configuration{
			WorkspaceId:           "100",
			SpaceId:               "200",
			FolderId:              "300",
			OnlyTasksWithPriority: false,
			Mappings: []BudgetItemMapping{
				{
					ClickupSpaceId: "200",
					ClickupTagName: "feature",
					BudgetItemId:   budgetItemId,
					Position:       0,
				},
			},
		}
		err := repo.StoreConfiguration(ctx, testUserId, 20, config)
		require.NoError(t, err)
		client.SetGetFilteredTeamTasksError(ErrClientTestError)

		// when
		_, err = service.GetTasksByBudgetItemId(ctx, budgetItemId)

		// then
		assert.Error(t, err)
		assert.Equal(t, ErrClientTestError, err)
	})
}

func TestServiceImpl_DisableIntegration(t *testing.T) {
	t.Run("should delete all configurations and auth data", func(t *testing.T) {
		// given
		service, repo, _, ctx := setupServiceTest(t)
		config1 := Configuration{
			WorkspaceId: "100",
			SpaceId:     "200",
			FolderId:    "300",
		}
		config2 := Configuration{
			WorkspaceId: "101",
			SpaceId:     "201",
			FolderId:    "301",
		}
		err := repo.StoreConfiguration(ctx, testUserId, 1, config1)
		require.NoError(t, err)
		err = repo.StoreConfiguration(ctx, testUserId, 2, config2)
		require.NoError(t, err)
		repo.SetAuthData(testUserId)

		// when
		err = service.DisableIntegration(ctx)

		// then
		require.NoError(t, err)
		allConfigs := repo.GetAllConfigs()
		assert.Empty(t, allConfigs)
		assert.False(t, repo.HasAuthData(testUserId))
	})

	t.Run("should fail when user ID is missing", func(t *testing.T) {
		// given
		service, _, _, _ := setupServiceTest(t)

		// when
		err := service.DisableIntegration(context.Background())

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get current user")
	})

	t.Run("should fail when delete configurations returns error", func(t *testing.T) {
		// given
		service, repo, _, ctx := setupServiceTest(t)
		repoWithError := NewRepositoryStubWithError(repo)
		repoWithError.SetDeleteAllConfigurationsError(ErrRepositoryTestError)
		service.repo = repoWithError

		// when
		err := service.DisableIntegration(ctx)

		// then
		assert.Error(t, err)
		assert.Equal(t, ErrRepositoryTestError, err)
	})

	t.Run("should fail when delete auth data returns error", func(t *testing.T) {
		// given
		service, repo, _, ctx := setupServiceTest(t)
		repoWithError := NewRepositoryStubWithError(repo)
		repoWithError.SetDeleteAuthDataError(ErrRepositoryTestError)
		service.repo = repoWithError

		// when
		err := service.DisableIntegration(ctx)

		// then
		assert.Error(t, err)
		assert.Equal(t, ErrRepositoryTestError, err)
	})
}

func TestServiceImpl_DeleteBudgetPlanConfiguration(t *testing.T) {
	t.Run("should delete specific budget plan configuration", func(t *testing.T) {
		// given
		service, repo, _, ctx := setupServiceTest(t)
		config1 := Configuration{WorkspaceId: "100", SpaceId: "200", FolderId: "300"}
		config2 := Configuration{WorkspaceId: "101", SpaceId: "201", FolderId: "301"}
		err := repo.StoreConfiguration(ctx, testUserId, 1, config1)
		require.NoError(t, err)
		err = repo.StoreConfiguration(ctx, testUserId, 2, config2)
		require.NoError(t, err)

		// when
		err = service.DeleteBudgetPlanConfiguration(ctx, 1)

		// then
		require.NoError(t, err)
		deletedConfig, err := repo.GetConfiguration(ctx, testUserId, 1)
		require.NoError(t, err)
		assert.Nil(t, deletedConfig)
		remainingConfig, err := repo.GetConfiguration(ctx, testUserId, 2)
		require.NoError(t, err)
		assert.NotNil(t, remainingConfig)
		assert.Equal(t, "101", remainingConfig.WorkspaceId)
	})

	t.Run("should succeed when configuration does not exist", func(t *testing.T) {
		// given
		service, _, _, ctx := setupServiceTest(t)

		// when
		err := service.DeleteBudgetPlanConfiguration(ctx, 999)

		// then
		require.NoError(t, err)
	})

	t.Run("should not delete configurations for other users", func(t *testing.T) {
		// given
		repo := NewRepositoryStub()
		client := NewClientStub()
		service := NewServiceImpl(repo, client)
		user1Id := 100
		user2Id := 200
		ctx1 := ctxWithUserId(user1Id)
		ctx2 := ctxWithUserId(user2Id)

		config1 := Configuration{WorkspaceId: "1000", SpaceId: "2000", FolderId: "3000"}
		config2 := Configuration{WorkspaceId: "1001", SpaceId: "2001", FolderId: "3001"}
		err := service.StoreConfiguration(ctx1, 1, config1)
		require.NoError(t, err)
		err = service.StoreConfiguration(ctx2, 1, config2)
		require.NoError(t, err)

		// when
		err = service.DeleteBudgetPlanConfiguration(ctx1, 1)

		// then
		require.NoError(t, err)
		deletedConfig, err := service.GetConfiguration(ctx1, 1)
		require.NoError(t, err)
		assert.Equal(t, Configuration{}, deletedConfig)
		stillExistsConfig, err := service.GetConfiguration(ctx2, 1)
		require.NoError(t, err)
		assert.Equal(t, "1001", stillExistsConfig.WorkspaceId)
	})

	t.Run("should fail with invalid budget plan ID", func(t *testing.T) {
		// given
		service, _, _, ctx := setupServiceTest(t)

		// when
		err := service.DeleteBudgetPlanConfiguration(ctx, 0)

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid budget plan ID")

		err = service.DeleteBudgetPlanConfiguration(ctx, -1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid budget plan ID")
	})

	t.Run("should fail when user ID is missing", func(t *testing.T) {
		// given
		service, _, _, _ := setupServiceTest(t)

		// when
		err := service.DeleteBudgetPlanConfiguration(context.Background(), 1)

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get current user")
	})

	t.Run("should fail when repository returns error", func(t *testing.T) {
		// given
		service, repo, _, ctx := setupServiceTest(t)
		repoWithError := NewRepositoryStubWithError(repo)
		repoWithError.SetDeleteBudgetPlanConfigurationError(ErrRepositoryTestError)
		service.repo = repoWithError

		// when
		err := service.DeleteBudgetPlanConfiguration(ctx, 1)

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete configuration")
	})
}
