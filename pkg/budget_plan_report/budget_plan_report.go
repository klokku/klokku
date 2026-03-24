package budget_plan_report

import (
	"time"
)

// ReportItem holds the three metrics for a single budget item within a period.
type ReportItem struct {
	BudgetItemId   int
	Name           string
	Icon           string
	Color          string
	Position       int
	BudgetPlanTime time.Duration // from budget plan item WeeklyDuration
	WeeklyPlanTime time.Duration // from weekly plan (or fallback to budget plan)
	ActualTime     time.Duration // from calendar events
	AveragePerWeek time.Duration // ActualTime / WeekCount (totals only)
	AveragePerDay  time.Duration // ActualTime / (WeekCount * WeeklyOccurrences) (totals only)
}

// WeeklyReportEntry represents one week's data for all budget items.
type WeeklyReportEntry struct {
	WeekNumber          string
	StartDate           time.Time
	EndDate             time.Time
	Items               []ReportItem
	TotalBudgetPlanTime time.Duration
	TotalWeeklyPlanTime time.Duration
	TotalActualTime     time.Duration
}

// Report holds per-week breakdown and aggregated totals for a budget plan over a time period.
type Report struct {
	PlanId              int
	PlanName            string
	StartDate           time.Time
	EndDate             time.Time
	WeekCount           int
	ExcludedWeekCount   int // number of off-weeks within the period that were excluded
	Weeks               []WeeklyReportEntry
	TotalItems          []ReportItem
	TotalBudgetPlanTime time.Duration
	TotalWeeklyPlanTime time.Duration
	TotalActualTime     time.Duration
}

// ItemDetailReport holds detailed statistics for a single budget item over a period.
type ItemDetailReport struct {
	PlanId    int
	PlanName  string
	ItemId    int
	ItemName  string
	ItemIcon  string
	ItemColor string

	StartDate time.Time
	EndDate   time.Time

	TotalActualTime     time.Duration
	TotalBudgetPlanTime time.Duration
	TotalWeeklyPlanTime time.Duration
	CompletionPercent   float64
	RemainingTime       time.Duration // max(0, budget - actual)
	OverBudgetTime      time.Duration // max(0, actual - budget)

	AveragePerDay       time.Duration
	AveragePerActiveDay time.Duration
	AveragePerWeek      time.Duration
	MedianPerDay        time.Duration
	MedianPerActiveDay  time.Duration
	MedianPerWeek       time.Duration

	ActiveDaysCount   int
	TotalDaysCount    int
	WeekCount         int
	ExcludedWeekCount int

	Weeks        []ItemWeekEntry
	Days         []ItemDayEntry
	DayOfWeekAvg []DayOfWeekEntry
}

// ItemWeekEntry is one week of data for a single budget item (including off-weeks).
type ItemWeekEntry struct {
	WeekNumber     string
	StartDate      time.Time
	EndDate        time.Time
	BudgetPlanTime time.Duration
	WeeklyPlanTime time.Duration
	ActualTime     time.Duration
	IsOffWeek      bool
}

// ItemDayEntry is one calendar day of tracked time for the heatmap.
type ItemDayEntry struct {
	Date       time.Time // truncated to midnight in user timezone
	ActualTime time.Duration
	DayOfWeek  time.Weekday
}

// DayOfWeekEntry holds the total and average time for a single day of the week.
type DayOfWeekEntry struct {
	DayOfWeek   time.Weekday
	AverageTime time.Duration
	TotalTime   time.Duration
}
