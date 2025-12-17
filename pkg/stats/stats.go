package stats

import (
	"time"

	"github.com/klokku/klokku/pkg/weekly_plan"
)

type DailyStats struct {
	Date             time.Time
	StatsPerPlanItem []PlanItemStats
	TotalTime        time.Duration
}

type PlanItemStats struct {
	PlanItem  weekly_plan.WeeklyPlanItem
	Duration  time.Duration
	Remaining time.Duration
}

type StatsSummary struct {
	StartDate      time.Time
	EndDate        time.Time
	PerDay         []DailyStats
	PerPlanItem    []PlanItemStats
	TotalPlanned   time.Duration
	TotalTime      time.Duration
	TotalRemaining time.Duration
}
