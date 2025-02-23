package budget

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
)

type BudgetRepo interface {
	// Store stores a new Budget to the database
	Store(ctx context.Context, userId int, budget Budget) (int, error)
	GetAll(ctx context.Context, userId int, includeInactive bool) ([]Budget, error)
	Update(ctx context.Context, userId int, budget Budget) (bool, error)
	UpdatePosition(ctx context.Context, userId int, budget Budget) (bool, error)
	FindMaxPosition(ctx context.Context, userId int) (int, error)
}

type BudgetRepoImpl struct {
	db *sql.DB
}

func NewBudgetRepo(db *sql.DB) *BudgetRepoImpl {
	return &BudgetRepoImpl{db: db}
}

func (bi BudgetRepoImpl) Store(ctx context.Context, userId int, budget Budget) (int, error) {

	query := "INSERT INTO budget (name, weekly_time, weekly_occurrences, status, position, icon, user_id) VALUES (?, ?, ?, ?, ?, ?, ?)"

	stmt, err := bi.db.PrepareContext(ctx, query)
	if err != nil {
		err := fmt.Errorf("could not prepare query: %v", err)
		log.Error(err)
		return 0, err
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx,
		budget.Name,
		budget.WeeklyTime.Milliseconds()/1000,
		budget.WeeklyOccurrences,
		budget.Status,
		budget.Position,
		budget.Icon,
		userId,
	)
	if err != nil {
		err := fmt.Errorf("could not execute query: %v", err)
		log.Error(err)
		return 0, err
	}

	lastInsertID, err := result.LastInsertId()
	if err != nil {
		err := fmt.Errorf("could not retrieve last insert id: %w", err)
		log.Error(err)
		return 0, err
	}

	return int(lastInsertID), nil
}

func (bi BudgetRepoImpl) GetAll(ctx context.Context, userId int, includeInactive bool) ([]Budget, error) {
	expectedStatuses := "'active'"
	if includeInactive {
		expectedStatuses = "'active','inactive'"
	}
	query := fmt.Sprintf(
		"SELECT id, name, weekly_time, weekly_occurrences, position, icon, status FROM budget WHERE budget.user_id = ? "+
			"AND status in (%s) ORDER BY status, position",
		expectedStatuses,
	)
	rows, err := bi.db.QueryContext(ctx, query, userId)
	if err != nil {
		err := fmt.Errorf("could not query budgets: %w", err)
		log.Error(err)
		return nil, err
	}
	defer rows.Close()

	var budgets []Budget
	for rows.Next() {
		var budget Budget
		var weeklyTime int64
		var status string
		if err := rows.Scan(&budget.ID, &budget.Name, &weeklyTime, &budget.WeeklyOccurrences, &budget.Position, &budget.Icon, &status); err != nil {
			err := fmt.Errorf("could not scan budget: %w", err)
			log.Error(err)
			return nil, err
		}
		budget.WeeklyTime = time.Duration(weeklyTime) * time.Second
		budget.Status = BudgetStatus(status)
		budgets = append(budgets, budget)
	}

	if err := rows.Err(); err != nil {
		err := fmt.Errorf("error iterating over rows: %w", err)
		log.Error(err)
		return nil, err
	}

	return budgets, nil
}

func (bi BudgetRepoImpl) UpdatePosition(ctx context.Context, userId int, budget Budget) (bool, error) {
	query := "UPDATE budget SET position = ? WHERE id = ? and user_id = ?"
	stmt, err := bi.db.PrepareContext(ctx, query)
	if err != nil {
		err := fmt.Errorf("could not prepare query: %v", err)
		log.Error(err)
		return false, err
	}
	defer stmt.Close()
	result, err := stmt.ExecContext(ctx, budget.Position, budget.ID, userId)
	if err != nil {
		err := fmt.Errorf("could not execute query: %v", err)
		log.Error(err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		err := fmt.Errorf("could not get rows affected: %w", err)
		log.Error(err)
	}
	return rowsAffected == 1, nil
}

func (bi BudgetRepoImpl) Update(ctx context.Context, userId int, budget Budget) (bool, error) {
	query := "UPDATE budget SET name = ?, weekly_time = ?, weekly_occurrences = ?, icon = ?, status = ? WHERE id = ? and user_id = ?"
	stmt, err := bi.db.PrepareContext(ctx, query)
	if err != nil {
		err := fmt.Errorf("could not prepare query: %v", err)
		log.Error(err)
		return false, err
	}
	defer stmt.Close()
	result, err := stmt.ExecContext(ctx,
		budget.Name,
		budget.WeeklyTime.Milliseconds()/1000,
		budget.WeeklyOccurrences,
		budget.Icon,
		budget.Status,
		budget.ID,
		userId,
	)
	if err != nil {
		err := fmt.Errorf("could not execute query: %v", err)
		log.Error(err)
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		err := fmt.Errorf("could not get rows affected: %w", err)
		log.Error(err)
	}

	return rowsAffected == 1, nil
}

func (bi BudgetRepoImpl) FindMaxPosition(ctx context.Context, userId int) (int, error) {
	query := "SELECT MAX(position) FROM budget WHERE user_id = ?"
	row := bi.db.QueryRowContext(ctx, query, userId)
	var maxPosition int
	err := row.Scan(&maxPosition)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		err := fmt.Errorf("could not scan row: %w", err)
		log.Error(err)
		return 0, err
	}
	return maxPosition, nil
}
