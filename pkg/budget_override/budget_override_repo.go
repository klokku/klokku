package budget_override

import (
	"context"
	"database/sql"
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
)

type BudgetOverrideRepo interface {
	Store(ctx context.Context, userId int, override BudgetOverride) (int, error)
	GetAllForWeek(ctx context.Context, userId int, weekStartDate time.Time) ([]BudgetOverride, error)
	Delete(ctx context.Context, userId int, id int) error
	Update(ctx context.Context, userId int, override BudgetOverride) error
}

type BudgetOverrideRepoImpl struct {
	db *sql.DB
}

func NewBudgetOverrideRepo(db *sql.DB) *BudgetOverrideRepoImpl {
	return &BudgetOverrideRepoImpl{db: db}
}

// Store stores a new BudgetOverride to the database
func (bi *BudgetOverrideRepoImpl) Store(ctx context.Context, userId int, override BudgetOverride) (int, error) {
	query := "INSERT INTO budget_override (budget_id, start_date, weekly_time, notes, user_id) VALUES (?, ?, ?, ?, ?)"

	stmt, err := bi.db.PrepareContext(ctx, query)
	if err != nil {
		err := fmt.Errorf("could not prepare query: %v", err)
		log.Error(err)
		return 0, err
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx,
		override.BudgetID,
		override.StartDate.Format("2006-01-02"),
		override.WeeklyTime.Minutes(),
		override.Notes,
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

func (bi *BudgetOverrideRepoImpl) GetAllForWeek(ctx context.Context, userId int, weekStartDate time.Time) ([]BudgetOverride, error) {
	weekStartDateString := weekStartDate.Format("2006-01-02")
	query := "SELECT id, budget_id, weekly_time, start_date, notes FROM budget_override WHERE start_date = ? and user_id = ?"
	rows, err := bi.db.QueryContext(ctx, query, weekStartDateString, userId)
	if err != nil {
		err := fmt.Errorf("could not query budget overrides: %w", err)
		log.Error(err)
		return nil, err
	}
	defer rows.Close()

	overrides := make([]BudgetOverride, 0, 10)
	for rows.Next() {
		var override BudgetOverride
		var weeklyTime int64
		var startDateString string

		if err := rows.Scan(&override.ID, &override.BudgetID, &weeklyTime, &startDateString, &override.Notes); err != nil {
			err := fmt.Errorf("could not scan budget override: %w", err)
			log.Error(err)
			return nil, err
		}
		override.StartDate, err = time.ParseInLocation("2006-01-02", startDateString, weekStartDate.Location())
		if err != nil {
			err := fmt.Errorf("could not parse start date from database")
			log.Error(err)
			return nil, err
		}

		override.WeeklyTime = time.Duration(weeklyTime) * time.Minute
		overrides = append(overrides, override)
	}

	if err := rows.Err(); err != nil {
		err := fmt.Errorf("error iterating over rows: %w", err)
		log.Error(err)
		return nil, err
	}

	return overrides, nil
}

func (bi *BudgetOverrideRepoImpl) Delete(ctx context.Context, userId int, id int) error {
	query := "DELETE FROM budget_override WHERE id = ? and user_id = ?"
	stmt, err := bi.db.PrepareContext(ctx, query)
	if err != nil {
		err := fmt.Errorf("could not prepare query: %v", err)
		log.Error(err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, id, userId)
	if err != nil {
		err := fmt.Errorf("could not execute query: %v", err)
		log.Error(err)
		return err
	}
	return nil
}

func (bi *BudgetOverrideRepoImpl) Update(ctx context.Context, userId int, override BudgetOverride) error {
	query := "UPDATE budget_override SET weekly_time = ?, start_date = ?, notes = ? WHERE id = ? and user_id = ?"
	stmt, err := bi.db.PrepareContext(ctx, query)
	if err != nil {
		err := fmt.Errorf("could not prepare query: %v", err)
		log.Error(err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx,
		override.WeeklyTime.Minutes(),
		override.StartDate.Format("2006-01-02"),
		override.Notes,
		override.ID,
		userId,
	)
	if err != nil {
		err := fmt.Errorf("could not execute query: %v", err)
		log.Error(err)
		return err
	}
	return nil
}
