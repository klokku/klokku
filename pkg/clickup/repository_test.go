package clickup

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klokku/klokku/internal/test_utils"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var pgContainer *postgres.PostgresContainer
var openDb func() *pgxpool.Pool

func TestMain(m *testing.M) {
	pgContainer, openDb = test_utils.TestWithDB()
	defer func() {
		if err := testcontainers.TerminateContainer(pgContainer); err != nil {
			log.Errorf("failed to terminate container: %s", err)
		}
	}()
	code := m.Run()
	os.Exit(code)
}

func setupTestRepository(t *testing.T) (context.Context, Repository, int) {
	ctx := context.Background()
	db := openDb()
	repository := NewRepository(db)
	t.Cleanup(func() {
		db.Close()
		err := pgContainer.Restore(ctx)
		require.NoError(t, err)
	})
	userId := 1001
	return ctx, repository, userId
}

func TestRepositoryImpl_StoreConfiguration(t *testing.T) {
	t.Run("should store configuration with mappings", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		config := Configuration{
			WorkspaceId:           "10",
			SpaceId:               "20",
			FolderId:              "30",
			OnlyTasksWithPriority: true,
			Mappings: []BudgetItemMapping{
				{
					ClickupSpaceId: "20",
					ClickupTagName: "clickup-tag-1",
					BudgetItemId:   101,
					Position:       1,
				},
			},
		}
		budgetPlanId := 1

		// when
		err := repo.StoreConfiguration(ctx, userId, budgetPlanId, config)

		// then
		require.NoError(t, err)
		storedConfig, err := repo.GetConfiguration(ctx, userId, budgetPlanId)
		require.NoError(t, err)
		assert.Equal(t, config, *storedConfig)
	})

	t.Run("should store configuration without mappings", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		config := Configuration{
			WorkspaceId:           "15",
			SpaceId:               "25",
			FolderId:              "35",
			OnlyTasksWithPriority: false,
			Mappings:              []BudgetItemMapping{},
		}
		budgetPlanId := 2

		// when
		err := repo.StoreConfiguration(ctx, userId, budgetPlanId, config)

		// then
		require.NoError(t, err)
		storedConfig, err := repo.GetConfiguration(ctx, userId, budgetPlanId)
		require.NoError(t, err)
		assert.Equal(t, config, *storedConfig)
	})

	t.Run("should update existing configuration", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		budgetPlanId := 3
		initialConfig := Configuration{
			WorkspaceId:           "10",
			SpaceId:               "20",
			FolderId:              "30",
			OnlyTasksWithPriority: true,
			Mappings: []BudgetItemMapping{
				{
					ClickupSpaceId: "20",
					ClickupTagName: "old-tag",
					BudgetItemId:   101,
					Position:       1,
				},
			},
		}
		err := repo.StoreConfiguration(ctx, userId, budgetPlanId, initialConfig)
		require.NoError(t, err)

		updatedConfig := Configuration{
			WorkspaceId:           "50",
			SpaceId:               "60",
			FolderId:              "70",
			OnlyTasksWithPriority: false,
			Mappings: []BudgetItemMapping{
				{
					ClickupSpaceId: "60",
					ClickupTagName: "new-tag",
					BudgetItemId:   201,
					Position:       1,
				},
			},
		}

		// when
		err = repo.StoreConfiguration(ctx, userId, budgetPlanId, updatedConfig)

		// then
		require.NoError(t, err)
		storedConfig, err := repo.GetConfiguration(ctx, userId, budgetPlanId)
		require.NoError(t, err)
		assert.Equal(t, updatedConfig, *storedConfig)
	})

	t.Run("should store multiple mappings with correct positions", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		config := Configuration{
			WorkspaceId:           "10",
			SpaceId:               "20",
			FolderId:              "30",
			OnlyTasksWithPriority: true,
			Mappings: []BudgetItemMapping{
				{
					ClickupSpaceId: "20",
					ClickupTagName: "tag-1",
					BudgetItemId:   101,
					Position:       2,
				},
				{
					ClickupSpaceId: "20",
					ClickupTagName: "tag-2",
					BudgetItemId:   102,
					Position:       1,
				},
				{
					ClickupSpaceId: "20",
					ClickupTagName: "tag-3",
					BudgetItemId:   103,
					Position:       3,
				},
			},
		}
		budgetPlanId := 4

		// when
		err := repo.StoreConfiguration(ctx, userId, budgetPlanId, config)

		// then
		require.NoError(t, err)
		storedConfig, err := repo.GetConfiguration(ctx, userId, budgetPlanId)
		require.NoError(t, err)
		// Mappings should be ordered by position
		assert.Equal(t, "tag-2", storedConfig.Mappings[0].ClickupTagName)
		assert.Equal(t, "tag-1", storedConfig.Mappings[1].ClickupTagName)
		assert.Equal(t, "tag-3", storedConfig.Mappings[2].ClickupTagName)
	})
}

func TestRepositoryImpl_GetConfiguration(t *testing.T) {
	t.Run("should return configuration with mappings", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		budgetPlanId := 1
		config := Configuration{
			WorkspaceId:           "10",
			SpaceId:               "20",
			FolderId:              "30",
			OnlyTasksWithPriority: true,
			Mappings: []BudgetItemMapping{
				{
					ClickupSpaceId: "20",
					ClickupTagName: "tag-1",
					BudgetItemId:   101,
					Position:       1,
				},
			},
		}
		err := repo.StoreConfiguration(ctx, userId, budgetPlanId, config)
		require.NoError(t, err)

		// when
		result, err := repo.GetConfiguration(ctx, userId, budgetPlanId)

		// then
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, config, *result)
	})

	t.Run("should return nil when configuration does not exist", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		budgetPlanId := 999

		// when
		result, err := repo.GetConfiguration(ctx, userId, budgetPlanId)

		// then
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("should return configuration with empty mappings", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		budgetPlanId := 2
		config := Configuration{
			WorkspaceId:           "15",
			SpaceId:               "25",
			FolderId:              "35",
			OnlyTasksWithPriority: false,
			Mappings:              []BudgetItemMapping{},
		}
		err := repo.StoreConfiguration(ctx, userId, budgetPlanId, config)
		require.NoError(t, err)

		// when
		result, err := repo.GetConfiguration(ctx, userId, budgetPlanId)

		// then
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, config, *result)
	})

	t.Run("should return only mappings for specified budget plan when user has multiple plans", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		budgetPlanId1 := 1
		budgetPlanId2 := 2

		// Store configuration for budget plan 1 with its mappings
		config1 := Configuration{
			WorkspaceId:           "10",
			SpaceId:               "20",
			FolderId:              "30",
			OnlyTasksWithPriority: true,
			Mappings: []BudgetItemMapping{
				{
					ClickupSpaceId: "20",
					ClickupTagName: "plan1-tag1",
					BudgetItemId:   101,
					Position:       1,
				},
				{
					ClickupSpaceId: "20",
					ClickupTagName: "plan1-tag2",
					BudgetItemId:   102,
					Position:       2,
				},
			},
		}
		err := repo.StoreConfiguration(ctx, userId, budgetPlanId1, config1)
		require.NoError(t, err)

		// Store configuration for budget plan 2 with different mappings
		config2 := Configuration{
			WorkspaceId:           "40",
			SpaceId:               "50",
			FolderId:              "60",
			OnlyTasksWithPriority: false,
			Mappings: []BudgetItemMapping{
				{
					ClickupSpaceId: "50",
					ClickupTagName: "plan2-tag1",
					BudgetItemId:   201,
					Position:       1,
				},
				{
					ClickupSpaceId: "50",
					ClickupTagName: "plan2-tag2",
					BudgetItemId:   202,
					Position:       2,
				},
			},
		}
		err = repo.StoreConfiguration(ctx, userId, budgetPlanId2, config2)
		require.NoError(t, err)

		// when - retrieve config for budget plan 1
		result1, err := repo.GetConfiguration(ctx, userId, budgetPlanId1)

		// then - should only get mappings for plan 1
		require.NoError(t, err)
		require.NotNil(t, result1)
		assert.Equal(t, config1.WorkspaceId, result1.WorkspaceId)
		assert.Equal(t, config1.SpaceId, result1.SpaceId)
		assert.Equal(t, config1.FolderId, result1.FolderId)
		assert.Equal(t, config1.OnlyTasksWithPriority, result1.OnlyTasksWithPriority)
		assert.Len(t, result1.Mappings, 2)
		assert.Equal(t, "plan1-tag1", result1.Mappings[0].ClickupTagName)
		assert.Equal(t, "plan1-tag2", result1.Mappings[1].ClickupTagName)
		assert.Equal(t, 101, result1.Mappings[0].BudgetItemId)
		assert.Equal(t, 102, result1.Mappings[1].BudgetItemId)

		// when - retrieve config for budget plan 2
		result2, err := repo.GetConfiguration(ctx, userId, budgetPlanId2)

		// then - should only get mappings for plan 2
		require.NoError(t, err)
		require.NotNil(t, result2)
		assert.Equal(t, config2.WorkspaceId, result2.WorkspaceId)
		assert.Equal(t, config2.SpaceId, result2.SpaceId)
		assert.Equal(t, config2.FolderId, result2.FolderId)
		assert.Equal(t, config2.OnlyTasksWithPriority, result2.OnlyTasksWithPriority)
		assert.Len(t, result2.Mappings, 2)
		assert.Equal(t, "plan2-tag1", result2.Mappings[0].ClickupTagName)
		assert.Equal(t, "plan2-tag2", result2.Mappings[1].ClickupTagName)
		assert.Equal(t, 201, result2.Mappings[0].BudgetItemId)
		assert.Equal(t, 202, result2.Mappings[1].BudgetItemId)
	})
}

func TestRepositoryImpl_GetConfigurationWithMappingByBudgetItemId(t *testing.T) {
	t.Run("should return configuration with specific mapping", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		budgetPlanId := 1
		budgetItemId := 101
		config := Configuration{
			WorkspaceId:           "10",
			SpaceId:               "20",
			FolderId:              "30",
			OnlyTasksWithPriority: true,
			Mappings: []BudgetItemMapping{
				{
					ClickupSpaceId: "20",
					ClickupTagName: "tag-1",
					BudgetItemId:   budgetItemId,
					Position:       1,
				},
				{
					ClickupSpaceId: "20",
					ClickupTagName: "tag-2",
					BudgetItemId:   102,
					Position:       2,
				},
			},
		}
		err := repo.StoreConfiguration(ctx, userId, budgetPlanId, config)
		require.NoError(t, err)

		// when
		result, err := repo.GetConfigurationWithMappingByBudgetItemId(ctx, userId, budgetItemId)

		// then
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, config.WorkspaceId, result.WorkspaceId)
		assert.Equal(t, config.SpaceId, result.SpaceId)
		assert.Equal(t, config.FolderId, result.FolderId)
		assert.Equal(t, config.OnlyTasksWithPriority, result.OnlyTasksWithPriority)
		assert.Len(t, result.Mappings, 1)
		assert.Equal(t, budgetItemId, result.Mappings[0].BudgetItemId)
		assert.Equal(t, "tag-1", result.Mappings[0].ClickupTagName)
	})

	t.Run("should return nil when mapping does not exist", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		nonExistentBudgetItemId := 999

		// when
		result, err := repo.GetConfigurationWithMappingByBudgetItemId(ctx, userId, nonExistentBudgetItemId)

		// then
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("should return correct configuration for different users", func(t *testing.T) {
		// given
		ctx, repo, _ := setupTestRepository(t)
		userId1 := 1001
		userId2 := 1002
		budgetPlanId := 1
		budgetItemId := 101

		config1 := Configuration{
			WorkspaceId:           "10",
			SpaceId:               "20",
			FolderId:              "30",
			OnlyTasksWithPriority: true,
			Mappings: []BudgetItemMapping{
				{
					ClickupSpaceId: "20",
					ClickupTagName: "user1-tag",
					BudgetItemId:   budgetItemId,
					Position:       1,
				},
			},
		}
		err := repo.StoreConfiguration(ctx, userId1, budgetPlanId, config1)
		require.NoError(t, err)

		// when - query for different user
		result, err := repo.GetConfigurationWithMappingByBudgetItemId(ctx, userId2, budgetItemId)

		// then
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestRepositoryImpl_DeleteAllConfigurations(t *testing.T) {
	t.Run("should delete all configurations for user", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		config1 := Configuration{
			WorkspaceId:           "10",
			SpaceId:               "20",
			FolderId:              "30",
			OnlyTasksWithPriority: true,
			Mappings:              []BudgetItemMapping{},
		}
		config2 := Configuration{
			WorkspaceId:           "15",
			SpaceId:               "25",
			FolderId:              "35",
			OnlyTasksWithPriority: false,
			Mappings:              []BudgetItemMapping{},
		}
		err := repo.StoreConfiguration(ctx, userId, 1, config1)
		require.NoError(t, err)
		err = repo.StoreConfiguration(ctx, userId, 2, config2)
		require.NoError(t, err)

		// when
		err = repo.DeleteAllConfigurations(ctx, userId)

		// then
		require.NoError(t, err)
		result1, err := repo.GetConfiguration(ctx, userId, 1)
		require.NoError(t, err)
		assert.Nil(t, result1)
		result2, err := repo.GetConfiguration(ctx, userId, 2)
		require.NoError(t, err)
		assert.Nil(t, result2)
	})

	t.Run("should not delete configurations for other users", func(t *testing.T) {
		// given
		ctx, repo, _ := setupTestRepository(t)
		userId1 := 1001
		userId2 := 1002
		budgetPlanId := 1
		config := Configuration{
			WorkspaceId:           "10",
			SpaceId:               "20",
			FolderId:              "30",
			OnlyTasksWithPriority: true,
			Mappings:              []BudgetItemMapping{},
		}
		err := repo.StoreConfiguration(ctx, userId1, budgetPlanId, config)
		require.NoError(t, err)
		err = repo.StoreConfiguration(ctx, userId2, budgetPlanId, config)
		require.NoError(t, err)

		// when
		err = repo.DeleteAllConfigurations(ctx, userId1)

		// then
		require.NoError(t, err)
		result1, err := repo.GetConfiguration(ctx, userId1, budgetPlanId)
		require.NoError(t, err)
		assert.Nil(t, result1)
		result2, err := repo.GetConfiguration(ctx, userId2, budgetPlanId)
		require.NoError(t, err)
		assert.NotNil(t, result2)
	})

	t.Run("should succeed when no configurations exist", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)

		// when
		err := repo.DeleteAllConfigurations(ctx, userId)

		// then
		require.NoError(t, err)
	})
}

func TestRepositoryImpl_DeleteBudgetPlanConfigurations(t *testing.T) {
	t.Run("should delete specific budget plan configuration", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		budgetPlanId1 := 1
		budgetPlanId2 := 2
		config1 := Configuration{
			WorkspaceId:           "10",
			SpaceId:               "20",
			FolderId:              "30",
			OnlyTasksWithPriority: true,
			Mappings:              []BudgetItemMapping{},
		}
		config2 := Configuration{
			WorkspaceId:           "15",
			SpaceId:               "25",
			FolderId:              "35",
			OnlyTasksWithPriority: false,
			Mappings:              []BudgetItemMapping{},
		}
		err := repo.StoreConfiguration(ctx, userId, budgetPlanId1, config1)
		require.NoError(t, err)
		err = repo.StoreConfiguration(ctx, userId, budgetPlanId2, config2)
		require.NoError(t, err)

		// when
		err = repo.DeleteBudgetPlanConfiguration(ctx, userId, budgetPlanId1)

		// then
		require.NoError(t, err)
		result1, err := repo.GetConfiguration(ctx, userId, budgetPlanId1)
		require.NoError(t, err)
		assert.Nil(t, result1)
		result2, err := repo.GetConfiguration(ctx, userId, budgetPlanId2)
		require.NoError(t, err)
		assert.NotNil(t, result2)
	})

	t.Run("should not delete configurations for other users", func(t *testing.T) {
		// given
		ctx, repo, _ := setupTestRepository(t)
		userId1 := 1001
		userId2 := 1002
		budgetPlanId := 1
		config := Configuration{
			WorkspaceId:           "10",
			SpaceId:               "20",
			FolderId:              "30",
			OnlyTasksWithPriority: true,
			Mappings:              []BudgetItemMapping{},
		}
		err := repo.StoreConfiguration(ctx, userId1, budgetPlanId, config)
		require.NoError(t, err)
		err = repo.StoreConfiguration(ctx, userId2, budgetPlanId, config)
		require.NoError(t, err)

		// when
		err = repo.DeleteBudgetPlanConfiguration(ctx, userId1, budgetPlanId)

		// then
		require.NoError(t, err)
		result1, err := repo.GetConfiguration(ctx, userId1, budgetPlanId)
		require.NoError(t, err)
		assert.Nil(t, result1)
		result2, err := repo.GetConfiguration(ctx, userId2, budgetPlanId)
		require.NoError(t, err)
		assert.NotNil(t, result2)
	})

	t.Run("should succeed when configuration does not exist", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		nonExistentBudgetPlanId := 999

		// when
		err := repo.DeleteBudgetPlanConfiguration(ctx, userId, nonExistentBudgetPlanId)

		// then
		require.NoError(t, err)
	})

	t.Run("should delete mappings when configuration is deleted", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		budgetPlanId := 1
		budgetItemId := 101
		config := Configuration{
			WorkspaceId:           "10",
			SpaceId:               "20",
			FolderId:              "30",
			OnlyTasksWithPriority: true,
			Mappings: []BudgetItemMapping{
				{
					ClickupSpaceId: "20",
					ClickupTagName: "tag-1",
					BudgetItemId:   budgetItemId,
					Position:       1,
				},
			},
		}
		err := repo.StoreConfiguration(ctx, userId, budgetPlanId, config)
		require.NoError(t, err)

		// when
		err = repo.DeleteBudgetPlanConfiguration(ctx, userId, budgetPlanId)

		// then
		require.NoError(t, err)
		result, err := repo.GetConfigurationWithMappingByBudgetItemId(ctx, userId, budgetItemId)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestRepositoryImpl_DeleteAuthData(t *testing.T) {
	t.Run("should delete auth data for user", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		db := openDb()
		defer db.Close()

		// Insert auth data
		_, err := db.Exec(ctx,
			"INSERT INTO clickup_auth (user_id, access_token, refresh_token) VALUES ($1, $2, $3)",
			userId, "access-token", "refresh-token")
		require.NoError(t, err)

		// when
		err = repo.DeleteAuthData(ctx, userId)

		// then
		require.NoError(t, err)
		var count int
		err = db.QueryRow(ctx, "SELECT COUNT(*) FROM clickup_auth WHERE user_id = $1", userId).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("should not delete auth data for other users", func(t *testing.T) {
		// given
		ctx, repo, _ := setupTestRepository(t)
		db := openDb()
		defer db.Close()
		userId1 := 1001
		userId2 := 1002

		// Insert auth data for both users
		_, err := db.Exec(ctx,
			"INSERT INTO clickup_auth (user_id, access_token, refresh_token) VALUES ($1, $2, $3)",
			userId1, "access-token-1", "refresh-token-1")
		require.NoError(t, err)
		_, err = db.Exec(ctx,
			"INSERT INTO clickup_auth (user_id, access_token, refresh_token) VALUES ($1, $2, $3)",
			userId2, "access-token-2", "refresh-token-2")
		require.NoError(t, err)

		// when
		err = repo.DeleteAuthData(ctx, userId1)

		// then
		require.NoError(t, err)
		var count int
		err = db.QueryRow(ctx, "SELECT COUNT(*) FROM clickup_auth WHERE user_id = $1", userId1).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
		err = db.QueryRow(ctx, "SELECT COUNT(*) FROM clickup_auth WHERE user_id = $1", userId2).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("should succeed when auth data does not exist", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)

		// when
		err := repo.DeleteAuthData(ctx, userId)

		// then
		require.NoError(t, err)
	})
}
