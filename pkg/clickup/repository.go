package clickup

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	StoreConfiguration(ctx context.Context, userId, budgetPlanId int, config Configuration) error
	GetConfiguration(ctx context.Context, userId, budgetPlanId int) (*Configuration, error)
	GetConfigurationWithMappingByBudgetItemId(ctx context.Context, userId, budgetItemId int) (*Configuration, error)
	DeleteAllConfigurations(ctx context.Context, userId int) error
	DeleteBudgetPlanConfiguration(ctx context.Context, userId, budgetPlanId int) error
	DeleteAuthData(ctx context.Context, userId int) error
}

type RepositoryImpl struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

// StoreConfiguration stores the ClickUp integration configuration for a user
func (r *RepositoryImpl) StoreConfiguration(ctx context.Context, userId, budgetPlanId int, config Configuration) error {
	// Begin a transaction to ensure atomicity
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Upsert configuration and GET the ID (needed for mappings)
	var configId int
	const upsertConfig = `
		INSERT INTO clickup_config (user_id, budget_plan_id, workspace_id, space_id, folder_id, only_tasks_with_priority)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, budget_plan_id) 
		DO UPDATE SET 
			workspace_id = EXCLUDED.workspace_id,
			space_id = EXCLUDED.space_id,
			folder_id = EXCLUDED.folder_id,
			only_tasks_with_priority = EXCLUDED.only_tasks_with_priority
		RETURNING id`

	err = tx.QueryRow(ctx, upsertConfig,
		userId,
		budgetPlanId,
		config.WorkspaceId,
		config.SpaceId,
		config.FolderId,
		config.OnlyTasksWithPriority,
	).Scan(&configId)
	if err != nil {
		return fmt.Errorf("failed to upsert configuration: %w", err)
	}

	// 2. Delete existing mappings for this specific configuration
	_, err = tx.Exec(ctx, "DELETE FROM clickup_tag_mapping WHERE clickup_config_id = $1", configId)
	if err != nil {
		return fmt.Errorf("failed to delete existing budget mappings: %w", err)
	}

	// 3. Batch insert mappings using the returned configId
	if len(config.Mappings) > 0 {
		values := []any{}
		query := "INSERT INTO clickup_tag_mapping (clickup_config_id, clickup_space_id, clickup_tag_name, budget_item_id, position, user_id) VALUES "

		for i, m := range config.Mappings {
			p := i * 6
			query += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d),", p+1, p+2, p+3, p+4, p+5, p+6)
			values = append(values, configId, m.ClickupSpaceId, m.ClickupTagName, m.BudgetItemId, m.Position, userId)
		}
		query = query[:len(query)-1]

		if _, err := tx.Exec(ctx, query, values...); err != nil {
			return fmt.Errorf("failed to insert budget mappings: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// GetConfiguration retrieves the ClickUp integration configuration for a user
func (r *RepositoryImpl) GetConfiguration(ctx context.Context, userId, budgetPlanId int) (*Configuration, error) {
	config := &Configuration{}

	// Query for the basic configuration
	err := r.db.QueryRow(ctx,
		"SELECT workspace_id, space_id, folder_id, only_tasks_with_priority FROM clickup_config WHERE user_id = $1 AND budget_plan_id = $2",
		userId, budgetPlanId).Scan(&config.WorkspaceId, &config.SpaceId, &config.FolderId, &config.OnlyTasksWithPriority)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No configuration found
		}
		return nil, fmt.Errorf("failed to retrieve configuration: %w", err)
	}

	// Query for the budget mappings
	rows, err := r.db.Query(ctx,
		`SELECT m.clickup_space_id, m.clickup_tag_name, m.budget_item_id, m.position 
				FROM clickup_tag_mapping m
				INNER JOIN clickup_config c ON m.clickup_config_id = c.id
				WHERE c.user_id = $1 AND c.budget_plan_id = $2 ORDER BY m.position`,
		userId, budgetPlanId)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve budget mappings: %w", err)
	}
	defer rows.Close()

	// Process the mapping rows
	mappings := make([]BudgetItemMapping, 0, 20)
	for rows.Next() {
		var mapping BudgetItemMapping
		err := rows.Scan(
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

func (r *RepositoryImpl) GetConfigurationWithMappingByBudgetItemId(ctx context.Context, userId, budgetItemId int) (*Configuration, error) {
	config := &Configuration{}
	var mapping BudgetItemMapping

	err := r.db.QueryRow(ctx,
		`SELECT m.clickup_space_id, m.clickup_tag_name, m.position, c.workspace_id, c.space_id, c.folder_id, c.only_tasks_with_priority
				FROM clickup_tag_mapping m
				INNER JOIN clickup_config c ON m.clickup_config_id = c.id
				WHERE m.user_id = $1 AND m.budget_item_id = $2`,
		userId, budgetItemId).Scan(
		&mapping.ClickupSpaceId,
		&mapping.ClickupTagName,
		&mapping.Position,
		&config.WorkspaceId,
		&config.SpaceId,
		&config.FolderId,
		&config.OnlyTasksWithPriority,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan budget mapping: %w", err)
	}

	mapping.BudgetItemId = budgetItemId
	config.Mappings = []BudgetItemMapping{mapping}

	return config, nil
}

func (r *RepositoryImpl) DeleteAllConfigurations(ctx context.Context, userId int) error {
	_, err := r.db.Exec(ctx, "DELETE FROM clickup_config WHERE user_id = $1", userId)
	if err != nil {
		return fmt.Errorf("failed to delete configuration: %w", err)
	}
	return nil
}

func (r *RepositoryImpl) DeleteBudgetPlanConfiguration(ctx context.Context, userId, budgetPlanId int) error {
	var deletedConfigId int
	err := r.db.QueryRow(ctx,
		`DELETE FROM clickup_config WHERE user_id = $1 AND budget_plan_id = $2 RETURNING id`,
		userId,
		budgetPlanId,
	).Scan(&deletedConfigId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Handle case where nothing was deleted if necessary
			return nil
		}
		return fmt.Errorf("failed to delete configuration: %w", err)
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
