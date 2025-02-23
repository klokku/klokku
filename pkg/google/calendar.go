package google

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/klokku/klokku/pkg/calendar"
	log "github.com/sirupsen/logrus"
	gcal "google.golang.org/api/calendar/v3"
	"slices"
	"time"
)

var ErrUnathenticated = fmt.Errorf("user is unauthenticated, authentication is required")

type Calendar struct {
	service    *gcal.Service
	userId     int
	calendarId string
}

func newGoogleCalendar(service *gcal.Service, userId int, calendarId string) *Calendar {
	return &Calendar{
		service:    service,
		userId:     userId,
		calendarId: calendarId,
	}
}

func (c *Calendar) AddEvent(_ context.Context, event calendar.Event) (*calendar.Event, error) {
	log.Debugf("Adding event: %+v, to calendar: %s", event, c.calendarId)
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		err := fmt.Errorf("unable to marshal event metadata: %v", err)
		log.Error(err)
		return nil, err
	}
	result, err := c.service.Events.Insert(c.calendarId, &gcal.Event{
		Summary:     event.Summary,
		Description: string(metadata),
		Start: &gcal.EventDateTime{
			DateTime: event.StartTime.Format(time.RFC3339),
		},
		End: &gcal.EventDateTime{
			DateTime: event.EndTime.Format(time.RFC3339),
		},
	}).Do()

	if err != nil {
		err := fmt.Errorf("unable to insert event in Google Calendar: %v", err)
		log.Error(err)
		return nil, err
	}

	event.UID = &result.Id

	return &event, nil
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
			UID:       &item.Id,
			Summary:   item.Summary,
			StartTime: startTime,
			EndTime:   endTime,
			Metadata:  metadata,
		})
	}
	return events, nil
}

func (c *Calendar) ModifyEvent(_ context.Context, event calendar.Event) (*calendar.Event, error) {
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		err := fmt.Errorf("unable to marshal event metadata: %v", err)
		log.Error(err)
		return nil, err
	}

	updatedGoogleEvent, err := c.service.Events.Update(c.calendarId, *event.UID, &gcal.Event{
		Summary:     event.Summary,
		Description: string(metadata),
		Start: &gcal.EventDateTime{
			DateTime: event.StartTime.Format(time.RFC3339),
		},
		End: &gcal.EventDateTime{
			DateTime: event.EndTime.Format(time.RFC3339),
		},
	}).Do()

	if err != nil {
		err := fmt.Errorf("unable to update event in Google Calendar: %v", err)
		log.Error(err)
		return nil, err
	}

	events, err := c.googleEventsToEvents([]*gcal.Event{updatedGoogleEvent})
	if err != nil {
		return nil, err
	}

	return &events[0], nil
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
