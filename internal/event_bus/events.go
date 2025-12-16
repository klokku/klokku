package event_bus

import "time"

type BudgetPlanItemUpdated struct {
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

type CalendarEventCreated struct {
	UID          string
	Summary      string
	StartTime    time.Time
	EndTime      time.Time
	BudgetItemId int
}
