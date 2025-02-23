package user

import (
	"context"
	"errors"
	log "github.com/sirupsen/logrus"
	"strconv"
)

type contextKey string

const UserIDKey contextKey = "userId"

var ErrNoUser = errors.New("user not found")

// CurrentId retrieves the current user's ID from the context. Returns ErrNoUser if ID not present in context.
func CurrentId(ctx context.Context) (int, error) {
	idString, ok := ctx.Value(UserIDKey).(string)
	if !ok {
		log.Trace("user not found in context")
		return 0, ErrNoUser
	}
	id, err := strconv.Atoi(idString)
	if err != nil {
		log.Error("user id is malformed in context: ", idString)
		return 0, errors.New("userId is not an integer")
	}
	return id, nil
}

func WithId(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, UserIDKey, id)
}
