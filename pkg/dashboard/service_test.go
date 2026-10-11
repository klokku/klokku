package dashboard

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/klokku/klokku/pkg/user"
	"github.com/stretchr/testify/require"
)

func TestService_SaveConfigurationCardLimit(t *testing.T) {
	for _, count := range []int{100, 101} {
		t.Run(fmt.Sprintf("%d cards", count), func(t *testing.T) {
			service := NewService(&memoryRepository{configurations: make(map[int]*Configuration)})
			ctx := user.WithUser(context.Background(), user.User{Id: 1})
			configuration := &Configuration{Cards: make([]Card, count)}
			for index := range configuration.Cards {
				configuration.Cards[index] = Card{Id: fmt.Sprintf("card-%d", index), Visible: true}
			}
			err := service.SaveConfiguration(ctx, configuration)
			if count == 100 {
				require.NoError(t, err)
				stored, err := service.FindConfiguration(ctx)
				require.NoError(t, err)
				require.Equal(t, configuration, stored)
			} else {
				require.ErrorIs(t, err, ErrConfigurationInvalid)
			}
		})
	}
}

type failingRepository struct{ err error }

func (r failingRepository) Find(context.Context, int) (*Configuration, error) { return nil, r.err }
func (r failingRepository) Store(context.Context, int, *Configuration) error  { return r.err }

func TestService_RequiresCurrentUserAndPropagatesRepositoryErrors(t *testing.T) {
	repoError := errors.New("database unavailable")
	service := NewService(failingRepository{err: repoError})
	_, err := service.FindConfiguration(context.Background())
	require.Error(t, err)
	require.NotErrorIs(t, err, repoError)
	err = service.SaveConfiguration(context.Background(), nil)
	require.Error(t, err)
	require.NotErrorIs(t, err, repoError)
	ctx := user.WithUser(context.Background(), user.User{Id: 1})
	_, err = service.FindConfiguration(ctx)
	require.ErrorIs(t, err, repoError)
	require.ErrorIs(t, service.SaveConfiguration(ctx, nil), repoError)
}
