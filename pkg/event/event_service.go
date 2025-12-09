package event

import (
	"context"
	"fmt"
	"time"

	"github.com/klokku/klokku/internal/utils"
	"github.com/klokku/klokku/pkg/budget_plan"
	"github.com/klokku/klokku/pkg/calendar"
	"github.com/klokku/klokku/pkg/user"
	log "github.com/sirupsen/logrus"
)

var ErrNoCurrentEvent = fmt.Errorf("no current event")

type EventService interface {
	FindCurrentEvent(ctx context.Context) (*Event, error)
	StartNewEvent(ctx context.Context, event Event) (Event, error)
	DeleteCurrentEvent(ctx context.Context) (*Event, error)
	FinishCurrentEvent(ctx context.Context) ([]Event, error)
	ModifyCurrentEventStartTime(ctx context.Context, newStartTime time.Time) (Event, error)
	GetLastPreviousEvents(ctx context.Context, limit int) ([]Event, error)
}

type EventServiceImpl struct {
	repo         EventRepository
	calendar     calendar.Calendar
	clock        utils.Clock
	userProvider user.Provider
}

func NewEventService(repo EventRepository, calendar calendar.Calendar, userService user.Service) *EventServiceImpl {
	return &EventServiceImpl{repo, calendar, &utils.SystemClock{}, userService}
}

func (s *EventServiceImpl) FindCurrentEvent(ctx context.Context) (*Event, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	return s.repo.FindCurrentEvent(ctx, userId)
}

func (s *EventServiceImpl) StartNewEvent(ctx context.Context, event Event) (Event, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return Event{}, fmt.Errorf("failed to get current user: %w", err)
	}
	currentEvent, err := s.FindCurrentEvent(ctx)
	if err != nil {
		return Event{}, err
	}
	if currentEvent != nil {
		log.Debug("Event already started, finishing it before starting a new one")
		_, err := s.FinishCurrentEvent(ctx)
		if err != nil {
			return Event{}, err
		}
	}

	return s.repo.StoreEvent(ctx, userId, event)
}

func (s *EventServiceImpl) DeleteCurrentEvent(ctx context.Context) (*Event, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	event, err := s.repo.FindCurrentEvent(ctx, userId)
	if err != nil {
		return nil, err
	}
	if event == nil {
		log.Debug("No current event to delete")
		return nil, nil
	}
	err = s.repo.DeleteCurrentEvent(ctx, userId)
	if err != nil {
		return nil, err
	}

	return event, nil
}

func (s *EventServiceImpl) FinishCurrentEvent(ctx context.Context) ([]Event, error) {
	currentEvent, err := s.FindCurrentEvent(ctx)
	if err != nil {
		return nil, err
	}
	if currentEvent == nil {
		log.Debug("No current event to finish")
		return nil, nil
	}

	endTime := s.clock.Now()

	currentUser, err := s.userProvider.GetCurrentUser(ctx)
	if err != nil {
		log.Error("Could not retrieve current user")
		return nil, err
	}
	userTimezone := currentUser.Settings.Timezone
	eventsForCalendar, err := prepareEventsForCalendar(currentEvent, endTime, userTimezone)
	if err != nil {
		return nil, err
	}

	savedEvents := make([]Event, 0, len(eventsForCalendar))
	for _, event := range eventsForCalendar {
		calendarEvent, err := s.calendar.AddEvent(ctx, event)
		if err != nil {
			return nil, err
		}
		savedEvents = append(savedEvents, calendarEventToEvent(*calendarEvent))
	}

	_, err = s.DeleteCurrentEvent(ctx)
	if err != nil {
		return nil, err
	}

	return savedEvents, nil
}

func prepareEventsForCalendar(event *Event, endTime time.Time, userTimezone string) ([]calendar.Event, error) {
	calendarEvent := calendar.Event{
		Summary:   event.Budget.Name,
		StartTime: event.StartTime,
		EndTime:   endTime,
		Metadata: calendar.EventMetadata{
			BudgetId: event.Budget.Id,
		},
	}

	return splitEventIfNeeded(&calendarEvent, userTimezone)
}

func splitEventIfNeeded(event *calendar.Event, userTimezone string) ([]calendar.Event, error) {
	location, err := time.LoadLocation(userTimezone)
	if err != nil {
		err := fmt.Errorf("could not load location for timezone %s", userTimezone)
		log.Error(err)
		return nil, err
	}
	if crossesDateBoundary(event.StartTime, event.EndTime, location) {
		log.Debug("Event crosses date boundary, splitting it into two events")
		eventA := calendar.Event{
			Summary:   event.Summary,
			StartTime: event.StartTime,
			EndTime:   endOfDay(event.StartTime, location),
			Metadata:  event.Metadata,
		}
		eventB := calendar.Event{
			Summary:   event.Summary,
			StartTime: startOfNextDay(event.StartTime, location),
			EndTime:   event.EndTime,
			Metadata:  event.Metadata,
		}
		resultEvents := []calendar.Event{eventA}
		splitEventB, err := splitEventIfNeeded(&eventB, userTimezone)
		if err != nil {
			return nil, err
		}
		return append(resultEvents, splitEventB...), nil
	} else {
		return []calendar.Event{
			{
				Summary:   event.Summary,
				StartTime: event.StartTime,
				EndTime:   event.EndTime,
				Metadata:  event.Metadata,
			},
		}, nil
	}
}

func crossesDateBoundary(start, end time.Time, location *time.Location) bool {
	startDate := start.In(location).YearDay()
	endDate := end.In(location).YearDay()

	return startDate != endDate
}

func endOfDay(t time.Time, location *time.Location) time.Time {
	day := t.In(location)
	return time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 999999999, location)
}
func startOfNextDay(t time.Time, location *time.Location) time.Time {
	day := t.In(location)
	return time.Date(day.Year(), day.Month(), day.Day()+1, 0, 0, 0, 0, location)
}

func (s *EventServiceImpl) ModifyCurrentEventStartTime(ctx context.Context, newStartTime time.Time) (Event, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return Event{}, fmt.Errorf("failed to get current user: %w", err)
	}

	currentEvent, err := s.FindCurrentEvent(ctx)
	if err != nil {
		return Event{}, err
	}
	if currentEvent == nil {
		log.Infof("No current event to modify for user %d", userId)
		return Event{}, ErrNoCurrentEvent
	}

	// change previous event endTime to given time in calendar
	fromTime := currentEvent.StartTime.Add(time.Hour * -24)
	previousEvents, err := s.calendar.GetEvents(ctx, fromTime, currentEvent.StartTime)
	if err != nil {
		return Event{}, err
	}
	if len(previousEvents) > 0 {
		previousEvent := previousEvents[len(previousEvents)-1]
		previousEvent.EndTime = newStartTime
		_, err := s.calendar.ModifyEvent(ctx, previousEvent)
		if err != nil {
			return Event{}, err
		}
	} else {
		log.Debug("No previous calendar event found to modify")
	}

	currentEvent.StartTime = newStartTime
	err = s.repo.DeleteCurrentEvent(ctx, userId)
	if err != nil {
		return Event{}, err
	}
	return s.repo.StoreEvent(ctx, userId, *currentEvent)
}

func (s *EventServiceImpl) GetLastPreviousEvents(ctx context.Context, limit int) ([]Event, error) {
	calendarEvents, err := s.calendar.GetLastEvents(ctx, limit)
	if err != nil {
		return nil, err
	}
	events := make([]Event, 0, len(calendarEvents))
	for _, calendarEvent := range calendarEvents {
		events = append(events, calendarEventToEvent(calendarEvent))
	}
	return events, nil
}

func calendarEventToEvent(e calendar.Event) Event {
	return Event{
		UID: e.UID,
		Budget: budget_plan.BudgetItem{
			Id:   e.Metadata.BudgetId,
			Name: e.Summary,
		},
		StartTime: e.StartTime,
		EndTime:   e.EndTime,
	}
}
