package dashboard

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/klokku/klokku/pkg/user"
	"github.com/stretchr/testify/require"
)

func TestServiceImpl_SaveConfigurationValidation(t *testing.T) {
	tests := []struct {
		name          string
		configuration *Configuration
		wantErr       error
	}{
		{name: "nil restores defaults"},
		{name: "empty card list is valid", configuration: &Configuration{Cards: []Card{}}},
		{name: "preserves ordered visibility and unknown cards", configuration: &Configuration{Cards: []Card{{Id: "future-card", Visible: false}, {Id: "time-today", Visible: true}}}},
		{name: "rejects missing cards", configuration: &Configuration{}, wantErr: ErrConfigurationInvalid},
		{name: "rejects duplicate cards", configuration: &Configuration{Cards: []Card{{Id: "time-today", Visible: true}, {Id: "time-today", Visible: false}}}, wantErr: ErrConfigurationInvalid},
		{name: "rejects empty card id", configuration: &Configuration{Cards: []Card{{Id: "", Visible: true}}}, wantErr: ErrConfigurationInvalid},
		{name: "rejects malformed card id", configuration: &Configuration{Cards: []Card{{Id: "bad id", Visible: true}}}, wantErr: ErrConfigurationInvalid},
		{name: "rejects overly long card id", configuration: &Configuration{Cards: []Card{{Id: strings.Repeat("a", 65), Visible: true}}}, wantErr: ErrConfigurationInvalid},
		{name: "tracking card cannot be configured", configuration: &Configuration{Cards: []Card{{Id: "tracking-now", Visible: false}}}, wantErr: ErrConfigurationInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := user.WithUser(context.Background(), user.User{Id: 1})
			repo := newRepositoryStub()
			service := NewService(repo)
			previous := &Configuration{Cards: []Card{{Id: "budget-balance", Visible: true}}}
			require.NoError(t, service.SaveConfiguration(ctx, previous))

			err := service.SaveConfiguration(ctx, test.configuration)

			require.ErrorIs(t, err, test.wantErr)
			stored, err := service.FindConfiguration(ctx)
			require.NoError(t, err)
			if test.wantErr != nil {
				require.Equal(t, previous, stored)
			} else {
				require.Equal(t, test.configuration, stored)
			}
		})
	}
}

func TestServiceImpl_SaveConfigurationCardLimit(t *testing.T) {
	tests := []struct {
		name    string
		count   int
		wantErr error
	}{
		{name: "accepts 100 cards", count: 100},
		{name: "rejects 101 cards", count: 101, wantErr: ErrConfigurationInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewService(newRepositoryStub())
			ctx := user.WithUser(context.Background(), user.User{Id: 1})
			configuration := &Configuration{Cards: make([]Card, test.count)}
			for index := range configuration.Cards {
				configuration.Cards[index] = Card{Id: fmt.Sprintf("card-%d", index), Visible: true}
			}

			err := service.SaveConfiguration(ctx, configuration)

			require.ErrorIs(t, err, test.wantErr)
			if test.wantErr == nil {
				stored, err := service.FindConfiguration(ctx)
				require.NoError(t, err)
				require.Equal(t, configuration, stored)
			}
		})
	}
}

func TestServiceImpl_FindConfigurationErrors(t *testing.T) {
	repoError := errors.New("database unavailable")
	tests := []struct {
		name    string
		ctx     context.Context
		repoErr error
		wantErr error
	}{
		{name: "missing current user", ctx: context.Background(), repoErr: repoError, wantErr: user.ErrNoUser},
		{name: "repository failure", ctx: user.WithUser(context.Background(), user.User{Id: 1}), repoErr: repoError, wantErr: repoError},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := newRepositoryStub()
			repo.err = test.repoErr
			service := NewService(repo)

			_, err := service.FindConfiguration(test.ctx)

			require.ErrorIs(t, err, test.wantErr)
		})
	}
}

func TestServiceImpl_SaveConfigurationErrors(t *testing.T) {
	repoError := errors.New("database unavailable")
	configuration := &Configuration{Cards: []Card{{Id: "time-today", Visible: true}}}
	tests := []struct {
		name          string
		ctx           context.Context
		configuration *Configuration
		repoErr       error
		wantErr       error
	}{
		{name: "save without current user", ctx: context.Background(), configuration: configuration, repoErr: repoError, wantErr: user.ErrNoUser},
		{name: "reset without current user", ctx: context.Background(), repoErr: repoError, wantErr: user.ErrNoUser},
		{name: "save repository failure", ctx: user.WithUser(context.Background(), user.User{Id: 1}), configuration: configuration, repoErr: repoError, wantErr: repoError},
		{name: "reset repository failure", ctx: user.WithUser(context.Background(), user.User{Id: 1}), repoErr: repoError, wantErr: repoError},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := newRepositoryStub()
			repo.err = test.repoErr
			service := NewService(repo)

			err := service.SaveConfiguration(test.ctx, test.configuration)

			require.ErrorIs(t, err, test.wantErr)
		})
	}
}
