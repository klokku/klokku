package budget_plan_report

import (
	"encoding/json"
	"fmt"
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
	PlanId            int                    `json:"planId"`
	PlanName          string                 `json:"planName"`
	StartDate         time.Time              `json:"startDate"`
	EndDate           time.Time              `json:"endDate"`
	WeekCount         int                    `json:"weekCount"`
	ExcludedWeekCount int                    `json:"excludedWeekCount"`
	Weeks             []WeeklyReportEntryDTO `json:"weeks"`
	Totals            ReportTotalsDTO        `json:"totals"`
}

// --- Item Detail Report DTOs ---

type ItemDetailReportDTO struct {
	PlanId    int    `json:"planId"`
	PlanName  string `json:"planName"`
	ItemId    int    `json:"itemId"`
	ItemName  string `json:"itemName"`
	ItemIcon  string `json:"itemIcon"`
	ItemColor string `json:"itemColor"`

	StartDate time.Time `json:"startDate"`
	EndDate   time.Time `json:"endDate"`

	TotalActualTime     int     `json:"totalActualTime"`
	TotalBudgetPlanTime int     `json:"totalBudgetPlanTime"`
	TotalWeeklyPlanTime int     `json:"totalWeeklyPlanTime"`
	CompletionPercent   float64 `json:"completionPercent"`
	RemainingTime       int     `json:"remainingTime"`
	OverBudgetTime      int     `json:"overBudgetTime"`

	AveragePerDay       int `json:"averagePerDay"`
	AveragePerActiveDay int `json:"averagePerActiveDay"`
	AveragePerWeek      int `json:"averagePerWeek"`
	MedianPerDay        int `json:"medianPerDay"`
	MedianPerActiveDay  int `json:"medianPerActiveDay"`
	MedianPerWeek       int `json:"medianPerWeek"`

	ActiveDaysCount   int `json:"activeDaysCount"`
	TotalDaysCount    int `json:"totalDaysCount"`
	WeekCount         int `json:"weekCount"`
	ExcludedWeekCount int `json:"excludedWeekCount"`

	Weeks        []ItemWeekEntryDTO  `json:"weeks"`
	Days         []ItemDayEntryDTO   `json:"days"`
	DayOfWeekAvg []DayOfWeekEntryDTO `json:"dayOfWeekAvg"`
}

type ItemWeekEntryDTO struct {
	WeekNumber     string    `json:"weekNumber"`
	StartDate      time.Time `json:"startDate"`
	EndDate        time.Time `json:"endDate"`
	BudgetPlanTime int       `json:"budgetPlanTime"`
	WeeklyPlanTime int       `json:"weeklyPlanTime"`
	ActualTime     int       `json:"actualTime"`
	IsOffWeek      bool      `json:"isOffWeek"`
}

type ItemDayEntryDTO struct {
	Date       time.Time `json:"date"`
	ActualTime int       `json:"actualTime"`
	DayOfWeek  int       `json:"dayOfWeek"`
}

type DayOfWeekEntryDTO struct {
	DayOfWeek   int `json:"dayOfWeek"`
	AverageTime int `json:"averageTime"`
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
		writeBadRequest(w, "Invalid plan ID", "planId must be an integer")
		return
	}

	from, to, err := parseDateRange(r)
	if err != nil {
		writeBadRequest(w, err.Error(), "provide both 'from' and 'to' query parameters in RFC3339 format, or neither for full lifetime report")
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

// GetItemReport godoc
// @Summary Get detailed report for a single budget plan item
// @Description Retrieve detailed statistics, weekly breakdown, daily breakdown, and day-of-week averages for a single budget plan item.
// @Tags BudgetPlanReport
// @Produce json
// @Param planId path int true "Budget Plan ID"
// @Param itemId path int true "Budget Item ID"
// @Param from query string false "Start date in RFC3339 format (must be provided together with 'to')"
// @Param to query string false "End date in RFC3339 format (must be provided together with 'from')"
// @Success 200 {object} ItemDetailReportDTO
// @Failure 400 {object} rest.ErrorResponse "Invalid parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /api/budgetplan/{planId}/report/item/{itemId} [get]
// @Security XUserId
func (h *Handler) GetItemReport(w http.ResponseWriter, r *http.Request) {
	planId, err := parsePlanId(r)
	if err != nil {
		writeBadRequest(w, "Invalid plan ID", "planId must be an integer")
		return
	}

	vars := mux.Vars(r)
	itemId, err := strconv.Atoi(vars["itemId"])
	if err != nil {
		writeBadRequest(w, "Invalid item ID", "itemId must be an integer")
		return
	}

	from, to, err := parseDateRange(r)
	if err != nil {
		writeBadRequest(w, err.Error(), "provide both 'from' and 'to' query parameters in RFC3339 format, or neither for full lifetime report")
		return
	}

	report, err := h.service.GetItemReport(r.Context(), planId, itemId, from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(itemDetailReportToDTO(report))
}

func parsePlanId(r *http.Request) (int, error) {
	vars := mux.Vars(r)
	return strconv.Atoi(vars["planId"])
}

func parseDateRange(r *http.Request) (*time.Time, *time.Time, error) {
	query := r.URL.Query()
	fromStr := query.Get("from")
	toStr := query.Get("to")

	if fromStr == "" && toStr == "" {
		return nil, nil, nil
	}
	if fromStr == "" || toStr == "" {
		return nil, nil, fmt.Errorf("both 'from' and 'to' must be provided")
	}

	fromTime, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid 'from' date format")
	}
	toTime, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid 'to' date format")
	}
	return &fromTime, &toTime, nil
}

func writeBadRequest(w http.ResponseWriter, errMsg, details string) {
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(rest.ErrorResponse{
		Error:   errMsg,
		Details: details,
	})
}

func reportToDTO(report Report) ReportDTO {
	weeks := make([]WeeklyReportEntryDTO, 0, len(report.Weeks))
	for _, w := range report.Weeks {
		weeks = append(weeks, weeklyEntryToDTO(w))
	}

	return ReportDTO{
		PlanId:            report.PlanId,
		PlanName:          report.PlanName,
		StartDate:         report.StartDate,
		EndDate:           report.EndDate,
		WeekCount:         report.WeekCount,
		ExcludedWeekCount: report.ExcludedWeekCount,
		Weeks:             weeks,
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

func itemDetailReportToDTO(report ItemDetailReport) ItemDetailReportDTO {
	weeks := make([]ItemWeekEntryDTO, 0, len(report.Weeks))
	for _, w := range report.Weeks {
		weeks = append(weeks, ItemWeekEntryDTO{
			WeekNumber:     w.WeekNumber,
			StartDate:      w.StartDate,
			EndDate:        w.EndDate,
			BudgetPlanTime: int(w.BudgetPlanTime.Seconds()),
			WeeklyPlanTime: int(w.WeeklyPlanTime.Seconds()),
			ActualTime:     int(w.ActualTime.Seconds()),
			IsOffWeek:      w.IsOffWeek,
		})
	}

	days := make([]ItemDayEntryDTO, 0, len(report.Days))
	for _, d := range report.Days {
		days = append(days, ItemDayEntryDTO{
			Date:       d.Date,
			ActualTime: int(d.ActualTime.Seconds()),
			DayOfWeek:  int(d.DayOfWeek),
		})
	}

	dayOfWeekAvg := make([]DayOfWeekEntryDTO, 0, len(report.DayOfWeekAvg))
	for _, d := range report.DayOfWeekAvg {
		dayOfWeekAvg = append(dayOfWeekAvg, DayOfWeekEntryDTO{
			DayOfWeek:   int(d.DayOfWeek),
			AverageTime: int(d.AverageTime.Seconds()),
		})
	}

	return ItemDetailReportDTO{
		PlanId:              report.PlanId,
		PlanName:            report.PlanName,
		ItemId:              report.ItemId,
		ItemName:            report.ItemName,
		ItemIcon:            report.ItemIcon,
		ItemColor:           report.ItemColor,
		StartDate:           report.StartDate,
		EndDate:             report.EndDate,
		TotalActualTime:     int(report.TotalActualTime.Seconds()),
		TotalBudgetPlanTime: int(report.TotalBudgetPlanTime.Seconds()),
		TotalWeeklyPlanTime: int(report.TotalWeeklyPlanTime.Seconds()),
		CompletionPercent:   report.CompletionPercent,
		RemainingTime:       int(report.RemainingTime.Seconds()),
		OverBudgetTime:      int(report.OverBudgetTime.Seconds()),
		AveragePerDay:       int(report.AveragePerDay.Seconds()),
		AveragePerActiveDay: int(report.AveragePerActiveDay.Seconds()),
		AveragePerWeek:      int(report.AveragePerWeek.Seconds()),
		MedianPerDay:        int(report.MedianPerDay.Seconds()),
		MedianPerActiveDay:  int(report.MedianPerActiveDay.Seconds()),
		MedianPerWeek:       int(report.MedianPerWeek.Seconds()),
		ActiveDaysCount:     report.ActiveDaysCount,
		TotalDaysCount:      report.TotalDaysCount,
		WeekCount:           report.WeekCount,
		ExcludedWeekCount:   report.ExcludedWeekCount,
		Weeks:               weeks,
		Days:                days,
		DayOfWeekAvg:        dayOfWeekAvg,
	}
}
