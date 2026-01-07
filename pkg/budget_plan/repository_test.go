package budget_plan

import (
	"context"
	"os"
	"testing"
	"time"

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
	repository := NewBudgetPlanRepo(db)
	t.Cleanup(func() {
		db.Close()
		err := pgContainer.Restore(ctx)
		require.NoError(t, err)
	})
	userId := 1
	return ctx, repository, userId
}

func TestRepositoryImpl_CreatePlan(t *testing.T) {
	// given
	ctx, repo, userId := setupTestRepository(t)

	// when
	plan, err := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Test Plan"})
	assert.NoError(t, err)

	// then
	storedPlans, err := repo.ListPlans(ctx, userId)
	assert.NoError(t, err)

	assert.Equal(t, "Test Plan", plan.Name)
	assert.Len(t, storedPlans, 1)
	assert.Equal(t, plan.Name, storedPlans[0].Name)

	// should set the plan as current when it is the first one created
	assert.Equal(t, storedPlans[0].IsCurrent, true)
}

func TestRepositoryImpl_DeletePlan(t *testing.T) {

	t.Run("should fail when deleting the only plan", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		_, err := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Some Plan"})
		require.NoError(t, err)
		currentPlan, err := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Current Plan"})
		require.NoError(t, err)
		_, err = repo.UpdatePlan(ctx, userId, BudgetPlan{Id: currentPlan.Id, IsCurrent: true})
		require.NoError(t, err)

		// when
		ok, err := repo.DeletePlan(ctx, userId, currentPlan.Id)

		// then
		assert.False(t, ok)
		assert.ErrorIs(t, err, ErrDeletingCurrentPlan)
	})

	t.Run("should delete non-current plan", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		plan1, err := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Plan 1"})
		assert.NoError(t, err)
		plan2, err := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Plan 2"})
		assert.NoError(t, err)

		// when
		ok, err := repo.DeletePlan(ctx, userId, plan2.Id)

		// then
		assert.NoError(t, err)
		assert.True(t, ok)
		plans, _ := repo.ListPlans(ctx, userId)
		assert.Len(t, plans, 1, "expected only one plan left")
		assert.Equal(t, plan1.Id, plans[0].Id)
	})
}

func TestRepositoryImpl_GetPlan(t *testing.T) {
	t.Run("should return the current plan", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		createdPlan, err := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Test Plan"})
		assert.NoError(t, err)

		// when
		plan, err := repo.GetPlan(ctx, userId, createdPlan.Id)

		// then
		assert.NoError(t, err)
		assert.Equal(t, createdPlan.Id, plan.Id)
		assert.Equal(t, "Test Plan", plan.Name)
		assert.True(t, plan.IsCurrent)
		assert.Empty(t, plan.Items)
	})

	t.Run("should error when plan is not found", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)

		// when
		_, err := repo.GetPlan(ctx, userId, 99999)

		// then
		assert.ErrorIs(t, err, ErrPlanNotFound)
	})

	t.Run("should return plan with items", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)

		// when
		plan1, _ := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Test Plan 1"})
		repo.StoreItem(ctx, userId, BudgetItem{PlanId: plan1.Id, Name: "Plan 1 Item 1", WeeklyDuration: time.Duration(1) * time.Hour})
		repo.StoreItem(ctx, userId, BudgetItem{PlanId: plan1.Id, Name: "Plan 1 Item 2", WeeklyDuration: time.Duration(2) * time.Hour})
		plan2, _ := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Test Plan 2"})
		repo.StoreItem(ctx, userId, BudgetItem{PlanId: plan2.Id, Name: "Plan 2 Item A", WeeklyDuration: time.Duration(3) * time.Hour})
		repo.StoreItem(ctx, userId, BudgetItem{PlanId: plan2.Id, Name: "Plan 2 Item B", WeeklyDuration: time.Duration(4) * time.Hour})

		// then
		storedPlan, _ := repo.GetPlan(ctx, userId, plan1.Id)
		assert.Len(t, storedPlan.Items, 2)
		assert.Equal(t, "Plan 1 Item 1", storedPlan.Items[0].Name)
		assert.Equal(t, "Plan 1 Item 2", storedPlan.Items[1].Name)
	})
}

func TestRepositoryImpl_ListPlans(t *testing.T) {
	// given
	ctx, repo, userId := setupTestRepository(t)
	plan1, _ := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Plan 1"})
	plan2, _ := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Plan 2"})

	// when
	plans, err := repo.ListPlans(ctx, userId)

	// then
	assert.NoError(t, err)
	assert.Len(t, plans, 2)
	assert.Equal(t, plan1.Id, plans[0].Id)
	assert.Equal(t, plan2.Id, plans[1].Id)
	assert.True(t, plans[0].IsCurrent)
	assert.False(t, plans[1].IsCurrent)
	assert.Empty(t, plans[0].Items)
	assert.Empty(t, plans[1].Items)
}

func TestRepositoryImpl_UpdatePlan(t *testing.T) {
	t.Run("should update the plan name", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		plan, err := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Original Name"})
		assert.NoError(t, err)

		// when
		updatedPlan, err := repo.UpdatePlan(ctx, userId, BudgetPlan{
			Id:   plan.Id,
			Name: "Updated Name",
		})

		// then
		assert.NoError(t, err)
		assert.Equal(t, "Updated Name", updatedPlan.Name)
		storedPlan, _ := repo.GetPlan(ctx, userId, plan.Id)
		assert.Equal(t, "Updated Name", storedPlan.Name)
	})

	t.Run("should return error for non-existent plan", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)

		// when
		_, err := repo.UpdatePlan(ctx, userId, BudgetPlan{Id: 99999, Name: "Test"})

		// then
		assert.ErrorIs(t, err, ErrPlanNotFound)
	})

	t.Run("should set the plan as current", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		plan1, _ := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Plan 1"})
		plan2, _ := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Plan 2"})

		// when
		_, err := repo.UpdatePlan(ctx, userId, BudgetPlan{
			Id:        plan2.Id,
			Name:      "Plan 2",
			IsCurrent: true,
		})

		// then
		assert.NoError(t, err)
		storedPlan1, _ := repo.GetPlan(ctx, userId, plan1.Id)
		storedPlan2, _ := repo.GetPlan(ctx, userId, plan2.Id)
		assert.False(t, storedPlan1.IsCurrent)
		assert.True(t, storedPlan2.IsCurrent)
	})
}

func TestRepositoryImpl_StoreItem(t *testing.T) {
	// given
	ctx, repo, userId := setupTestRepository(t)
	plan, _ := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Test Plan"})
	item := BudgetItem{
		PlanId:            plan.Id,
		Name:              "Test Item",
		WeeklyDuration:    3600000000000, // 1 hour in nanoseconds
		WeeklyOccurrences: 5,
		Icon:              "icon",
		Color:             "#FF0000",
		Position:          0,
	}

	// when
	itemId, _, err := repo.StoreItem(ctx, userId, item)

	// then
	assert.NoError(t, err)
	assert.Greater(t, itemId, 0)
	storedPlan, _ := repo.GetPlan(ctx, userId, plan.Id)
	assert.Len(t, storedPlan.Items, 1)
	assert.Equal(t, "Test Item", storedPlan.Items[0].Name)
	assert.Equal(t, 5, storedPlan.Items[0].WeeklyOccurrences)
	assert.Equal(t, "#FF0000", storedPlan.Items[0].Color)
	assert.Equal(t, "icon", storedPlan.Items[0].Icon)
}

func TestRepositoryImpl_UpdateItem(t *testing.T) {
	t.Run("should update budget item properties", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		plan, _ := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Test Plan"})
		itemId, position, _ := repo.StoreItem(ctx, userId, BudgetItem{
			PlanId:            plan.Id,
			Name:              "Original",
			WeeklyDuration:    3600, // 1 hour in seconds
			WeeklyOccurrences: 3,
			Icon:              "old",
			Color:             "#000000",
			Position:          0,
		})

		// when
		_, err := repo.UpdateItem(ctx, userId, BudgetItem{
			Id:                itemId,
			Name:              "Updated",
			WeeklyDuration:    7200, // 2 hours in seconds
			WeeklyOccurrences: 7,
			Icon:              "new",
			Color:             "#FFFFFF",
		})

		// then
		assert.NoError(t, err)
		storedPlan, _ := repo.GetPlan(ctx, userId, plan.Id)
		assert.Equal(t, "Updated", storedPlan.Items[0].Name)
		assert.Equal(t, 7, storedPlan.Items[0].WeeklyOccurrences)
		assert.Equal(t, "#FFFFFF", storedPlan.Items[0].Color)
		assert.Equal(t, "new", storedPlan.Items[0].Icon)
		assert.Equal(t, position, storedPlan.Items[0].Position)
	})

	t.Run("should return updated item", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		plan, _ := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Test Plan"})
		itemId, position, _ := repo.StoreItem(ctx, userId, BudgetItem{
			PlanId:            plan.Id,
			Name:              "Original",
			WeeklyDuration:    3600, // 1 hour in seconds
			WeeklyOccurrences: 3,
			Icon:              "old",
			Color:             "#000000",
			Position:          0,
		})

		// when
		updatedItem, err := repo.UpdateItem(ctx, userId, BudgetItem{
			Id:                itemId,
			Name:              "Updated",
			WeeklyDuration:    7200, // 2 hours in seconds
			WeeklyOccurrences: 7,
			Icon:              "new",
			Color:             "#FFFFFF",
		})

		// then
		assert.NoError(t, err)
		assert.Equal(t, "Updated", updatedItem.Name)
		assert.Equal(t, 7, updatedItem.WeeklyOccurrences)
		assert.Equal(t, "#FFFFFF", updatedItem.Color)
		assert.Equal(t, "new", updatedItem.Icon)
		assert.Equal(t, position, updatedItem.Position)
	})
}

func TestRepositoryImpl_UpdateItemPosition(t *testing.T) {
	// given
	ctx, repo, userId := setupTestRepository(t)
	plan, _ := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Test Plan"})
	itemId, _, _ := repo.StoreItem(ctx, userId, BudgetItem{
		PlanId: plan.Id,
		Name:   "Item",
	})

	// when
	ok, err := repo.UpdateItemPosition(ctx, userId, BudgetItem{
		Id:       itemId,
		Position: 5,
	})

	// then
	assert.NoError(t, err)
	assert.True(t, ok)
	storedPlan, _ := repo.GetPlan(ctx, userId, plan.Id)
	assert.Equal(t, 5, storedPlan.Items[0].Position)
}

func TestRepositoryImpl_DeleteItem(t *testing.T) {
	// given
	ctx, repo, userId := setupTestRepository(t)
	plan, _ := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Test Plan"})
	itemId, _, _ := repo.StoreItem(ctx, userId, BudgetItem{
		PlanId:   plan.Id,
		Name:     "Item to Delete",
		Position: 0,
	})

	// when
	ok, err := repo.DeleteItem(ctx, userId, itemId)

	// then
	assert.NoError(t, err)
	assert.True(t, ok)
	storedPlan, _ := repo.GetPlan(ctx, userId, plan.Id)
	assert.Empty(t, storedPlan.Items)
}

func TestRepositoryImpl_GetCurrentPlan(t *testing.T) {
	t.Run("should return the current plan", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		_, err := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Test Plan 1"})
		require.NoError(t, err)
		currentPlan, err := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Test Plan 2"})
		require.NoError(t, err)
		_, err = repo.UpdatePlan(ctx, userId, BudgetPlan{Id: currentPlan.Id, Name: currentPlan.Name, IsCurrent: true})
		require.NoError(t, err)

		// when
		plan, err := repo.GetCurrentPlan(ctx, userId)

		// then
		assert.NoError(t, err)
		assert.Equal(t, currentPlan.Id, plan.Id)
		assert.Equal(t, "Test Plan 2", plan.Name)
		assert.True(t, plan.IsCurrent)
	})

	t.Run("should error when plan is not found", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)

		// when
		_, err := repo.GetCurrentPlan(ctx, userId)

		// then
		assert.ErrorIs(t, err, ErrPlanNotFound)
	})

	t.Run("should return plan with items", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)

		// when
		plan1, _ := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Test Plan 1"})
		repo.StoreItem(ctx, userId, BudgetItem{PlanId: plan1.Id, Name: "Plan 1 Item 1", WeeklyDuration: time.Duration(1) * time.Hour})
		repo.StoreItem(ctx, userId, BudgetItem{PlanId: plan1.Id, Name: "Plan 1 Item 2", WeeklyDuration: time.Duration(2) * time.Hour})

		// then
		storedPlan, _ := repo.GetCurrentPlan(ctx, userId)
		assert.Len(t, storedPlan.Items, 2)
		assert.Equal(t, "Plan 1 Item 1", storedPlan.Items[0].Name)
		assert.Equal(t, "Plan 1 Item 2", storedPlan.Items[1].Name)
	})
}

func TestRepositoryImpl_GetItem(t *testing.T) {
	// given
	ctx, repo, userId := setupTestRepository(t)
	plan, _ := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Test Plan"})
	itemId, _, _ := repo.StoreItem(ctx, userId, BudgetItem{
		PlanId:            plan.Id,
		Name:              "Item 1",
		WeeklyDuration:    30 * time.Minute,
		WeeklyOccurrences: 2,
		Icon:              "some-icon",
		Color:             "#DDDDFF",
		Position:          2,
	})

	// when
	item, err := repo.GetItem(ctx, userId, itemId)

	// then
	assert.NoError(t, err)
	assert.Equal(t, itemId, item.Id)
	assert.Equal(t, "Item 1", item.Name)
	assert.Equal(t, 30*time.Minute, item.WeeklyDuration)
	assert.Equal(t, 2, item.WeeklyOccurrences)
	assert.Equal(t, "#DDDDFF", item.Color)
	assert.Equal(t, "some-icon", item.Icon)
}
