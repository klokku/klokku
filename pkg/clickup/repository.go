package clickup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type Repository interface {
	StoreConfiguration(ctx context.Context, userId int, config Configuration) error
	GetConfiguration(ctx context.Context, userId int) (*Configuration, error)
	DeleteConfiguration(ctx context.Context, userId int) error
	DeleteAuthData(ctx context.Context, userId int) error
}

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

// StoreConfiguration stores the ClickUp integration configuration for a user
func (r *RepositoryImpl) StoreConfiguration(ctx context.Context, userId int, config Configuration) error {
	// Begin a transaction to ensure atomicity
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Check if configuration already exists for this user
	var exists bool
	err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM clickup_config WHERE user_id = ?)", userId).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check for existing configuration: %w", err)
	}

	// Update or insert configuration
	if exists {
		_, err = tx.ExecContext(ctx,
			"UPDATE clickup_config SET workspace_id = ?, space_id = ?, folder_id = ? WHERE user_id = ?",
			config.WorkspaceId, config.SpaceId, config.FolderId, userId)
		if err != nil {
			return fmt.Errorf("failed to update configuration: %w", err)
		}
	} else {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO clickup_config (user_id, workspace_id, space_id, folder_id) VALUES (?, ?, ?, ?)",
			userId, config.WorkspaceId, config.SpaceId, config.FolderId)
		if err != nil {
			return fmt.Errorf("failed to insert configuration: %w", err)
		}
	}

	// Delete existing budget mappings for this user
	_, err = tx.ExecContext(ctx, "DELETE FROM clickup_tag_mapping WHERE user_id = ?", userId)
	if err != nil {
		return fmt.Errorf("failed to delete existing budget mappings: %w", err)
	}

	// Insert new budget mappings
	for _, mapping := range config.Mappings {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO clickup_tag_mapping (user_id, clickup_space_id, clickup_tag_name, budget_id, position) VALUES (?, ?, ?, ?, ?)",
			userId, mapping.ClickupSpaceId, mapping.ClickupTagName, mapping.BudgetId, mapping.Position)
		if err != nil {
			return fmt.Errorf("failed to insert budget mapping: %w", err)
		}
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetConfiguration retrieves the ClickUp integration configuration for a user
func (r *RepositoryImpl) GetConfiguration(ctx context.Context, userId int) (*Configuration, error) {
	config := &Configuration{}

	// Query for the basic configuration
	err := r.db.QueryRowContext(ctx,
		"SELECT workspace_id, space_id, folder_id FROM clickup_config WHERE user_id = ?",
		userId).Scan(&config.WorkspaceId, &config.SpaceId, &config.FolderId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No configuration found
		}
		return nil, fmt.Errorf("failed to retrieve configuration: %w", err)
	}

	// Query for the budget mappings
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, clickup_space_id, clickup_tag_name, budget_id, position FROM clickup_tag_mapping WHERE user_id = ? ORDER BY position",
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
			&mapping.BudgetId,
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
	_, err := r.db.ExecContext(ctx, "DELETE FROM clickup_config WHERE user_id = ?", userId)
	if err != nil {
		return fmt.Errorf("failed to delete configuration: %w", err)
	}
	_, err = r.db.ExecContext(ctx, "DELETE FROM clickup_tag_mapping WHERE user_id = ?", userId)
	if err != nil {
		return fmt.Errorf("failed to delete budget mappings: %w", err)
	}
	return nil
}

func (g *RepositoryImpl) DeleteAuthData(ctx context.Context, userId int) error {
	_, err := g.db.ExecContext(ctx, "DELETE FROM clickup_auth WHERE user_id = ?", userId)
	if err != nil {
		return fmt.Errorf("failed to delete auth data: %w", err)
	}
	return nil
}
