package calendar

import (
	"context"
	"fmt"
	"time"

	"github.com/klokku/klokku/pkg/user"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) AddEvent(ctx context.Context, event Event) (*Event, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	eventUid, err := s.repo.StoreEvent(ctx, userId, event)
	if err != nil {
		return nil, fmt.Errorf("failed to store event: %w", err)
	}

	event.UID = eventUid

	return &event, nil
}

func (s *Service) AddStickyEvent(ctx context.Context, event Event) (*Event, error) {
	overlappingEvents, err := s.GetEvents(ctx, event.StartTime, event.EndTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}
	eventsToModify, eventsToDelete, eventsToCreate := calculateStickyEventsChanges(overlappingEvents, event)
	var newEvent *Event
	err = s.repo.WithTransaction(ctx, func(repo Repository) error {
		s := NewService(repo)
		for _, event := range eventsToModify {
			_, err := s.ModifyEvent(ctx, event)
			if err != nil {
				return fmt.Errorf("failed to update event: %w", err)
			}
		}
		for _, event := range eventsToDelete {
			err := s.DeleteEvent(ctx, event.UID)
			if err != nil {
				return fmt.Errorf("failed to delete event: %w", err)
			}
		}
		for _, event := range eventsToCreate {
			_, err := s.AddEvent(ctx, event)
			if err != nil {
				return fmt.Errorf("failed to add event: %w", err)
			}
		}
		newEvent, err = s.AddEvent(ctx, event)
		if err != nil {
			return fmt.Errorf("failed to add event: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to perform transaction: %w", err)
	}

	return newEvent, nil
}

func (s *Service) GetEvents(ctx context.Context, from time.Time, to time.Time) ([]Event, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	return s.repo.GetEvents(ctx, userId, from, to)
}

func (s *Service) ModifyEvent(ctx context.Context, event Event) (*Event, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	err = s.repo.UpdateEvent(ctx, userId, event)
	if err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}
	return &event, nil
}

func (s *Service) ModifyStickyEvent(ctx context.Context, event Event) (*Event, error) {
	overlappingEvents, err := s.GetEvents(ctx, event.StartTime, event.EndTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}
	eventsToModify, eventsToDelete, eventsToCreate := calculateStickyEventsChanges(overlappingEvents, event)
	var modifiedEvent *Event
	err = s.repo.WithTransaction(ctx, func(repo Repository) error {
		s := NewService(repo)
		for _, event := range eventsToModify {
			_, err := s.ModifyEvent(ctx, event)
			if err != nil {
				return fmt.Errorf("failed to update event: %w", err)
			}
		}
		for _, event := range eventsToDelete {
			err := s.DeleteEvent(ctx, event.UID)
			if err != nil {
				return fmt.Errorf("failed to delete event: %w", err)
			}
		}
		for _, event := range eventsToCreate {
			_, err := s.AddEvent(ctx, event)
			if err != nil {
				return fmt.Errorf("failed to add event: %w", err)
			}
		}

		modifiedEvent, err = s.ModifyEvent(ctx, event)
		if err != nil {
			return fmt.Errorf("failed to modify event: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to perform transaction: %w", err)
	}

	return modifiedEvent, nil
}

func calculateStickyEventsChanges(overlappingEvents []Event, event Event) ([]Event, []Event, []Event) {
	eventsToModify := make([]Event, 0, len(overlappingEvents))
	eventsToDelete := make([]Event, 0, len(overlappingEvents))
	eventsToCreate := make([]Event, 0, len(overlappingEvents))
	if len(overlappingEvents) != 0 {
		for _, overlappingEvent := range overlappingEvents {
			if overlappingEvent.StartTime.Before(event.StartTime) && overlappingEvent.EndTime.Before(event.EndTime) {
				overlappingEvent.EndTime = event.StartTime
				eventsToModify = append(eventsToModify, overlappingEvent)
			} else if overlappingEvent.StartTime.Before(event.EndTime) && overlappingEvent.StartTime.After(event.StartTime) && overlappingEvent.EndTime.After(event.EndTime) {
				overlappingEvent.StartTime = event.EndTime
				eventsToModify = append(eventsToModify, overlappingEvent)
			} else if overlappingEvent.StartTime.After(event.StartTime) && overlappingEvent.EndTime.Before(event.EndTime) {
				eventsToDelete = append(eventsToDelete, overlappingEvent)
			} else if overlappingEvent.StartTime.Before(event.StartTime) && overlappingEvent.EndTime.After(event.EndTime) {
				newEvent := Event{
					Summary:   overlappingEvent.Summary,
					StartTime: event.EndTime,
					EndTime:   overlappingEvent.EndTime,
					Metadata:  overlappingEvent.Metadata,
				}
				eventsToCreate = append(eventsToCreate, newEvent)
				overlappingEvent.EndTime = event.StartTime
				eventsToModify = append(eventsToModify, overlappingEvent)
			}
		}
	}
	return eventsToModify, eventsToDelete, eventsToCreate
}

func (s *Service) GetLastEvents(ctx context.Context, limit int) ([]Event, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	return s.repo.GetLastEvents(ctx, userId, limit)
}

func (s *Service) DeleteEvent(ctx context.Context, eventUid string) error {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}
	return s.repo.DeleteEvent(ctx, userId, eventUid)
}
