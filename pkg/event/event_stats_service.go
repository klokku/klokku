package event

import (
	"context"
	"github.com/klokku/klokku/internal/utils"
	"github.com/klokku/klokku/pkg/calendar"
	"time"
)

type EventStatsService interface {
	GetStats(ctx context.Context, from time.Time, to time.Time) (Stats, error)
}

type EventStatsServiceImpl struct {
	calendar calendar.Calendar
	clock    utils.Clock
}

func NewEventStatsServiceImpl(calendar calendar.Calendar, clock utils.Clock) EventStatsService {
	return &EventStatsServiceImpl{
		calendar: calendar,
		clock:    clock,
	}
}

func (s *EventStatsServiceImpl) GetStats(ctx context.Context, from time.Time, to time.Time) (Stats, error) {
	events, err := s.calendar.GetEvents(ctx, from, to)
	if err != nil {
		return Stats{}, err
	}

	eventsByDate := make(map[time.Time]map[int]time.Duration)
	eventsByBudget := make(map[int]time.Duration)

	for _, event := range events {
		eventsByBudget[event.Metadata.BudgetId] += duration(event)

		loc := event.StartTime.Location()
		year, month, day := event.StartTime.In(loc).Date()
		date := time.Date(year, month, day, 0, 0, 0, 0, loc)
		if eventsByDate[date] == nil {
			eventsByDate[date] = make(map[int]time.Duration)
		}
		eventsByDate[date][event.Metadata.BudgetId] += duration(event)
	}

	return Stats{
		StartDate: from,
		EndDate:   to,
		ByDate:    eventsByDate,
		ByBudget:  eventsByBudget,
	}, nil
}

func duration(event calendar.Event) time.Duration {
	return event.EndTime.Sub(event.StartTime)
}
