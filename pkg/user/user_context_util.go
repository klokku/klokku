package user

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"
)

type contextKey string

const UserKey contextKey = "user"

var ErrNoUser = errors.New("user not found")

// CurrentId retrieves the current user's ID from the context. Returns ErrNoUser if ID not present in context.
func CurrentId(ctx context.Context) (int, error) {
	user, ok := ctx.Value(UserKey).(User)
	if !ok {
		log.Trace("user not found in context")
		return 0, ErrNoUser
	}
	return user.Id, nil
}

func CurrentUser(ctx context.Context) (User, error) {
	user, ok := ctx.Value(UserKey).(User)
	if !ok {
		log.Trace("user not found in context")
		return User{}, ErrNoUser
	}
	return user, nil
}

func WithUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, UserKey, user)
}
