package stats

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/klokku/klokku/internal/rest"
	"github.com/klokku/klokku/pkg/weekly_plan"
)

type DailyStatsDTO struct {
	Date        time.Time          `json:"date"`
	PerPlanItem []PlanItemStatsDTO `json:"perPlanItem"`
	TotalTime   int                `json:"totalTime"`
}

type PlanItemStatsDTO struct {
	WeeklyPlanItem weekly_plan.WeeklyPlanItemDTO `json:"weeklyPlanItem"`
	Duration       int                           `json:"duration"`
	Remaining      int                           `json:"remaining"`
}

type StatsSummaryDTO struct {
	StartDate      time.Time          `json:"startDate"`
	EndDate        time.Time          `json:"endDate"`
	PerDay         []DailyStatsDTO    `json:"perDay"`
	PerPlanItem    []PlanItemStatsDTO `json:"perPlanItem"`
	TotalPlanned   int                `json:"totalPlanned"`
	TotalTime      int                `json:"totalTime"`
	TotalRemaining int                `json:"totalRemaining"`
}

type StatsHandler struct {
	statsService StatsService
}

func NewStatsHandler(statsService StatsService) *StatsHandler {
	return &StatsHandler{statsService}
}

func (handler *StatsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	weekDateString := r.URL.Query().Get("date")
	weekDate, err := time.Parse(time.RFC3339, weekDateString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Invalid date format",
			Details: "date must be in RFC3339 format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
	stats, err := handler.statsService.GetStats(r.Context(), weekDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	statsSummaryDTO := statsSummaryToDTO(&stats)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(statsSummaryDTO); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	return
}

func statsSummaryToDTO(stats *StatsSummary) *StatsSummaryDTO {
	budgetStats := make([]PlanItemStatsDTO, 0, len(stats.PerPlanItem))
	for _, budgetStat := range stats.PerPlanItem {
		budgetStatsDTO := PlanItemStatsDTO{
			WeeklyPlanItem: weekly_plan.WeeklyPlanItemToDTO(budgetStat.PlanItem),
			Duration:       int(budgetStat.Duration.Seconds()),
			Remaining:      int(budgetStat.Remaining.Seconds()),
		}
		budgetStats = append(budgetStats, budgetStatsDTO)
	}

	days := make([]DailyStatsDTO, 0, len(stats.PerDay))
	for _, day := range stats.PerDay {
		dailyStatsDTO := DailyStatsDTO{
			Date:      day.Date,
			TotalTime: int(day.TotalTime.Seconds()),
		}
		for _, dayBudget := range day.StatsPerPlanItem {
			budgetStatsDTO := PlanItemStatsDTO{
				WeeklyPlanItem: weekly_plan.WeeklyPlanItemToDTO(dayBudget.PlanItem),
				Duration:       int(dayBudget.Duration.Seconds()),
				Remaining:      int(dayBudget.Remaining.Seconds()),
			}
			dailyStatsDTO.PerPlanItem = append(dailyStatsDTO.PerPlanItem, budgetStatsDTO)
		}
		days = append(days, dailyStatsDTO)
	}

	return &StatsSummaryDTO{
		StartDate:      stats.StartDate,
		EndDate:        stats.EndDate,
		PerDay:         days,
		PerPlanItem:    budgetStats,
		TotalPlanned:   int(stats.TotalPlanned.Seconds()),
		TotalTime:      int(stats.TotalTime.Seconds()),
		TotalRemaining: int(stats.TotalRemaining.Seconds()),
	}

}
