package dashboard

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klokku/klokku/internal/test_utils"
	"github.com/klokku/klokku/pkg/user"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type repositoryTestFixture struct {
	ctx    context.Context
	repo   Repository
	users  *user.UserRepoImpl
	userId int
}

func setupTestRepository(t *testing.T, container *postgres.PostgresContainer, openDb func() *pgxpool.Pool) repositoryTestFixture {
	t.Helper()
	ctx := context.Background()
	db := openDb()
	t.Cleanup(func() {
		db.Close()
		require.NoError(t, container.Restore(ctx))
	})
	users := user.NewUserRepo(db)
	userId, err := users.CreateUser(ctx, user.User{
		Uid: "dashboard-a", Username: "dashboard-a", DisplayName: "A",
		Settings: user.Settings{Timezone: "UTC"},
	})
	require.NoError(t, err)
	return repositoryTestFixture{ctx: ctx, repo: NewRepository(db), users: users, userId: userId}
}

func TestRepositoryImpl(t *testing.T) {
	// Scope the shared container to repository tests so service-only runs do not need Docker.
	container, openDb := test_utils.TestWithDB()
	t.Cleanup(func() { require.NoError(t, testcontainers.TerminateContainer(container)) })

	t.Run("Find returns defaults for an uncustomized profile", func(t *testing.T) {
		fixture := setupTestRepository(t, container, openDb)

		configuration, err := fixture.repo.Find(fixture.ctx, fixture.userId)

		require.NoError(t, err)
		require.Nil(t, configuration)
	})

	t.Run("Store preserves card order and visibility across repository instances", func(t *testing.T) {
		fixture := setupTestRepository(t, container, openDb)
		want := &Configuration{Cards: []Card{{Id: "future-card", Visible: false}, {Id: "time-today", Visible: true}}}

		require.NoError(t, fixture.repo.Store(fixture.ctx, fixture.userId, want))

		db := openDb()
		t.Cleanup(db.Close)
		configuration, err := NewRepository(db).Find(fixture.ctx, fixture.userId)
		require.NoError(t, err)
		require.Equal(t, want, configuration)
	})

	t.Run("Store replaces existing preferences", func(t *testing.T) {
		fixture := setupTestRepository(t, container, openDb)
		require.NoError(t, fixture.repo.Store(fixture.ctx, fixture.userId, &Configuration{Cards: []Card{{Id: "time-today", Visible: true}}}))
		want := &Configuration{Cards: []Card{{Id: "budget-balance", Visible: true}, {Id: "time-today", Visible: false}}}

		require.NoError(t, fixture.repo.Store(fixture.ctx, fixture.userId, want))

		configuration, err := fixture.repo.Find(fixture.ctx, fixture.userId)
		require.NoError(t, err)
		require.Equal(t, want, configuration)
	})

	t.Run("Store keeps profiles isolated", func(t *testing.T) {
		fixture := setupTestRepository(t, container, openDb)
		secondId, err := fixture.users.CreateUser(fixture.ctx, user.User{
			Uid: "dashboard-b", Username: "dashboard-b", DisplayName: "B",
			Settings: user.Settings{Timezone: "UTC"},
		})
		require.NoError(t, err)
		first := &Configuration{Cards: []Card{{Id: "time-today", Visible: false}}}
		second := &Configuration{Cards: []Card{{Id: "budget-balance", Visible: true}}}

		require.NoError(t, fixture.repo.Store(fixture.ctx, fixture.userId, first))
		configuration, err := fixture.repo.Find(fixture.ctx, secondId)
		require.NoError(t, err)
		require.Nil(t, configuration)
		require.NoError(t, fixture.repo.Store(fixture.ctx, secondId, second))

		configuration, err = fixture.repo.Find(fixture.ctx, fixture.userId)
		require.NoError(t, err)
		require.Equal(t, first, configuration)
		configuration, err = fixture.repo.Find(fixture.ctx, secondId)
		require.NoError(t, err)
		require.Equal(t, second, configuration)
	})

	t.Run("Profile updates preserve dashboard preferences", func(t *testing.T) {
		fixture := setupTestRepository(t, container, openDb)
		want := &Configuration{Cards: []Card{{Id: "time-today", Visible: true}}}
		require.NoError(t, fixture.repo.Store(fixture.ctx, fixture.userId, want))
		profile, err := fixture.users.GetUser(fixture.ctx, fixture.userId)
		require.NoError(t, err)
		profile.DisplayName = "Changed name"
		profile.Settings.IgnoreShortEvents = true

		_, err = fixture.users.UpdateUser(fixture.ctx, fixture.userId, profile)

		require.NoError(t, err)
		configuration, err := fixture.repo.Find(fixture.ctx, fixture.userId)
		require.NoError(t, err)
		require.Equal(t, want, configuration)
	})

	t.Run("Store nil resets preferences without changing the profile", func(t *testing.T) {
		fixture := setupTestRepository(t, container, openDb)
		profile, err := fixture.users.GetUser(fixture.ctx, fixture.userId)
		require.NoError(t, err)
		require.NoError(t, fixture.repo.Store(fixture.ctx, fixture.userId, &Configuration{Cards: []Card{{Id: "time-today", Visible: false}}}))
		afterSave, err := fixture.users.GetUser(fixture.ctx, fixture.userId)
		require.NoError(t, err)
		require.Equal(t, profile, afterSave)

		require.NoError(t, fixture.repo.Store(fixture.ctx, fixture.userId, nil))

		configuration, err := fixture.repo.Find(fixture.ctx, fixture.userId)
		require.NoError(t, err)
		require.Nil(t, configuration)
		afterReset, err := fixture.users.GetUser(fixture.ctx, fixture.userId)
		require.NoError(t, err)
		require.Equal(t, profile, afterReset)
	})

	t.Run("Find rejects an unknown profile", func(t *testing.T) {
		fixture := setupTestRepository(t, container, openDb)

		_, err := fixture.repo.Find(fixture.ctx, 999999)

		require.ErrorIs(t, err, user.ErrUserNotFound)
	})

	t.Run("Store rejects an unknown profile", func(t *testing.T) {
		fixture := setupTestRepository(t, container, openDb)
		configuration := &Configuration{Cards: []Card{{Id: "time-today", Visible: true}}}

		err := fixture.repo.Store(fixture.ctx, 999999, configuration)

		require.ErrorIs(t, err, user.ErrUserNotFound)
	})
}
