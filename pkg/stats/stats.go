package stats

import (
	"github.com/klokku/klokku/pkg/budget_override"
	"github.com/klokku/klokku/pkg/budget_plan"
	"time"
)

type DailyStats struct {
	Date      time.Time
	Budgets   []BudgetStats
	TotalTime time.Duration
}

type BudgetStats struct {
	Budget         budget_plan.BudgetItem
	BudgetOverride *budget_override.BudgetOverride
	Duration       time.Duration
	Remaining      time.Duration
}

type StatsSummary struct {
	StartDate      time.Time
	EndDate        time.Time
	Days           []DailyStats
	Budgets        []BudgetStats
	TotalPlanned   time.Duration
	TotalTime      time.Duration
	TotalRemaining time.Duration
}
