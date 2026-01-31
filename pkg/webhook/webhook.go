package webhook

import (
	"encoding/json"
	"time"
)

// WebhookType represents the type of webhook action
type WebhookType string

const (
	TypeStartCurrentEvent WebhookType = "START_CURRENT_EVENT"
)

// Webhook represents a webhook configuration
type Webhook struct {
	Id        int
	Type      WebhookType
	Token     string
	UserId    int
	Data      json.RawMessage
	CreatedAt time.Time
	UpdatedAt time.Time
}

// StartCurrentEventData is the data structure for START_CURRENT_EVENT webhook type
type StartCurrentEventData struct {
	BudgetItemId int `json:"budgetItemId"`
}
