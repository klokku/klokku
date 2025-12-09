package budget_plan

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/klokku/klokku/internal/test_utils"
	"github.com/stretchr/testify/assert"
)

var db *pgx.Conn

func TestMain(m *testing.M) {
	var cleanup func()
	db, cleanup = test_utils.TestWithDB()
	defer cleanup()
	code := m.Run()
	os.Exit(code)
}

func setupTestRepository(t *testing.T) (context.Context, Repository, int) {
	ctx := context.Background()
	repository := NewBudgetRepo(db)
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

func TestRepositoryImpl_DeletePlan_ShouldFailWhenDeletingTheOnlyPlan(t *testing.T) {
	// given
	ctx, repo, userId := setupTestRepository(t)
	_, err := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Some Plan"})
	currentPlan, err := repo.CreatePlan(ctx, userId, BudgetPlan{Name: "Current Plan"})
	assert.NoError(t, err)
	_, err = repo.UpdatePlan(ctx, userId, BudgetPlan{Id: currentPlan.Id, IsCurrent: true})
	assert.NoError(t, err)

	// when
	ok, err := repo.DeletePlan(ctx, userId, currentPlan.Id)

	// then
	assert.False(t, ok)
	assert.ErrorIs(t, err, ErrDeletingCurrentPlan)
}
