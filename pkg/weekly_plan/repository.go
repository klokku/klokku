package weekly_plan

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	log "github.com/sirupsen/logrus"
)

var ErrWeeklyPlanItemNotFound = errors.New("weekly plan item not found")

type Repository interface {
	WithTransaction(ctx context.Context, fn func(repo Repository) error) error
	GetItemsForWeek(ctx context.Context, userId int, weekNumber WeekNumber) ([]WeeklyPlanItem, error)
	GetItem(ctx context.Context, userId int, id int) (WeeklyPlanItem, error)
	// UpdateAllItemsByBudgetItemId updates name, icon, and color of all weekly plan items for a given budget item.
	UpdateAllItemsByBudgetItemId(ctx context.Context, userId int, budgetItemId int, name string, icon string, color string) (int, error)
	UpdateItem(ctx context.Context, userId int, id int, weeklyDuration time.Duration, notes string) (WeeklyPlanItem, error)
	createItems(ctx context.Context, userId int, items []WeeklyPlanItem) ([]WeeklyPlanItem, error)
	// DeleteWeekItems deletes all weekly plan items for a given week.
	DeleteWeekItems(ctx context.Context, userId int, weekNumber WeekNumber) (int, error)
}

type repositoryImpl struct {
	db *pgx.Conn
	tx pgx.Tx
}

func NewRepo(db *pgx.Conn) Repository {
	return &repositoryImpl{db: db}
}

// getQueryer returns the appropriate database interface for queries (either tx or db)
func (r *repositoryImpl) getQueryer() interface {
	Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row
} {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

func (r *repositoryImpl) WithTransaction(ctx context.Context, fn func(repo Repository) error) error {
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

	// Create a repository that uses the transaction
	txRepo := &repositoryImpl{db: r.db, tx: tx}

	if err := fn(txRepo); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *repositoryImpl) GetItemsForWeek(ctx context.Context, userId int, weekNumber WeekNumber) ([]WeeklyPlanItem, error) {

	query := `SELECT 
    			item.id,
    			item.budget_item_id,
    			item.week_number,
    			item.name,
    			item.weekly_duration_sec,
    			item.weekly_occurrences,
    			item.icon,
    			item.color,
    			item.notes,
    			item.position
			  FROM weekly_plan_item item 
			  WHERE user_id = $1 AND week_number = $2 
			  ORDER BY item.position`
	rows, err := r.getQueryer().Query(ctx, query, userId, weekNumber.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []WeeklyPlanItem

	for rows.Next() {
		var itemWeekNumberString string
		var weeklyDurationSec int
		var item WeeklyPlanItem
		if err := rows.Scan(
			&item.Id,
			&item.BudgetItemId,
			&itemWeekNumberString,
			&item.Name,
			&weeklyDurationSec,
			&item.WeeklyOccurrences,
			&item.Icon,
			&item.Color,
			&item.Notes,
			&item.Position,
		); err != nil {
			return nil, err
		}
		item.WeeklyDuration = time.Duration(weeklyDurationSec) * time.Second
		item.WeekNumber, err = WeekNumberFromString(itemWeekNumberString)
		if err != nil {
			return nil, fmt.Errorf("could not parse week number: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *repositoryImpl) UpdateAllItemsByBudgetItemId(
	ctx context.Context,
	userId int,
	budgetItemId int,
	name string,
	icon string,
	color string,
) (int, error) {
	query := `UPDATE weekly_plan_item SET name = $1, icon = $2, color = $3 WHERE user_id = $4 AND budget_item_id = $5`
	result, err := r.getQueryer().Exec(ctx, query, name, icon, color, userId, budgetItemId)
	if err != nil {
		return 0, err
	}
	rowsAffected := result.RowsAffected()
	return int(rowsAffected), nil
}

func (r *repositoryImpl) GetItem(ctx context.Context, userId int, id int) (WeeklyPlanItem, error) {
	query := `SELECT
    			item.id,
    			item.budget_item_id,
    			item.week_number,
    			item.name,
    			item.weekly_duration_sec,
    			item.weekly_occurrences,
    			item.icon,
    			item.color,
    			item.notes,
    			item.position
 			  FROM weekly_plan_item item WHERE item.user_id = $1 AND item.id = $2`
	var itemWeekNumberString string
	var weeklyDurationSec int
	var item WeeklyPlanItem
	err := r.getQueryer().QueryRow(ctx, query, userId, id).Scan(
		&item.Id,
		&item.BudgetItemId,
		&itemWeekNumberString,
		&item.Name,
		&weeklyDurationSec,
		&item.WeeklyOccurrences,
		&item.Icon,
		&item.Color,
		&item.Notes,
		&item.Position,
	)
	if err != nil {
		return WeeklyPlanItem{}, err
	}
	item.WeeklyDuration = time.Duration(weeklyDurationSec) * time.Second
	item.WeekNumber, err = WeekNumberFromString(itemWeekNumberString)
	if err != nil {
		return WeeklyPlanItem{}, fmt.Errorf("could not parse week number: %w", err)
	}
	return item, nil
}

func (r *repositoryImpl) UpdateItem(ctx context.Context, userId int, id int, weeklyDuration time.Duration, notes string) (WeeklyPlanItem, error) {
	query := `UPDATE weekly_plan_item item
	 			SET weekly_duration_sec = $1, notes = $2
     			WHERE item.user_id = $3 AND item.id = $4
     			RETURNING
     			     item.id,
    					 item.budget_item_id,
    					 item.week_number,
    					 item.name,
    					 item.weekly_duration_sec,
    					 item.weekly_occurrences,
    					 item.icon,
    					 item.color,
    					 item.notes `
	var itemWeekNumberString string
	var weeklyDurationSec int
	var item WeeklyPlanItem
	err := r.getQueryer().QueryRow(ctx, query, weeklyDuration.Seconds(), notes, userId, id).Scan(
		&item.Id,
		&item.BudgetItemId,
		&itemWeekNumberString,
		&item.Name,
		&weeklyDurationSec,
		&item.WeeklyOccurrences,
		&item.Icon,
		&item.Color,
		&item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WeeklyPlanItem{}, ErrWeeklyPlanItemNotFound
		}
		return WeeklyPlanItem{}, fmt.Errorf("could not update item: %w", err)
	}
	item.WeeklyDuration = time.Duration(weeklyDurationSec) * time.Second
	item.WeekNumber, err = WeekNumberFromString(itemWeekNumberString)
	if err != nil {
		return WeeklyPlanItem{}, fmt.Errorf("could not parse week number: %w", err)
	}
	return item, nil
}

func (r *repositoryImpl) createItems(ctx context.Context, userId int, items []WeeklyPlanItem) ([]WeeklyPlanItem, error) {
	if len(items) == 0 {
		return nil, nil
	}

	var valuesBuilder strings.Builder
	args := make([]any, 0, len(items)*10)
	placeholder := 1
	for idx, item := range items {
		if idx > 0 {
			valuesBuilder.WriteByte(',')
		}
		valuesBuilder.WriteString("(")
		for i := 0; i < 10; i++ {
			if i > 0 {
				valuesBuilder.WriteByte(',')
			}
			fmt.Fprintf(&valuesBuilder, "$%d", placeholder)
			placeholder++
		}
		valuesBuilder.WriteString(")")

		args = append(args,
			userId,
			item.BudgetItemId,
			item.WeekNumber.String(),
			item.Name,
			item.WeeklyDuration.Seconds(),
			item.WeeklyOccurrences,
			item.Icon,
			item.Color,
			item.Notes,
			item.Position,
		)
	}

	query := fmt.Sprintf(`INSERT INTO weekly_plan_item (
                            user_id,
                            budget_item_id,
                            week_number,
                            name,
                            weekly_duration_sec,
                            weekly_occurrences,
                            icon,
                            color,
                            notes,
                            position
                  ) VALUES %s RETURNING 
                            id,
                            budget_item_id,
                            week_number,
                            name,
                            weekly_duration_sec,
                            weekly_occurrences,
                            icon,
                            color,
                            notes,
                            position`, valuesBuilder.String())

	rows, err := r.getQueryer().Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var created []WeeklyPlanItem
	for rows.Next() {
		var weekNumberString string
		var weeklyDurationSec int
		var item WeeklyPlanItem
		err := rows.Scan(
			&item.Id,
			&item.BudgetItemId,
			&weekNumberString,
			&item.Name,
			&weeklyDurationSec,
			&item.WeeklyOccurrences,
			&item.Icon,
			&item.Color,
			&item.Notes,
			&item.Position,
		)
		if err != nil {
			return nil, err
		}
		item.WeeklyDuration = time.Duration(weeklyDurationSec) * time.Second
		item.WeekNumber, err = WeekNumberFromString(weekNumberString)
		if err != nil {
			return nil, fmt.Errorf("could not parse week number: %w", err)
		}
		created = append(created, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return created, nil
}

func (r *repositoryImpl) DeleteWeekItems(ctx context.Context, userId int, weekNumber WeekNumber) (int, error) {
	query := `DELETE FROM weekly_plan_item WHERE user_id = $1 AND week_number = $2`
	result, err := r.getQueryer().Exec(ctx, query, userId, weekNumber.String())
	if err != nil {
		return 0, err
	}
	return int(result.RowsAffected()), nil
}
