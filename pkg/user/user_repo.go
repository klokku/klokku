package user

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
)

type Repo interface {
	CreateUser(ctx context.Context, user User) (int, error)
	GetUser(ctx context.Context, id int) (User, error)
	GetUserByUid(ctx context.Context, uid string) (User, error)
	UpdateUser(ctx context.Context, userId int, user User) (User, error)
	DeleteUser(ctx context.Context, id int) error
	GetAllUsers(ctx context.Context) ([]User, error)
	IsUsernameAvailable(ctx context.Context, username string) (bool, error)
}

type UserRepoImpl struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepoImpl {
	return &UserRepoImpl{db: db}
}

func (u *UserRepoImpl) CreateUser(ctx context.Context, user User) (int, error) {
	eventCalendarType := user.Settings.EventCalendarType
	if eventCalendarType == "" {
		eventCalendarType = KlokkuCalendar
	}
	query := `INSERT INTO users (uid, username, display_name, photo_url, timezone, week_first_day, event_calendar_type, 
				event_calendar_google_calendar_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	var id int
	err := u.db.QueryRow(ctx, query,
		user.Uid,
		user.Username,
		user.DisplayName,
		user.PhotoUrl,
		user.Settings.Timezone,
		user.Settings.WeekFirstDay,
		eventCalendarType,
		user.Settings.GoogleCalendar.CalendarId,
	).Scan(&id)
	if err != nil {
		log.Errorf("failed to create user: %v", err)
		return 0, err
	}
	return int(id), nil
}

func (u *UserRepoImpl) GetUser(ctx context.Context, id int) (User, error) {
	query := `SELECT id, uid, username, display_name, photo_url, timezone, week_first_day, event_calendar_type,
				event_calendar_google_calendar_id, ignore_short_events FROM users WHERE id = $1`
	var user User
	var googleCalendarId sql.NullString
	err := u.db.QueryRow(ctx, query, id).
		Scan(
			&user.Id,
			&user.Uid,
			&user.Username,
			&user.DisplayName,
			&user.PhotoUrl,
			&user.Settings.Timezone,
			&user.Settings.WeekFirstDay,
			&user.Settings.EventCalendarType,
			&googleCalendarId,
			&user.Settings.IgnoreShortEvents,
		)
	if errors.Is(err, sql.ErrNoRows) {
		log.Errorf("user with id %d not found: %v", id, err)
		return User{}, err
	} else if err != nil {
		log.Errorf("failed to get user: %v", err)
		return User{}, err
	}
	if googleCalendarId.Valid {
		user.Settings.GoogleCalendar.CalendarId = googleCalendarId.String
	}
	return user, nil
}

func (u *UserRepoImpl) GetUserByUid(ctx context.Context, uid string) (User, error) {
	query := `SELECT id, uid, username, display_name, photo_url, timezone, week_first_day, event_calendar_type,
				event_calendar_google_calendar_id, ignore_short_events FROM users WHERE uid = $1`

	var user User
	var googleCalendarId sql.NullString
	err := u.db.QueryRow(ctx, query, uid).
		Scan(
			&user.Id,
			&user.Uid,
			&user.Username,
			&user.DisplayName,
			&user.PhotoUrl,
			&user.Settings.Timezone,
			&user.Settings.WeekFirstDay,
			&user.Settings.EventCalendarType,
			&googleCalendarId,
			&user.Settings.IgnoreShortEvents,
		)
	if errors.Is(err, sql.ErrNoRows) {
		log.Infof("user with uid %s not found: %v", uid, err)
		return User{}, err
	} else if err != nil {
		log.Errorf("failed to get user: %v", err)
		return User{}, err
	}
	if googleCalendarId.Valid {
		user.Settings.GoogleCalendar.CalendarId = googleCalendarId.String
	}
	return user, nil
}

func (u *UserRepoImpl) UpdateUser(ctx context.Context, userId int, user User) (User, error) {
	query := `UPDATE users SET display_name = $1, timezone = $2, week_first_day = $3, event_calendar_type = $4, 
				event_calendar_google_calendar_id = $5, ignore_short_events = $6 WHERE id = $7`
	result, err := u.db.Exec(ctx, query,
		user.DisplayName,
		user.Settings.Timezone,
		user.Settings.WeekFirstDay,
		user.Settings.EventCalendarType,
		user.Settings.GoogleCalendar.CalendarId,
		user.Settings.IgnoreShortEvents,
		userId,
	)
	if err != nil {
		return User{}, err
	}
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		log.Info("no rows affected of updating user")
		return User{}, errors.New("User with id " + strconv.Itoa(user.Id) + " not found")
	}
	return user, nil
}

func (u *UserRepoImpl) DeleteUser(ctx context.Context, id int) error {
	query := `DELETE FROM users WHERE id = $1`
	result, err := u.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		log.Info("no rows affected of deleting user")
		return errors.New("User with id " + strconv.Itoa(id) + " not found")
	}
	return nil
}

func (u *UserRepoImpl) GetAllUsers(ctx context.Context) ([]User, error) {
	query := `SELECT id, uid, username, display_name, photo_url, timezone, week_first_day, event_calendar_type, 
		        event_calendar_google_calendar_id, ignore_short_events FROM users`
	rows, err := u.db.Query(ctx, query)
	if err != nil {
		log.Errorf("failed to get users: %v", err)
		return nil, err
	}
	defer rows.Close()
	users := make([]User, 0, 10)
	for rows.Next() {
		var user User
		var googleCalendarId sql.NullString
		err := rows.Scan(&user.Id, &user.Uid, &user.Username, &user.DisplayName, &user.PhotoUrl, &user.Settings.Timezone,
			&user.Settings.WeekFirstDay, &user.Settings.EventCalendarType, &googleCalendarId, &user.Settings.IgnoreShortEvents)
		if err != nil {
			log.Errorf("failed to scan user: %v", err)
			return nil, err
		}
		if googleCalendarId.Valid {
			user.Settings.GoogleCalendar.CalendarId = googleCalendarId.String
		}
		users = append(users, user)
		if err := rows.Err(); err != nil {
			log.Errorf("error iterating over rows: %v", err)
			return nil, err
		}
	}
	return users, nil
}

func (u *UserRepoImpl) IsUsernameAvailable(ctx context.Context, username string) (bool, error) {
	query := `SELECT COUNT(*) FROM users WHERE username = $1`
	var count int
	err := u.db.QueryRow(ctx, query, username).Scan(&count)
	if err != nil {
		log.Errorf("failed to check username availability: %v", err)
		return false, err
	}
	return count == 0, nil
}
