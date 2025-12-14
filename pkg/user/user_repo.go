package user

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
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
	db *pgx.Conn
}

func NewUserRepo(db *pgx.Conn) *UserRepoImpl {
	return &UserRepoImpl{db: db}
}

func (u *UserRepoImpl) CreateUser(ctx context.Context, user User) (int, error) {
	eventCalendarType := user.Settings.EventCalendarType
	if eventCalendarType == "" {
		eventCalendarType = KlokkuCalendar
	}
	query := `INSERT INTO "user" (uid, username, display_name, photo_url, timezone, week_first_day, event_calendar_type, 
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
				event_calendar_google_calendar_id FROM "user" WHERE id = $1`
	var user User
	err := u.db.QueryRow(context.Background(), query, id).
		Scan(
			&user.Id,
			&user.Uid,
			&user.Username,
			&user.DisplayName,
			&user.PhotoUrl,
			&user.Settings.Timezone,
			&user.Settings.WeekFirstDay,
			&user.Settings.EventCalendarType,
			&user.Settings.GoogleCalendar.CalendarId,
		)
	if errors.Is(err, sql.ErrNoRows) {
		log.Errorf("user with id %d not found: %v", id, err)
		return User{}, err
	} else if err != nil {
		log.Errorf("failed to get user: %v", err)
		return User{}, err
	}
	return user, nil
}

func (u *UserRepoImpl) GetUserByUid(ctx context.Context, uid string) (User, error) {
	query := `SELECT id, uid, username, display_name, photo_url, timezone, week_first_day, event_calendar_type,
				event_calendar_google_calendar_id FROM "user" WHERE uid = $1`

	var user User
	err := u.db.QueryRow(context.Background(), query, uid).
		Scan(
			&user.Id,
			&user.Uid,
			&user.Username,
			&user.DisplayName,
			&user.PhotoUrl,
			&user.Settings.Timezone,
			&user.Settings.WeekFirstDay,
			&user.Settings.EventCalendarType,
			&user.Settings.GoogleCalendar.CalendarId,
		)
	if errors.Is(err, sql.ErrNoRows) {
		log.Infof("user with uid %s not found: %v", uid, err)
		return User{}, err
	} else if err != nil {
		log.Errorf("failed to get user: %v", err)
		return User{}, err
	}
	return user, nil
}

func (u *UserRepoImpl) UpdateUser(ctx context.Context, userId int, user User) (User, error) {
	query := `UPDATE "user" SET display_name = $1, timezone = $2, week_first_day = $3, event_calendar_type = $4, 
				event_calendar_google_calendar_id = $5 WHERE id = $6`
	result, err := u.db.Exec(ctx, query,
		user.DisplayName,
		user.Settings.Timezone,
		user.Settings.WeekFirstDay,
		user.Settings.EventCalendarType,
		user.Settings.GoogleCalendar.CalendarId,
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
	query := `DELETE FROM "user" WHERE id = $1`
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
		        event_calendar_google_calendar_id FROM "user"`
	rows, err := u.db.Query(ctx, query)
	if err != nil {
		log.Errorf("failed to get users: %v", err)
		return nil, err
	}
	defer rows.Close()
	users := make([]User, 0, 10)
	for rows.Next() {
		var user User
		err := rows.Scan(&user.Id, &user.Uid, &user.Username, &user.DisplayName, &user.PhotoUrl, &user.Settings.Timezone,
			&user.Settings.WeekFirstDay, &user.Settings.EventCalendarType, &user.Settings.GoogleCalendar.CalendarId)
		if err != nil {
			log.Errorf("failed to scan user: %v", err)
			return nil, err
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
	query := `SELECT COUNT(*) FROM "user" WHERE username = $1`
	var count int
	err := u.db.QueryRow(ctx, query, username).Scan(&count)
	if err != nil {
		log.Errorf("failed to check username availability: %v", err)
		return false, err
	}
	return count == 0, nil
}
