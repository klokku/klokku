package event

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
)

type EventRepository interface {
	StoreEvent(ctx context.Context, userId int, event Event) (Event, error)
	DeleteCurrentEvent(ctx context.Context, userId int) error
	FindCurrentEvent(ctx context.Context, userId int) (*Event, error)
}

type EventRepositoryImpl struct {
	db *sql.DB
}

func NewEventRepo(db *sql.DB) *EventRepositoryImpl {
	return &EventRepositoryImpl{db: db}
}

// StoreEvent stores a new Event to the database
func (bie *EventRepositoryImpl) StoreEvent(ctx context.Context, userId int, event Event) (Event, error) {
	query := "INSERT INTO event (budget_id, start_time, end_time, user_id) VALUES (?, ?, ?, ?)"

	stmt, err := bie.db.PrepareContext(ctx, query)
	if err != nil {
		err := fmt.Errorf("could not prepare query: %v", err)
		log.Error(err)
		return Event{}, err
	}
	defer stmt.Close()

	var endTimeUnix *int64 = nil
	if !event.EndTime.IsZero() {
		unixValue := event.EndTime.Unix()
		endTimeUnix = &unixValue
	}
	result, err := stmt.ExecContext(ctx, event.Budget.ID, event.StartTime.Unix(), endTimeUnix, userId)
	if err != nil {
		err := fmt.Errorf("could not execute query: %v", err)
		log.Error(err)
		return Event{}, err
	}

	lastInsertID, err := result.LastInsertId()
	if err != nil {
		err := fmt.Errorf("could not retrieve last insert id: %w", err)
		log.Error(err)
		return Event{}, err
	}

	event.ID = int(lastInsertID)

	return event, nil
}

func (bie *EventRepositoryImpl) DeleteCurrentEvent(ctx context.Context, userId int) error {
	query := "DELETE FROM event WHERE end_time IS NULL AND user_id = ?"
	_, err := bie.db.ExecContext(ctx, query, userId)
	if err != nil {
		err := fmt.Errorf("could not execute query: %w", err)
		log.Error(err)
		return err
	}
	return nil
}

func (bie *EventRepositoryImpl) FindCurrentEvent(ctx context.Context, userId int) (*Event, error) {
	query := `
		SELECT e.id, e.budget_id, e.start_time,
			   bi.name, bi.weekly_time
		FROM event e
		JOIN budget bi ON e.budget_id = bi.id
		WHERE e.end_time IS NULL AND e.user_id = ? LIMIT 1`

	row := bie.db.QueryRowContext(ctx, query, userId)

	var event Event
	var startTimeUnix int64
	var weeklyTime int
	err := row.Scan(&event.ID, &event.Budget.ID, &startTimeUnix, &event.Budget.Name, &weeklyTime)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		err := fmt.Errorf("failed when trying to find current event: %w", err)
		log.Error(err)
		return nil, err
	}

	if startTimeUnix != 0 {
		startTime := time.Unix(startTimeUnix, 0)
		event.StartTime = startTime
	} else {
		err := fmt.Errorf("could not parse start time")
		log.Error(err)
		return nil, err
	}
	event.Budget.WeeklyTime = time.Duration(weeklyTime) * time.Second

	return &event, nil
}
