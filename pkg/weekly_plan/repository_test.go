package weekly_plan

import (
	"context"
	"errors"
	"math/rand"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/klokku/klokku/internal/test_utils"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var pgContainer *postgres.PostgresContainer
var openDb func() *pgx.Conn

func TestMain(m *testing.M) {
	pgContainer, openDb = test_utils.TestWithDB()
	defer func() {
		if err := testcontainers.TerminateContainer(pgContainer); err != nil {
			log.Errorf("failed to terminate container: %s", err)
		}
	}()
	code := m.Run()
	os.Exit(code)
}

func setupTestRepository(t *testing.T) (context.Context, Repository, int) {
	ctx := context.Background()
	db := openDb()
	repository := NewRepo(db)
	t.Cleanup(func() {
		db.Close(ctx)
		err := pgContainer.Restore(ctx)
		require.NoError(t, err)
	})
	userId := 1
	return ctx, repository, userId
}

func TestRepositoryImpl_CreateItems(t *testing.T) {
	t.Run("should create items for a week", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		var items []WeeklyPlanItem
		for i := 1; i <= 10; i++ {
			items = append(items, weeklyItem(WeeklyPlanItem{BudgetItemId: i}))
		}

		// when
		createdItems, err := repo.createItems(ctx, userId, items)

		// then
		require.NoError(t, err)
		require.Len(t, createdItems, 10)
		for i, createdItem := range createdItems {
			require.NotEmpty(t, createdItem.Id)
			assertWeeklyPlanItemsEqual(t, items[i], createdItem)
		}
	})

	t.Run("should return nil when creating empty list of items", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		var items []WeeklyPlanItem

		// when
		createdItems, err := repo.createItems(ctx, userId, items)

		// then
		require.NoError(t, err)
		require.Nil(t, createdItems)
	})
}

func TestRepositoryImpl_DeleteWeekItems(t *testing.T) {
	t.Run("should delete items for a specific week", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		currentWeek := WeekNumberFromDate(time.Now(), time.Monday)
		nextWeek := WeekNumber{Year: currentWeek.Year, Week: currentWeek.Week + 1}

		var currentWeekItems []WeeklyPlanItem
		for i := 1; i <= 5; i++ {
			currentWeekItems = append(currentWeekItems, weeklyItem(WeeklyPlanItem{
				BudgetItemId: i,
				WeekNumber:   currentWeek,
			}))
		}

		var nextWeekItems []WeeklyPlanItem
		for i := 6; i <= 10; i++ {
			nextWeekItems = append(nextWeekItems, weeklyItem(WeeklyPlanItem{
				BudgetItemId: i,
				WeekNumber:   nextWeek,
			}))
		}

		allItems := append(currentWeekItems, nextWeekItems...)
		_, err := repo.createItems(ctx, userId, allItems)
		require.NoError(t, err)

		// when
		deletedCount, err := repo.DeleteWeekItems(ctx, userId, currentWeek)

		// then
		require.NoError(t, err)
		require.Equal(t, 5, deletedCount)

		// verify current week items are deleted
		remainingCurrentWeek, err := repo.GetItemsForWeek(ctx, userId, currentWeek)
		require.NoError(t, err)
		require.Len(t, remainingCurrentWeek, 0)

		// verify next week items still exist
		remainingNextWeek, err := repo.GetItemsForWeek(ctx, userId, nextWeek)
		require.NoError(t, err)
		require.Len(t, remainingNextWeek, 5)
	})

	t.Run("should return 0 when no items exist for the week", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		emptyWeek := WeekNumber{Year: 2025, Week: 50}

		// when
		deletedCount, err := repo.DeleteWeekItems(ctx, userId, emptyWeek)

		// then
		require.NoError(t, err)
		require.Equal(t, 0, deletedCount)
	})
}

func TestRepositoryImpl_GetItem(t *testing.T) {
	t.Run("should return a single item by id", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		items := []WeeklyPlanItem{
			weeklyItem(WeeklyPlanItem{BudgetItemId: 1}),
			weeklyItem(WeeklyPlanItem{BudgetItemId: 2}),
		}
		createdItems, err := repo.createItems(ctx, userId, items)
		require.NoError(t, err)
		require.Len(t, createdItems, 2)

		// when
		item, err := repo.GetItem(ctx, userId, createdItems[0].Id)

		// then
		require.NoError(t, err)
		assertWeeklyPlanItemsEqual(t, createdItems[0], item)
	})

	t.Run("should return error when item does not exist", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		nonExistentId := 99999

		// when
		_, err := repo.GetItem(ctx, userId, nonExistentId)

		// then
		require.Error(t, err)
	})

	t.Run("should not return items belonging to other users", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		items := []WeeklyPlanItem{weeklyItem(WeeklyPlanItem{BudgetItemId: 1})}
		createdItems, err := repo.createItems(ctx, userId, items)
		require.NoError(t, err)

		// when
		differentUserId := userId + 1
		_, err = repo.GetItem(ctx, differentUserId, createdItems[0].Id)

		// then
		require.Error(t, err)
	})
}

func TestRepositoryImpl_GetItemsForWeek(t *testing.T) {
	t.Run("should return items for a week", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		var items []WeeklyPlanItem
		currentWeek := WeekNumberFromDate(time.Now(), time.Monday)
		for i := 1; i <= 10; i++ {
			items = append(items, weeklyItem(
				WeeklyPlanItem{
					BudgetItemId: i,
					WeekNumber:   currentWeek,
					Position:     100 - i, // Position in the reverse to test that GetItemsForWeek returns items in the correct order
				}))
		}
		differentWeekItem := weeklyItem(
			WeeklyPlanItem{
				BudgetItemId: 1,
				WeekNumber:   WeekNumber{Year: currentWeek.Year, Week: currentWeek.Week + 1},
				Position:     110,
			})
		items = append(items, differentWeekItem)

		// when
		_, err := repo.createItems(ctx, userId, items)
		require.NoError(t, err)
		weeklyItems, err := repo.GetItemsForWeek(ctx, userId, currentWeek)

		// then
		require.NoError(t, err)
		require.Len(t, weeklyItems, 10)
		// sort created items by position
		sort.Slice(items, func(i, j int) bool {
			return items[i].Position < items[j].Position
		})
		for i, weeklyItem := range weeklyItems {
			assertWeeklyPlanItemsEqual(t, items[i], weeklyItem)
		}
		require.NotContains(t, weeklyItems, differentWeekItem)
	})
}

func TestRepositoryImpl_UpdateAllItemsByBudgetItemId(t *testing.T) {
	t.Run("should update name, icon, and color for all items with the same budget item id", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		budgetItemId := 42
		currentWeek := WeekNumberFromDate(time.Now(), time.Monday)

		var items []WeeklyPlanItem
		for i := 0; i < 3; i++ {
			items = append(items, weeklyItem(WeeklyPlanItem{
				BudgetItemId: budgetItemId,
				WeekNumber:   WeekNumber{Year: currentWeek.Year, Week: currentWeek.Week + i},
				Name:         "Old Name",
				Icon:         "old-icon",
				Color:        "old-color",
			}))
		}

		// Add an item with a different budget item id
		items = append(items, weeklyItem(WeeklyPlanItem{
			BudgetItemId: 99,
			WeekNumber:   WeekNumber{Year: currentWeek.Year, Week: currentWeek.Week + 10},
			Name:         "Different Item",
			Icon:         "different-icon",
			Color:        "different-color",
		}))

		createdItems, err := repo.createItems(ctx, userId, items)
		require.NoError(t, err)

		// when
		newName := "Updated Name"
		newIcon := "updated-icon"
		newColor := "updated-color"
		updatedCount, err := repo.UpdateAllItemsByBudgetItemId(ctx, userId, budgetItemId, newName, newIcon, newColor)

		// then
		require.NoError(t, err)
		require.Equal(t, 3, updatedCount)

		// verify the updates
		for i := 0; i < 3; i++ {
			item, err := repo.GetItem(ctx, userId, createdItems[i].Id)
			require.NoError(t, err)
			require.Equal(t, newName, item.Name)
			require.Equal(t, newIcon, item.Icon)
			require.Equal(t, newColor, item.Color)
		}

		// verify the item with different budget id was not updated
		differentItem, err := repo.GetItem(ctx, userId, createdItems[3].Id)
		require.NoError(t, err)
		require.Equal(t, "Different Item", differentItem.Name)
		require.Equal(t, "different-icon", differentItem.Icon)
		require.Equal(t, "different-color", differentItem.Color)
	})

	t.Run("should return 0 when no items match the budget item id", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		nonExistentBudgetItemId := 99999

		// when
		updatedCount, err := repo.UpdateAllItemsByBudgetItemId(ctx, userId, nonExistentBudgetItemId, "name", "icon", "color")

		// then
		require.NoError(t, err)
		require.Equal(t, 0, updatedCount)
	})

	t.Run("should only update items for the specified user", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		budgetItemId := 42

		items := []WeeklyPlanItem{
			weeklyItem(WeeklyPlanItem{BudgetItemId: budgetItemId}),
		}

		createdItems, err := repo.createItems(ctx, userId, items)
		require.NoError(t, err)

		// when - try to update with different user id
		differentUserId := userId + 1
		updatedCount, err := repo.UpdateAllItemsByBudgetItemId(ctx, differentUserId, budgetItemId, "new name", "new icon", "new color")

		// then
		require.NoError(t, err)
		require.Equal(t, 0, updatedCount)

		// verify original item was not updated
		item, err := repo.GetItem(ctx, userId, createdItems[0].Id)
		require.NoError(t, err)
		require.NotEqual(t, "new name", item.Name)
	})
}

func TestRepositoryImpl_WithTransaction(t *testing.T) {
	t.Run("should commit transaction on success", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		item := weeklyItem(WeeklyPlanItem{BudgetItemId: 1})

		// when
		err := repo.WithTransaction(ctx, func(txRepo Repository) error {
			_, err := txRepo.createItems(ctx, userId, []WeeklyPlanItem{item})
			return err
		})

		// then
		require.NoError(t, err)

		// verify item was created
		items, err := repo.GetItemsForWeek(ctx, userId, item.WeekNumber)
		require.NoError(t, err)
		require.Len(t, items, 1)
	})

	t.Run("should rollback transaction on error", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		item := weeklyItem(WeeklyPlanItem{BudgetItemId: 1})

		// when
		err := repo.WithTransaction(ctx, func(txRepo Repository) error {
			_, err := txRepo.createItems(ctx, userId, []WeeklyPlanItem{item})
			if err != nil {
				return err
			}
			return errors.New("intentional error to trigger rollback")
		})

		// then
		require.Error(t, err)
		require.Equal(t, "intentional error to trigger rollback", err.Error())

		// verify item was not created
		items, err := repo.GetItemsForWeek(ctx, userId, item.WeekNumber)
		require.NoError(t, err)
		require.Len(t, items, 0)
	})

	t.Run("should support multiple operations in transaction", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		currentWeek := WeekNumberFromDate(time.Now(), time.Monday)

		item1 := weeklyItem(WeeklyPlanItem{BudgetItemId: 1, WeekNumber: currentWeek})
		item2 := weeklyItem(WeeklyPlanItem{BudgetItemId: 2, WeekNumber: currentWeek})

		var firstItemId int

		// when
		err := repo.WithTransaction(ctx, func(txRepo Repository) error {
			// Create items
			created, err := txRepo.createItems(ctx, userId, []WeeklyPlanItem{item1, item2})
			if err != nil {
				return err
			}

			firstItemId = created[0].Id

			// Update one of them
			_, err = txRepo.UpdateItem(ctx, userId, created[0].Id, 10*time.Hour, "Updated in transaction")
			if err != nil {
				return err
			}

			return nil
		})

		// then
		require.NoError(t, err)

		// verify both operations succeeded
		items, err := repo.GetItemsForWeek(ctx, userId, currentWeek)
		require.NoError(t, err)
		require.Len(t, items, 2)

		// verify the update by getting the specific item
		updatedItem, err := repo.GetItem(ctx, userId, firstItemId)
		require.NoError(t, err)
		require.Equal(t, "Updated in transaction", updatedItem.Notes)
	})
}

func TestRepositoryImpl_UpdateItem(t *testing.T) {
	t.Run("should update weekly duration and notes", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		originalItem := weeklyItem(WeeklyPlanItem{
			WeeklyDuration: 2 * time.Hour,
			Notes:          "Original notes",
		})

		createdItems, err := repo.createItems(ctx, userId, []WeeklyPlanItem{originalItem})
		require.NoError(t, err)
		createdItem := createdItems[0]

		// when
		newDuration := 5 * time.Hour
		newNotes := "Updated notes"
		updatedItem, err := repo.UpdateItem(ctx, userId, createdItem.Id, newDuration, newNotes)

		// then
		require.NoError(t, err)
		require.Equal(t, createdItem.Id, updatedItem.Id)
		require.Equal(t, newDuration, updatedItem.WeeklyDuration)
		require.Equal(t, newNotes, updatedItem.Notes)

		// verify other fields remain unchanged
		require.Equal(t, createdItem.BudgetItemId, updatedItem.BudgetItemId)
		require.Equal(t, createdItem.WeekNumber, updatedItem.WeekNumber)
		require.Equal(t, createdItem.Name, updatedItem.Name)
		require.Equal(t, createdItem.Icon, updatedItem.Icon)
		require.Equal(t, createdItem.Color, updatedItem.Color)
	})

	t.Run("should return ErrWeeklyPlanItemNotFound when item does not exist", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		nonExistentId := 99999

		// when
		_, err := repo.UpdateItem(ctx, userId, nonExistentId, 1*time.Hour, "notes")

		// then
		require.ErrorIs(t, err, ErrWeeklyPlanItemNotFound)
	})

	t.Run("should not update items belonging to other users", func(t *testing.T) {
		// given
		ctx, repo, userId := setupTestRepository(t)
		item := weeklyItem(WeeklyPlanItem{Notes: "Original"})
		createdItems, err := repo.createItems(ctx, userId, []WeeklyPlanItem{item})
		require.NoError(t, err)

		// when
		differentUserId := userId + 1
		_, err = repo.UpdateItem(ctx, differentUserId, createdItems[0].Id, 3*time.Hour, "Should not update")

		// then
		require.ErrorIs(t, err, ErrWeeklyPlanItemNotFound)

		// verify original item was not updated
		originalItem, err := repo.GetItem(ctx, userId, createdItems[0].Id)
		require.NoError(t, err)
		require.Equal(t, "Original", originalItem.Notes)
	})
}

func weeklyItem(itemPartial WeeklyPlanItem) WeeklyPlanItem {
	budgetItemId := 0
	if itemPartial.BudgetItemId != 0 {
		budgetItemId = itemPartial.BudgetItemId
	}

	date := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	date.Add(time.Duration(rand.Intn(365)) * 24 * time.Hour)
	weekNumber := WeekNumberFromDate(date, time.Monday)
	if itemPartial.WeekNumber.Year != 0 {
		weekNumber = itemPartial.WeekNumber
	}

	name := "Test week item " + uuid.NewString()
	if itemPartial.Name != "" {
		name = itemPartial.Name
	}

	weeklyDuration := time.Duration(rand.Intn(30)+1) * time.Hour
	if itemPartial.WeeklyDuration != 0 {
		weeklyDuration = itemPartial.WeeklyDuration
	}

	weeklyOccurrences := rand.Intn(7) + 1
	if itemPartial.WeeklyOccurrences != 0 {
		weeklyOccurrences = itemPartial.WeeklyOccurrences
	}

	icon := "test-icon-" + (uuid.NewString())[0:4]
	if itemPartial.Icon != "" {
		icon = itemPartial.Icon
	}

	color := "test-color " + (uuid.NewString())[0:4]
	if itemPartial.Color != "" {
		color = itemPartial.Color
	}

	notes := "Some notes " + (uuid.NewString())[0:4]
	if itemPartial.Notes != "" {
		notes = itemPartial.Notes
	}

	position := rand.Intn(30)
	if itemPartial.Position != 0 {
		position = itemPartial.Position
	}

	return WeeklyPlanItem{
		BudgetItemId:      budgetItemId,
		WeekNumber:        weekNumber,
		Name:              name,
		WeeklyDuration:    weeklyDuration,
		WeeklyOccurrences: weeklyOccurrences,
		Icon:              icon,
		Color:             color,
		Notes:             notes,
		Position:          position,
	}
}

func assertWeeklyPlanItemsEqual(t *testing.T, expected WeeklyPlanItem, actual WeeklyPlanItem) {
	require.Equal(t, expected.BudgetItemId, actual.BudgetItemId)
	require.Equal(t, expected.WeekNumber, actual.WeekNumber)
	require.Equal(t, expected.Name, actual.Name)
	require.Equal(t, expected.WeeklyDuration, actual.WeeklyDuration)
	require.Equal(t, expected.WeeklyOccurrences, actual.WeeklyOccurrences)
	require.Equal(t, expected.Icon, actual.Icon)
	require.Equal(t, expected.Color, actual.Color)
	require.Equal(t, expected.Notes, actual.Notes)
	require.Equal(t, expected.Position, actual.Position)
}
