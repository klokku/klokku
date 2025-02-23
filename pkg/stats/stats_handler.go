package stats

import (
	"encoding/json"
	"github.com/klokku/klokku/internal/rest"
	"github.com/klokku/klokku/pkg/budget"
	"net/http"
	"time"
)

type DailyStatsDTO struct {
	Date      time.Time        `json:"date"`
	Budgets   []BudgetStatsDTO `json:"budgets"`
	TotalTime int              `json:"totalTime"`
}

type BudgetStatsDTO struct {
	Budget         budget.BudgetDTO   `json:"budget"`
	BudgetOverride *BudgetOverrideDTO `json:"budgetOverride,omitempty"`
	Duration       int                `json:"duration"`
	Remaining      int                `json:"remaining"`
}

type BudgetOverrideDTO struct {
	ID         int       `json:"id"`
	BudgetID   int       `json:"budgetId"`
	StartDate  time.Time `json:"startDate"`
	WeeklyTime int       `json:"weeklyTime"`
	Notes      *string   `json:"notes"`
}

type StatsSummaryDTO struct {
	StartDate      time.Time        `json:"startDate"`
	EndDate        time.Time        `json:"endDate"`
	Days           []DailyStatsDTO  `json:"days"`
	Budgets        []BudgetStatsDTO `json:"budgets"`
	TotalPlanned   int              `json:"totalPlanned"`
	TotalTime      int              `json:"totalTime"`
	TotalRemaining int              `json:"totalRemaining"`
}

type StatsHandler struct {
	statsService     StatsService
	csvStatsRenderer StatsRenderer
}

func NewStatsHandler(statsService StatsService, csvStatsRenderer StatsRenderer) *StatsHandler {
	return &StatsHandler{statsService, csvStatsRenderer}
}

func (handler *StatsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	fromDateString := r.URL.Query().Get("fromDate")
	toDateString := r.URL.Query().Get("toDate")
	fromDate, err := time.Parse(time.RFC3339, fromDateString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Invalid fromDate format",
			Details: "fromDate must be in RFC3339 format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
	toDate, err := time.Parse(time.RFC3339, toDateString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Invalid toDate format",
			Details: "toDate must be in RFC3339 format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
	stats, err := handler.statsService.GetStats(r.Context(), fromDate, toDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("Accept") == "text/csv" {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		csv, err := handler.csvStatsRenderer.RenderStats(stats)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err := w.Write([]byte(csv)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	} else {
		w.Header().Set("Content-Type", "application/json")
		responseStats := convertToJsonResponse(&stats)
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(responseStats); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
}

func convertToJsonResponse(stats *StatsSummary) *StatsSummaryDTO {
	budgetStats := make([]BudgetStatsDTO, 0, len(stats.Budgets))
	for _, budgetStat := range stats.Budgets {
		budgetStatsDTO := BudgetStatsDTO{
			Budget:    budget.BudgetToDTO(budgetStat.Budget),
			Duration:  int(budgetStat.Duration.Seconds()),
			Remaining: int(budgetStat.Remaining.Seconds()),
		}
		if budgetStat.BudgetOverride != nil {
			budgetStatsDTO.BudgetOverride = &BudgetOverrideDTO{
				ID:         budgetStat.BudgetOverride.ID,
				BudgetID:   budgetStat.BudgetOverride.BudgetID,
				StartDate:  budgetStat.BudgetOverride.StartDate,
				WeeklyTime: int(budgetStat.BudgetOverride.WeeklyTime.Seconds()),
				Notes:      budgetStat.BudgetOverride.Notes,
			}
		}

		budgetStats = append(budgetStats, budgetStatsDTO)
	}

	days := make([]DailyStatsDTO, 0, len(stats.Days))
	for _, day := range stats.Days {
		dailyStatsDTO := DailyStatsDTO{
			Date:      day.Date,
			TotalTime: int(day.TotalTime.Seconds()),
		}
		for _, dayBudget := range day.Budgets {
			budgetStatsDTO := BudgetStatsDTO{
				Budget:    budget.BudgetToDTO(dayBudget.Budget),
				Duration:  int(dayBudget.Duration.Seconds()),
				Remaining: int(dayBudget.Remaining.Seconds()),
			}
			if dayBudget.BudgetOverride != nil {
				budgetStatsDTO.BudgetOverride = &BudgetOverrideDTO{
					ID:         dayBudget.BudgetOverride.ID,
					BudgetID:   dayBudget.BudgetOverride.BudgetID,
					StartDate:  dayBudget.BudgetOverride.StartDate,
					WeeklyTime: int(dayBudget.BudgetOverride.WeeklyTime.Seconds()),
					Notes:      dayBudget.BudgetOverride.Notes,
				}
			}
			dailyStatsDTO.Budgets = append(dailyStatsDTO.Budgets, budgetStatsDTO)
		}
		days = append(days, dailyStatsDTO)
	}

	return &StatsSummaryDTO{
		StartDate:      stats.StartDate,
		EndDate:        stats.EndDate,
		Days:           days,
		Budgets:        budgetStats,
		TotalPlanned:   int(stats.TotalPlanned.Seconds()),
		TotalTime:      int(stats.TotalTime.Seconds()),
		TotalRemaining: int(stats.TotalRemaining.Seconds()),
	}

}
