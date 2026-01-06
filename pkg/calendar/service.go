package calendar

import (
	"context"
	"fmt"
	"time"

	"github.com/klokku/klokku/internal/event_bus"
	"github.com/klokku/klokku/pkg/user"
	"github.com/klokku/klokku/pkg/weekly_plan"
	log "github.com/sirupsen/logrus"
)

type PlanItemsProviderFunc func(ctx context.Context, date time.Time) ([]weekly_plan.WeeklyPlanItem, error)

type Service struct {
	repo              Repository
	eventBus          *event_bus.EventBus
	planItemsProvider PlanItemsProviderFunc
}

func NewService(repo Repository, eventBus *event_bus.EventBus, planItemsProvider PlanItemsProviderFunc) *Service {
	return &Service{
		repo:              repo,
		eventBus:          eventBus,
		planItemsProvider: planItemsProvider,
	}
}

func (s *Service) AddEvent(ctx context.Context, event Event) ([]Event, error) {
	err := validateEvent(event)
	if err != nil {
		return nil, err
	}
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	var storedEvents []Event
	err = s.repo.WithTransaction(ctx, func(repo Repository) error {
		currentUser, err := user.CurrentUser(ctx)
		if err != nil {
			return fmt.Errorf("failed to get current user: %w", err)
		}
		events, err := splitEventIfNeeded(&event, currentUser.Settings.Timezone)
		if err != nil {
			return err
		}
		for _, e := range events {
			planItemName, err := s.getEventName(ctx, e.StartTime, e.Metadata.BudgetItemId)
			if err != nil {
				return err
			}
			e.Summary = planItemName

			storedEvent, err := repo.StoreEvent(ctx, userId, e)
			if err != nil {
				return err
			}
			storedEvents = append(storedEvents, storedEvent)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to perform transaction: %w", err)
	}

	for _, e := range storedEvents {
		err = s.eventBus.Publish(event_bus.NewEvent(ctx, "calendar.event.created", event_bus.CalendarEventCreated{
			UID:          e.UID,
			Summary:      e.Summary,
			StartTime:    e.StartTime,
			EndTime:      e.EndTime,
			BudgetItemId: e.Metadata.BudgetItemId,
		}))
		if err != nil {
			return nil, fmt.Errorf("failed to publish event creation: %w", err)
		}
	}

	return storedEvents, nil
}

func splitEventIfNeeded(event *Event, userTimezone string) ([]Event, error) {
	location, err := time.LoadLocation(userTimezone)
	if err != nil {
		err := fmt.Errorf("could not load location for timezone %s", userTimezone)
		log.Error(err)
		return nil, err
	}
	if crossesDateBoundary(event.StartTime, event.EndTime, location) {
		log.Debug("Event crosses date boundary, splitting it into two events")
		eventA := Event{
			UID:       event.UID,
			Summary:   event.Summary,
			StartTime: event.StartTime,
			EndTime:   endOfDay(event.StartTime, location),
			Metadata:  event.Metadata,
		}
		eventB := Event{
			Summary:   event.Summary,
			StartTime: startOfNextDay(event.StartTime, location),
			EndTime:   event.EndTime,
			Metadata:  event.Metadata,
		}
		resultEvents := []Event{eventA}
		splitEventB, err := splitEventIfNeeded(&eventB, userTimezone)
		if err != nil {
			return nil, err
		}
		return append(resultEvents, splitEventB...), nil
	} else {
		return []Event{
			{
				UID:       event.UID,
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

func (s *Service) AddStickyEvent(ctx context.Context, event Event) ([]Event, error) {
	err := validateEvent(event)
	if err != nil {
		return nil, err
	}
	overlappingEvents, err := s.GetEvents(ctx, event.StartTime, event.EndTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}
	eventsToModify, eventsToDelete, eventsToCreate := calculateStickyEventsChanges(overlappingEvents, event)
	var newEvents []Event
	err = s.repo.WithTransaction(ctx, func(repo Repository) error {
		s := NewService(repo, s.eventBus, s.planItemsProvider)
		for _, e := range eventsToModify {
			_, err := s.ModifyEvent(ctx, e)
			if err != nil {
				return fmt.Errorf("failed to update event: %w", err)
			}
		}
		for _, e := range eventsToDelete {
			err := s.DeleteEvent(ctx, e.UID)
			if err != nil {
				return fmt.Errorf("failed to delete event: %w", err)
			}
		}
		for _, e := range eventsToCreate {
			_, err := s.AddEvent(ctx, e)
			if err != nil {
				return fmt.Errorf("failed to add event: %w", err)
			}
		}
		newEvents, err = s.AddEvent(ctx, event)
		if err != nil {
			return fmt.Errorf("failed to add event: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to perform transaction: %w", err)
	}

	return newEvents, nil
}

func (s *Service) GetEvents(ctx context.Context, from time.Time, to time.Time) ([]Event, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	return s.repo.GetEvents(ctx, userId, from, to)
}

func (s *Service) ModifyEvent(ctx context.Context, event Event) ([]Event, error) {
	err := validateEvent(event)
	if err != nil {
		return nil, err
	}
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	var updatedEvents []Event
	err = s.repo.WithTransaction(ctx, func(repo Repository) error {
		currentUser, err := user.CurrentUser(ctx)
		if err != nil {
			return fmt.Errorf("failed to get current user: %w", err)
		}
		events, err := splitEventIfNeeded(&event, currentUser.Settings.Timezone)
		if err != nil {
			return err
		}
		eventToUpdate := events[0]
		eventsToAdd := events[1:]

		planItemName, err := s.getEventName(ctx, eventToUpdate.StartTime, eventToUpdate.Metadata.BudgetItemId)
		if err != nil {
			return err
		}
		eventToUpdate.Summary = planItemName

		updatedEvent, err := repo.UpdateEvent(ctx, userId, eventToUpdate)
		if err != nil {
			log.Errorf("failed to update event: %v", err)
			return err
		}
		updatedEvents = append(updatedEvents, updatedEvent)
		for _, e := range eventsToAdd {
			planItemName, err := s.getEventName(ctx, e.StartTime, e.Metadata.BudgetItemId)
			if err != nil {
				return err
			}
			e.Summary = planItemName
			newEvent, err := repo.StoreEvent(ctx, userId, e)
			if err != nil {
				log.Errorf("failed to store event: %v", err)
				return err
			}
			updatedEvents = append(updatedEvents, newEvent)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to perform transaction: %w", err)
	}

	return updatedEvents, nil
}

func (s *Service) getEventName(ctx context.Context, startTime time.Time, budgetItemId int) (string, error) {
	planItems, err := s.planItemsProvider(ctx, startTime)
	if err != nil {
		log.Errorf("failed to get plan items: %v", err)
		return "", err
	}
	var planItemInfo *weekly_plan.WeeklyPlanItem
	for _, planItem := range planItems {
		if planItem.BudgetItemId == budgetItemId {
			planItemInfo = &planItem
			break
		}
	}
	if planItemInfo == nil {
		return "", fmt.Errorf("invalid budget item id")
	}
	return planItemInfo.Name, nil
}

func (s *Service) ModifyStickyEvent(ctx context.Context, event Event) ([]Event, error) {
	err := validateEvent(event)
	if err != nil {
		return nil, err
	}
	overlappingEvents, err := s.GetEvents(ctx, event.StartTime, event.EndTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}
	eventsToModify, eventsToDelete, eventsToCreate := calculateStickyEventsChanges(overlappingEvents, event)
	var modifiedEvents []Event
	err = s.repo.WithTransaction(ctx, func(repo Repository) error {
		s := NewService(repo, s.eventBus, s.planItemsProvider)
		for _, e := range eventsToModify {
			_, err := s.ModifyEvent(ctx, e)
			if err != nil {
				return fmt.Errorf("failed to update event: %w", err)
			}
		}
		for _, e := range eventsToDelete {
			err := s.DeleteEvent(ctx, e.UID)
			if err != nil {
				return fmt.Errorf("failed to delete event: %w", err)
			}
		}
		for _, e := range eventsToCreate {
			_, err := s.AddEvent(ctx, e)
			if err != nil {
				return fmt.Errorf("failed to add event: %w", err)
			}
		}

		modifiedEvents, err = s.ModifyEvent(ctx, event)
		if err != nil {
			return fmt.Errorf("failed to modify event: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to perform transaction: %w", err)
	}

	return modifiedEvents, nil
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

func validateEvent(event Event) error {
	if event.StartTime.IsZero() {
		return fmt.Errorf("start time cannot be zero")
	}
	if event.EndTime.IsZero() {
		return fmt.Errorf("end time cannot be zero")
	}
	if !event.EndTime.After(event.StartTime) {
		return fmt.Errorf("end time must be after start time")
	}
	if event.Metadata.BudgetItemId == 0 {
		return fmt.Errorf("budget item id cannot be zero")
	}
	return nil
}
