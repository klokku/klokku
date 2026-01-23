package current_event

import (
	"context"
	"fmt"
	"time"

	"github.com/klokku/klokku/internal/utils"
	"github.com/klokku/klokku/pkg/calendar"
	"github.com/klokku/klokku/pkg/user"
	log "github.com/sirupsen/logrus"
)

var ErrNoCurrentEvent = fmt.Errorf("no current event")

type Service interface {
	FindCurrentEvent(ctx context.Context) (CurrentEvent, error)
	StartNewEvent(ctx context.Context, event CurrentEvent) (CurrentEvent, error)
	ModifyCurrentEventStartTime(ctx context.Context, newStartTime time.Time) (CurrentEvent, error)
}

type EventServiceImpl struct {
	repo     Repository
	calendar calendar.Calendar
	clock    utils.Clock
}

func NewEventService(repo Repository, calendar calendar.Calendar) *EventServiceImpl {
	return &EventServiceImpl{repo, calendar, &utils.SystemClock{}}
}

func (s *EventServiceImpl) FindCurrentEvent(ctx context.Context) (CurrentEvent, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return CurrentEvent{}, fmt.Errorf("failed to get current user: %w", err)
	}
	return s.repo.FindCurrentEvent(ctx, userId)
}

func (s *EventServiceImpl) StartNewEvent(ctx context.Context, event CurrentEvent) (CurrentEvent, error) {
	currentUser, err := user.CurrentUser(ctx)
	if err != nil {
		return CurrentEvent{}, fmt.Errorf("failed to get current user: %w", err)
	}
	currentEvent, err := s.FindCurrentEvent(ctx)
	if err != nil {
		return CurrentEvent{}, err
	}
	if currentEvent.Id != 0 {
		eventDuration := s.clock.Now().Sub(currentEvent.StartTime)
		isShortEvent := eventDuration < time.Minute

		if currentUser.Settings.IgnoreShortEvents && isShortEvent {
			log.Debugf("Ignoring short event (duration: %v), not storing to calendar", eventDuration)
			// Use the start time of the previous event for the new event
			event.StartTime = currentEvent.StartTime
		} else {
			log.Debug("Storing previous event to calendar before starting new one")
			err := s.storeEventToCalendar(ctx, currentEvent)
			if err != nil {
				return CurrentEvent{}, err
			}
		}
	}

	return s.repo.ReplaceCurrentEvent(ctx, currentUser.Id, event)
}

func (s *EventServiceImpl) storeEventToCalendar(ctx context.Context, event CurrentEvent) error {
	endTime := s.clock.Now()
	calEvent := calendar.Event{
		Summary:   event.PlanItem.Name,
		StartTime: event.StartTime,
		EndTime:   endTime,
		Metadata: calendar.EventMetadata{
			BudgetItemId: event.PlanItem.BudgetItemId,
		},
	}

	_, err := s.calendar.AddEvent(ctx, calEvent)
	if err != nil {
		return err
	}

	return nil
}

func (s *EventServiceImpl) ModifyCurrentEventStartTime(ctx context.Context, newStartTime time.Time) (CurrentEvent, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return CurrentEvent{}, fmt.Errorf("failed to get current user: %w", err)
	}

	if newStartTime.After(s.clock.Now()) {
		return CurrentEvent{}, fmt.Errorf("new start time cannot be in the future")
	}

	currentEvent, err := s.FindCurrentEvent(ctx)
	if err != nil {
		return CurrentEvent{}, err
	}
	if currentEvent.Id == 0 {
		log.Infof("No current event to modify for user %d", userId)
		return CurrentEvent{}, ErrNoCurrentEvent
	}

	var previousEvents []calendar.Event
	if newStartTime.After(currentEvent.StartTime) { // Moving the current event start time forward
		previousEvents, err = s.calendar.GetLastEvents(ctx, 1)
		if err != nil {
			return CurrentEvent{}, err
		}
	} else { // Moving the current event start time backward
		previousEvents, err = s.calendar.GetEvents(ctx, newStartTime, time.Now())
		if err != nil {
			return CurrentEvent{}, err
		}
	}

	if len(previousEvents) > 0 {
		previousEvent := previousEvents[0] // the most early one
		otherEvents := previousEvents[1:]  // the rest between previousEvent and currentEvent that need to be deleted

		for _, event := range otherEvents {
			log.Debugf("Deleting event %v from calendar", event)
			err := s.calendar.DeleteEvent(ctx, event.UID)
			if err != nil {
				return CurrentEvent{}, err
			}
		}

		previousEvent.EndTime = newStartTime
		log.Debugf("Modifying event %v in calendar", previousEvent)
		_, err := s.calendar.ModifyEvent(ctx, previousEvent)
		if err != nil {
			return CurrentEvent{}, err
		}

	} else {
		log.Debug("No previous calendar events found to modify/delete")
	}

	currentEvent.StartTime = newStartTime
	return s.repo.ReplaceCurrentEvent(ctx, userId, currentEvent)
}
