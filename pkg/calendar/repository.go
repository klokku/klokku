package calendar

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type Repository interface {
	WithTransaction(ctx context.Context, fn func(repo Repository) error) error
	StoreEvent(ctx context.Context, userId int, event Event) (uuid.UUID, error)
	GetEvents(ctx context.Context, userId int, from, to time.Time) ([]Event, error)
	GetLastEvents(ctx context.Context, userId int, limit int) ([]Event, error)
	UpdateEvent(ctx context.Context, userId int, event Event) error
	DeleteEvent(ctx context.Context, userId int, eventId uuid.UUID) error
}
type RepositoryImpl struct {
	db *sql.DB
	tx *sql.Tx
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db, tx: nil}
}

// getQueryer returns the appropriate database interface for queries (either tx or db)
func (r *RepositoryImpl) getQueryer() interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
} {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

func (r *RepositoryImpl) WithTransaction(ctx context.Context, fn func(repo Repository) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		// The Rollback will be a no-op if the transaction was already committed
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			// Just log rollback errors
			log.Errorf("rollback error: %v", rbErr)
		}
	}()

	// Create a repository that uses the transaction
	txRepo := &RepositoryImpl{db: r.db, tx: tx}

	if err := fn(txRepo); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *RepositoryImpl) StoreEvent(ctx context.Context, userId int, event Event) (uuid.UUID, error) {
	query := `INSERT INTO calendar_event (
                            uid,
                            summary,
                            start_time,
                            end_time,
                            budget_id,
                            user_id
						) VALUES (?, ?, ?, ?, ?, ?)`

	stmt, err := r.getQueryer().PrepareContext(ctx, query)
	if err != nil {
		err := fmt.Errorf("could not prepare query: %v", err)
		log.Error(err)
		return uuid.Nil, err
	}
	defer stmt.Close()

	uid := uuid.New()
	_, err = stmt.ExecContext(ctx, uid.String(), event.Summary, event.StartTime.UnixMilli(), event.EndTime.UnixMilli(), event.Metadata.BudgetId, userId)
	if err != nil {
		err := fmt.Errorf("could not execute query: %v", err)
		log.Error(err)
		return uuid.Nil, err
	}

	return uid, nil
}

func (r *RepositoryImpl) GetEvents(ctx context.Context, userId int, from, to time.Time) ([]Event, error) {
	// Return all events that overlap with the given period:
	// 1. Events that start before the end of the period (start_time <= to)
	// 2. AND end after the start of the period (end_time >= from)
	query := `SELECT uid, summary, start_time, end_time, budget_id 
              FROM calendar_event 
              WHERE user_id = ? 
                AND start_time <= ? 
                AND end_time >= ?
			  ORDER BY start_time`

	rows, err := r.getQueryer().QueryContext(ctx, query, userId, to.UnixMilli(), from.UnixMilli())
	if err != nil {
		err := fmt.Errorf("could not query calendar events: %w", err)
		log.Error(err)
		return nil, err
	}
	defer rows.Close()

	events := make([]Event, 0, 10)
	for rows.Next() {
		var uid uuid.NullUUID
		var summary string
		var startTimeMillis int64
		var endTimeMillis int64
		var budgetId int
		err := rows.Scan(&uid, &summary, &startTimeMillis, &endTimeMillis, &budgetId)
		if err != nil {
			err := fmt.Errorf("could not scan row: %w", err)
			log.Error(err)
			return nil, err
		}
		events = append(events, Event{
			Summary:   summary,
			StartTime: time.UnixMilli(startTimeMillis),
			EndTime:   time.UnixMilli(endTimeMillis),
			Metadata: EventMetadata{
				BudgetId: budgetId,
			},
			UID: uid,
		})
	}
	return events, nil
}

// GetLastEvents retrieves the most recent calendar events for a specific user, limited by the specified number of records.
func (r *RepositoryImpl) GetLastEvents(ctx context.Context, userId int, limit int) ([]Event, error) {
	query := `SELECT uid, summary, start_time, end_time, budget_id
				FROM calendar_event 
				WHERE user_id = ? AND
				      end_time <= ?
				ORDER BY end_time DESC
				LIMIT ?`

	rows, err := r.getQueryer().QueryContext(ctx, query, userId, time.Now().UnixMilli(), limit)
	if err != nil {
		err := fmt.Errorf("could not query calendar events: %w", err)
		log.Error(err)
		return nil, err
	}
	defer rows.Close()

	events := make([]Event, 0, 10)
	for rows.Next() {
		var uid uuid.NullUUID
		var summary string
		var startTimeMillis int64
		var endTimeMillis int64
		var budgetId int
		err := rows.Scan(&uid, &summary, &startTimeMillis, &endTimeMillis, &budgetId)
		if err != nil {
			err := fmt.Errorf("could not scan row: %w", err)
			log.Error(err)
			return nil, err
		}
		events = append(events, Event{
			Summary:   summary,
			StartTime: time.UnixMilli(startTimeMillis),
			EndTime:   time.UnixMilli(endTimeMillis),
			Metadata: EventMetadata{
				BudgetId: budgetId,
			},
			UID: uid,
		})
	}
	return events, nil
}

func (r *RepositoryImpl) UpdateEvent(ctx context.Context, userId int, event Event) error {
	query := `UPDATE calendar_event SET summary = ?, start_time = ?, end_time = ?, budget_id = ? WHERE uid = ? AND user_id = ?`
	stmt, err := r.getQueryer().PrepareContext(ctx, query)
	if err != nil {
		err := fmt.Errorf("could not prepare query: %v", err)
		log.Error(err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, event.Summary, event.StartTime.UnixMilli(), event.EndTime.UnixMilli(), event.Metadata.BudgetId, event.UID.UUID.String(),
		userId)
	if err != nil {
		err := fmt.Errorf("could not execute query: %v", err)
		log.Error(err)
		return err
	}
	return nil
}

func (r *RepositoryImpl) DeleteEvent(ctx context.Context, userId int, eventUid uuid.UUID) error {
	query := `DELETE FROM calendar_event WHERE uid = ? AND user_id = ?`
	stmt, err := r.getQueryer().PrepareContext(ctx, query)
	if err != nil {
		err := fmt.Errorf("could not prepare query: %v", err)
		log.Error(err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, eventUid.String(), userId)
	if err != nil {
		err := fmt.Errorf("could not execute query: %v", err)
		log.Error(err)
		return err
	}
	return nil
}
