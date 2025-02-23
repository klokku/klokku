package budget_override

import (
	"time"
)

type BudgetOverride struct {
	ID int
	// BudgetID is the unique identifier of the related budget.
	BudgetID int
	// StartDate indicates the start of the week date when the budget override becomes active for a week
	StartDate time.Time
	// WeeklyTime represents the total time allocated weekly for a budget, specified as a duration.
	WeeklyTime time.Duration
	// Notes provide additional information or comments about the budget override.
	Notes *string
}
