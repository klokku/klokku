package calendar_provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/klokku/klokku/pkg/calendar"
	"github.com/klokku/klokku/pkg/google"
	"github.com/klokku/klokku/pkg/user"
	"time"
)

type CalendarProvider struct {
	userService   user.Service
	googleService google.Service
}

func NewCalendarProvider(userService user.Service, googleService google.Service) *CalendarProvider {
	return &CalendarProvider{
		userService:   userService,
		googleService: googleService,
	}
}

func (c *CalendarProvider) getCalendar(ctx context.Context) (calendar.Calendar, error) {
	currentUser, err := c.userService.GetCurrentUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user when getting calendar: %w", err)
	}
	switch u := currentUser; u.Settings.EventCalendarType {
	case "google":
		return c.googleService.GetCalendar(ctx, u.Settings.GoogleCalendar.CalendarId)
	}
	return nil, errors.New("unknown calendar type")
}

func (c *CalendarProvider) AddEvent(ctx context.Context, event calendar.Event) (*calendar.Event, error) {
	cal, err := c.getCalendar(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get calendar when adding event: %w", err)
	}
	return cal.AddEvent(ctx, event)
}

func (c *CalendarProvider) GetEvents(ctx context.Context, from time.Time, to time.Time) ([]calendar.Event, error) {
	cal, err := c.getCalendar(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get calendar when getting events: %w", err)
	}
	return cal.GetEvents(ctx, from, to)
}

func (c *CalendarProvider) ModifyEvent(ctx context.Context, event calendar.Event) (*calendar.Event, error) {
	cal, err := c.getCalendar(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get calendar when modifying event: %w", err)
	}
	return cal.ModifyEvent(ctx, event)
}

func (c *CalendarProvider) GetLastEvents(ctx context.Context, limit int) ([]calendar.Event, error) {
	cal, err := c.getCalendar(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get calendar when getting last events: %w", err)
	}
	return cal.GetLastEvents(ctx, limit)
}
