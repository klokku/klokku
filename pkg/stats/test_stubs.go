package stats

import (
	"context"
	"sort"
	"time"

	"github.com/klokku/klokku/pkg/event"
	"github.com/klokku/klokku/pkg/weekly_plan"
)

type weeklyPlanItemsReaderStub struct {
	items map[int]weekly_plan.WeeklyPlanItem // id -> item
}

func newWeeklyPlanItemsReaderStub() *weeklyPlanItemsReaderStub {
	return &weeklyPlanItemsReaderStub{
		items: make(map[int]weekly_plan.WeeklyPlanItem),
	}
}

func (s *weeklyPlanItemsReaderStub) setItems(items []weekly_plan.WeeklyPlanItem) {
	s.reset()
	for _, item := range items {
		s.items[item.Id] = item
	}
}

func (s *weeklyPlanItemsReaderStub) GetItemsForWeek(ctx context.Context, date time.Time) ([]weekly_plan.WeeklyPlanItem, error) {
	var items []weekly_plan.WeeklyPlanItem
	for _, item := range s.items {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Position < items[j].Position
	})
	return items, nil
}

func (s *weeklyPlanItemsReaderStub) reset() {
	s.items = make(map[int]weekly_plan.WeeklyPlanItem)
}

type currentEventProviderStub struct {
	event *event.Event
}

func newCurrentEventProviderStub() *currentEventProviderStub {
	return &currentEventProviderStub{
		event: nil,
	}
}

func (s *currentEventProviderStub) FindCurrentEvent(ctx context.Context) (*event.Event, error) {
	return s.event, nil
}

func (s *currentEventProviderStub) set(event *event.Event) {
	s.event = event
}

func (s *currentEventProviderStub) reset() {
	s.event = nil
}
