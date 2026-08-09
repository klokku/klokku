package calendar

import (
	"time"
)

type Event struct {
	UID       string
	Summary   string
	StartTime time.Time
	EndTime   time.Time
	Notes     string
	Metadata  EventMetadata
}

type EventPatch struct {
	Summary      *string
	StartTime    *time.Time
	EndTime      *time.Time
	Notes        *string
	BudgetItemId *int
}

type EventMetadata struct {
	BudgetItemId int `json:"budgetItemId"`
}
