package budget

import (
	"time"
)

type BudgetStatus string

const (
	BudgetStatusActive   BudgetStatus = "active"
	BudgetStatusInactive BudgetStatus = "inactive"
	BudgetStatusArchived BudgetStatus = "archived"
)

type Budget struct {
	ID   int
	Name string
	// WeeklyTime represents the total time allocated weekly for a budget, specified as a duration.
	WeeklyTime        time.Duration
	WeeklyOccurrences int
	Status            BudgetStatus
	Icon              string
	Position          int
}
