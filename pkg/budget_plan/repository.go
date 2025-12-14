package budget_plan

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	log "github.com/sirupsen/logrus"
)

var ErrPlanNotFound = errors.New("plan not found")
var ErrDeletingCurrentPlan = errors.New("cannot delete current plan")
var ErrBudgetPlanItemNotFound = errors.New("budget plan item not found")

type Repository interface {
	StoreItem(ctx context.Context, userId int, budget BudgetItem) (int, error)
	GetPlan(ctx context.Context, userId int, planId int) (BudgetPlan, error)
	GetCurrentPlan(ctx context.Context, userId int) (BudgetPlan, error)
	ListPlans(ctx context.Context, userId int) ([]BudgetPlan, error)
	CreatePlan(ctx context.Context, userId int, plan BudgetPlan) (BudgetPlan, error)
	UpdatePlan(ctx context.Context, userId int, plan BudgetPlan) (BudgetPlan, error)
	DeletePlan(ctx context.Context, userId int, planId int) (bool, error)
	GetItem(ctx context.Context, userId int, itemId int) (BudgetItem, error)
	UpdateItem(ctx context.Context, userId int, budget BudgetItem) (bool, error)
	UpdateItemPosition(ctx context.Context, userId int, budget BudgetItem) (bool, error)
	FindMaxPlanItemPosition(ctx context.Context, planId int, userId int) (int, error)
	DeleteItem(ctx context.Context, userId int, itemId int) (bool, error)
}

type RepositoryImpl struct {
	db *pgx.Conn
}

func NewBudgetPlanRepo(db *pgx.Conn) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (bi RepositoryImpl) StoreItem(ctx context.Context, userId int, budget BudgetItem) (int, error) {

	query := `INSERT INTO budget_item (
                    budget_plan_id,
					name, 
                    weekly_duration_sec, 
                    weekly_occurrences, 
                    icon,
                    color,
                    position, 
                    user_id
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`

	var lastInsertID int
	err := bi.db.QueryRow(ctx, query,
		budget.PlanId,
		budget.Name,
		budget.WeeklyDuration.Milliseconds()/1000,
		budget.WeeklyOccurrences,
		budget.Icon,
		budget.Color,
		budget.Position,
		userId,
	).Scan(&lastInsertID)
	if err != nil {
		err := fmt.Errorf("could not execute query: %v", err)
		log.Error(err)
		return 0, err
	}

	return lastInsertID, nil
}

func (bi RepositoryImpl) GetPlan(ctx context.Context, userId int, planId int) (BudgetPlan, error) {
	// Get a Tx for making transaction requests.
	tx, err := bi.db.Begin(ctx)
	if err != nil {
		return BudgetPlan{}, err
	}
	defer tx.Rollback(ctx)

	query := `SELECT 
    			plan.name as plan_name,
    			item.id as item_id, 
    			item.budget_plan_id, 
    			item.name as item_name, 
    			item.weekly_duration_sec,
    			item.weekly_occurrences,
    			item.icon,
    			item.color,
    			item.position
               FROM budget_plan plan 
			   LEFT JOIN budget_item item on plan.id = item.budget_plan_id
               WHERE plan.user_id = $1 AND plan.id = $2 ORDER BY item.position`
	rows, err := tx.Query(ctx, query, userId, planId)
	if err != nil {
		err := fmt.Errorf("could not query budgets: %w", err)
		log.Error(err)
		return BudgetPlan{}, err
	}
	defer rows.Close()

	var planName string

	foundPlan := false
	var items []BudgetItem
	for rows.Next() {
		foundPlan = true
		var (
			itemId            sql.NullInt64
			itemPlanId        sql.NullInt64
			itemName          sql.NullString
			weeklyDurationSec sql.NullInt64
			itemOccurrences   sql.NullInt64
			itemIcon          sql.NullString
			itemColor         sql.NullString
			itemPosition      sql.NullInt64
		)

		if err := rows.Scan(
			&planName,          // plan.name AS plan_name
			&itemId,            // item.id AS item_id
			&itemPlanId,        // item.budget_plan_id
			&itemName,          // item.name AS item_name
			&weeklyDurationSec, // item.weekly_duration_sec
			&itemOccurrences,
			&itemIcon,
			&itemColor,
			&itemPosition,
		); err != nil {
			err := fmt.Errorf("error scanning row: %w", err)
			log.Error(err)
			return BudgetPlan{}, err
		}

		// If there's no item (LEFT JOIN), item_id will be NULL
		if !itemId.Valid {
			continue
		}

		var item BudgetItem
		item.Id = int(itemId.Int64)
		item.PlanId = int(itemPlanId.Int64)
		item.Name = itemName.String
		item.WeeklyDuration = time.Duration(weeklyDurationSec.Int64) * time.Second
		if itemOccurrences.Valid {
			item.WeeklyOccurrences = int(itemOccurrences.Int64)
		}
		if itemIcon.Valid {
			item.Icon = itemIcon.String
		}
		if itemColor.Valid {
			item.Color = itemColor.String
		}
		item.Position = int(itemPosition.Int64)

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		err := fmt.Errorf("error iterating over rows: %w", err)
		log.Error(err)
		return BudgetPlan{}, err
	}

	if !foundPlan {
		return BudgetPlan{}, ErrPlanNotFound
	}

	currentPlanId, err := bi.getCurrentPlanId(ctx, tx, userId)
	if err != nil {
		return BudgetPlan{}, err
	}
	isCurrentPlan := currentPlanId == planId

	plan := BudgetPlan{Id: planId, Name: planName, IsCurrent: isCurrentPlan, Items: items}

	if err := tx.Commit(ctx); err != nil {
		return BudgetPlan{}, fmt.Errorf("could not commit transaction: %w", err)
	}
	return plan, nil
}

func (bi RepositoryImpl) GetCurrentPlan(ctx context.Context, userId int) (BudgetPlan, error) {
	// Get a Tx for making transaction requests.
	tx, err := bi.db.Begin(ctx)
	if err != nil {
		return BudgetPlan{}, err
	}
	defer tx.Rollback(ctx)

	currentPlanId, err := bi.getCurrentPlanId(ctx, tx, userId)
	if err != nil {
		return BudgetPlan{}, err
	}
	if currentPlanId == 0 {
		return BudgetPlan{}, ErrPlanNotFound
	}
	return bi.GetPlan(ctx, userId, currentPlanId)
}

func (bi RepositoryImpl) ListPlans(ctx context.Context, userId int) ([]BudgetPlan, error) {
	// Get a Tx for making transaction requests.
	tx, err := bi.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	currentPlanId, err := bi.getCurrentPlanId(ctx, tx, userId)
	if err != nil {
		return nil, err
	}

	query := `SELECT plan.id, plan.name FROM budget_plan plan WHERE plan.user_id = $1 ORDER BY plan.created`
	rows, err := tx.Query(ctx, query, userId)
	if err != nil {
		err := fmt.Errorf("could not query budget plans: %w", err)
		log.Error(err)
		return nil, err
	}
	defer rows.Close()

	var plans []BudgetPlan
	for rows.Next() {
		var planId int
		var planName string
		if err := rows.Scan(&planId, &planName); err != nil {
			err := fmt.Errorf("error scanning row: %w", err)
			log.Error(err)
			return nil, err
		}
		plans = append(plans, BudgetPlan{Id: planId, IsCurrent: currentPlanId == planId, Name: planName})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}
	return plans, nil
}

func (bi RepositoryImpl) CreatePlan(ctx context.Context, userId int, plan BudgetPlan) (BudgetPlan, error) {
	// Get a Tx for making transaction requests.
	tx, err := bi.db.Begin(ctx)
	if err != nil {
		return BudgetPlan{}, err
	}
	defer tx.Rollback(ctx)

	plansCount, err := bi.countPlans(ctx, tx, userId)
	if err != nil {
		return BudgetPlan{}, err
	}
	if plansCount == 0 {
		plan.IsCurrent = true
	}

	var planId int
	query := `INSERT INTO budget_plan (name, user_id) VALUES ($1, $2) RETURNING id`
	err = tx.QueryRow(ctx, query, plan.Name, userId).Scan(&planId)
	if err != nil {
		err := fmt.Errorf("could not execute query: %w", err)
		log.Error(err)
		return BudgetPlan{}, err
	}
	if plan.IsCurrent {
		ok, err := bi.setCurrentPlan(ctx, tx, userId, planId)
		if err != nil {
			err := fmt.Errorf("could not execute query: %w", err)
			log.Error(err)
			return BudgetPlan{}, err
		}
		if !ok {
			return BudgetPlan{}, fmt.Errorf("could not set current plan to %d", planId)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return BudgetPlan{}, fmt.Errorf("could not commit transaction: %w", err)
	}

	return BudgetPlan{Id: planId, Name: plan.Name}, nil
}

func (bi RepositoryImpl) UpdatePlan(ctx context.Context, userId int, plan BudgetPlan) (BudgetPlan, error) {
	// Get a Tx for making transaction requests.
	tx, err := bi.db.Begin(ctx)
	if err != nil {
		return BudgetPlan{}, err
	}
	defer tx.Rollback(ctx)

	query := `UPDATE budget_plan SET name = $1 WHERE id = $2 and user_id = $3`
	result, err := tx.Exec(ctx, query, plan.Name, plan.Id, userId)
	if err != nil {
		err := fmt.Errorf("could not execute query: %v", err)
		log.Error(err)
		return BudgetPlan{}, err
	}
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return BudgetPlan{}, ErrPlanNotFound
	}

	if plan.IsCurrent {
		ok, err := bi.setCurrentPlan(ctx, tx, userId, plan.Id)
		if err != nil {
			err := fmt.Errorf("could not execute query: %v", err)
			log.Error(err)
			return BudgetPlan{}, err
		}
		if !ok {
			return BudgetPlan{}, fmt.Errorf("could not set current plan to %d", plan.Id)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return BudgetPlan{}, fmt.Errorf("could not commit transaction: %w", err)
	}
	return plan, nil
}

func (bi RepositoryImpl) setCurrentPlan(ctx context.Context, tx pgx.Tx, userId int, planId int) (bool, error) {
	query := `INSERT INTO 
					budget_plan_current (budget_plan_id, user_id) VALUES ($1, $2) 
					ON CONFLICT (user_id) DO UPDATE SET budget_plan_id = EXCLUDED.budget_plan_id`
	_, err := tx.Exec(ctx, query, planId, userId)
	if err != nil {
		err := fmt.Errorf("could not execute query: %v", err)
		log.Error(err)
		return false, err
	}
	return true, nil
}

func (bi RepositoryImpl) DeletePlan(ctx context.Context, userId int, planId int) (bool, error) {
	// Get a Tx for making transaction requests.
	tx, err := bi.db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	currentPlanId, err := bi.getCurrentPlanId(ctx, tx, userId)
	if err != nil {
		return false, err
	}
	if planId == currentPlanId {
		return false, ErrDeletingCurrentPlan
	}

	query := "DELETE FROM budget_plan WHERE id = $1 and user_id = $2"
	result, err := tx.Exec(ctx, query, planId, userId)
	if err != nil {
		err := fmt.Errorf("could not execute query: %v", err)
		log.Error(err)
		return false, err
	}
	rowsAffected := result.RowsAffected()
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("could not commit transaction: %w", err)
	}
	return rowsAffected == 1, nil
}

// TODO test this function
func (bi RepositoryImpl) GetItem(ctx context.Context, userId int, itemId int) (BudgetItem, error) {
	query := `SELECT 
    			item.budget_plan_id, 
    			item.name, 
    			item.weekly_duration_sec,
    			item.weekly_occurrences,
    			item.icon,
    			item.color,
    			item.position
               FROM budget_item item
               WHERE item.id = $1 AND item.user_id = $2`

	var (
		itemPlanId        int
		itemName          string
		weeklyDurationSec int
		weeklyOccurrences sql.NullInt64
		itemIcon          sql.NullString
		itemColor         sql.NullString
		itemPosition      int
	)

	err := bi.db.QueryRow(ctx, query, itemId, userId).
		Scan(
			&itemPlanId,
			&itemName,
			&weeklyDurationSec,
			&weeklyOccurrences,
			&itemIcon,
			&itemColor,
			&itemPosition,
		)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return BudgetItem{}, ErrBudgetPlanItemNotFound
		}
		err := fmt.Errorf("error scanning row: %w", err)
		log.Error(err)
		return BudgetItem{}, err
	}

	var item BudgetItem
	item.Id = itemId
	item.PlanId = itemPlanId
	item.Name = itemName
	item.WeeklyDuration = time.Duration(weeklyDurationSec) * time.Second
	if weeklyOccurrences.Valid {
		item.WeeklyOccurrences = int(weeklyOccurrences.Int64)
	}
	if itemIcon.Valid {
		item.Icon = itemIcon.String
	}
	if itemColor.Valid {
		item.Color = itemColor.String
	}
	item.Position = itemPosition

	return item, nil
}

func (bi RepositoryImpl) UpdateItemPosition(ctx context.Context, userId int, budget BudgetItem) (bool, error) {
	query := "UPDATE budget_item SET position = $1 WHERE id = $2 and user_id = $3"
	result, err := bi.db.Exec(ctx, query, budget.Position, budget.Id, userId)
	if err != nil {
		err := fmt.Errorf("could not execute query: %v", err)
		log.Error(err)
		return false, err
	}
	rowsAffected := result.RowsAffected()
	return rowsAffected == 1, nil
}

func (bi RepositoryImpl) UpdateItem(ctx context.Context, userId int, budget BudgetItem) (bool, error) {
	query := `UPDATE budget_item SET 
                  name = $1, 
                  weekly_duration_sec = $2, 
                  weekly_occurrences = $3, 
                  icon = $4,
                  color = $5
              WHERE id = $6 and user_id = $7`
	result, err := bi.db.Exec(ctx, query,
		budget.Name,
		budget.WeeklyDuration.Milliseconds()/1000,
		budget.WeeklyOccurrences,
		budget.Icon,
		budget.Color,
		budget.Id,
		userId,
	)
	if err != nil {
		err := fmt.Errorf("could not execute query: %v", err)
		log.Error(err)
		return false, err
	}

	rowsAffected := result.RowsAffected()
	return rowsAffected == 1, nil
}

func (bi RepositoryImpl) DeleteItem(ctx context.Context, userId int, budgetId int) (bool, error) {
	query := "DELETE FROM budget_item WHERE id = $1 and user_id = $2"
	result, err := bi.db.Exec(ctx, query, budgetId, userId)
	if err != nil {
		err := fmt.Errorf("could not execute query: %v", err)
		log.Error(err)
		return false, err
	}
	rowsAffected := result.RowsAffected()
	return rowsAffected == 1, nil
}

func (bi RepositoryImpl) FindMaxPlanItemPosition(ctx context.Context, planId int, userId int) (int, error) {
	query := "SELECT MAX(position) FROM budget_item WHERE budget_item.budget_plan_id = $1 AND user_id = $2"
	var maxPosition sql.NullInt64
	err := bi.db.QueryRow(ctx, query, planId, userId).Scan(&maxPosition)
	if err != nil {
		err := fmt.Errorf("could not find max position: %w", err)
		log.Error(err)
		return 0, err
	}

	if !maxPosition.Valid {
		log.Debugf(
			"could not find max position for user %d, returning 0",
			userId)
		return 0, nil
	}

	return int(maxPosition.Int64), nil
}

func (bi RepositoryImpl) getCurrentPlanId(ctx context.Context, tx pgx.Tx, userId int) (int, error) {
	query := "SELECT budget_plan_current.budget_plan_id FROM budget_plan_current WHERE budget_plan_current.user_id = $1"
	var planId sql.NullInt64
	err := tx.QueryRow(ctx, query, userId).Scan(&planId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Debugf("no current plan found for user %d, returning 0", userId)
			return 0, nil
		}
		err := fmt.Errorf("could not get current plan id: %w", err)
		log.Error(err)
		return 0, err
	}
	return int(planId.Int64), nil
}

func (bi RepositoryImpl) countPlans(ctx context.Context, tx pgx.Tx, userId int) (int, error) {
	query := "SELECT COUNT(*) FROM budget_plan WHERE user_id = $1"
	var count int
	err := tx.QueryRow(ctx, query, userId).Scan(&count)
	if err != nil {
		err := fmt.Errorf("could not count plans: %w", err)
		log.Error(err)
		return 0, err
	}
	return count, nil
}
