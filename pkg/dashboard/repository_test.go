package dashboard

import (
	"context"
	"testing"

	"github.com/klokku/klokku/internal/test_utils"
	"github.com/klokku/klokku/pkg/user"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

func TestRepository_ConfigurationSurvivesProfileUpdates(t *testing.T) {
	container, openDb := test_utils.TestWithDB()
	t.Cleanup(func() { require.NoError(t, testcontainers.TerminateContainer(container)) })
	db := openDb()
	t.Cleanup(db.Close)
	ctx := context.Background()
	users := user.NewUserRepo(db)
	firstId, err := users.CreateUser(ctx, user.User{Uid: "dashboard-a", Username: "dashboard-a", DisplayName: "A", Settings: user.Settings{Timezone: "UTC"}})
	require.NoError(t, err)
	secondId, err := users.CreateUser(ctx, user.User{Uid: "dashboard-b", Username: "dashboard-b", DisplayName: "B", Settings: user.Settings{Timezone: "UTC"}})
	require.NoError(t, err)
	repo := NewRepository(db)
	configuration, err := repo.Find(ctx, firstId)
	require.NoError(t, err)
	require.Nil(t, configuration)
	want := &Configuration{Cards: []Card{{Id: "future-card", Visible: false}, {Id: "time-today", Visible: true}}}
	require.NoError(t, repo.Store(ctx, firstId, want))
	// A fresh repository represents a different browser reading the same profile.
	configuration, err = NewRepository(db).Find(ctx, firstId)
	require.NoError(t, err)
	require.Equal(t, want, configuration)
	configuration, err = repo.Find(ctx, secondId)
	require.NoError(t, err)
	require.Nil(t, configuration)
	profile, err := users.GetUser(ctx, firstId)
	require.NoError(t, err)
	profile.DisplayName = "Changed name"
	_, err = users.UpdateUser(ctx, firstId, profile)
	require.NoError(t, err)
	configuration, err = repo.Find(ctx, firstId)
	require.NoError(t, err)
	require.Equal(t, want, configuration)
	require.NoError(t, repo.Store(ctx, firstId, nil))
	configuration, err = repo.Find(ctx, firstId)
	require.NoError(t, err)
	require.Nil(t, configuration)
	profile, err = users.GetUser(ctx, firstId)
	require.NoError(t, err)
	require.Equal(t, "Changed name", profile.DisplayName)
	require.ErrorIs(t, repo.Store(ctx, 999999, want), user.ErrUserNotFound)
}
