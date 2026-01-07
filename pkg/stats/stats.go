package stats

import (
	"time"
)

type DailyStats struct {
	Date             time.Time
	StatsPerPlanItem []PlanItemStats
	TotalTime        time.Duration
}

type PlanItem struct {
	BudgetPlanId       int
	BudgetItemId       int
	WeeklyItemId       int
	Name               string
	Icon               string
	Color              string
	Position           int
	WeeklyItemDuration time.Duration
	BudgetItemDuration time.Duration
	WeeklyOccurrences  int
	Notes              string
}

type PlanItemStats struct {
	PlanItem  PlanItem
	Duration  time.Duration
	Remaining time.Duration
	StartDate time.Time
	EndDate   time.Time
}

type PlanItemHistoryStats struct {
	StartDate    time.Time
	EndDate      time.Time
	StatsPerWeek []PlanItemStats
}

type WeeklyStatsSummary struct {
	StartDate      time.Time
	EndDate        time.Time
	PerDay         []DailyStats
	PerPlanItem    []PlanItemStats
	TotalPlanned   time.Duration
	TotalTime      time.Duration
	TotalRemaining time.Duration
}
