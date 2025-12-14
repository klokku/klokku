package weekly_plan

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type WeeklyPlanItem struct {
	Id           int
	BudgetItemId int
	WeekNumber   WeekNumber
	Name         string // copy - as long as BudgetItem exist, updated with value from there
	// WeeklyDuration represents the total time allocated weekly for a budget, specified as a duration.
	WeeklyDuration time.Duration // updatable - independent copy
	// WeeklyOccurrences represents the number of days in a week that a budget is expected to be used.
	WeeklyOccurrences int    // immutable - created and never updated
	Icon              string // copy - as long as BudgetItem exist, updated with value from there
	Color             string // copy - as long as BudgetItem exist, updated with value from there
	Notes             string // updatable - independent - does not exist on BudgetItem
	Position          int    // copy - as long as BudgetItem exist, updated with value from there
}

type WeekNumber struct {
	Week int
	Year int
}

// WeekNumberFromDate returns the ISO week number that corresponds to the week containing
// the provided date, taking the desired week start day into account. The week start day
// can shift the ISO week into the previous calendar week when it is earlier than Monday.
func WeekNumberFromDate(date time.Time, weekStartDay time.Weekday) WeekNumber {
	if weekStartDay < time.Sunday || weekStartDay > time.Saturday {
		weekStartDay = time.Monday
	}

	delta := (int(date.Weekday()) - int(weekStartDay) + 7) % 7
	startOfWeek := date.AddDate(0, 0, -delta)

	year, week := startOfWeek.ISOWeek()
	return WeekNumber{Year: year, Week: week}
}

// WeekNumberFromString converts ISO week format ISO 8601 e.g. "2025-W03" to WeekNumber
func WeekNumberFromString(isoWeekString string) (WeekNumber, error) {
	parts := strings.Split(isoWeekString, "-")
	if len(parts) != 2 {
		return WeekNumber{}, fmt.Errorf("invalid ISO week format: %s", isoWeekString)
	}
	year, err := strconv.Atoi(parts[0])
	if err != nil {
		return WeekNumber{}, fmt.Errorf("invalid year: %w", err)
	}
	week, err := strconv.Atoi(parts[1][1:])
	if err != nil {
		return WeekNumber{}, fmt.Errorf("invalid week: %w", err)
	}
	return WeekNumber{Year: year, Week: week}, nil
}

// Equal returns true when both the year and week match.
func (w WeekNumber) Equal(other WeekNumber) bool {
	return w.Year == other.Year && w.Week == other.Week
}

// Before reports whether w refers to a week that occurs before other.
func (w WeekNumber) Before(other WeekNumber) bool {
	if w.Year != other.Year {
		return w.Year < other.Year
	}
	return w.Week < other.Week
}

// After reports whether w refers to a week that occurs after other.
func (w WeekNumber) After(other WeekNumber) bool {
	if w.Year != other.Year {
		return w.Year > other.Year
	}
	return w.Week > other.Week
}

// String returns the ISO week format ISO 8601 e.g. "2025-W03"
func (w WeekNumber) String() string {
	return fmt.Sprintf("%04d-W%02d", w.Year, w.Week)
}
