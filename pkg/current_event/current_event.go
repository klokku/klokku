package current_event

import (
	"time"
)

type CurrentEvent struct {
	Id        int
	PlanItem  PlanItem
	StartTime time.Time
}

type PlanItem struct {
	BudgetItemId   int
	Name           string
	WeeklyDuration time.Duration
}
