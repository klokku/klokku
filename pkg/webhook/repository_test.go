package webhook

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klokku/klokku/internal/test_utils"
	log "github.com/sirupsen/logrus"
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
	userId := 1
	return ctx, repository, userId
}

func TestRepositoryImpl_Create(t *testing.T) {
	t.Run("should create webhook with generated token", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		data := StartCurrentEventData{BudgetItemId: 42}
		dataJSON, err := json.Marshal(data)
		require.NoError(t, err)

		webhook := Webhook{
			Type:   TypeStartCurrentEvent,
			UserId: userId,
			Data:   dataJSON,
		}

		// when
		created, err := repo.Create(ctx, webhook)

		// then
		require.NoError(t, err)
		require.NotZero(t, created.Id)
		require.NotEmpty(t, created.Token)
		require.Equal(t, 64, len(created.Token)) // 32 bytes = 64 hex chars
		require.Equal(t, TypeStartCurrentEvent, created.Type)
		require.Equal(t, userId, created.UserId)
		require.JSONEq(t, string(dataJSON), string(created.Data))
		require.NotZero(t, created.CreatedAt)
		require.NotZero(t, created.UpdatedAt)
	})

	t.Run("should generate unique tokens for multiple webhooks", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		data := StartCurrentEventData{BudgetItemId: 42}
		dataJSON, err := json.Marshal(data)
		require.NoError(t, err)

		webhook := Webhook{
			Type:   TypeStartCurrentEvent,
			UserId: userId,
			Data:   dataJSON,
		}

		// when
		webhook1, err1 := repo.Create(ctx, webhook)
		webhook2, err2 := repo.Create(ctx, webhook)

		// then
		require.NoError(t, err1)
		require.NoError(t, err2)
		require.NotEqual(t, webhook1.Token, webhook2.Token)
	})

	t.Run("should create webhooks for different users", func(t *testing.T) {
		// given
		ctx, repo, _ := setupTestRepository(t)
		data := StartCurrentEventData{BudgetItemId: 42}
		dataJSON, err := json.Marshal(data)
		require.NoError(t, err)

		// when
		webhook1, err1 := repo.Create(ctx, Webhook{
			Type:   TypeStartCurrentEvent,
			UserId: 1,
			Data:   dataJSON,
		})
		webhook2, err2 := repo.Create(ctx, Webhook{
			Type:   TypeStartCurrentEvent,
			UserId: 2,
			Data:   dataJSON,
		})

		// then
		require.NoError(t, err1)
		require.NoError(t, err2)
		require.Equal(t, 1, webhook1.UserId)
		require.Equal(t, 2, webhook2.UserId)
		require.NotEqual(t, webhook1.Token, webhook2.Token)
	})
}

func TestRepositoryImpl_GetByToken(t *testing.T) {
	t.Run("should retrieve webhook by token", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		data := StartCurrentEventData{BudgetItemId: 123}
		dataJSON, err := json.Marshal(data)
		require.NoError(t, err)

		created, err := repo.Create(ctx, Webhook{
			Type:   TypeStartCurrentEvent,
			UserId: userId,
			Data:   dataJSON,
		})
		require.NoError(t, err)

		// when
		retrieved, err := repo.GetByToken(ctx, created.Token)

		// then
		require.NoError(t, err)
		require.Equal(t, created.Id, retrieved.Id)
		require.Equal(t, created.Token, retrieved.Token)
		require.Equal(t, created.Type, retrieved.Type)
		require.Equal(t, created.UserId, retrieved.UserId)
		require.JSONEq(t, string(created.Data), string(retrieved.Data))
	})

	t.Run("should return ErrWebhookNotFound for non-existent token", func(t *testing.T) {
		// given
		ctx, repo, _ := setupTestRepository(t)
		nonExistentToken := "nonexistent1234567890abcdef1234567890abcdef1234567890abcdef12"

		// when
		_, err := repo.GetByToken(ctx, nonExistentToken)

		// then
		require.ErrorIs(t, err, ErrWebhookNotFound)
	})

	t.Run("should return ErrWebhookNotFound for empty token", func(t *testing.T) {
		// given
		ctx, repo, _ := setupTestRepository(t)

		// when
		_, err := repo.GetByToken(ctx, "")

		// then
		require.ErrorIs(t, err, ErrWebhookNotFound)
	})
}

func TestRepositoryImpl_GetByUserIdAndType(t *testing.T) {
	t.Run("should return webhooks for user and type", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)

		// Create multiple webhooks
		for i := 1; i <= 3; i++ {
			data := StartCurrentEventData{BudgetItemId: i}
			dataJSON, _ := json.Marshal(data)
			_, err := repo.Create(ctx, Webhook{
				Type:   TypeStartCurrentEvent,
				UserId: userId,
				Data:   dataJSON,
			})
			require.NoError(t, err)
		}

		// Create webhook for different user
		data := StartCurrentEventData{BudgetItemId: 999}
		dataJSON, _ := json.Marshal(data)
		_, err := repo.Create(ctx, Webhook{
			Type:   TypeStartCurrentEvent,
			UserId: userId + 1,
			Data:   dataJSON,
		})
		require.NoError(t, err)

		// when
		webhooks, err := repo.GetByUserIdAndType(ctx, userId, TypeStartCurrentEvent)

		// then
		require.NoError(t, err)
		require.Len(t, webhooks, 3)
		for _, webhook := range webhooks {
			require.Equal(t, userId, webhook.UserId)
			require.Equal(t, TypeStartCurrentEvent, webhook.Type)
		}
	})

	t.Run("should return empty list when no webhooks exist", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)

		// when
		webhooks, err := repo.GetByUserIdAndType(ctx, userId, TypeStartCurrentEvent)

		// then
		require.NoError(t, err)
		require.Empty(t, webhooks)
	})

	t.Run("should return webhooks ordered by creation time descending", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)

		var createdWebhooks []Webhook
		for i := 1; i <= 3; i++ {
			data := StartCurrentEventData{BudgetItemId: i}
			dataJSON, _ := json.Marshal(data)
			created, err := repo.Create(ctx, Webhook{
				Type:   TypeStartCurrentEvent,
				UserId: userId,
				Data:   dataJSON,
			})
			require.NoError(t, err)
			createdWebhooks = append(createdWebhooks, created)
			time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		}

		// when
		webhooks, err := repo.GetByUserIdAndType(ctx, userId, TypeStartCurrentEvent)

		// then
		require.NoError(t, err)
		require.Len(t, webhooks, 3)
		// Should be in reverse order (newest first)
		require.Equal(t, createdWebhooks[2].Id, webhooks[0].Id)
		require.Equal(t, createdWebhooks[1].Id, webhooks[1].Id)
		require.Equal(t, createdWebhooks[0].Id, webhooks[2].Id)
	})
}

func TestRepositoryImpl_RotateToken(t *testing.T) {
	t.Run("should generate new token", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		data := StartCurrentEventData{BudgetItemId: 42}
		dataJSON, _ := json.Marshal(data)

		created, err := repo.Create(ctx, Webhook{
			Type:   TypeStartCurrentEvent,
			UserId: userId,
			Data:   dataJSON,
		})
		require.NoError(t, err)
		originalToken := created.Token

		// when
		newToken, err := repo.RotateToken(ctx, created.Id, userId)

		// then
		require.NoError(t, err)
		require.NotEmpty(t, newToken)
		require.NotEqual(t, originalToken, newToken)
		require.Equal(t, 64, len(newToken))

		// Verify old token no longer works
		_, err = repo.GetByToken(ctx, originalToken)
		require.ErrorIs(t, err, ErrWebhookNotFound)

		// Verify new token works
		retrieved, err := repo.GetByToken(ctx, newToken)
		require.NoError(t, err)
		require.Equal(t, created.Id, retrieved.Id)
	})

	t.Run("should return ErrWebhookNotFound for non-existent webhook", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		nonExistentId := 99999

		// when
		_, err := repo.RotateToken(ctx, nonExistentId, userId)

		// then
		require.ErrorIs(t, err, ErrWebhookNotFound)
	})

	t.Run("should not rotate token for different user", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		data := StartCurrentEventData{BudgetItemId: 42}
		dataJSON, _ := json.Marshal(data)

		created, err := repo.Create(ctx, Webhook{
			Type:   TypeStartCurrentEvent,
			UserId: userId,
			Data:   dataJSON,
		})
		require.NoError(t, err)

		// when
		differentUserId := userId + 1
		_, err = repo.RotateToken(ctx, created.Id, differentUserId)

		// then
		require.ErrorIs(t, err, ErrWebhookNotFound)
	})

	t.Run("should update updated_at timestamp", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		data := StartCurrentEventData{BudgetItemId: 42}
		dataJSON, _ := json.Marshal(data)

		created, err := repo.Create(ctx, Webhook{
			Type:   TypeStartCurrentEvent,
			UserId: userId,
			Data:   dataJSON,
		})
		require.NoError(t, err)
		originalUpdatedAt := created.UpdatedAt

		time.Sleep(100 * time.Millisecond) // Ensure different timestamp

		// when
		newToken, err := repo.RotateToken(ctx, created.Id, userId)
		require.NoError(t, err)

		// then
		retrieved, err := repo.GetByToken(ctx, newToken)
		require.NoError(t, err)
		require.True(t, retrieved.UpdatedAt.After(originalUpdatedAt))
	})
}

func TestRepositoryImpl_Delete(t *testing.T) {
	t.Run("should delete webhook", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		data := StartCurrentEventData{BudgetItemId: 42}
		dataJSON, _ := json.Marshal(data)

		created, err := repo.Create(ctx, Webhook{
			Type:   TypeStartCurrentEvent,
			UserId: userId,
			Data:   dataJSON,
		})
		require.NoError(t, err)

		// when
		err = repo.Delete(ctx, created.Id, userId)

		// then
		require.NoError(t, err)

		// Verify webhook is deleted
		_, err = repo.GetByToken(ctx, created.Token)
		require.ErrorIs(t, err, ErrWebhookNotFound)
	})

	t.Run("should return ErrWebhookNotFound for non-existent webhook", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		nonExistentId := 99999

		// when
		err := repo.Delete(ctx, nonExistentId, userId)

		// then
		require.ErrorIs(t, err, ErrWebhookNotFound)
	})

	t.Run("should not delete webhook for different user", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		data := StartCurrentEventData{BudgetItemId: 42}
		dataJSON, _ := json.Marshal(data)

		created, err := repo.Create(ctx, Webhook{
			Type:   TypeStartCurrentEvent,
			UserId: userId,
			Data:   dataJSON,
		})
		require.NoError(t, err)

		// when
		differentUserId := userId + 1
		err = repo.Delete(ctx, created.Id, differentUserId)

		// then
		require.ErrorIs(t, err, ErrWebhookNotFound)

		// Verify webhook still exists for original user
		retrieved, err := repo.GetByToken(ctx, created.Token)
		require.NoError(t, err)
		require.Equal(t, created.Id, retrieved.Id)
	})

	t.Run("should only delete specified webhook", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)

		// Create two webhooks
		webhook1, err := repo.Create(ctx, Webhook{
			Type:   TypeStartCurrentEvent,
			UserId: userId,
			Data:   json.RawMessage(`{"budgetItemId":1}`),
		})
		require.NoError(t, err)

		webhook2, err := repo.Create(ctx, Webhook{
			Type:   TypeStartCurrentEvent,
			UserId: userId,
			Data:   json.RawMessage(`{"budgetItemId":2}`),
		})
		require.NoError(t, err)

		// when
		err = repo.Delete(ctx, webhook1.Id, userId)

		// then
		require.NoError(t, err)

		// Verify webhook1 is deleted
		_, err = repo.GetByToken(ctx, webhook1.Token)
		require.ErrorIs(t, err, ErrWebhookNotFound)

		// Verify webhook2 still exists
		retrieved, err := repo.GetByToken(ctx, webhook2.Token)
		require.NoError(t, err)
		require.Equal(t, webhook2.Id, retrieved.Id)
	})
}

func TestGenerateToken(t *testing.T) {
	t.Run("should generate 64 character hex token", func(t *testing.T) {
		// when
		token, err := generateToken()

		// then
		require.NoError(t, err)
		require.Equal(t, 64, len(token))
		require.Regexp(t, "^[0-9a-f]{64}$", token)
	})

	t.Run("should generate unique tokens", func(t *testing.T) {
		// when
		token1, err1 := generateToken()
		token2, err2 := generateToken()

		// then
		require.NoError(t, err1)
		require.NoError(t, err2)
		require.NotEqual(t, token1, token2)
	})
}
