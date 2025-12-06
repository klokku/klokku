package event

import (
	"time"

	"github.com/klokku/klokku/pkg/budget"
)

type Event struct {
	UID       string
	Budget    budget.Budget
	StartTime time.Time
	EndTime   time.Time
}

type Stats struct {
	StartDate time.Time
	EndDate   time.Time
	ByDate    map[time.Time]map[int]time.Duration
	ByBudget  map[int]time.Duration
}
