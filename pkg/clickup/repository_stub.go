package clickup

import (
	"context"
	"errors"
	"sync"
)

type RepositoryStub struct {
	mu             sync.RWMutex
	configs        map[configKey]*Configuration // (userId, budgetPlanId) -> config
	authData       map[int]bool                 // userId -> has auth data
	nextMappingPos int
}

type configKey struct {
	userId       int
	budgetPlanId int
}

func NewRepositoryStub() *RepositoryStub {
	return &RepositoryStub{
		configs:        make(map[configKey]*Configuration),
		authData:       make(map[int]bool),
		nextMappingPos: 1,
	}
}

func (r *RepositoryStub) StoreConfiguration(ctx context.Context, userId, budgetPlanId int, config Configuration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := configKey{userId: userId, budgetPlanId: budgetPlanId}
	// Create a deep copy to avoid shared reference issues
	configCopy := Configuration{
		WorkspaceId:           config.WorkspaceId,
		SpaceId:               config.SpaceId,
		FolderId:              config.FolderId,
		OnlyTasksWithPriority: config.OnlyTasksWithPriority,
		Mappings:              make([]BudgetItemMapping, len(config.Mappings)),
	}
	copy(configCopy.Mappings, config.Mappings)

	r.configs[key] = &configCopy
	return nil
}

func (r *RepositoryStub) GetConfiguration(ctx context.Context, userId, budgetPlanId int) (*Configuration, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := configKey{userId: userId, budgetPlanId: budgetPlanId}
	config, exists := r.configs[key]
	if !exists {
		return nil, nil
	}

	// Return a copy to avoid external modifications
	configCopy := &Configuration{
		WorkspaceId:           config.WorkspaceId,
		SpaceId:               config.SpaceId,
		FolderId:              config.FolderId,
		OnlyTasksWithPriority: config.OnlyTasksWithPriority,
		Mappings:              make([]BudgetItemMapping, len(config.Mappings)),
	}
	copy(configCopy.Mappings, config.Mappings)

	return configCopy, nil
}

func (r *RepositoryStub) GetConfigurationWithMappingByBudgetItemId(ctx context.Context, userId, budgetItemId int) (*Configuration, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Search through all configs for this user to find the one with the matching budget item
	for key, config := range r.configs {
		if key.userId != userId {
			continue
		}

		for _, mapping := range config.Mappings {
			if mapping.BudgetItemId == budgetItemId {
				// Found the mapping, return config with just this mapping
				configCopy := &Configuration{
					WorkspaceId:           config.WorkspaceId,
					SpaceId:               config.SpaceId,
					FolderId:              config.FolderId,
					OnlyTasksWithPriority: config.OnlyTasksWithPriority,
					Mappings:              []BudgetItemMapping{mapping},
				}
				return configCopy, nil
			}
		}
	}

	return nil, nil
}

func (r *RepositoryStub) DeleteAllConfigurations(ctx context.Context, userId int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Delete all configs for this user
	for key := range r.configs {
		if key.userId == userId {
			delete(r.configs, key)
		}
	}

	return nil
}

func (r *RepositoryStub) DeleteBudgetPlanConfiguration(ctx context.Context, userId, budgetPlanId int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := configKey{userId: userId, budgetPlanId: budgetPlanId}
	delete(r.configs, key)

	return nil
}

func (r *RepositoryStub) DeleteAuthData(ctx context.Context, userId int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.authData, userId)
	return nil
}

// Helper methods for testing

// SetAuthData sets auth data for a user (useful for test setup)
func (r *RepositoryStub) SetAuthData(userId int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.authData[userId] = true
}

// HasAuthData checks if auth data exists for a user
func (r *RepositoryStub) HasAuthData(userId int) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.authData[userId]
}

// GetAllConfigs returns all stored configurations (useful for test assertions)
func (r *RepositoryStub) GetAllConfigs() map[configKey]*Configuration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[configKey]*Configuration, len(r.configs))
	for k, v := range r.configs {
		configCopy := &Configuration{
			WorkspaceId:           v.WorkspaceId,
			SpaceId:               v.SpaceId,
			FolderId:              v.FolderId,
			OnlyTasksWithPriority: v.OnlyTasksWithPriority,
			Mappings:              make([]BudgetItemMapping, len(v.Mappings)),
		}
		copy(configCopy.Mappings, v.Mappings)
		result[k] = configCopy
	}
	return result
}

// Reset clears all data (useful between tests)
func (r *RepositoryStub) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.configs = make(map[configKey]*Configuration)
	r.authData = make(map[int]bool)
	r.nextMappingPos = 1
}

// SetError can be used to simulate errors
type RepositoryStubWithError struct {
	*RepositoryStub
	storeConfigErr                        error
	getConfigErr                          error
	getConfigWithMappingByBudgetItemIdErr error
	deleteAllConfigurationsErr            error
	deleteBudgetPlanConfigurationErr      error
	deleteAuthDataErr                     error
}

func NewRepositoryStubWithError(stub *RepositoryStub) *RepositoryStubWithError {
	return &RepositoryStubWithError{RepositoryStub: stub}
}

func (r *RepositoryStubWithError) StoreConfiguration(ctx context.Context, userId, budgetPlanId int, config Configuration) error {
	if r.storeConfigErr != nil {
		return r.storeConfigErr
	}
	return r.RepositoryStub.StoreConfiguration(ctx, userId, budgetPlanId, config)
}

func (r *RepositoryStubWithError) GetConfiguration(ctx context.Context, userId, budgetPlanId int) (*Configuration, error) {
	if r.getConfigErr != nil {
		return nil, r.getConfigErr
	}
	return r.RepositoryStub.GetConfiguration(ctx, userId, budgetPlanId)
}

func (r *RepositoryStubWithError) GetConfigurationWithMappingByBudgetItemId(ctx context.Context, userId, budgetItemId int) (*Configuration, error) {
	if r.getConfigWithMappingByBudgetItemIdErr != nil {
		return nil, r.getConfigWithMappingByBudgetItemIdErr
	}
	return r.RepositoryStub.GetConfigurationWithMappingByBudgetItemId(ctx, userId, budgetItemId)
}

func (r *RepositoryStubWithError) DeleteAllConfigurations(ctx context.Context, userId int) error {
	if r.deleteAllConfigurationsErr != nil {
		return r.deleteAllConfigurationsErr
	}
	return r.RepositoryStub.DeleteAllConfigurations(ctx, userId)
}

func (r *RepositoryStubWithError) DeleteBudgetPlanConfiguration(ctx context.Context, userId, budgetPlanId int) error {
	if r.deleteBudgetPlanConfigurationErr != nil {
		return r.deleteBudgetPlanConfigurationErr
	}
	return r.RepositoryStub.DeleteBudgetPlanConfiguration(ctx, userId, budgetPlanId)
}

func (r *RepositoryStubWithError) DeleteAuthData(ctx context.Context, userId int) error {
	if r.deleteAuthDataErr != nil {
		return r.deleteAuthDataErr
	}
	return r.RepositoryStub.DeleteAuthData(ctx, userId)
}

// Error setters for testing error scenarios
func (r *RepositoryStubWithError) SetStoreConfigurationError(err error) {
	r.storeConfigErr = err
}

func (r *RepositoryStubWithError) SetGetConfigurationError(err error) {
	r.getConfigErr = err
}

func (r *RepositoryStubWithError) SetGetConfigurationWithMappingByBudgetItemIdError(err error) {
	r.getConfigWithMappingByBudgetItemIdErr = err
}

func (r *RepositoryStubWithError) SetDeleteAllConfigurationsError(err error) {
	r.deleteAllConfigurationsErr = err
}

func (r *RepositoryStubWithError) SetDeleteBudgetPlanConfigurationError(err error) {
	r.deleteBudgetPlanConfigurationErr = err
}

func (r *RepositoryStubWithError) SetDeleteAuthDataError(err error) {
	r.deleteAuthDataErr = err
}

var ErrRepositoryTestError = errors.New("repository test error")
