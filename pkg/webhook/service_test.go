package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klokku/klokku/pkg/budget_plan"
	"github.com/klokku/klokku/pkg/current_event"
	"github.com/klokku/klokku/pkg/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var ctx = context.WithValue(context.Background(), user.UserKey, user.User{
	Id:          10,
	Uid:         uuid.NewString(),
	Username:    "test-user-1",
	DisplayName: "Test User 1",
	PhotoUrl:    "",
	Settings: user.Settings{
		Timezone:          "Europe/Warsaw",
		WeekFirstDay:      time.Monday,
		EventCalendarType: user.KlokkuCalendar,
		GoogleCalendar:    user.GoogleCalendarSettings{},
	},
})

// Mock implementations
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, webhook Webhook) (Webhook, error) {
	args := m.Called(ctx, webhook)
	return args.Get(0).(Webhook), args.Error(1)
}

func (m *MockRepository) GetByToken(ctx context.Context, token string) (Webhook, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return Webhook{}, args.Error(1)
	}
	return args.Get(0).(Webhook), args.Error(1)
}

func (m *MockRepository) GetByUserIdAndType(ctx context.Context, userId int, webhookType WebhookType) ([]Webhook, error) {
	args := m.Called(ctx, userId, webhookType)
	return args.Get(0).([]Webhook), args.Error(1)
}

func (m *MockRepository) RotateToken(ctx context.Context, webhookId int, userId int) (string, error) {
	args := m.Called(ctx, webhookId, userId)
	return args.String(0), args.Error(1)
}

func (m *MockRepository) Delete(ctx context.Context, webhookId int, userId int) error {
	args := m.Called(ctx, webhookId, userId)
	return args.Error(0)
}

// MockEventStarter mocks the EventStarter interface
type MockEventStarter struct {
	mock.Mock
}

func (m *MockEventStarter) StartNewEvent(ctx context.Context, event current_event.CurrentEvent) (current_event.CurrentEvent, error) {
	args := m.Called(ctx, event)
	return args.Get(0).(current_event.CurrentEvent), args.Error(1)
}

// MockBudgetItemProvider mocks the BudgetItemProvider interface
type MockBudgetItemProvider struct {
	mock.Mock
}

func (m *MockBudgetItemProvider) GetItem(ctx context.Context, id int) (budget_plan.BudgetItem, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return budget_plan.BudgetItem{}, args.Error(1)
	}
	return args.Get(0).(budget_plan.BudgetItem), args.Error(1)
}

// MockUserProvider mocks the UserProvider interface
type MockUserProvider struct {
	mock.Mock
}

func (m *MockUserProvider) GetUser(ctx context.Context, id int) (user.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(user.User), args.Error(1)
}

func setupService() (*ServiceImpl, *MockRepository, *MockEventStarter, *MockBudgetItemProvider, *MockUserProvider) {
	mockRepo := new(MockRepository)
	mockEventStarter := new(MockEventStarter)
	mockBudgetProvider := new(MockBudgetItemProvider)
	mockUserProvider := new(MockUserProvider)

	service := &ServiceImpl{
		repo:          mockRepo,
		eventStarter:  mockEventStarter,
		budgetService: mockBudgetProvider,
		userService:   mockUserProvider,
	}

	return service, mockRepo, mockEventStarter, mockBudgetProvider, mockUserProvider
}

func TestServiceImpl_Create(t *testing.T) {
	t.Run("should create webhook with data", func(t *testing.T) {
		// given
		service, mockRepo, _, _, _ := setupService()

		data := StartCurrentEventData{BudgetItemId: 42}
		dataJSON, _ := json.Marshal(data)

		expectedWebhook := Webhook{
			Id:        1,
			Type:      TypeStartCurrentEvent,
			Token:     "abc123def456",
			UserId:    10,
			Data:      dataJSON,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockRepo.On("Create", ctx, mock.MatchedBy(func(w Webhook) bool {
			return w.Type == TypeStartCurrentEvent && w.UserId == 10
		})).Return(expectedWebhook, nil)

		// when
		webhook, webhookURL, err := service.Create(ctx, TypeStartCurrentEvent, data)

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedWebhook.Id, webhook.Id)
		assert.Equal(t, expectedWebhook.Token, webhook.Token)
		assert.Equal(t, "https://app.klokku.com/webhook/abc123def456", webhookURL)
		mockRepo.AssertExpectations(t)
	})

	t.Run("should return error when user not in context", func(t *testing.T) {
		// given
		service, _, _, _, _ := setupService()
		emptyCtx := context.Background()

		// when
		_, _, err := service.Create(emptyCtx, TypeStartCurrentEvent, StartCurrentEventData{})

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get current user")
	})
}

func TestServiceImpl_GetByUserIdAndType(t *testing.T) {
	t.Run("should return user's webhooks", func(t *testing.T) {
		// given
		service, mockRepo, _, _, _ := setupService()

		expectedWebhooks := []Webhook{
			{Id: 1, Type: TypeStartCurrentEvent, Token: "token1", UserId: 10},
			{Id: 2, Type: TypeStartCurrentEvent, Token: "token2", UserId: 10},
		}

		mockRepo.On("GetByUserIdAndType", ctx, 10, TypeStartCurrentEvent).
			Return(expectedWebhooks, nil)

		// when
		webhooks, err := service.GetByUserIdAndType(ctx, TypeStartCurrentEvent)

		// then
		require.NoError(t, err)
		assert.Len(t, webhooks, 2)
		assert.Equal(t, expectedWebhooks, webhooks)
		mockRepo.AssertExpectations(t)
	})

	t.Run("should return error when user not in context", func(t *testing.T) {
		// given
		service, _, _, _, _ := setupService()
		emptyCtx := context.Background()

		// when
		_, err := service.GetByUserIdAndType(emptyCtx, TypeStartCurrentEvent)

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get current user")
	})
}

func TestServiceImpl_RotateToken(t *testing.T) {
	t.Run("should rotate token", func(t *testing.T) {
		// given
		service, mockRepo, _, _, _ := setupService()
		webhookId := 1
		newToken := "newtoken123456"

		mockRepo.On("RotateToken", ctx, webhookId, 10).Return(newToken, nil)

		// when
		token, err := service.RotateToken(ctx, webhookId)

		// then
		require.NoError(t, err)
		assert.Equal(t, newToken, token)
		mockRepo.AssertExpectations(t)
	})

	t.Run("should return error when user not in context", func(t *testing.T) {
		// given
		service, _, _, _, _ := setupService()
		emptyCtx := context.Background()

		// when
		_, err := service.RotateToken(emptyCtx, 1)

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get current user")
	})
}

func TestServiceImpl_Delete(t *testing.T) {
	t.Run("should delete webhook", func(t *testing.T) {
		// given
		service, mockRepo, _, _, _ := setupService()
		webhookId := 1

		mockRepo.On("Delete", ctx, webhookId, 10).Return(nil)

		// when
		err := service.Delete(ctx, webhookId)

		// then
		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("should return error when user not in context", func(t *testing.T) {
		// given
		service, _, _, _, _ := setupService()
		emptyCtx := context.Background()

		// when
		err := service.Delete(emptyCtx, 1)

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get current user")
	})
}

func TestServiceImpl_Execute(t *testing.T) {
	t.Run("should execute START_CURRENT_EVENT webhook", func(t *testing.T) {
		// given
		service, mockRepo, mockEventStarter, mockBudgetProvider, mockUserProvider := setupService()

		budgetItemId := 42
		data := StartCurrentEventData{BudgetItemId: budgetItemId}
		dataJSON, _ := json.Marshal(data)

		webhook := Webhook{
			Id:     1,
			Type:   TypeStartCurrentEvent,
			Token:  "test-token",
			UserId: 10,
			Data:   dataJSON,
		}

		testUser := user.User{
			Id:          10,
			Uid:         "user-uid",
			Username:    "testuser",
			DisplayName: "Test User",
		}

		budgetItem := budget_plan.BudgetItem{
			Id:             budgetItemId,
			Name:           "Work",
			WeeklyDuration: 40 * time.Hour,
		}

		expectedEvent := current_event.CurrentEvent{
			Id: 1,
			PlanItem: current_event.PlanItem{
				BudgetItemId:   budgetItemId,
				Name:           "Work",
				WeeklyDuration: 40 * time.Hour,
			},
			StartTime: time.Now(),
		}

		mockRepo.On("GetByToken", mock.Anything, "test-token").Return(webhook, nil)
		mockUserProvider.On("GetUser", mock.Anything, 10).Return(testUser, nil)
		mockBudgetProvider.On("GetItem", mock.Anything, budgetItemId).Return(budgetItem, nil)
		mockEventStarter.On("StartNewEvent", mock.Anything, mock.MatchedBy(func(e current_event.CurrentEvent) bool {
			return e.PlanItem.BudgetItemId == budgetItemId && e.PlanItem.Name == "Work"
		})).Return(expectedEvent, nil)

		// when
		err := service.Execute(context.Background(), "test-token")

		// then
		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockUserProvider.AssertExpectations(t)
		mockBudgetProvider.AssertExpectations(t)
		mockEventStarter.AssertExpectations(t)
	})

	t.Run("should return error for invalid token", func(t *testing.T) {
		// given
		service, mockRepo, _, _, _ := setupService()

		mockRepo.On("GetByToken", mock.Anything, "invalid-token").
			Return(nil, ErrWebhookNotFound)

		// when
		err := service.Execute(context.Background(), "invalid-token")

		// then
		require.ErrorIs(t, err, ErrWebhookNotFound)
		mockRepo.AssertExpectations(t)
	})

	t.Run("should return error for unsupported webhook type", func(t *testing.T) {
		// given
		service, mockRepo, _, _, mockUserProvider := setupService()

		webhook := Webhook{
			Id:     1,
			Type:   WebhookType("UNSUPPORTED_TYPE"),
			Token:  "test-token",
			UserId: 10,
			Data:   json.RawMessage(`{}`),
		}

		testUser := user.User{Id: 10}

		mockRepo.On("GetByToken", mock.Anything, "test-token").Return(webhook, nil)
		mockUserProvider.On("GetUser", mock.Anything, 10).Return(testUser, nil)

		// when
		err := service.Execute(context.Background(), "test-token")

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported webhook type")
		mockRepo.AssertExpectations(t)
		mockUserProvider.AssertExpectations(t)
	})

	t.Run("should return error when user not found", func(t *testing.T) {
		// given
		service, mockRepo, _, _, mockUserProvider := setupService()

		webhook := Webhook{
			Id:     1,
			Type:   TypeStartCurrentEvent,
			Token:  "test-token",
			UserId: 999,
			Data:   json.RawMessage(`{"budgetItemId": 42}`),
		}

		mockRepo.On("GetByToken", mock.Anything, "test-token").Return(webhook, nil)
		mockUserProvider.On("GetUser", mock.Anything, 999).
			Return(user.User{}, user.ErrNoUser)

		// when
		err := service.Execute(context.Background(), "test-token")

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get user")
		mockRepo.AssertExpectations(t)
		mockUserProvider.AssertExpectations(t)
	})

	t.Run("should return error when budget item not found", func(t *testing.T) {
		// given
		service, mockRepo, _, mockBudgetProvider, mockUserProvider := setupService()

		data := StartCurrentEventData{BudgetItemId: 999}
		dataJSON, _ := json.Marshal(data)

		webhook := Webhook{
			Id:     1,
			Type:   TypeStartCurrentEvent,
			Token:  "test-token",
			UserId: 10,
			Data:   dataJSON,
		}

		testUser := user.User{Id: 10}

		mockRepo.On("GetByToken", mock.Anything, "test-token").Return(webhook, nil)
		mockUserProvider.On("GetUser", mock.Anything, 10).Return(testUser, nil)
		mockBudgetProvider.On("GetItem", mock.Anything, 999).
			Return(nil, errors.New("item not found"))

		// when
		err := service.Execute(context.Background(), "test-token")

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get budget item")
		mockRepo.AssertExpectations(t)
		mockUserProvider.AssertExpectations(t)
		mockBudgetProvider.AssertExpectations(t)
	})

	t.Run("should return error when invalid JSON data", func(t *testing.T) {
		// given
		service, mockRepo, _, _, mockUserProvider := setupService()

		webhook := Webhook{
			Id:     1,
			Type:   TypeStartCurrentEvent,
			Token:  "test-token",
			UserId: 10,
			Data:   json.RawMessage(`invalid json`),
		}

		testUser := user.User{Id: 10}

		mockRepo.On("GetByToken", mock.Anything, "test-token").Return(webhook, nil)
		mockUserProvider.On("GetUser", mock.Anything, 10).Return(testUser, nil)

		// when
		err := service.Execute(context.Background(), "test-token")

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to unmarshal webhook data")
		mockRepo.AssertExpectations(t)
		mockUserProvider.AssertExpectations(t)
	})
}
