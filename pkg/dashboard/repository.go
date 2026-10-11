package dashboard

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klokku/klokku/pkg/user"
)

// Repository persists dashboard preferences independently of general profile updates.
type Repository interface {
	// Find returns nil for application defaults, or user.ErrUserNotFound for a missing user.
	Find(ctx context.Context, userId int) (*Configuration, error)
	// Store replaces preferences, or resets them when nil; missing users return user.ErrUserNotFound.
	Store(ctx context.Context, userId int, configuration *Configuration) error
}

type RepositoryImpl struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) Find(ctx context.Context, userId int) (*Configuration, error) {
	var configuration *Configuration
	err := r.db.QueryRow(ctx, `SELECT dashboard_configuration FROM users WHERE id = $1`, userId).Scan(&configuration)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, user.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read dashboard configuration: %w", err)
	}
	return configuration, nil
}

func (r *RepositoryImpl) Store(ctx context.Context, userId int, configuration *Configuration) error {
	result, err := r.db.Exec(ctx, `UPDATE users SET dashboard_configuration = $1 WHERE id = $2`, configuration, userId)
	if err != nil {
		return fmt.Errorf("store dashboard configuration: %w", err)
	}
	if result.RowsAffected() == 0 {
		return user.ErrUserNotFound
	}
	return nil
}
