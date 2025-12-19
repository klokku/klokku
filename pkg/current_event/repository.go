package current_event

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	log "github.com/sirupsen/logrus"
)

type Repository interface {
	ReplaceCurrentEvent(ctx context.Context, userId int, event CurrentEvent) (CurrentEvent, error)
	DeleteCurrentEvent(ctx context.Context, userId int) error
	FindCurrentEvent(ctx context.Context, userId int) (CurrentEvent, error)
}

type repositoryImpl struct {
	db *pgx.Conn
}

func NewEventRepo(db *pgx.Conn) Repository {
	return &repositoryImpl{db: db}
}

// ReplaceCurrentEvent replaces the current event with the given event
func (r *repositoryImpl) ReplaceCurrentEvent(ctx context.Context, userId int, event CurrentEvent) (CurrentEvent, error) {
	query := `INSERT INTO current_event (budget_item_id, budget_item_name, plan_item_weekly_duration_sec, start_time, user_id) 
				VALUES ($1, $2, $3, $4, $5) 
				ON CONFLICT (user_id) DO UPDATE SET 
					budget_item_id = EXCLUDED.budget_item_id,
					budget_item_name = EXCLUDED.budget_item_name,
					plan_item_weekly_duration_sec = EXCLUDED.plan_item_weekly_duration_sec,
					start_time = EXCLUDED.start_time`

	_, err := r.db.Exec(ctx, query, event.PlanItem.BudgetItemId, event.PlanItem.Name, event.PlanItem.WeeklyDuration.Seconds(), event.StartTime, userId)
	if err != nil {
		err := fmt.Errorf("could not execute query: %v", err)
		log.Error(err)
		return CurrentEvent{}, err
	}

	return event, nil
}

func (r *repositoryImpl) DeleteCurrentEvent(ctx context.Context, userId int) error {
	query := "DELETE FROM current_event WHERE user_id = $1"
	_, err := r.db.Exec(ctx, query, userId)
	if err != nil {
		err := fmt.Errorf("could not execute query: %w", err)
		log.Error(err)
		return err
	}
	return nil
}

func (r *repositoryImpl) FindCurrentEvent(ctx context.Context, userId int) (CurrentEvent, error) {
	query := `
		SELECT budget_item_id, budget_item_name, plan_item_weekly_duration_sec, start_time
		FROM current_event e
		WHERE e.user_id = $1 LIMIT 1`

	var weeklyTime int
	var event CurrentEvent
	err := r.db.QueryRow(ctx, query, userId).Scan(&event.PlanItem.BudgetItemId, &event.PlanItem.Name, &weeklyTime, &event.StartTime)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CurrentEvent{}, nil
		}
		err := fmt.Errorf("failed when trying to find current event: %w", err)
		log.Error(err)
		return CurrentEvent{}, err
	}

	event.PlanItem.WeeklyDuration = time.Duration(weeklyTime) * time.Second

	return event, nil
}
