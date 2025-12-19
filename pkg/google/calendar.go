package google

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/klokku/klokku/pkg/calendar"
	"github.com/klokku/klokku/pkg/user"
	"github.com/klokku/klokku/pkg/weekly_plan"
	log "github.com/sirupsen/logrus"
	gcal "google.golang.org/api/calendar/v3"
)

var ErrUnathenticated = fmt.Errorf("user is unauthenticated, authentication is required")

type Calendar struct {
	service           *gcal.Service
	planItemsProvider calendar.PlanItemsProviderFunc
	userId            int
	calendarId        string
}

func newGoogleCalendar(service *gcal.Service, userId int, calendarId string, planItemsProvider calendar.PlanItemsProviderFunc) *Calendar {
	return &Calendar{
		service:           service,
		planItemsProvider: planItemsProvider,
		userId:            userId,
		calendarId:        calendarId,
	}
}

func (c *Calendar) AddEvent(ctx context.Context, event calendar.Event) ([]calendar.Event, error) {
	log.Debugf("Adding event: %+v, to calendar: %s", event, c.calendarId)
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		err := fmt.Errorf("unable to marshal event metadata: %v", err)
		log.Error(err)
		return nil, err
	}

	currentUser, err := user.CurrentUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	events, err := splitEventIfNeeded(&event, currentUser.Settings.Timezone)
	if err != nil {
		return nil, err
	}
	var storedEvents []calendar.Event
	for _, e := range events {
		planItemName, err := c.getEventName(err, ctx, e.StartTime, e.Metadata.BudgetItemId)
		if err != nil {
			return nil, err
		}
		e.Summary = planItemName

		result, err := c.service.Events.Insert(c.calendarId, &gcal.Event{
			Summary:     e.Summary,
			Description: string(metadata),
			Start: &gcal.EventDateTime{
				DateTime: e.StartTime.Format(time.RFC3339),
			},
			End: &gcal.EventDateTime{
				DateTime: e.EndTime.Format(time.RFC3339),
			},
		}).Do()

		if err != nil {
			err := fmt.Errorf("unable to insert event in Google Calendar: %v", err)
			log.Error(err)
			return nil, err
		}

		e.UID = result.Id
		storedEvents = append(storedEvents, e)
	}
	return storedEvents, nil
}

func (c *Calendar) GetEvents(_ context.Context, from time.Time, to time.Time) ([]calendar.Event, error) {
	googleEvents, err := c.service.Events.List(c.calendarId).
		TimeMin(from.Format(time.RFC3339)).
		TimeMax(to.Format(time.RFC3339)).
		SingleEvents(true).
		OrderBy("startTime").
		Do()

	if err != nil {
		err := fmt.Errorf("unable to retrieve events from Google Calendar: %v", err)
		log.Error(err)
		return nil, err
	}

	events, err := c.googleEventsToEvents(googleEvents.Items)
	if err != nil {
		return nil, err
	}

	return events, nil
}

func (c *Calendar) googleEventsToEvents(googleEvents []*gcal.Event) ([]calendar.Event, error) {
	events := make([]calendar.Event, 0, len(googleEvents))
	for _, item := range googleEvents {

		startTime, _ := time.Parse(time.RFC3339, item.Start.DateTime)
		endTime, _ := time.Parse(time.RFC3339, item.End.DateTime)
		var metadata calendar.EventMetadata
		if item.Description != "" {
			err := json.Unmarshal([]byte(item.Description), &metadata)
			if err != nil {
				log.Errorf("unable to unmarshal event metadata: %v", err)
				return nil, err
			}
		} else {
			log.Warnf("found calendar event without metadata - ignoring: %s (%s - %s)", item.Summary, item.Start.DateTime, item.End.DateTime)
		}

		events = append(events, calendar.Event{
			UID:       item.Id,
			Summary:   item.Summary,
			StartTime: startTime,
			EndTime:   endTime,
			Metadata:  metadata,
		})
	}
	return events, nil
}

func (c *Calendar) ModifyEvent(ctx context.Context, event calendar.Event) ([]calendar.Event, error) {
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		err := fmt.Errorf("unable to marshal event metadata: %v", err)
		log.Error(err)
		return nil, err
	}

	var updatedEvents []calendar.Event
	currentUser, err := user.CurrentUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	events, err := splitEventIfNeeded(&event, currentUser.Settings.Timezone)
	if err != nil {
		return nil, err
	}
	eventToUpdate := events[0]
	eventsToAdd := events[1:]

	planItemName, err := c.getEventName(err, ctx, eventToUpdate.StartTime, eventToUpdate.Metadata.BudgetItemId)
	if err != nil {
		return nil, err
	}
	eventToUpdate.Summary = planItemName

	updatedGoogleEvent, err := c.service.Events.Update(c.calendarId, event.UID, &gcal.Event{
		Summary:     eventToUpdate.Summary,
		Description: string(metadata),
		Start: &gcal.EventDateTime{
			DateTime: eventToUpdate.StartTime.Format(time.RFC3339),
		},
		End: &gcal.EventDateTime{
			DateTime: eventToUpdate.EndTime.Format(time.RFC3339),
		},
	}).Do()
	if err != nil {
		err := fmt.Errorf("unable to update event in Google Calendar: %v", err)
		log.Error(err)
		return nil, err
	}
	eventToUpdate.UID = updatedGoogleEvent.Id
	updatedEvents = append(updatedEvents, eventToUpdate)
	for _, e := range eventsToAdd {
		planItemName, err := c.getEventName(err, ctx, e.StartTime, e.Metadata.BudgetItemId)
		if err != nil {
			return nil, err
		}
		e.Summary = planItemName

		result, err := c.service.Events.Insert(c.calendarId, &gcal.Event{
			Summary:     e.Summary,
			Description: string(metadata),
			Start: &gcal.EventDateTime{
				DateTime: e.StartTime.Format(time.RFC3339),
			},
			End: &gcal.EventDateTime{
				DateTime: e.EndTime.Format(time.RFC3339),
			},
		}).Do()

		if err != nil {
			err := fmt.Errorf("unable to insert event in Google Calendar: %v", err)
			log.Error(err)
			return nil, err
		}

		e.UID = result.Id
		updatedEvents = append(updatedEvents, e)
	}

	return updatedEvents, nil
}

func (c *Calendar) GetLastEvents(ctx context.Context, limit int) ([]calendar.Event, error) {
	events, err := c.GetEvents(ctx, time.Now().AddDate(0, 0, -2), time.Now())
	if err != nil {
		return nil, err
	}

	slices.Reverse(events)

	if len(events) > limit {
		events = events[:limit]
	}
	return events, nil
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
			UID:       event.UID,
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
	}

	return []calendar.Event{
		{
			UID:       event.UID,
			Summary:   event.Summary,
			StartTime: event.StartTime,
			EndTime:   event.EndTime,
			Metadata:  event.Metadata,
		},
	}, nil
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

func (c *Calendar) getEventName(err error, ctx context.Context, startTime time.Time, budgetItemId int) (string, error) {
	planItems, err := c.planItemsProvider(ctx, startTime)
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

func (c *Calendar) DeleteEvent(ctx context.Context, eventUid string) error {
	//TODO implement me
	panic("implement me")
}
