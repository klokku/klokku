package clickup

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	log "github.com/sirupsen/logrus"
)

type Repository interface {
	StoreConfiguration(ctx context.Context, userId int, config Configuration) error
	GetConfiguration(ctx context.Context, userId int) (*Configuration, error)
	DeleteConfiguration(ctx context.Context, userId int) error
	DeleteAuthData(ctx context.Context, userId int) error
}

type RepositoryImpl struct {
	db *pgx.Conn
}

func NewRepository(db *pgx.Conn) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

// StoreConfiguration stores the ClickUp integration configuration for a user
func (r *RepositoryImpl) StoreConfiguration(ctx context.Context, userId int, config Configuration) error {
	// Begin a transaction to ensure atomicity
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		// The Rollback will be a no-op if the transaction was already committed
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			// Just log rollback errors
			log.Errorf("rollback error: %v", rbErr)
		}
	}()

	// Check if configuration already exists for this user
	var exists bool
	err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM clickup_config WHERE user_id = $1)", userId).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check for existing configuration: %w", err)
	}

	// Update or insert configuration
	if exists {
		_, err = tx.Exec(ctx,
			"UPDATE clickup_config SET workspace_id = $1, space_id = $2, folder_id = $3 WHERE user_id = $4",
			config.WorkspaceId, config.SpaceId, config.FolderId, userId)
		if err != nil {
			return fmt.Errorf("failed to update configuration: %w", err)
		}
	} else {
		_, err = tx.Exec(ctx,
			"INSERT INTO clickup_config (user_id, workspace_id, space_id, folder_id) VALUES ($1, $2, $3, $4)",
			userId, config.WorkspaceId, config.SpaceId, config.FolderId)
		if err != nil {
			return fmt.Errorf("failed to insert configuration: %w", err)
		}
	}

	// Delete existing budget mappings for this user
	_, err = tx.Exec(ctx, "DELETE FROM clickup_tag_mapping WHERE user_id = $1", userId)
	if err != nil {
		return fmt.Errorf("failed to delete existing budget mappings: %w", err)
	}

	// Insert new budget mappings
	for _, mapping := range config.Mappings {
		_, err = tx.Exec(ctx,
			"INSERT INTO clickup_tag_mapping (user_id, clickup_space_id, clickup_tag_name, budget_item_id, position) VALUES ($1, $2, $3, $4, $5)",
			userId, mapping.ClickupSpaceId, mapping.ClickupTagName, mapping.BudgetItemId, mapping.Position)
		if err != nil {
			return fmt.Errorf("failed to insert budget mapping: %w", err)
		}
	}

	// Commit the transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetConfiguration retrieves the ClickUp integration configuration for a user
func (r *RepositoryImpl) GetConfiguration(ctx context.Context, userId int) (*Configuration, error) {
	config := &Configuration{}

	// Query for the basic configuration
	err := r.db.QueryRow(ctx,
		"SELECT workspace_id, space_id, folder_id FROM clickup_config WHERE user_id = $1",
		userId).Scan(&config.WorkspaceId, &config.SpaceId, &config.FolderId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No configuration found
		}
		return nil, fmt.Errorf("failed to retrieve configuration: %w", err)
	}

	// Query for the budget mappings
	rows, err := r.db.Query(ctx,
		"SELECT id, clickup_space_id, clickup_tag_name, budget_item_id, position FROM clickup_tag_mapping WHERE user_id = $1 ORDER BY position",
		userId)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve budget mappings: %w", err)
	}
	defer rows.Close()

	// Process the mapping rows
	var mappings []BudgetMapping
	for rows.Next() {
		var mapping BudgetMapping
		err := rows.Scan(
			&mapping.Id,
			&mapping.ClickupSpaceId,
			&mapping.ClickupTagName,
			&mapping.BudgetItemId,
			&mapping.Position,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan budget mapping: %w", err)
		}
		mappings = append(mappings, mapping)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	config.Mappings = mappings
	return config, nil
}

func (r *RepositoryImpl) DeleteConfiguration(ctx context.Context, userId int) error {
	_, err := r.db.Exec(ctx, "DELETE FROM clickup_config WHERE user_id = $1", userId)
	if err != nil {
		return fmt.Errorf("failed to delete configuration: %w", err)
	}
	_, err = r.db.Exec(ctx, "DELETE FROM clickup_tag_mapping WHERE user_id = $1", userId)
	if err != nil {
		return fmt.Errorf("failed to delete budget mappings: %w", err)
	}
	return nil
}

func (g *RepositoryImpl) DeleteAuthData(ctx context.Context, userId int) error {
	_, err := g.db.Exec(ctx, "DELETE FROM clickup_auth WHERE user_id = $1", userId)
	if err != nil {
		return fmt.Errorf("failed to delete auth data: %w", err)
	}
	return nil
}
