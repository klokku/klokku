package dashboard

import (
	"context"
	"slices"
)

type repositoryStub struct {
	configurations map[int]*Configuration
	err            error
}

func newRepositoryStub() *repositoryStub {
	return &repositoryStub{configurations: make(map[int]*Configuration)}
}

func (r *repositoryStub) Find(_ context.Context, userId int) (*Configuration, error) {
	if r.err != nil {
		return nil, r.err
	}
	return cloneConfiguration(r.configurations[userId]), nil
}

func (r *repositoryStub) Store(_ context.Context, userId int, configuration *Configuration) error {
	if r.err != nil {
		return r.err
	}
	r.configurations[userId] = cloneConfiguration(configuration)
	return nil
}

func cloneConfiguration(configuration *Configuration) *Configuration {
	if configuration == nil {
		return nil
	}
	return &Configuration{Cards: slices.Clone(configuration.Cards)}
}
