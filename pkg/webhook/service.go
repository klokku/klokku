package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/klokku/klokku/pkg/budget_plan"
	"github.com/klokku/klokku/pkg/current_event"
	"github.com/klokku/klokku/pkg/user"
	log "github.com/sirupsen/logrus"
)

type EventStarter interface {
	StartNewEvent(ctx context.Context, event current_event.CurrentEvent) (current_event.CurrentEvent, error)
}

type BudgetItemProvider interface {
	GetItem(ctx context.Context, id int) (budget_plan.BudgetItem, error)
}

type UserProvider interface {
	GetUser(ctx context.Context, id int) (user.User, error)
}

type Service interface {
	Create(ctx context.Context, webhookType WebhookType, data interface{}) (Webhook, error)
	GetByUserIdAndType(ctx context.Context, webhookType WebhookType) ([]Webhook, error)
	RotateToken(ctx context.Context, webhookId int) (string, error)
	Delete(ctx context.Context, webhookId int) error
	Execute(ctx context.Context, token string) error
}

type ServiceImpl struct {
	repo          Repository
	eventStarter  EventStarter
	budgetService BudgetItemProvider
	userService   UserProvider
}

func NewService(repo Repository, eventStarter EventStarter, budgetService BudgetItemProvider, userService UserProvider) Service {
	return &ServiceImpl{
		repo:          repo,
		eventStarter:  eventStarter,
		budgetService: budgetService,
		userService:   userService,
	}
}

func (s *ServiceImpl) Create(ctx context.Context, webhookType WebhookType, data interface{}) (Webhook, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return Webhook{}, fmt.Errorf("failed to get current user: %w", err)
	}

	// Marshal data to JSON
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return Webhook{}, fmt.Errorf("failed to marshal webhook data: %w", err)
	}

	webhook := Webhook{
		Type:   webhookType,
		UserId: userId,
		Data:   dataJSON,
	}

	created, err := s.repo.Create(ctx, webhook)
	if err != nil {
		return Webhook{}, err
	}

	return created, nil
}

func (s *ServiceImpl) GetByUserIdAndType(ctx context.Context, webhookType WebhookType) ([]Webhook, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	return s.repo.GetByUserIdAndType(ctx, userId, webhookType)
}

func (s *ServiceImpl) RotateToken(ctx context.Context, webhookId int) (string, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	return s.repo.RotateToken(ctx, webhookId, userId)
}

func (s *ServiceImpl) Delete(ctx context.Context, webhookId int) error {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	return s.repo.Delete(ctx, webhookId, userId)
}

func (s *ServiceImpl) Execute(ctx context.Context, token string) error {
	// Get webhook by token (no user context required)
	webhook, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return err
	}

	// Get user to create proper context for service calls
	userObj, err := s.userService.GetUser(ctx, webhook.UserId)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Create context with user
	userCtx := user.WithUser(ctx, userObj)

	// Execute based on webhook type
	switch webhook.Type {
	case TypeStartCurrentEvent:
		return s.executeStartCurrentEvent(userCtx, webhook)
	default:
		return fmt.Errorf("unsupported webhook type: %s", webhook.Type)
	}
}

func (s *ServiceImpl) executeStartCurrentEvent(ctx context.Context, webhook Webhook) error {
	// Parse webhook data
	var data StartCurrentEventData
	if err := json.Unmarshal(webhook.Data, &data); err != nil {
		return fmt.Errorf("failed to unmarshal webhook data: %w", err)
	}

	// Get budget item to retrieve all necessary information
	budgetItem, err := s.budgetService.GetItem(ctx, data.BudgetItemId)
	if err != nil {
		return fmt.Errorf("failed to get budget item: %w", err)
	}

	// Create and start event
	event := current_event.CurrentEvent{
		StartTime: time.Now(),
		PlanItem: current_event.PlanItem{
			BudgetItemId:   budgetItem.Id,
			Name:           budgetItem.Name,
			WeeklyDuration: budgetItem.WeeklyDuration,
		},
	}

	_, err = s.eventStarter.StartNewEvent(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to start event: %w", err)
	}

	log.Infof("Event started via webhook for user %d, item %d (%s)", webhook.UserId, budgetItem.Id, budgetItem.Name)
	return nil
}
