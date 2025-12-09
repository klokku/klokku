package budget_plan

import "time"

type BudgetPlan struct {
	Id        int
	Name      string
	IsCurrent bool
	Items     []BudgetItem
}

type BudgetItem struct {
	Id     int
	PlanId int
	Name   string
	// WeeklyDuration represents the total time allocated weekly for a budget, specified as a duration.
	WeeklyDuration time.Duration
	// WeeklyOccurrences represents the number of days in a week that a budget is expected to be used.
	WeeklyOccurrences int
	Icon              string
	Color             string
	Position          int
}
