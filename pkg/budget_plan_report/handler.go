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

type MonthlyReportEntryDTO struct {
	PeriodNumber        int             `json:"periodNumber"`
	StartDate           time.Time       `json:"startDate"`
	EndDate             time.Time       `json:"endDate"`
	WeekCount           int             `json:"weekCount"`
	Items               []ReportItemDTO `json:"items"`
	TotalBudgetPlanTime int             `json:"totalBudgetPlanTime"`
	TotalWeeklyPlanTime int             `json:"totalWeeklyPlanTime"`
	TotalActualTime     int             `json:"totalActualTime"`
}

type SummaryReportDTO struct {
	PlanId              int             `json:"planId"`
	PlanName            string          `json:"planName"`
	StartDate           time.Time       `json:"startDate"`
	EndDate             time.Time       `json:"endDate"`
	WeekCount           int             `json:"weekCount"`
	Items               []ReportItemDTO `json:"items"`
	TotalBudgetPlanTime int             `json:"totalBudgetPlanTime"`
	TotalWeeklyPlanTime int             `json:"totalWeeklyPlanTime"`
	TotalActualTime     int             `json:"totalActualTime"`
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// GetSummaryReport godoc
// @Summary Get budget plan summary report
// @Description Retrieve aggregated totals for the entire lifetime of a budget plan
// @Tags BudgetPlanReport
// @Produce json
// @Param planId path int true "Budget Plan ID"
// @Success 200 {object} SummaryReportDTO
// @Failure 400 {object} rest.ErrorResponse "Invalid plan ID"
// @Failure 500 {string} string "Internal server error"
// @Router /api/budgetplan/{planId}/report/summary [get]
// @Security XUserId
func (h *Handler) GetSummaryReport(w http.ResponseWriter, r *http.Request) {
	planId, err := parsePlanId(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Invalid plan ID",
			Details: "planId must be an integer",
		})
		return
	}

	report, err := h.service.GetSummaryReport(r.Context(), planId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(summaryReportToDTO(report))
}

// GetWeeklyReport godoc
// @Summary Get budget plan weekly report
// @Description Retrieve per-week breakdown for the entire lifetime of a budget plan
// @Tags BudgetPlanReport
// @Produce json
// @Param planId path int true "Budget Plan ID"
// @Success 200 {array} WeeklyReportEntryDTO
// @Failure 400 {object} rest.ErrorResponse "Invalid plan ID"
// @Failure 500 {string} string "Internal server error"
// @Router /api/budgetplan/{planId}/report/weekly [get]
// @Security XUserId
func (h *Handler) GetWeeklyReport(w http.ResponseWriter, r *http.Request) {
	planId, err := parsePlanId(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Invalid plan ID",
			Details: "planId must be an integer",
		})
		return
	}

	entries, err := h.service.GetWeeklyReport(r.Context(), planId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dtos := make([]WeeklyReportEntryDTO, 0, len(entries))
	for _, e := range entries {
		dtos = append(dtos, weeklyEntryToDTO(e))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(dtos)
}

// GetMonthlyReport godoc
// @Summary Get budget plan monthly (4-week) report
// @Description Retrieve per-4-week-period breakdown for the entire lifetime of a budget plan
// @Tags BudgetPlanReport
// @Produce json
// @Param planId path int true "Budget Plan ID"
// @Success 200 {array} MonthlyReportEntryDTO
// @Failure 400 {object} rest.ErrorResponse "Invalid plan ID"
// @Failure 500 {string} string "Internal server error"
// @Router /api/budgetplan/{planId}/report/monthly [get]
// @Security XUserId
func (h *Handler) GetMonthlyReport(w http.ResponseWriter, r *http.Request) {
	planId, err := parsePlanId(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Invalid plan ID",
			Details: "planId must be an integer",
		})
		return
	}

	entries, err := h.service.GetMonthlyReport(r.Context(), planId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dtos := make([]MonthlyReportEntryDTO, 0, len(entries))
	for _, e := range entries {
		dtos = append(dtos, monthlyEntryToDTO(e))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(dtos)
}

func parsePlanId(r *http.Request) (int, error) {
	vars := mux.Vars(r)
	return strconv.Atoi(vars["planId"])
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
	}
}

func reportItemsToDTO(items []ReportItem) []ReportItemDTO {
	dtos := make([]ReportItemDTO, 0, len(items))
	for _, item := range items {
		dtos = append(dtos, reportItemToDTO(item))
	}
	return dtos
}

func summaryReportToDTO(report SummaryReport) SummaryReportDTO {
	return SummaryReportDTO{
		PlanId:              report.PlanId,
		PlanName:            report.PlanName,
		StartDate:           report.StartDate,
		EndDate:             report.EndDate,
		WeekCount:           report.WeekCount,
		Items:               reportItemsToDTO(report.Items),
		TotalBudgetPlanTime: int(report.TotalBudgetPlanTime.Seconds()),
		TotalWeeklyPlanTime: int(report.TotalWeeklyPlanTime.Seconds()),
		TotalActualTime:     int(report.TotalActualTime.Seconds()),
	}
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

func monthlyEntryToDTO(entry MonthlyReportEntry) MonthlyReportEntryDTO {
	return MonthlyReportEntryDTO{
		PeriodNumber:        entry.PeriodNumber,
		StartDate:           entry.StartDate,
		EndDate:             entry.EndDate,
		WeekCount:           entry.WeekCount,
		Items:               reportItemsToDTO(entry.Items),
		TotalBudgetPlanTime: int(entry.TotalBudgetPlanTime.Seconds()),
		TotalWeeklyPlanTime: int(entry.TotalWeeklyPlanTime.Seconds()),
		TotalActualTime:     int(entry.TotalActualTime.Seconds()),
	}
}
