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
	Weeks               []WeeklyReportEntry
	TotalItems          []ReportItem
	TotalBudgetPlanTime time.Duration
	TotalWeeklyPlanTime time.Duration
	TotalActualTime     time.Duration
}
