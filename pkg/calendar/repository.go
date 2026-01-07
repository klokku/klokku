package calendar

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
)

type Repository interface {
	WithTransaction(ctx context.Context, fn func(repo Repository) error) error
	StoreEvent(ctx context.Context, userId int, event Event) (Event, error)
	GetEvents(ctx context.Context, userId int, from, to time.Time) ([]Event, error)
	GetLastEvents(ctx context.Context, userId int, limit int) ([]Event, error)
	UpdateEvent(ctx context.Context, userId int, event Event) (Event, error)
	DeleteEvent(ctx context.Context, userId int, eventId string) error
}
type repositoryImpl struct {
	db *pgxpool.Pool
	tx pgx.Tx
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repositoryImpl{db: db, tx: nil}
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

func (r *repositoryImpl) StoreEvent(ctx context.Context, userId int, event Event) (Event, error) {
	query := `INSERT INTO calendar_event (
                            uid,
                            summary,
                            start_time,
                            end_time,
                            budget_item_id,
                            user_id
						) VALUES ($1, $2, $3, $4, $5, $6) RETURNING uid, summary, start_time, end_time, budget_item_id`

	uid := uuid.NewString()
	var createdEvent Event
	err := r.getQueryer().QueryRow(ctx, query,
		uid,
		event.Summary,
		event.StartTime,
		event.EndTime,
		event.Metadata.BudgetItemId,
		userId,
	).Scan(&createdEvent.UID, &createdEvent.Summary, &createdEvent.StartTime, &createdEvent.EndTime, &createdEvent.Metadata.BudgetItemId)
	if err != nil {
		err := fmt.Errorf("could not execute query: %v", err)
		log.Error(err)
		return Event{}, err
	}

	return createdEvent, nil
}

func (r *repositoryImpl) GetEvents(ctx context.Context, userId int, from, to time.Time) ([]Event, error) {
	// Return all events that overlap with the given period:
	// 1. Events that start before the end of the period (start_time <= to)
	// 2. AND end after the start of the period (end_time >= from)
	query := `SELECT uid, summary, start_time, end_time, budget_item_id 
              FROM calendar_event 
              WHERE user_id = $1 
                AND start_time <= $2 
                AND end_time >= $3
			  ORDER BY start_time`

	rows, err := r.getQueryer().Query(ctx, query, userId, to, from)
	if err != nil {
		err := fmt.Errorf("could not query calendar events: %w", err)
		log.Error(err)
		return nil, err
	}
	defer rows.Close()

	events := make([]Event, 0, 10)
	for rows.Next() {
		var event Event
		err := rows.Scan(&event.UID, &event.Summary, &event.StartTime, &event.EndTime, &event.Metadata.BudgetItemId)
		if err != nil {
			err := fmt.Errorf("could not scan row: %w", err)
			log.Error(err)
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

// GetLastEvents retrieves the most recent calendar events for a specific user, limited by the specified number of records.
func (r *repositoryImpl) GetLastEvents(ctx context.Context, userId int, limit int) ([]Event, error) {
	query := `SELECT uid, summary, start_time, end_time, budget_item_id
				FROM calendar_event 
				WHERE user_id = $1 AND
				      end_time <= $2
				ORDER BY end_time DESC
				LIMIT $3`

	rows, err := r.getQueryer().Query(ctx, query, userId, time.Now(), limit)
	if err != nil {
		err := fmt.Errorf("could not query calendar events: %w", err)
		log.Error(err)
		return nil, err
	}
	defer rows.Close()

	events := make([]Event, 0, limit)
	for rows.Next() {
		var event Event
		err := rows.Scan(&event.UID, &event.Summary, &event.StartTime, &event.EndTime, &event.Metadata.BudgetItemId)
		if err != nil {
			err := fmt.Errorf("could not scan row: %w", err)
			log.Error(err)
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func (r *repositoryImpl) UpdateEvent(ctx context.Context, userId int, event Event) (Event, error) {
	query := `UPDATE calendar_event 
				SET summary = $1, start_time = $2, end_time = $3, budget_item_id = $4 
				WHERE uid = $5 AND user_id = $6
				RETURNING uid, summary, start_time, end_time, budget_item_id`
	var updatedEvent Event
	err := r.getQueryer().QueryRow(ctx, query,
		event.Summary,
		event.StartTime,
		event.EndTime,
		event.Metadata.BudgetItemId,
		event.UID,
		userId).Scan(&updatedEvent.UID, &updatedEvent.Summary, &updatedEvent.StartTime, &updatedEvent.EndTime, &updatedEvent.Metadata.BudgetItemId)
	if err != nil {
		err := fmt.Errorf("could not execute query: %v", err)
		log.Error(err)
		return Event{}, err
	}
	return updatedEvent, nil
}

func (r *repositoryImpl) DeleteEvent(ctx context.Context, userId int, eventUid string) error {
	query := `DELETE FROM calendar_event WHERE uid = $1 AND user_id = $2`
	result, err := r.getQueryer().Exec(ctx, query, eventUid, userId)
	if err != nil {
		err := fmt.Errorf("could not execute query: %v", err)
		log.Error(err)
		return err
	}
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no event found with uid %s for user %d", eventUid, userId)
	}
	return nil
}
