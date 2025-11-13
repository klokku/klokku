package user

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"
)

type contextKey string

const UserIDKey contextKey = "userId"

var ErrNoUser = errors.New("user not found")

// CurrentId retrieves the current user's ID from the context. Returns ErrNoUser if ID not present in context.
func CurrentId(ctx context.Context) (int, error) {
	id, ok := ctx.Value(UserIDKey).(int)
	if !ok {
		log.Trace("user not found in context")
		return 0, ErrNoUser
	}
	return id, nil
}

func WithId(ctx context.Context, id int) context.Context {
	return context.WithValue(ctx, UserIDKey, id)
}
