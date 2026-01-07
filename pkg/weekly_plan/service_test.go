package weekly_plan

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klokku/klokku/internal/event_bus"
	"github.com/klokku/klokku/pkg/budget_plan"
	"github.com/klokku/klokku/pkg/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.WithValue(context.Background(), user.UserKey, user.User{
	Id:          10,
	Uid:         uuid.NewString(),
	Username:    "test-user-1",
	DisplayName: "Test User 1",
	PhotoUrl:    "",
	Settings: user.Settings{
		Timezone:          "Europe/Warsaw",
		WeekFirstDay:      time.Monday,
		EventCalendarType: user.KlokkuCalendar,
		GoogleCalendar:    user.GoogleCalendarSettings{},
	},
})

var repoStub = NewRepositoryStub()
var bpReaderStub = NewBudgetPlanReaderStub()
var eventBus = event_bus.NewEventBus()

var service Service

func setup(t *testing.T) func() {
	service = NewService(repoStub, bpReaderStub, eventBus)
	return func() {
		t.Log("Teardown after test")
		repoStub.Reset()
		bpReaderStub.Reset()
	}
}

func TestServiceImpl_GetItemsForWeek(t *testing.T) {
	t.Run("returns existing weekly items when they exist", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		weekDate := time.Date(2025, 1, 13, 0, 0, 0, 0, time.UTC)

		// Set up budget plan and create items through service
		plan := budget_plan.BudgetPlan{
			Id:        1,
			Name:      "My Plan",
			IsCurrent: true,
			Items: []budget_plan.BudgetItem{
				{
					Id:                101,
					PlanId:            1,
					Name:              "Work",
					WeeklyDuration:    40 * time.Hour,
					WeeklyOccurrences: 5,
					Icon:              "💼",
					Color:             "#FF5733",
					Position:          0,
				},
				{
					Id:                102,
					PlanId:            1,
					Name:              "Exercise",
					WeeklyDuration:    5 * time.Hour,
					WeeklyOccurrences: 3,
					Icon:              "🏃",
					Color:             "#33FF57",
					Position:          1,
				},
			},
		}
		bpReaderStub.SetCurrentPlan(plan)
		bpReaderStub.SetPlan(plan)

		// Create items through UpdateItem (which creates all items for the week)
		_, err := service.UpdateItem(ctx, weekDate, 0, 101, 30*time.Hour, "Some note")
		require.NoError(t, err, "failed to create initial items")

		// Now get the items
		items, err := service.GetItemsForWeek(ctx, weekDate)

		require.NoError(t, err)
		require.Len(t, items, 2)
		assert.Equal(t, "Work", items[0].Name)
		assert.Equal(t, 101, items[0].BudgetItemId)
		assert.Equal(t, 30*time.Hour, items[0].WeeklyDuration)
		assert.Equal(t, "Some note", items[0].Notes)
	})

	t.Run("returns items from current budget plan when no weekly items exist", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		weekDate := time.Date(2025, 1, 13, 0, 0, 0, 0, time.UTC)

		// Set up budget plan
		plan := budget_plan.BudgetPlan{
			Id:        1,
			Name:      "My Plan",
			IsCurrent: true,
			Items: []budget_plan.BudgetItem{
				{
					Id:                101,
					PlanId:            1,
					Name:              "Work",
					WeeklyDuration:    40 * time.Hour,
					WeeklyOccurrences: 5,
					Icon:              "💼",
					Color:             "#FF5733",
					Position:          0,
				},
				{
					Id:                102,
					PlanId:            1,
					Name:              "Exercise",
					WeeklyDuration:    5 * time.Hour,
					WeeklyOccurrences: 3,
					Icon:              "🏃",
					Color:             "#33FF57",
					Position:          1,
				},
			},
		}
		bpReaderStub.SetCurrentPlan(plan)

		items, err := service.GetItemsForWeek(ctx, weekDate)

		require.NoError(t, err)
		require.Len(t, items, 2)
		assert.Equal(t, "Work", items[0].Name)
		assert.Equal(t, 0, items[0].Id)
	})

	t.Run("returns error when no current plan exists", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		weekDate := time.Date(2025, 1, 13, 0, 0, 0, 0, time.UTC)

		items, err := service.GetItemsForWeek(ctx, weekDate)

		require.ErrorIs(t, err, ErrNoCurrentPlan)
		assert.Nil(t, items)
	})
}

func TestServiceImpl_ResetWeekItemToBudgetPlanItem(t *testing.T) {
	t.Run("resets weekly item to budget plan values", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		weekDate := time.Date(2025, 1, 13, 0, 0, 0, 0, time.UTC)

		// Set up budget plan
		plan := budget_plan.BudgetPlan{
			Id:        1,
			Name:      "My Plan",
			IsCurrent: true,
			Items: []budget_plan.BudgetItem{
				{
					Id:                101,
					PlanId:            1,
					Name:              "Work",
					WeeklyDuration:    40 * time.Hour,
					WeeklyOccurrences: 5,
					Icon:              "💼",
					Color:             "#FF5733",
					Position:          0,
				},
			},
		}
		bpReaderStub.SetCurrentPlan(plan)
		bpReaderStub.SetPlan(plan)

		// Create item through service
		createdItem, err := service.UpdateItem(ctx, weekDate, 0, 101, 35*time.Hour, "Custom notes")
		require.NoError(t, err, "failed to create item")
		require.Equal(t, 35*time.Hour, createdItem.WeeklyDuration, "setup failed: item not modified correctly")

		// Reset to budget plan values
		resetItem, err := service.ResetWeekItemToBudgetPlanItem(ctx, createdItem.Id)

		require.NoError(t, err)
		assert.Equal(t, 40*time.Hour, resetItem.WeeklyDuration)
		assert.Empty(t, resetItem.Notes)
	})

	t.Run("returns error when weekly item not found", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		_, err := service.ResetWeekItemToBudgetPlanItem(ctx, 999)

		require.ErrorIs(t, err, ErrWeeklyItemNotFound)
	})

	t.Run("returns error when budget item not found", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		weekDate := time.Date(2025, 1, 13, 0, 0, 0, 0, time.UTC)

		// Set up budget plan with one item
		plan := budget_plan.BudgetPlan{
			Id:        1,
			Name:      "My Plan",
			IsCurrent: true,
			Items: []budget_plan.BudgetItem{
				{
					Id:                101,
					PlanId:            1,
					Name:              "Work",
					WeeklyDuration:    40 * time.Hour,
					WeeklyOccurrences: 5,
					Icon:              "💼",
					Color:             "#FF5733",
					Position:          0,
				},
			},
		}
		bpReaderStub.SetCurrentPlan(plan)
		bpReaderStub.SetPlan(plan)

		// Create item through service
		createdItem, err := service.UpdateItem(ctx, weekDate, 0, 101, 35*time.Hour, "Some note")
		require.NoError(t, err, "failed to create initial item")

		// Remove the budget item from stub
		bpReaderStub.Reset()

		_, err = service.ResetWeekItemToBudgetPlanItem(ctx, createdItem.Id)

		require.ErrorIs(t, err, ErrBudgetItemNotFound)
	})
}

func TestServiceImpl_ResetWeekItemsToBudgetPlan(t *testing.T) {
	t.Run("deletes items for future weeks", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// Create items for a future week
		futureDate := time.Now().AddDate(0, 0, 14) // 2 weeks in future

		// Set up budget plan
		plan := budget_plan.BudgetPlan{
			Id:        1,
			Name:      "My Plan",
			IsCurrent: true,
			Items: []budget_plan.BudgetItem{
				{
					Id:                101,
					PlanId:            1,
					Name:              "Work",
					WeeklyDuration:    40 * time.Hour,
					WeeklyOccurrences: 5,
				},
			},
		}
		bpReaderStub.SetCurrentPlan(plan)
		bpReaderStub.SetPlan(plan)

		// Create the item by modifying the original budget item data
		modifiedItem, err := service.UpdateItem(ctx, futureDate, 0, 101, 35*time.Hour, "Custom notes")
		require.NoError(t, err, "failed to modify item")
		require.Equal(t, 35*time.Hour, modifiedItem.WeeklyDuration, "setup failed: item not modified correctly")
		require.Equal(t, "Custom notes", modifiedItem.Notes, "setup failed: item not modified correctly")

		items, err := service.ResetWeekItemsToBudgetPlan(ctx, futureDate)

		require.NoError(t, err)
		// Should return fresh items from budget plan
		require.Len(t, items, 1)
		assert.Equal(t, 40*time.Hour, items[0].WeeklyDuration)
		assert.Equal(t, 0, items[0].Id)
	})

	t.Run("resets duration and notes for current week", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		currentDate := time.Now()

		// Set up budget plan
		plan := budget_plan.BudgetPlan{
			Id:        1,
			Name:      "My Plan",
			IsCurrent: true,
			Items: []budget_plan.BudgetItem{
				{
					Id:                101,
					PlanId:            1,
					Name:              "Work",
					WeeklyDuration:    40 * time.Hour,
					WeeklyOccurrences: 5,
					Position:          0,
				},
				{
					Id:                102,
					PlanId:            1,
					Name:              "Sport",
					WeeklyDuration:    6 * time.Hour,
					WeeklyOccurrences: 3,
					Position:          1,
				},
			},
		}
		bpReaderStub.SetCurrentPlan(plan)
		bpReaderStub.SetPlan(plan)

		// Create and modify item through service
		createdItem, err := service.UpdateItem(ctx, currentDate, 0, 102, 7*time.Hour, "One hour more for sports")
		require.NoError(t, err, "failed to create initial item")

		items, err := service.ResetWeekItemsToBudgetPlan(ctx, currentDate)

		require.NoError(t, err)
		require.Len(t, items, 2)
		assert.Equal(t, 6*time.Hour, items[1].WeeklyDuration)
		assert.Empty(t, items[1].Notes)
		assert.Equal(t, createdItem.Id, items[1].Id)
	})

	t.Run("resets duration and notes for past weeks", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		pastDate := time.Now().AddDate(0, 0, -7) // 1 week in past

		// Set up budget plan
		plan := budget_plan.BudgetPlan{
			Id:        1,
			Name:      "My Plan",
			IsCurrent: true,
			Items: []budget_plan.BudgetItem{
				{
					Id:                101,
					PlanId:            1,
					Name:              "Work",
					WeeklyDuration:    40 * time.Hour,
					WeeklyOccurrences: 5,
				},
			},
		}
		bpReaderStub.SetCurrentPlan(plan)
		bpReaderStub.SetPlan(plan)

		// Create and modify item through service
		_, err := service.UpdateItem(ctx, pastDate, 0, 101, 35*time.Hour, "Custom notes")
		require.NoError(t, err, "failed to create initial item")

		items, err := service.ResetWeekItemsToBudgetPlan(ctx, pastDate)

		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, 40*time.Hour, items[0].WeeklyDuration)
		assert.Empty(t, items[0].Notes)
	})
}

func TestServiceImpl_UpdateItem(t *testing.T) {
	t.Run("updates existing weekly item", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		weekDate := time.Date(2025, 1, 13, 0, 0, 0, 0, time.UTC)

		// Set up budget plan
		plan := budget_plan.BudgetPlan{
			Id:        1,
			Name:      "My Plan",
			IsCurrent: true,
			Items: []budget_plan.BudgetItem{
				{
					Id:                101,
					PlanId:            1,
					Name:              "Work",
					WeeklyDuration:    40 * time.Hour,
					WeeklyOccurrences: 5,
				},
			},
		}
		bpReaderStub.SetCurrentPlan(plan)
		bpReaderStub.SetPlan(plan)

		// Update the item
		updatedItem, err := service.UpdateItem(ctx, weekDate, 0, 101, 45*time.Hour, "New notes")

		require.NoError(t, err)
		assert.Equal(t, 45*time.Hour, updatedItem.WeeklyDuration)
		assert.Equal(t, "New notes", updatedItem.Notes)
	})

	t.Run("creates items from budget plan when id is 0 and no items exist", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		weekDate := time.Date(2025, 1, 13, 0, 0, 0, 0, time.UTC)

		// Set up budget plan
		plan := budget_plan.BudgetPlan{
			Id:        1,
			Name:      "My Plan",
			IsCurrent: true,
			Items: []budget_plan.BudgetItem{
				{
					Id:                101,
					PlanId:            1,
					Name:              "Work",
					WeeklyDuration:    40 * time.Hour,
					WeeklyOccurrences: 5,
					Icon:              "💼",
					Color:             "#FF5733",
					Position:          0,
				},
				{
					Id:                102,
					PlanId:            1,
					Name:              "Exercise",
					WeeklyDuration:    5 * time.Hour,
					WeeklyOccurrences: 3,
					Icon:              "🏃",
					Color:             "#33FF57",
					Position:          1,
				},
			},
		}
		bpReaderStub.SetCurrentPlan(plan)
		bpReaderStub.SetPlan(plan)

		// Try to update item with id=0 (create mode)
		updatedItem, err := service.UpdateItem(ctx, weekDate, 0, 101, 45*time.Hour, "New notes")

		require.NoError(t, err)
		assert.Equal(t, 101, updatedItem.BudgetItemId)
		assert.Equal(t, "Work", updatedItem.Name)
		assert.Equal(t, 45*time.Hour, updatedItem.WeeklyDuration)
		assert.Equal(t, "New notes", updatedItem.Notes)
		assert.Equal(t, 0, updatedItem.Position)

		// Verify all items were created by querying through service
		items, err := service.GetItemsForWeek(ctx, weekDate)
		require.NoError(t, err, "failed to get items")
		assert.Len(t, items, 2)
	})

	t.Run("returns error when id is 0 but items already exist", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		weekDate := time.Date(2025, 1, 13, 0, 0, 0, 0, time.UTC)

		// Set up budget plan
		plan := budget_plan.BudgetPlan{
			Id:        1,
			Name:      "My Plan",
			IsCurrent: true,
			Items: []budget_plan.BudgetItem{
				{
					Id:                101,
					PlanId:            1,
					Name:              "Work",
					WeeklyDuration:    40 * time.Hour,
					WeeklyOccurrences: 5,
				},
			},
		}
		bpReaderStub.SetCurrentPlan(plan)
		bpReaderStub.SetPlan(plan)

		// Create items first
		_, err := service.UpdateItem(ctx, weekDate, 0, 101, 35*time.Hour, "Something")
		require.NoError(t, err, "failed to create initial item")

		// Try to create again with id=0
		_, err = service.UpdateItem(ctx, weekDate, 0, 101, 45*time.Hour, "New notes")

		require.ErrorIs(t, err, ErrWeeklyItemAlreadyExists)
	})

	t.Run("returns error when budget item not found", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		weekDate := time.Date(2025, 1, 13, 0, 0, 0, 0, time.UTC)

		_, err := service.UpdateItem(ctx, weekDate, 0, 999, 45*time.Hour, "New notes")

		require.ErrorIs(t, err, ErrBudgetItemNotFound)
	})
}

func TestServiceImpl_createItemsFromBudgetPlan(t *testing.T) {
	t.Run("creates items from budget plan", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		weekNumber := WeekNumber{Year: 2025, Week: 3}

		// Set up budget plan
		plan := budget_plan.BudgetPlan{
			Id:        1,
			Name:      "My Plan",
			IsCurrent: true,
			Items: []budget_plan.BudgetItem{
				{
					Id:                101,
					PlanId:            1,
					Name:              "Work",
					WeeklyDuration:    40 * time.Hour,
					WeeklyOccurrences: 5,
					Icon:              "💼",
					Color:             "#FF5733",
					Position:          0,
				},
				{
					Id:                102,
					PlanId:            1,
					Name:              "Exercise",
					WeeklyDuration:    5 * time.Hour,
					WeeklyOccurrences: 3,
					Icon:              "🏃",
					Color:             "#33FF57",
					Position:          1,
				},
			},
		}
		bpReaderStub.SetPlan(plan)

		items, err := service.(*ServiceImpl).createItemsFromBudgetPlan(ctx, 1, weekNumber)

		require.NoError(t, err)
		require.Len(t, items, 2)
		assert.Equal(t, "Work", items[0].Name)
		assert.Equal(t, weekNumber, items[0].WeekNumber)
		assert.NotEqual(t, 0, items[0].Id)
		assert.Equal(t, 0, items[0].Position)
		assert.Equal(t, "Exercise", items[1].Name)
		assert.Equal(t, weekNumber, items[1].WeekNumber)
		assert.NotEqual(t, 0, items[1].Id)
		assert.Equal(t, 1, items[1].Position)
	})

	t.Run("returns error when budget plan not found", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		weekNumber := WeekNumber{Year: 2025, Week: 3}

		_, err := service.(*ServiceImpl).createItemsFromBudgetPlan(ctx, 999, weekNumber)

		require.Error(t, err)
	})
}

func TestServiceImpl_handleBudgetPlanItemUpdated(t *testing.T) {
	t.Run("updates all weekly items' name, color and icon when original budget item is updated", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		weekDate1 := time.Date(2025, 1, 13, 0, 0, 0, 0, time.UTC)
		weekDate2 := time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)

		// Set up budget plan with multiple items
		plan := budget_plan.BudgetPlan{
			Id:        1,
			Name:      "My Plan",
			IsCurrent: true,
			Items: []budget_plan.BudgetItem{
				{
					Id:                101,
					PlanId:            1,
					Name:              "Old Work",
					WeeklyDuration:    40 * time.Hour,
					WeeklyOccurrences: 5,
					Icon:              "📝",
					Color:             "#000000",
					Position:          0,
				},
				{
					Id:                102,
					PlanId:            1,
					Name:              "Exercise",
					WeeklyDuration:    5 * time.Hour,
					WeeklyOccurrences: 3,
					Icon:              "🏃",
					Color:             "#33FF57",
					Position:          1,
				},
			},
		}
		bpReaderStub.SetCurrentPlan(plan)
		bpReaderStub.SetPlan(plan)

		// Create weekly items for two different weeks
		_, err := service.UpdateItem(ctx, weekDate1, 0, 101, 35*time.Hour, "custom note 1")
		require.NoError(t, err, "failed to create items for week 1")

		_, err = service.UpdateItem(ctx, weekDate2, 0, 101, 42*time.Hour, "custom note 2")
		require.NoError(t, err, "failed to create items for week 2")

		// Simulate budget plan item update
		updatedBudgetItem := event_bus.BudgetPlanItemUpdated{
			Id:    101,
			Name:  "Updated Work",
			Icon:  "💼",
			Color: "#FF5733",
		}

		err = eventBus.Publish(event_bus.NewEvent(ctx, "budget_plan.item.updated", updatedBudgetItem))
		require.NoError(t, err)

		// Verify items were updated by retrieving them through service
		items1, err := service.GetItemsForWeek(ctx, weekDate1)
		require.NoError(t, err, "failed to get week 1 items")
		require.Equal(t, len(items1), 2)

		workItemWeek1 := items1[0]
		assert.Equal(t, "Updated Work", workItemWeek1.Name, "Name should be changed")
		assert.Equal(t, "💼", workItemWeek1.Icon, "Icon should be changed")
		assert.Equal(t, "#FF5733", workItemWeek1.Color, "Color should be changed")
		assert.Equal(t, "custom note 1", workItemWeek1.Notes, "Note should not be changed")
		assert.Equal(t, 35*time.Hour, workItemWeek1.WeeklyDuration, "Weekly duration should not be changed")

		// Verify items were updated by retrieving them through service
		items2, err := service.GetItemsForWeek(ctx, weekDate2)
		require.NoError(t, err, "failed to get week 2 items")
		require.Equal(t, len(items2), 2)

		workItemWeek2 := items2[0]
		assert.Equal(t, "Updated Work", workItemWeek2.Name, "Name should be changed")
		assert.Equal(t, "💼", workItemWeek2.Icon, "Icon should be changed")
		assert.Equal(t, "#FF5733", workItemWeek2.Color, "Color should be changed")
		assert.Equal(t, "custom note 2", workItemWeek2.Notes, "Note should not be changed")
		assert.Equal(t, 42*time.Hour, workItemWeek2.WeeklyDuration, "Weekly duration should not be changed")

		// Verify item with different budget item was not updated
		exerciseItem := items1[1]
		if exerciseItem.BudgetItemId == 102 {
			assert.Equal(t, "Exercise", exerciseItem.Name)
		}
	})

	t.Run("returns 0 when no items match", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		updatedBudgetItem := event_bus.BudgetPlanItemUpdated{
			Id:    999,
			Name:  "Updated Work",
			Icon:  "💼",
			Color: "#FF5733",
		}

		count, err := service.(*ServiceImpl).handleBudgetPlanItemUpdated(ctx, updatedBudgetItem)

		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestServiceImpl_handleCalendarEventChanged(t *testing.T) {
	t.Run("creates weekly items", func(t *testing.T) {
		teardown := setup(t)
		defer teardown()

		// given
		location, err := time.LoadLocation("Europe/Warsaw")
		require.NoError(t, err)
		nextWeekMonday := time.Date(2025, 1, 17, 0, 0, 0, 0, location)
		calendarEvent := event_bus.CalendarEventCreated{
			UID:          uuid.NewString(),
			Summary:      "Calendar event 1",
			StartTime:    nextWeekMonday,
			EndTime:      nextWeekMonday.Add(1 * time.Hour),
			BudgetItemId: 122,
		}
		item1 := budget_plan.BudgetItem{
			Id:                122,
			PlanId:            3,
			Name:              "Plan item 122",
			WeeklyDuration:    30 * time.Hour,
			WeeklyOccurrences: 5,
			Icon:              "some-icon",
			Color:             "#FF5733",
			Position:          4,
		}
		item2 := budget_plan.BudgetItem{
			Id:                128,
			PlanId:            3,
			Name:              "Plan item 128",
			WeeklyDuration:    13 * time.Hour,
			WeeklyOccurrences: 2,
			Icon:              "some-icon",
			Color:             "#FF5733",
			Position:          5,
		}
		plan := budget_plan.BudgetPlan{
			Id:        3,
			Name:      "Test Plan 3",
			IsCurrent: true,
			Items:     []budget_plan.BudgetItem{item1, item2},
		}
		bpReaderStub.SetPlan(plan)
		bpReaderStub.SetItem(item1)
		bpReaderStub.SetItem(item2)
		bpReaderStub.SetCurrentPlan(plan)

		// when
		eventBus.Publish(event_bus.NewEvent(ctx, "calendar.event.created", calendarEvent))

		// then
		items, err := service.GetItemsForWeek(ctx, nextWeekMonday)
		require.NoError(t, err)
		require.Len(t, items, 2)
		assert.Equal(t, calendarEvent.BudgetItemId, items[0].BudgetItemId)
		assert.Equal(t, item2.Id, items[1].BudgetItemId)
	})
}
