package current_event

import (
	"context"
	"testing"
	"time"

	"github.com/klokku/klokku/internal/test_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

func TestRepositoryImpl_CurrentEventNotes(t *testing.T) {
	container, openDb := test_utils.TestWithDB()
	ctx := context.Background()
	t.Cleanup(func() {
		require.NoError(t, testcontainers.TerminateContainer(container))
	})
	db := openDb()
	t.Cleanup(db.Close)

	var budgetPlanId int
	err := db.QueryRow(ctx, `INSERT INTO budget_plan (name, user_id) VALUES ($1, $2) RETURNING id`, "Test plan", 1).Scan(&budgetPlanId)
	require.NoError(t, err)
	var budgetItemId int
	err = db.QueryRow(ctx, `INSERT INTO budget_item
		(budget_plan_id, name, weekly_duration_sec, position, user_id)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`, budgetPlanId, "Test item", 3600, 1, 1).Scan(&budgetItemId)
	require.NoError(t, err)

	repository := NewEventRepo(db)
	event := CurrentEvent{
		StartTime: time.Now().Truncate(time.Millisecond),
		Notes:     "Persisted note",
		PlanItem: PlanItem{
			BudgetItemId:   budgetItemId,
			Name:           "Test item",
			WeeklyDuration: time.Hour,
		},
	}

	_, err = repository.ReplaceCurrentEvent(ctx, 1, event)
	require.NoError(t, err)
	found, err := repository.FindCurrentEvent(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, "Persisted note", found.Notes)
	assert.Equal(t, event.StartTime, found.StartTime)
	assert.Equal(t, event.PlanItem, found.PlanItem)

	event.Notes = ""
	_, err = repository.ReplaceCurrentEvent(ctx, 1, event)
	require.NoError(t, err)
	found, err = repository.FindCurrentEvent(ctx, 1)
	require.NoError(t, err)
	assert.Empty(t, found.Notes)

	otherUserEvent, err := repository.FindCurrentEvent(ctx, 2)
	require.NoError(t, err)
	assert.Zero(t, otherUserEvent.Id)
}
