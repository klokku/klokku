package calendar_provider

import (
	"context"
	"fmt"
	"time"

	"github.com/klokku/klokku/pkg/calendar"
	"github.com/klokku/klokku/pkg/google"
	"github.com/klokku/klokku/pkg/user"
	log "github.com/sirupsen/logrus"
)

type EventsMigrator interface {
	MigrateFromGoogleToKlokku() error
}

type EventsMigratorImpl struct {
	userService    user.Service
	googleService  google.Service
	klokkuCalendar calendar.Calendar
}

func NewEventsMigratorImpl(calendarProvider *CalendarProvider) *EventsMigratorImpl {
	return &EventsMigratorImpl{
		userService:    calendarProvider.userService,
		googleService:  calendarProvider.googleService,
		klokkuCalendar: calendarProvider.klokkuCalendar,
	}
}

func (m *EventsMigratorImpl) MigrateFromGoogleToKlokku(ctx context.Context, from time.Time, to time.Time) (int, error) {
	currentUser, err := m.userService.GetCurrentUser(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get current user: %w", err)
	}
	googleCalendar, err := m.getGoogleCalendar(ctx, &currentUser)
	if err != nil {
		return 0, fmt.Errorf("failed to get Google Calendar: %w", err)
	}
	googleEvents, err := googleCalendar.GetEvents(ctx, from, to)
	if err != nil {
		return 0, fmt.Errorf("failed to get events from Google Calendar: %w", err)
	}

	numberOfMigratedEvents := 0
	for _, event := range googleEvents {
		_, err := m.klokkuCalendar.AddEvent(ctx, event)
		if err != nil {
			log.Errorf("failed to add event %v to Klokku Calendar: %v. Trying to continue", event, err)
		} else {
			numberOfMigratedEvents++
		}
	}
	return numberOfMigratedEvents, nil
}

func (m *EventsMigratorImpl) MigrateFromKlokkuToGoogle(ctx context.Context, from time.Time, to time.Time) (int, error) {
	currentUser, err := m.userService.GetCurrentUser(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get current user: %w", err)
	}
	googleCalendar, err := m.getGoogleCalendar(ctx, &currentUser)
	if err != nil {
		return 0, fmt.Errorf("failed to get Google Calendar: %w", err)
	}
	events, err := m.klokkuCalendar.GetEvents(ctx, from, to)
	if err != nil {
		return 0, fmt.Errorf("failed to get events from Klokku Calendar: %w", err)
	}
	numberOfMigratedEvents := 0
	for _, event := range events {
		_, err := googleCalendar.AddEvent(ctx, event)
		if err != nil {
			log.Errorf("failed to add event %v to Google Calendar: %v. Trying to continue", event, err)
		} else {
			numberOfMigratedEvents++
		}
	}
	return numberOfMigratedEvents, nil
}

func (m *EventsMigratorImpl) getGoogleCalendar(ctx context.Context, user *user.User) (*google.Calendar, error) {
	if user.Settings.GoogleCalendar.CalendarId == "" {
		log.Info("No Google Calendar configured, skipping migration")
		return nil, fmt.Errorf("google calendar is not configured")
	}

	googleCalendar, err := m.googleService.GetCalendar(ctx, user.Settings.GoogleCalendar.CalendarId)
	if err != nil {
		return nil, fmt.Errorf("failed to get Google Calendar: %w", err)
	}
	return googleCalendar, nil
}
