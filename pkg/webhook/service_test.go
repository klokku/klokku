package webhook

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klokku/klokku/pkg/budget_plan"
	"github.com/klokku/klokku/pkg/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.WithValue(context.Background(), user.UserKey, user.User{
	Id:          10,
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

var repoStub = NewRepositoryStub()
var eventStarterStub = NewEventStarterStub()
var budgetProviderStub = NewBudgetProviderStub()
var userProviderStub = NewUserProviderStub()

var service Service

func setup(t *testing.T) func() {
	service = NewService(repoStub, eventStarterStub, budgetProviderStub, userProviderStub)
	return func() {
		t.Log("Teardown after test")
		repoStub.Reset()
		eventStarterStub.Reset()
		budgetProviderStub.Reset()
		userProviderStub.Reset()
	}
}

func TestServiceImpl_Create(t *testing.T) {
	t.Run("should create webhook with data", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		data := StartCurrentEventData{BudgetItemId: 42}

		// when
		webhook, err := service.Create(ctx, TypeStartCurrentEvent, data)

		// then
		require.NoError(t, err)
		assert.Equal(t, 1, webhook.Id)
		assert.Equal(t, TypeStartCurrentEvent, webhook.Type)
		assert.Equal(t, 10, webhook.UserId)
		assert.NotEmpty(t, webhook.Token)
		assert.Len(t, webhook.Token, 64) // 32 bytes = 64 hex chars

		// Verify data is stored correctly
		var storedData StartCurrentEventData
		err = json.Unmarshal(webhook.Data, &storedData)
		require.NoError(t, err)
		assert.Equal(t, 42, storedData.BudgetItemId)
	})

	t.Run("should return error when user not in context", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		emptyCtx := context.Background()

		// when
		_, err := service.Create(emptyCtx, TypeStartCurrentEvent, StartCurrentEventData{})

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get current user")
	})
}

func TestServiceImpl_GetByUserIdAndType(t *testing.T) {
	t.Run("should return user's webhooks", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// Create webhooks
		_, err := service.Create(ctx, TypeStartCurrentEvent, StartCurrentEventData{BudgetItemId: 1})
		require.NoError(t, err)
		_, err = service.Create(ctx, TypeStartCurrentEvent, StartCurrentEventData{BudgetItemId: 2})
		require.NoError(t, err)

		// when
		webhooks, err := service.GetByUserIdAndType(ctx, TypeStartCurrentEvent)

		// then
		require.NoError(t, err)
		assert.Len(t, webhooks, 2)
		assert.Equal(t, 10, webhooks[0].UserId)
		assert.Equal(t, 10, webhooks[1].UserId)
	})

	t.Run("should return empty list when no webhooks exist", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// when
		webhooks, err := service.GetByUserIdAndType(ctx, TypeStartCurrentEvent)

		// then
		require.NoError(t, err)
		assert.Len(t, webhooks, 0)
	})

	t.Run("should return error when user not in context", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		emptyCtx := context.Background()

		// when
		_, err := service.GetByUserIdAndType(emptyCtx, TypeStartCurrentEvent)

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get current user")
	})
}

func TestServiceImpl_RotateToken(t *testing.T) {
	t.Run("should rotate token", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// Create webhook first
		webhook, err := service.Create(ctx, TypeStartCurrentEvent, StartCurrentEventData{BudgetItemId: 42})
		require.NoError(t, err)
		originalToken := webhook.Token

		// when
		newToken, err := service.RotateToken(ctx, webhook.Id)

		// then
		require.NoError(t, err)
		assert.NotEmpty(t, newToken)
		assert.NotEqual(t, originalToken, newToken)
		assert.Len(t, newToken, 64) // 32 bytes = 64 hex chars

		// Verify old token doesn't work
		_, err = repoStub.GetByToken(ctx, originalToken)
		assert.ErrorIs(t, err, ErrWebhookNotFound)

		// Verify new token works
		retrieved, err := repoStub.GetByToken(ctx, newToken)
		require.NoError(t, err)
		assert.Equal(t, webhook.Id, retrieved.Id)
	})

	t.Run("should return error when user not in context", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		emptyCtx := context.Background()

		// when
		_, err := service.RotateToken(emptyCtx, 1)

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get current user")
	})
}

func TestServiceImpl_Delete(t *testing.T) {
	t.Run("should delete webhook", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// Create webhook first
		webhook, err := service.Create(ctx, TypeStartCurrentEvent, StartCurrentEventData{BudgetItemId: 42})
		require.NoError(t, err)

		// when
		err = service.Delete(ctx, webhook.Id)

		// then
		require.NoError(t, err)

		// Verify webhook is deleted
		_, err = repoStub.GetByToken(ctx, webhook.Token)
		assert.ErrorIs(t, err, ErrWebhookNotFound)
	})

	t.Run("should return error when user not in context", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		emptyCtx := context.Background()

		// when
		err := service.Delete(emptyCtx, 1)

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get current user")
	})
}

func TestServiceImpl_Execute(t *testing.T) {
	t.Run("should execute START_CURRENT_EVENT webhook", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		budgetItemId := 42

		// Create webhook
		webhook, err := service.Create(ctx, TypeStartCurrentEvent, StartCurrentEventData{BudgetItemId: budgetItemId})
		require.NoError(t, err)

		// Setup stubs
		testUser := user.User{
			Id:          10,
			Uid:         "user-uid",
			Username:    "testuser",
			DisplayName: "Test User",
		}
		userProviderStub.SetUser(10, testUser)

		budgetItem := budget_plan.BudgetItem{
			Id:             budgetItemId,
			Name:           "Work",
			WeeklyDuration: 40 * time.Hour,
		}
		budgetProviderStub.SetItem(budgetItemId, budgetItem)

		// when
		err = service.Execute(context.Background(), webhook.Token)

		// then
		require.NoError(t, err)

		// Verify event was started
		events := eventStarterStub.GetStartedEvents()
		require.Len(t, events, 1)
		assert.Equal(t, budgetItemId, events[0].PlanItem.BudgetItemId)
		assert.Equal(t, "Work", events[0].PlanItem.Name)
		assert.Equal(t, 40*time.Hour, events[0].PlanItem.WeeklyDuration)
	})

	t.Run("should return error for invalid token", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// when
		err := service.Execute(context.Background(), "invalid-token")

		// then
		require.ErrorIs(t, err, ErrWebhookNotFound)
	})

	t.Run("should return error for unsupported webhook type", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// Create webhook with unsupported type directly in repo
		webhook := Webhook{
			Type:   WebhookType("UNSUPPORTED_TYPE"),
			UserId: 10,
			Data:   json.RawMessage(`{}`),
		}
		created, err := repoStub.Create(ctx, webhook)
		require.NoError(t, err)

		// Setup user stub
		testUser := user.User{Id: 10}
		userProviderStub.SetUser(10, testUser)

		// when
		err = service.Execute(context.Background(), created.Token)

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported webhook type")
	})

	t.Run("should return error when user not found", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// Create webhook with non-existent user directly in repo
		webhook := Webhook{
			Type:   TypeStartCurrentEvent,
			UserId: 999,
			Data:   json.RawMessage(`{"budgetItemId": 42}`),
		}
		created, err := repoStub.Create(ctx, webhook)
		require.NoError(t, err)

		// Don't setup user stub - user not found

		// when
		err = service.Execute(context.Background(), created.Token)

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get user")
	})

	t.Run("should return error when budget item not found", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// Create webhook
		webhook, err := service.Create(ctx, TypeStartCurrentEvent, StartCurrentEventData{BudgetItemId: 999})
		require.NoError(t, err)

		// Setup user stub but not budget item
		testUser := user.User{Id: 10}
		userProviderStub.SetUser(10, testUser)

		// when
		err = service.Execute(context.Background(), webhook.Token)

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get budget item")
	})

	t.Run("should return error when invalid JSON data", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// Create webhook with invalid JSON directly in repo
		webhook := Webhook{
			Type:   TypeStartCurrentEvent,
			UserId: 10,
			Data:   json.RawMessage(`invalid json`),
		}
		created, err := repoStub.Create(ctx, webhook)
		require.NoError(t, err)

		// Setup user stub
		testUser := user.User{Id: 10}
		userProviderStub.SetUser(10, testUser)

		// when
		err = service.Execute(context.Background(), created.Token)

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to unmarshal webhook data")
	})
}
