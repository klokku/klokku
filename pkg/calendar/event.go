package calendar

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	UID       uuid.NullUUID
	Summary   string
	StartTime time.Time
	EndTime   time.Time
	Metadata  EventMetadata
}

type EventMetadata struct {
	BudgetId int `json:"budgetId"`
}
