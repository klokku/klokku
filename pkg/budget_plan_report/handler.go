package budget_plan_report

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/klokku/klokku/internal/rest"
)

type ReportItemDTO struct {
	BudgetItemId   int    `json:"budgetItemId"`
	Name           string `json:"name"`
	Icon           string `json:"icon"`
	Color          string `json:"color"`
	Position       int    `json:"position"`
	BudgetPlanTime int    `json:"budgetPlanTime"`
	WeeklyPlanTime int    `json:"weeklyPlanTime"`
	ActualTime     int    `json:"actualTime"`
	AveragePerWeek int    `json:"averagePerWeek"`
	AveragePerDay  int    `json:"averagePerDay"`
}

type WeeklyReportEntryDTO struct {
	WeekNumber          string          `json:"weekNumber"`
	StartDate           time.Time       `json:"startDate"`
	EndDate             time.Time       `json:"endDate"`
	Items               []ReportItemDTO `json:"items"`
	TotalBudgetPlanTime int             `json:"totalBudgetPlanTime"`
	TotalWeeklyPlanTime int             `json:"totalWeeklyPlanTime"`
	TotalActualTime     int             `json:"totalActualTime"`
}

type ReportTotalsDTO struct {
	Items               []ReportItemDTO `json:"items"`
	TotalBudgetPlanTime int             `json:"totalBudgetPlanTime"`
	TotalWeeklyPlanTime int             `json:"totalWeeklyPlanTime"`
	TotalActualTime     int             `json:"totalActualTime"`
}

type ReportDTO struct {
	PlanId    int                    `json:"planId"`
	PlanName  string                 `json:"planName"`
	StartDate time.Time              `json:"startDate"`
	EndDate   time.Time              `json:"endDate"`
	WeekCount int                    `json:"weekCount"`
	Weeks     []WeeklyReportEntryDTO `json:"weeks"`
	Totals    ReportTotalsDTO        `json:"totals"`
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// GetReport godoc
// @Summary Get budget plan report
// @Description Retrieve report for a budget plan. Without from/to params returns full lifetime data. With from/to returns data for the specified period.
// @Tags BudgetPlanReport
// @Produce json
// @Param planId path int true "Budget Plan ID"
// @Param from query string false "Start date in RFC3339 format (must be provided together with 'to')"
// @Param to query string false "End date in RFC3339 format (must be provided together with 'from')"
// @Success 200 {object} ReportDTO
// @Failure 400 {object} rest.ErrorResponse "Invalid parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /api/budgetplan/{planId}/report [get]
// @Security XUserId
func (h *Handler) GetReport(w http.ResponseWriter, r *http.Request) {
	planId, err := parsePlanId(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Invalid plan ID",
			Details: "planId must be an integer",
		})
		return
	}

	query := r.URL.Query()
	fromStr := query.Get("from")
	toStr := query.Get("to")

	var from, to *time.Time

	if fromStr != "" && toStr != "" {
		fromTime, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(rest.ErrorResponse{
				Error:   "Invalid 'from' date format",
				Details: "date must be in RFC3339 format",
			})
			return
		}
		toTime, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(rest.ErrorResponse{
				Error:   "Invalid 'to' date format",
				Details: "date must be in RFC3339 format",
			})
			return
		}
		from = &fromTime
		to = &toTime
	} else if fromStr != "" || toStr != "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Both 'from' and 'to' must be provided",
			Details: "provide both 'from' and 'to' query parameters, or neither for full lifetime report",
		})
		return
	}

	report, err := h.service.GetReport(r.Context(), planId, from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(reportToDTO(report))
}

func parsePlanId(r *http.Request) (int, error) {
	vars := mux.Vars(r)
	return strconv.Atoi(vars["planId"])
}

func reportToDTO(report Report) ReportDTO {
	weeks := make([]WeeklyReportEntryDTO, 0, len(report.Weeks))
	for _, w := range report.Weeks {
		weeks = append(weeks, weeklyEntryToDTO(w))
	}

	return ReportDTO{
		PlanId:    report.PlanId,
		PlanName:  report.PlanName,
		StartDate: report.StartDate,
		EndDate:   report.EndDate,
		WeekCount: report.WeekCount,
		Weeks:     weeks,
		Totals: ReportTotalsDTO{
			Items:               reportItemsToDTO(report.TotalItems),
			TotalBudgetPlanTime: int(report.TotalBudgetPlanTime.Seconds()),
			TotalWeeklyPlanTime: int(report.TotalWeeklyPlanTime.Seconds()),
			TotalActualTime:     int(report.TotalActualTime.Seconds()),
		},
	}
}

func reportItemToDTO(item ReportItem) ReportItemDTO {
	return ReportItemDTO{
		BudgetItemId:   item.BudgetItemId,
		Name:           item.Name,
		Icon:           item.Icon,
		Color:          item.Color,
		Position:       item.Position,
		BudgetPlanTime: int(item.BudgetPlanTime.Seconds()),
		WeeklyPlanTime: int(item.WeeklyPlanTime.Seconds()),
		ActualTime:     int(item.ActualTime.Seconds()),
		AveragePerWeek: int(item.AveragePerWeek.Seconds()),
		AveragePerDay:  int(item.AveragePerDay.Seconds()),
	}
}

func reportItemsToDTO(items []ReportItem) []ReportItemDTO {
	dtos := make([]ReportItemDTO, 0, len(items))
	for _, item := range items {
		dtos = append(dtos, reportItemToDTO(item))
	}
	return dtos
}

func weeklyEntryToDTO(entry WeeklyReportEntry) WeeklyReportEntryDTO {
	return WeeklyReportEntryDTO{
		WeekNumber:          entry.WeekNumber,
		StartDate:           entry.StartDate,
		EndDate:             entry.EndDate,
		Items:               reportItemsToDTO(entry.Items),
		TotalBudgetPlanTime: int(entry.TotalBudgetPlanTime.Seconds()),
		TotalWeeklyPlanTime: int(entry.TotalWeeklyPlanTime.Seconds()),
		TotalActualTime:     int(entry.TotalActualTime.Seconds()),
	}
}
