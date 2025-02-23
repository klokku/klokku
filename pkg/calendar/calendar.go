package calendar

import (
	"context"
	"time"
)

type Calendar interface {
	AddEvent(ctx context.Context, event Event) (*Event, error)
	GetEvents(ctx context.Context, from time.Time, to time.Time) ([]Event, error)
	ModifyEvent(ctx context.Context, event Event) (*Event, error)
	GetLastEvents(ctx context.Context, limit int) ([]Event, error)
}
