package calendar

import (
	"time"
)

type Event struct {
	UID       string
	Summary   string
	StartTime time.Time
	EndTime   time.Time
	Metadata  EventMetadata
}

type EventMetadata struct {
	BudgetId int `json:"budgetId"`
}
