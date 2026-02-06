package webhook

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

type RepositoryStub struct {
	mu       sync.RWMutex
	webhooks map[int]Webhook // id -> webhook
	tokens   map[string]int  // token -> id
	userIds  map[int]int     // id -> userId
	nextId   int
}

func NewRepositoryStub() *RepositoryStub {
	return &RepositoryStub{
		webhooks: make(map[int]Webhook),
		tokens:   make(map[string]int),
		userIds:  make(map[int]int),
		nextId:   1,
	}
}

func (r *RepositoryStub) Create(ctx context.Context, webhook Webhook) (Webhook, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Generate token if not provided
	if webhook.Token == "" {
		token, err := r.generateToken()
		if err != nil {
			return Webhook{}, err
		}
		webhook.Token = token
	}

	webhook.Id = r.nextId
	webhook.CreatedAt = time.Now()
	webhook.UpdatedAt = time.Now()

	r.webhooks[r.nextId] = webhook
	r.tokens[webhook.Token] = r.nextId
	r.userIds[r.nextId] = webhook.UserId
	r.nextId++

	return webhook, nil
}

func (r *RepositoryStub) GetByToken(ctx context.Context, token string) (Webhook, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if token == "" {
		return Webhook{}, ErrWebhookNotFound
	}

	id, exists := r.tokens[token]
	if !exists {
		return Webhook{}, ErrWebhookNotFound
	}

	webhook, exists := r.webhooks[id]
	if !exists {
		return Webhook{}, ErrWebhookNotFound
	}

	return webhook, nil
}

func (r *RepositoryStub) GetByUserIdAndType(ctx context.Context, userId int, webhookType WebhookType) ([]Webhook, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Webhook
	for id, webhook := range r.webhooks {
		if r.userIds[id] == userId && webhook.Type == webhookType {
			result = append(result, webhook)
		}
	}

	// Sort by creation time descending (newest first)
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].CreatedAt.Before(result[j].CreatedAt) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result, nil
}

func (r *RepositoryStub) RotateToken(ctx context.Context, webhookId int, userId int) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	webhook, exists := r.webhooks[webhookId]
	if !exists || r.userIds[webhookId] != userId {
		return "", ErrWebhookNotFound
	}

	// Generate new token
	newToken, err := r.generateToken()
	if err != nil {
		return "", err
	}

	// Remove old token mapping
	delete(r.tokens, webhook.Token)

	// Update webhook with new token
	webhook.Token = newToken
	webhook.UpdatedAt = time.Now()
	r.webhooks[webhookId] = webhook
	r.tokens[newToken] = webhookId

	return newToken, nil
}

func (r *RepositoryStub) Delete(ctx context.Context, webhookId int, userId int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	webhook, exists := r.webhooks[webhookId]
	if !exists || r.userIds[webhookId] != userId {
		return ErrWebhookNotFound
	}

	delete(r.tokens, webhook.Token)
	delete(r.webhooks, webhookId)
	delete(r.userIds, webhookId)

	return nil
}

func (r *RepositoryStub) generateToken() (string, error) {
	tokenBytes := make([]byte, 32)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", tokenBytes), nil
}

// Helper method to reset the stub (useful between tests)
func (r *RepositoryStub) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.webhooks = make(map[int]Webhook)
	r.tokens = make(map[string]int)
	r.userIds = make(map[int]int)
	r.nextId = 1
}

// Helper method to get all webhooks (useful for test assertions)
func (r *RepositoryStub) GetAllWebhooks() []Webhook {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Webhook, 0, len(r.webhooks))
	for _, webhook := range r.webhooks {
		result = append(result, webhook)
	}
	return result
}
