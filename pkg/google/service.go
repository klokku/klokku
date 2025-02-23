package google

import (
	"context"
	"fmt"
	"github.com/klokku/klokku/pkg/user"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type CalendarItem struct {
	ID      string
	Summary string
}

type Service interface {
	GetCalendar(ctx context.Context, calendarId string) (*Calendar, error)
	ListCalendars(ctx context.Context) ([]CalendarItem, error)
}

type ServiceImpl struct {
	auth *GoogleAuth
}

func NewService(auth *GoogleAuth) *ServiceImpl {
	return &ServiceImpl{
		auth: auth,
	}
}

func (s *ServiceImpl) GetCalendar(ctx context.Context, calendarId string) (*Calendar, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	service, err := s.prepareGoogleService(ctx, userId)
	if err != nil {
		return nil, err
	}
	return newGoogleCalendar(service, userId, calendarId), nil
}

func (s *ServiceImpl) ListCalendars(ctx context.Context) ([]CalendarItem, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	googleService, err := s.prepareGoogleService(ctx, userId)
	if err != nil {
		return nil, err
	}
	calendars, err := googleService.CalendarList.List().Do()
	if err != nil {
		err := fmt.Errorf("unable to retrieve calendars from Google Calendar: %v", err)
		log.Error(err)
		return nil, err
	}
	var googleCalendars []CalendarItem
	for _, cal := range calendars.Items {
		googleCalendars = append(googleCalendars, CalendarItem{
			ID:      cal.Id,
			Summary: cal.Summary,
		})
	}
	return googleCalendars, nil
}

func (s *ServiceImpl) prepareGoogleService(ctx context.Context, userId int) (*calendar.Service, error) {

	client, err := s.auth.getClient(ctx, userId)
	if err != nil {
		err := fmt.Errorf("unable to retrieve Google auth client: %v", err)
		log.Error(err)
		return nil, err
	}
	if client == nil {
		log.Debug("user is unauthenticated, authentication is required")
		return nil, ErrUnathenticated
	}
	service, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		err := fmt.Errorf("unable to retrieve Calendar client: %v", err)
		log.Error(err)
		return nil, err
	}

	return service, nil
}
