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

// MonthlyReportEntry represents one 4-week period's data for all budget items.
type MonthlyReportEntry struct {
	PeriodNumber        int
	StartDate           time.Time
	EndDate             time.Time
	WeekCount           int
	Items               []ReportItem
	TotalBudgetPlanTime time.Duration
	TotalWeeklyPlanTime time.Duration
	TotalActualTime     time.Duration
}

// SummaryReport holds the aggregated totals for the entire lifetime of a budget plan.
type SummaryReport struct {
	PlanId              int
	PlanName            string
	StartDate           time.Time
	EndDate             time.Time
	WeekCount           int
	Items               []ReportItem
	TotalBudgetPlanTime time.Duration
	TotalWeeklyPlanTime time.Duration
	TotalActualTime     time.Duration
}
