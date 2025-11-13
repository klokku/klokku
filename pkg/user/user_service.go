package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
)

var ErrUserNotFound = errors.New("user not found")
var ErrUserDataInvalid = errors.New("user data invalid")

const storagePath = "storage/user_photos/"

type Service interface {
	GetCurrentUser(ctx context.Context) (User, error)
	CreateUser(ctx context.Context, user User) (User, error)
	GetUser(ctx context.Context, id int) (User, error)
	UpdateUser(ctx context.Context, user User) (User, error)
	GetUserByUid(ctx context.Context, uid string) (User, error)
	DeleteUser(ctx context.Context, id int) error
	GetAllUsers(ctx context.Context) ([]User, error)
	StoreUserPhoto(ctx context.Context, photo []byte) error
	GetUserPhoto(ctx context.Context, id int) ([]byte, error)
	GetCurrentUserPhoto(ctx context.Context) ([]byte, error)
	DeleteUserPhoto(ctx context.Context) error
	IsUsernameAvailable(ctx context.Context, username string) (bool, error)
}

type Provider interface {
	GetCurrentUser(ctx context.Context) (User, error)
}

type UserServiceImpl struct {
	repo Repo
}

func NewUserService(repo Repo) *UserServiceImpl {
	return &UserServiceImpl{repo: repo}
}

func (u *UserServiceImpl) GetCurrentUser(ctx context.Context) (User, error) {
	userId, err := CurrentId(ctx)
	if err != nil {
		return User{}, fmt.Errorf("failed to get current user: %w", err)
	}
	return u.GetUser(ctx, userId)
}

func (u *UserServiceImpl) CreateUser(ctx context.Context, user User) (User, error) {
	userValid := u.validateUser(ctx, user)
	if !userValid {
		return User{}, ErrUserDataInvalid
	}
	userId, err := u.repo.CreateUser(ctx, user)
	if err != nil {
		return User{}, err
	}
	user.Id = userId
	return user, nil
}

func (u *UserServiceImpl) validateUser(ctx context.Context, user User) bool {
	if user.DisplayName == "" || user.Username == "" || user.Uid == "" {
		return false
	}
	available, err := u.IsUsernameAvailable(ctx, user.Username)
	if err != nil {
		log.Errorf("failed to check username availability: %v", err)
		return false
	}
	return available
}

func (u *UserServiceImpl) GetUser(ctx context.Context, id int) (User, error) {
	user, err := u.repo.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return user, nil
}

func (u *UserServiceImpl) GetUserByUid(ctx context.Context, uid string) (User, error) {
	user, err := u.repo.GetUserByUid(ctx, uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return user, nil
}

func (u *UserServiceImpl) UpdateUser(ctx context.Context, user User) (User, error) {
	userId, err := CurrentId(ctx)
	if err != nil {
		return User{}, fmt.Errorf("failed to get current user: %w", err)
	}
	return u.repo.UpdateUser(ctx, userId, user)
}

func (u *UserServiceImpl) DeleteUser(ctx context.Context, id int) error {
	return u.repo.DeleteUser(ctx, id)
}

func (u *UserServiceImpl) GetAllUsers(ctx context.Context) ([]User, error) {
	return u.repo.GetAllUsers(ctx)
}

func (u *UserServiceImpl) StoreUserPhoto(ctx context.Context, photo []byte) error {
	userId, err := CurrentId(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	err = os.MkdirAll(storagePath, 0755)
	if err != nil {
		return err
	}
	err = os.WriteFile(storagePath+"/"+strconv.Itoa(userId)+".jpg", photo, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (u *UserServiceImpl) GetUserPhoto(_ context.Context, id int) ([]byte, error) {
	expectedFile := storagePath + "/" + strconv.Itoa(id) + ".jpg"
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		return nil, nil
	}
	return os.ReadFile(expectedFile)
}

func (u *UserServiceImpl) GetCurrentUserPhoto(ctx context.Context) ([]byte, error) {
	userId, err := CurrentId(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	return u.GetUserPhoto(ctx, userId)
}

func (u *UserServiceImpl) DeleteUserPhoto(ctx context.Context) error {
	userId, err := CurrentId(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	expectedFile := storagePath + "/" + strconv.Itoa(userId) + ".jpg"
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(expectedFile)
}

func (u *UserServiceImpl) IsUsernameAvailable(ctx context.Context, username string) (bool, error) {
	available, err := u.repo.IsUsernameAvailable(ctx, username)
	if err != nil {
		return false, err
	}
	return available, nil
}
