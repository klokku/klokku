package budget

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

type BudgetRepo interface {
	// Store stores a new Budget to the database
	Store(ctx context.Context, userId int, budget Budget) (int, error)
	GetAll(ctx context.Context, userId int, includeInactive bool) ([]Budget, error)
	Update(ctx context.Context, userId int, budget Budget) (bool, error)
	UpdatePosition(ctx context.Context, userId int, budget Budget) (bool, error)
	FindMaxPosition(ctx context.Context, userId int) (int, error)
	Delete(ctx context.Context, userId int, budgetId int) (bool, error)
}

type BudgetRepoImpl struct {
	db *sql.DB
}

func NewBudgetRepo(db *sql.DB) *BudgetRepoImpl {
	return &BudgetRepoImpl{db: db}
}

func (bi BudgetRepoImpl) Store(ctx context.Context, userId int, budget Budget) (int, error) {

	query := `INSERT INTO budget (
                    name, 
                    weekly_time, 
                    weekly_occurrences, 
                    position, 
                    icon, 
                    start_date,
                    end_date,
                    user_id
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	stmt, err := bi.db.PrepareContext(ctx, query)
	if err != nil {
		err := fmt.Errorf("could not prepare query: %v", err)
		log.Error(err)
		return 0, err
	}
	defer stmt.Close()

	var startDateParam interface{}
	if !budget.StartDate.IsZero() {
		startDateParam = budget.StartDate.Format("2006-01-02")
	} else {
		startDateParam = nil
	}
	var endDateParam interface{}
	if !budget.EndDate.IsZero() {
		endDateParam = budget.EndDate.Format("2006-01-02")
	} else {
		endDateParam = nil
	}

	result, err := stmt.ExecContext(ctx,
		budget.Name,
		budget.WeeklyTime.Milliseconds()/1000,
		budget.WeeklyOccurrences,
		budget.Position,
		budget.Icon,
		startDateParam,
		endDateParam,
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
	activeWhereQuery := "AND (budget.start_date IS NULL OR budget.start_date <= date('now')) AND (budget.end_date IS NULL OR budget." +
		"end_date >= date('now'))"
	if includeInactive {
		activeWhereQuery = ""
	}
	query := fmt.Sprintf(
		`SELECT id, name, weekly_time, weekly_occurrences, position, icon, start_date, end_date 
				FROM budget WHERE budget.user_id = ? %s ORDER BY position`,
		activeWhereQuery,
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
		var startDateString, endDate sql.NullString
		if err := rows.Scan(
			&budget.ID,
			&budget.Name,
			&weeklyTime,
			&budget.WeeklyOccurrences,
			&budget.Position,
			&budget.Icon,
			&startDateString,
			&endDate,
		); err != nil {
			err := fmt.Errorf("could not scan budget: %w", err)
			log.Error(err)
			return nil, err
		}
		budget.WeeklyTime = time.Duration(weeklyTime) * time.Second
		if startDateString.Valid {
			startDate, err := time.Parse("2006-01-02", startDateString.String)
			if err != nil {
				err := fmt.Errorf("could not parse start date: %w", err)
				log.Error(err)
				return nil, err
			}
			budget.StartDate = startDate
		}
		if endDate.Valid {
			endDate, err := time.Parse("2006-01-02", endDate.String)
			if err != nil {
				err := fmt.Errorf("could not parse end date: %w", err)
				log.Error(err)
				return nil, err
			}
			budget.EndDate = endDate
		}
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
	query := `UPDATE budget SET 
                  name = ?, 
                  weekly_time = ?, 
                  weekly_occurrences = ?, 
                  icon = ?,
                  start_date = ?,
                  end_date = ?
              WHERE id = ? and user_id = ?`
	stmt, err := bi.db.PrepareContext(ctx, query)
	if err != nil {
		err := fmt.Errorf("could not prepare query: %v", err)
		log.Error(err)
		return false, err
	}
	defer stmt.Close()

	var startDateParam interface{}
	if !budget.StartDate.IsZero() {
		startDateParam = budget.StartDate.Format("2006-01-02")
	} else {
		startDateParam = nil
	}
	var endDateParam interface{}
	if !budget.EndDate.IsZero() {
		endDateParam = budget.EndDate.Format("2006-01-02")
	} else {
		endDateParam = nil
	}
	result, err := stmt.ExecContext(ctx,
		budget.Name,
		budget.WeeklyTime.Milliseconds()/1000,
		budget.WeeklyOccurrences,
		budget.Icon,
		startDateParam,
		endDateParam,
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

func (bi BudgetRepoImpl) Delete(ctx context.Context, userId int, budgetId int) (bool, error) {
	query := "DELETE FROM budget WHERE id = ? and user_id = ?"
	stmt, err := bi.db.PrepareContext(ctx, query)
	if err != nil {
		err := fmt.Errorf("could not prepare query: %v", err)
		log.Error(err)
		return false, err
	}
	defer stmt.Close()
	result, err := stmt.ExecContext(ctx, budgetId, userId)
	if err != nil {
		err := fmt.Errorf("could not execute query: %v", err)
		log.Error(err)
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		err := fmt.Errorf("could not get rows affected: %w", err)
		log.Error(err)
		return false, err
	}
	return rowsAffected == 1, nil
}

func (bi BudgetRepoImpl) FindMaxPosition(ctx context.Context, userId int) (int, error) {
	query := "SELECT MAX(position) FROM budget WHERE user_id = ?"
	row := bi.db.QueryRowContext(ctx, query, userId)
	var maxPosition sql.NullInt64
	err := row.Scan(&maxPosition)
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
