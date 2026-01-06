package stats

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/klokku/klokku/internal/rest"
)

type DailyStatsDTO struct {
	Date        time.Time          `json:"date"`
	PerPlanItem []PlanItemStatsDTO `json:"perPlanItem"`
	TotalTime   int                `json:"totalTime"`
}

type PlanItemDTO struct {
	BudgetPlanId       int    `json:"budgetPlanId"`
	BudgetItemId       int    `json:"budgetItemId"`
	WeeklyItemId       int    `json:"weeklyItemId"`
	Name               string `json:"name"`
	Icon               string `json:"icon"`
	Color              string `json:"color"`
	Position           int    `json:"position"`
	WeeklyItemDuration int    `json:"weeklyItemDuration"`
	BudgetItemDuration int    `json:"budgetItemDuration"`
	WeeklyOccurrences  int    `json:"weeklyOccurrences"`
	Notes              string `json:"notes"`
}

type PlanItemStatsDTO struct {
	PlanItem  PlanItemDTO `json:"planItem"`
	Duration  int         `json:"duration"`
	Remaining int         `json:"remaining"`
	StartDate time.Time   `json:"startDate"`
	EndDate   time.Time   `json:"endDate"`
}

type WeeklyStatsSummaryDTO struct {
	StartDate      time.Time          `json:"startDate"`
	EndDate        time.Time          `json:"endDate"`
	PerDay         []DailyStatsDTO    `json:"perDay"`
	PerPlanItem    []PlanItemStatsDTO `json:"perPlanItem"`
	TotalPlanned   int                `json:"totalPlanned"`
	TotalTime      int                `json:"totalTime"`
	TotalRemaining int                `json:"totalRemaining"`
}

type PlanItemHistoryStatsDTO struct {
	StartDate    time.Time          `json:"startDate"`
	EndDate      time.Time          `json:"endDate"`
	StatsPerWeek []PlanItemStatsDTO `json:"statsPerWeek"`
}

type StatsHandler struct {
	statsService StatsService
}

func NewStatsHandler(statsService StatsService) *StatsHandler {
	return &StatsHandler{statsService}
}

// GetWeeklyStats godoc
// @Summary Get weekly statistics
// @Description Retrieve statistics for a specific week including time spent per plan item
// @Tags Stats
// @Produce json
// @Param date query string true "Date in RFC3339 format (can be any day of the week)"
// @Success 200 {object} WeeklyStatsSummaryDTO
// @Failure 400 {object} rest.ErrorResponse "Invalid date format"
// @Failure 403 {string} string "User not found"
// @Router /api/stats/weekly [get]
// @Security XUserId
func (handler *StatsHandler) GetWeeklyStats(w http.ResponseWriter, r *http.Request) {
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
	stats, err := handler.statsService.GetWeeklyStats(r.Context(), weekDate)
	if err != nil {
		if errors.Is(err, ErrNoStatsFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
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

func statsSummaryToDTO(stats *WeeklyStatsSummary) *WeeklyStatsSummaryDTO {
	budgetStats := make([]PlanItemStatsDTO, 0, len(stats.PerPlanItem))
	for _, planItemStats := range stats.PerPlanItem {
		budgetStatsDTO := planItemStatsToDTO(planItemStats)
		budgetStats = append(budgetStats, budgetStatsDTO)
	}

	days := make([]DailyStatsDTO, 0, len(stats.PerDay))
	for _, day := range stats.PerDay {
		dailyStatsDTO := DailyStatsDTO{
			Date:      day.Date,
			TotalTime: int(day.TotalTime.Seconds()),
		}
		for _, dayItemStats := range day.StatsPerPlanItem {
			budgetStatsDTO := planItemStatsToDTO(dayItemStats)
			dailyStatsDTO.PerPlanItem = append(dailyStatsDTO.PerPlanItem, budgetStatsDTO)
		}
		days = append(days, dailyStatsDTO)
	}

	return &WeeklyStatsSummaryDTO{
		StartDate:      stats.StartDate,
		EndDate:        stats.EndDate,
		PerDay:         days,
		PerPlanItem:    budgetStats,
		TotalPlanned:   int(stats.TotalPlanned.Seconds()),
		TotalTime:      int(stats.TotalTime.Seconds()),
		TotalRemaining: int(stats.TotalRemaining.Seconds()),
	}
}

func planItemStatsToDTO(itemStats PlanItemStats) PlanItemStatsDTO {
	return PlanItemStatsDTO{
		PlanItem:  planItemToDTO(itemStats.PlanItem),
		Duration:  int(itemStats.Duration.Seconds()),
		Remaining: int(itemStats.Remaining.Seconds()),
		StartDate: itemStats.StartDate,
		EndDate:   itemStats.EndDate,
	}
}

func planItemToDTO(planItem PlanItem) PlanItemDTO {
	return PlanItemDTO{
		BudgetPlanId:       planItem.BudgetPlanId,
		BudgetItemId:       planItem.BudgetItemId,
		WeeklyItemId:       planItem.WeeklyItemId,
		Name:               planItem.Name,
		Icon:               planItem.Icon,
		Color:              planItem.Color,
		Position:           planItem.Position,
		WeeklyItemDuration: int(planItem.WeeklyItemDuration.Seconds()),
		BudgetItemDuration: int(planItem.BudgetItemDuration.Seconds()),
		WeeklyOccurrences:  planItem.WeeklyOccurrences,
		Notes:              planItem.Notes,
	}
}

// GetPlanItemByWeekHistoryStats godoc
// @Summary Get historical statistics for a specific budget item
// @Description Retrieve statistics for a specific budget item by week for a given period
// @Tags Stats
// @Produce json
// @Param from query string true "Start date in RFC3339 format"
// @Param to query string true "End date in RFC3339 format"
// @Param budgetItemId query int true "Budget Item ID"
// @Success 200 {object} PlanItemHistoryStatsDTO
// @Failure 400 {object} rest.ErrorResponse "Invalid parameters"
// @Failure 403 {string} string "User not found"
// @Router /api/stats/item-history [get]
// @Security XUserId
func (handler *StatsHandler) GetPlanItemByWeekHistoryStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := r.URL.Query()

	fromStr := query.Get("from")
	toStr := query.Get("to")
	budgetItemIdStr := query.Get("budgetItemId")

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Invalid 'from' date format",
			Details: "date must be in RFC3339 format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Invalid 'to' date format",
			Details: "date must be in RFC3339 format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	budgetItemId, err := strconv.Atoi(budgetItemIdStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Invalid 'budgetItemId' format",
			Details: "budgetItemId must be an integer",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	stats, err := handler.statsService.GetPlanItemByWeekHistoryStats(ctx, from, to, budgetItemId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	statsDTO := planItemHistoryStatsToDTO(stats)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(statsDTO); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	return
}

func planItemHistoryStatsToDTO(stats PlanItemHistoryStats) PlanItemHistoryStatsDTO {

	statsPerWeek := make([]PlanItemStatsDTO, 0, len(stats.StatsPerWeek))
	for _, weekStats := range stats.StatsPerWeek {
		statsPerWeek = append(statsPerWeek, planItemStatsToDTO(weekStats))
	}

	return PlanItemHistoryStatsDTO{
		StartDate:    stats.StartDate,
		EndDate:      stats.EndDate,
		StatsPerWeek: statsPerWeek,
	}
}
