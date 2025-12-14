package budget_plan

import (
	"context"
	"testing"
	"time"

	"github.com/klokku/klokku/pkg/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.WithValue(context.Background(), user.UserIDKey, 1)

var budgetRepoStub = NewStubBudgetRepo()

var service Service

func setup(t *testing.T) func() {
	service = NewBudgetPlanService(budgetRepoStub)
	return func() {
		t.Log("Teardown after test")
		budgetRepoStub.Cleanup()
	}
}

func TestServiceImpl_GetPlan(t *testing.T) {
	t.Run("should get a plan successfully", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// given
		createdPlan, _ := service.CreatePlan(ctx, BudgetPlan{Name: "Test Plan"})

		// when
		result, err := service.GetPlan(ctx, createdPlan.Id)

		// then
		assert.NoError(t, err)
		assert.Equal(t, createdPlan.Id, result.Id)
		assert.Equal(t, "Test Plan", result.Name)
	})

	t.Run("should return error when context has no user", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// when
		_, err := service.GetPlan(context.Background(), 1)

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get current user")
	})
}

func TestServiceImpl_GetCurrentPlan(t *testing.T) {
	t.Run("should get a plan successfully", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// given
		_, err := service.CreatePlan(ctx, BudgetPlan{Name: "Test Plan 1"})
		require.NoError(t, err)
		currentPlan, err := service.CreatePlan(ctx, BudgetPlan{Name: "Test Plan 2"})
		require.NoError(t, err)
		_, err = service.UpdatePlan(ctx, BudgetPlan{Id: currentPlan.Id, Name: currentPlan.Name, IsCurrent: true})
		require.NoError(t, err)

		// when
		result, err := service.GetCurrentPlan(ctx)

		// then
		assert.NoError(t, err)
		assert.Equal(t, currentPlan.Id, result.Id)
		assert.Equal(t, "Test Plan 2", result.Name)
	})

	t.Run("should return error when context has no user", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// when
		_, err := service.GetCurrentPlan(context.Background())

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get current user")
	})
}

func TestServiceImpl_ListPlans(t *testing.T) {
	t.Run("should list all plans for user", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// given
		service.CreatePlan(ctx, BudgetPlan{Name: "Plan 1"})
		service.CreatePlan(ctx, BudgetPlan{Name: "Plan 2"})

		// when
		plans, err := service.ListPlans(ctx)

		// then
		assert.NoError(t, err)
		assert.Len(t, plans, 2)
	})

	t.Run("should return error when context has no user", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// when
		_, err := service.ListPlans(context.Background())

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get current user")
	})
}

func TestServiceImpl_CreatePlan(t *testing.T) {
	t.Run("should create a new plan", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// when
		createdPlan, err := service.CreatePlan(ctx, BudgetPlan{Name: "Test Plan"})

		// then
		assert.NoError(t, err)
		assert.Equal(t, "Test Plan", createdPlan.Name)
		assert.True(t, createdPlan.IsCurrent) // The first created plan should be current
	})

	t.Run("should return error when context has no user", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// when
		_, err := service.CreatePlan(context.Background(), BudgetPlan{Name: "Test"})

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get current user")
	})
}

func TestServiceImpl_UpdatePlan(t *testing.T) {
	t.Run("should update an existing plan", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// given
		createdPlan, _ := service.CreatePlan(ctx, BudgetPlan{Name: "Original Name"})
		createdPlan.Name = "Updated Name"

		// when
		updatedPlan, err := service.UpdatePlan(ctx, createdPlan)

		// then
		assert.NoError(t, err)
		assert.Equal(t, "Updated Name", updatedPlan.Name)
		assert.Equal(t, createdPlan.Id, updatedPlan.Id)
	})

	t.Run("should return error when context has no user", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// when
		_, err := service.UpdatePlan(context.Background(), BudgetPlan{Id: 1, Name: "Test"})

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get current user")
	})
}

func TestServiceImpl_DeletePlan(t *testing.T) {
	t.Run("should delete an existing plan", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// given
		service.CreatePlan(ctx, BudgetPlan{Name: "Not to be deleted"})
		createdPlan, _ := service.CreatePlan(ctx, BudgetPlan{Name: "To Delete"})

		// when
		deleted, err := service.DeletePlan(ctx, createdPlan.Id)

		// then
		assert.NoError(t, err)
		assert.True(t, deleted)

		plans, err := service.ListPlans(ctx)
		require.NoError(t, err)
		assert.Len(t, plans, 1)
		assert.Equal(t, "Not to be deleted", plans[0].Name)
	})

	t.Run("should return error when this is the only plan left", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// given
		createdPlan, _ := service.CreatePlan(ctx, BudgetPlan{Name: "To Delete"})

		// when
		deleted, err := service.DeletePlan(ctx, createdPlan.Id)

		// then
		assert.Error(t, err)
		assert.False(t, deleted)
		assert.Contains(t, err.Error(), "cannot delete current plan")
	})

	t.Run("should return error when context has no user", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// when
		_, err := service.DeletePlan(context.Background(), 1)

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})
}

func TestServiceImpl_CreateItem(t *testing.T) {
	t.Run("should create a budget item with correct position", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// given
		plan, _ := service.CreatePlan(ctx, BudgetPlan{Name: "Test Plan"})

		// when
		item, err := service.CreateItem(ctx, BudgetItem{
			PlanId:            plan.Id,
			Name:              "Test Item",
			WeeklyDuration:    time.Duration(2) * time.Hour,
			WeeklyOccurrences: 3,
			Icon:              "SomeIcon",
			Color:             "#FF0000",
		})

		// then
		assert.NoError(t, err)
		assert.NotZero(t, item.Id)
		assert.Equal(t, "Test Item", item.Name)
		assert.Equal(t, time.Duration(2)*time.Hour, item.WeeklyDuration)
		assert.Equal(t, 3, item.WeeklyOccurrences)
		assert.Equal(t, "SomeIcon", item.Icon)
		assert.Equal(t, "#FF0000", item.Color)
		assert.Equal(t, 100, item.Position) // First item should have position 100
	})

	t.Run("should create items with incrementing positions", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// given
		plan, _ := service.CreatePlan(ctx, BudgetPlan{Name: "Test Plan"})
		item1, _ := service.CreateItem(ctx, BudgetItem{PlanId: plan.Id, Name: "Item 1"})

		// when
		item2, err := service.CreateItem(ctx, BudgetItem{PlanId: plan.Id, Name: "Item 2"})

		// then
		assert.NoError(t, err)
		assert.Greater(t, item2.Position, item1.Position)
	})

	t.Run("should return error when context has no user", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// when
		_, err := service.CreateItem(context.Background(), BudgetItem{Name: "Test"})

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get current user")
	})
}

func TestServiceImpl_UpdateItem(t *testing.T) {
	t.Run("should update an existing item", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// given
		plan, _ := service.CreatePlan(ctx, BudgetPlan{Name: "Test Plan"})
		item, _ := service.CreateItem(ctx, BudgetItem{
			PlanId:            plan.Id,
			Name:              "Original",
			WeeklyDuration:    time.Duration(2) * time.Hour,
			WeeklyOccurrences: 3,
			Icon:              "SomeIcon",
			Color:             "#FF0000"})
		item.Name = "Updated"
		item.WeeklyDuration = time.Duration(4) * time.Hour

		// when
		updated, err := service.UpdateItem(ctx, item)

		// then
		assert.NoError(t, err)
		assert.True(t, updated)
		assert.Equal(t, "Updated", item.Name)
		assert.Equal(t, time.Duration(4)*time.Hour, item.WeeklyDuration)
	})

	t.Run("should return error when context has no user", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// when
		_, err := service.UpdateItem(context.Background(), BudgetItem{Id: 1, Name: "Test"})

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get current user")
	})
}

func TestServiceImpl_DeleteItem(t *testing.T) {
	t.Run("should delete an existing item", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// given
		plan, _ := service.CreatePlan(ctx, BudgetPlan{Name: "Test Plan"})
		item, _ := service.CreateItem(ctx, BudgetItem{PlanId: plan.Id, Name: "To Delete"})

		// when
		deleted, err := service.DeleteItem(ctx, item.Id)

		// then
		assert.NoError(t, err)
		assert.True(t, deleted)
	})

	t.Run("should return error when context has no user", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// when
		_, err := service.DeleteItem(context.Background(), 1)

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get current user")
	})
}

func TestServiceImpl_MoveItemAfter(t *testing.T) {
	t.Run("should move item to new position with space", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// given
		plan, _ := service.CreatePlan(ctx, BudgetPlan{Name: "Test Plan"})
		item1, _ := service.CreateItem(ctx, BudgetItem{PlanId: plan.Id, Name: "Item 1"})
		service.CreateItem(ctx, BudgetItem{PlanId: plan.Id, Name: "Item 2"})
		item3, _ := service.CreateItem(ctx, BudgetItem{PlanId: plan.Id, Name: "Item 3"})

		// when - move item3 after item1
		moved, err := service.MoveItemAfter(ctx, plan.Id, item3.Id, item1.Id)

		// then
		assert.NoError(t, err)
		assert.True(t, moved)
	})

	t.Run("should move item to first position", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// given
		plan, _ := service.CreatePlan(ctx, BudgetPlan{Name: "Test Plan"})
		service.CreateItem(ctx, BudgetItem{PlanId: plan.Id, Name: "Item 1"})
		item2, _ := service.CreateItem(ctx, BudgetItem{PlanId: plan.Id, Name: "Item 2"})

		// when - move item2 to first position (precedingId = -1 or 0)
		moved, err := service.MoveItemAfter(ctx, plan.Id, item2.Id, -1)

		// then
		assert.NoError(t, err)
		assert.True(t, moved)
	})

	t.Run("should move item to last position", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// given
		plan, _ := service.CreatePlan(ctx, BudgetPlan{Name: "Test Plan"})
		item1, _ := service.CreateItem(ctx, BudgetItem{PlanId: plan.Id, Name: "Item 1"})
		item2, _ := service.CreateItem(ctx, BudgetItem{PlanId: plan.Id, Name: "Item 2"})

		// when - move item1 after item2 (to last position)
		moved, err := service.MoveItemAfter(ctx, plan.Id, item1.Id, item2.Id)

		// then
		assert.NoError(t, err)
		assert.True(t, moved)
	})

	t.Run("should return error when context has no user", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// when
		_, err := service.MoveItemAfter(context.Background(), 1, 1, 1)

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get current user")
	})
}
