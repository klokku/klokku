package database_test

import (
	"context"
	"testing"

	"github.com/klokku/klokku/internal/test_utils"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

func TestEventCalendarTypeMigration(t *testing.T) {
	container, openDatabase := test_utils.TestWithDB()
	t.Cleanup(func() {
		require.NoError(t, testcontainers.TerminateContainer(container))
	})
	database := openDatabase()
	t.Cleanup(database.Close)
	ctx := context.Background()

	t.Run("defaults omitted calendar type to klokku", func(t *testing.T) {
		var calendarType string
		err := database.QueryRow(ctx, `
			INSERT INTO users (uid, username, display_name, timezone, week_first_day)
			VALUES ('default-calendar-user', 'default-calendar-user', 'Default Calendar User', 'UTC', 1)
			RETURNING event_calendar_type
		`).Scan(&calendarType)
		require.NoError(t, err)
		require.Equal(t, "klokku", calendarType)
	})

	t.Run("rejects invalid calendar type", func(t *testing.T) {
		_, err := database.Exec(ctx, `
			INSERT INTO users (uid, username, display_name, timezone, week_first_day, event_calendar_type)
			VALUES ('invalid-calendar-user', 'invalid-calendar-user', 'Invalid Calendar User', 'UTC', 1, 'invalid')
		`)
		require.Error(t, err)
	})
}
