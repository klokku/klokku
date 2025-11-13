package user

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

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
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepoImpl {
	return &UserRepoImpl{db: db}
}

func (u *UserRepoImpl) CreateUser(ctx context.Context, user User) (int, error) {
	eventCalendarType := user.Settings.EventCalendarType
	if eventCalendarType == "" {
		eventCalendarType = KlokkuCalendar
	}
	query := `INSERT INTO user (uid, username, display_name, photo_url, timezone, week_first_day, event_calendar_type, 
				event_calendar_google_calendar_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := u.db.ExecContext(ctx, query,
		user.Uid,
		user.Username,
		user.DisplayName,
		user.PhotoUrl,
		user.Settings.Timezone,
		user.Settings.WeekFirstDay,
		eventCalendarType,
		user.Settings.GoogleCalendar.CalendarId,
	)
	if err != nil {
		log.Errorf("failed to create user: %v", err)
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		log.Errorf("failed to get last insert id: %v", err)
		return 0, err
	}
	return int(id), nil
}

func (u *UserRepoImpl) GetUser(ctx context.Context, id int) (User, error) {
	query := `SELECT id, uid, username, display_name, photo_url, timezone, week_first_day, event_calendar_type, 
				event_calendar_google_calendar_id FROM user WHERE id = ?`
	var user User
	err := u.db.QueryRowContext(context.Background(), query, id).
		Scan(&user.Id,
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
				event_calendar_google_calendar_id FROM user WHERE uid = ?`

	var user User
	err := u.db.QueryRowContext(context.Background(), query, uid).
		Scan(&user.Id,
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
	query := "UPDATE user SET display_name = ?, timezone = ?, week_first_day = ?, event_calendar_type = ?, event_calendar_google_calendar_id = ? WHERE id = ?"
	result, err := u.db.ExecContext(ctx, query,
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
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Errorf("failed to get rows affected of updating user: %v", err)
		return User{}, err
	}
	if rowsAffected == 0 {
		log.Info("no rows affected of updating user")
		return User{}, errors.New("User with id " + strconv.Itoa(user.Id) + " not found")
	}
	return user, nil
}

func (u *UserRepoImpl) DeleteUser(ctx context.Context, id int) error {
	query := "DELETE FROM user WHERE id = ?"
	result, err := u.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Errorf("failed to get rows affected of deleting user: %v", err)
		return err
	}
	if rowsAffected == 0 {
		log.Info("no rows affected of deleting user")
		return errors.New("User with id " + strconv.Itoa(id) + " not found")
	}
	return nil
}

func (u *UserRepoImpl) GetAllUsers(ctx context.Context) ([]User, error) {
	query := `SELECT id, uid, username, display_name, photo_url, timezone, week_first_day, event_calendar_type, 
		        event_calendar_google_calendar_id FROM user`
	rows, err := u.db.QueryContext(ctx, query)
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
	query := `SELECT COUNT(*) FROM user WHERE username = ?`
	var count int
	err := u.db.QueryRowContext(ctx, query, username).Scan(&count)
	if err != nil {
		log.Errorf("failed to check username availability: %v", err)
		return false, err
	}
	return count == 0, nil
}
