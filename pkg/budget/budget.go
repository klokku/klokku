package budget

import (
	"time"
)

type Budget struct {
	ID   int
	Name string
	// WeeklyTime represents the total time allocated weekly for a budget, specified as a duration.
	WeeklyTime        time.Duration
	WeeklyOccurrences int
	StartDate         time.Time
	EndDate           time.Time
	Icon              string
	Position          int
}

func (b Budget) IsActiveOn(date time.Time) bool {
	afterStartDate := b.StartDate.IsZero() || b.StartDate.Before(date)
	beforeEndDate := b.EndDate.IsZero() || b.EndDate.After(date)
	return afterStartDate && beforeEndDate
}

// IsActiveBetween returns true if the budget is active in any date between the given start and end dates (inclusive).
func (b Budget) IsActiveBetween(startDate, endDate time.Time) bool {
	if endDate.Before(b.StartDate) && !b.StartDate.IsZero() {
		return false
	}

	if startDate.After(b.EndDate) && !b.EndDate.IsZero() {
		return false
	}

	return true
}
