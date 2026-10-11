package dashboard

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/klokku/klokku/pkg/user"
)

var ErrConfigurationInvalid = errors.New("invalid dashboard configuration")

var cardIdPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

// Service manages dashboard preferences for the current user from context.
type Service interface {
	// FindConfiguration returns nil when the current user follows application defaults.
	FindConfiguration(ctx context.Context) (*Configuration, error)
	// SaveConfiguration validates preferences; nil restores application defaults.
	SaveConfiguration(ctx context.Context, configuration *Configuration) error
}

type ServiceImpl struct {
	repo Repository
}

func NewService(repo Repository) *ServiceImpl {
	return &ServiceImpl{repo: repo}
}

func (s *ServiceImpl) FindConfiguration(ctx context.Context) (*Configuration, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return nil, fmt.Errorf("get dashboard user: %w", err)
	}
	return s.repo.Find(ctx, userId)
}

func (s *ServiceImpl) SaveConfiguration(ctx context.Context, configuration *Configuration) error {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return fmt.Errorf("get dashboard user: %w", err)
	}
	if configuration != nil {
		if configuration.Cards == nil || len(configuration.Cards) > 100 {
			return ErrConfigurationInvalid
		}
		seen := make(map[string]bool, len(configuration.Cards))
		for _, card := range configuration.Cards {
			if !cardIdPattern.MatchString(card.Id) || card.Id == "tracking-now" || seen[card.Id] {
				return ErrConfigurationInvalid
			}
			seen[card.Id] = true
		}
	}
	return s.repo.Store(ctx, userId, configuration)
}
