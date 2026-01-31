package webhook

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrWebhookNotFound = errors.New("webhook not found")

type Repository interface {
	Create(ctx context.Context, webhook Webhook) (Webhook, error)
	GetByToken(ctx context.Context, token string) (Webhook, error)
	GetByUserIdAndType(ctx context.Context, userId int, webhookType WebhookType) ([]Webhook, error)
	RotateToken(ctx context.Context, webhookId int, userId int) (string, error)
	Delete(ctx context.Context, webhookId int, userId int) error
}

type RepositoryImpl struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) Create(ctx context.Context, webhook Webhook) (Webhook, error) {
	// Generate a secure random token
	token, err := generateToken()
	if err != nil {
		return Webhook{}, fmt.Errorf("failed to generate token: %w", err)
	}

	query := `INSERT INTO webhooks (type, token, user_id, data)
	          VALUES ($1, $2, $3, $4)
	          RETURNING id, type, token, user_id, data, created_at, updated_at`

	var result Webhook
	err = r.db.QueryRow(ctx, query, webhook.Type, token, webhook.UserId, webhook.Data).
		Scan(&result.Id, &result.Type, &result.Token, &result.UserId, &result.Data, &result.CreatedAt, &result.UpdatedAt)

	if err != nil {
		return Webhook{}, fmt.Errorf("failed to create webhook: %w", err)
	}

	return result, nil
}

func (r *RepositoryImpl) GetByToken(ctx context.Context, token string) (Webhook, error) {
	query := `SELECT id, type, token, user_id, data, created_at, updated_at
	          FROM webhooks
	          WHERE token = $1`

	var webhook Webhook
	err := r.db.QueryRow(ctx, query, token).
		Scan(&webhook.Id, &webhook.Type, &webhook.Token, &webhook.UserId, &webhook.Data, &webhook.CreatedAt, &webhook.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Webhook{}, ErrWebhookNotFound
		}
		return Webhook{}, fmt.Errorf("failed to get webhook by token: %w", err)
	}

	return webhook, nil
}

func (r *RepositoryImpl) GetByUserIdAndType(ctx context.Context, userId int, webhookType WebhookType) ([]Webhook, error) {
	query := `SELECT id, type, token, user_id, data, created_at, updated_at
	          FROM webhooks
	          WHERE user_id = $1 AND type = $2
	          ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, userId, webhookType)
	if err != nil {
		return nil, fmt.Errorf("failed to query webhooks: %w", err)
	}
	defer rows.Close()

	var webhooks []Webhook
	for rows.Next() {
		var webhook Webhook
		err := rows.Scan(&webhook.Id, &webhook.Type, &webhook.Token, &webhook.UserId, &webhook.Data, &webhook.CreatedAt, &webhook.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook: %w", err)
		}
		webhooks = append(webhooks, webhook)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating webhooks: %w", err)
	}

	return webhooks, nil
}

func (r *RepositoryImpl) RotateToken(ctx context.Context, webhookId int, userId int) (string, error) {
	// Generate a new secure random token
	token, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	query := `UPDATE webhooks
	          SET token = $1, updated_at = NOW()
	          WHERE id = $2 AND user_id = $3
	          RETURNING token`

	var newToken string
	err = r.db.QueryRow(ctx, query, token, webhookId, userId).Scan(&newToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrWebhookNotFound
		}
		return "", fmt.Errorf("failed to rotate token: %w", err)
	}

	return newToken, nil
}

func (r *RepositoryImpl) Delete(ctx context.Context, webhookId int, userId int) error {
	query := `DELETE FROM webhooks WHERE id = $1 AND user_id = $2`

	result, err := r.db.Exec(ctx, query, webhookId, userId)
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrWebhookNotFound
	}

	return nil
}

// generateToken generates a secure random token (32 bytes = 64 hex characters)
func generateToken() (string, error) {
	tokenBytes := make([]byte, 32)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", tokenBytes), nil
}
