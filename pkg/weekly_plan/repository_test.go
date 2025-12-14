package weekly_plan

import (
	"context"
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
}

func TestRepositoryImpl_DeleteWeekItems(t *testing.T) {

}

func TestRepositoryImpl_GetItem(t *testing.T) {

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

}

func TestRepositoryImpl_UpdateItem(t *testing.T) {

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
