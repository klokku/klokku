package api

import "encoding/json"

// ErrorResponse mirrors the server's rest.ErrorResponse.
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// --- User ---

type UserDTO struct {
	UID         string      `json:"uid"`
	Username    string      `json:"username"`
	DisplayName string      `json:"displayName"`
	PhotoURL    string      `json:"photoUrl"`
	Settings    SettingsDTO `json:"settings"`
}

type SettingsDTO struct {
	Timezone          string                    `json:"timezone"`
	WeekStartDay      string                    `json:"weekStartDay"`
	EventCalendarType string                    `json:"eventCalendarType"`
	GoogleCalendar    GoogleCalendarSettingsDTO `json:"googleCalendar"`
	IgnoreShortEvents bool                      `json:"ignoreShortEvents"`
}

type GoogleCalendarSettingsDTO struct {
	CalendarID string `json:"calendarId"`
}

// --- Budget Plan ---

type BudgetPlanDTO struct {
	ID        int             `json:"id"`
	Name      string          `json:"name"`
	IsCurrent bool            `json:"isCurrent"`
	Items     []BudgetItemDTO `json:"items,omitempty"`
}

type BudgetItemDTO struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	WeeklyDuration    int    `json:"weeklyDuration"`
	WeeklyOccurrences int    `json:"weeklyOccurrences,omitempty"`
	Icon              string `json:"icon,omitempty"`
	Color             string `json:"color,omitempty"`
}

type SetItemPositionRequest struct {
	ID          int `json:"id"`
	PrecedingID int `json:"precedingId"`
}

// --- Weekly Plan ---

type WeeklyPlanDTO struct {
	BudgetPlanID int                 `json:"budgetPlanId"`
	IsOffWeek    bool                `json:"isOffWeek"`
	Items        []WeeklyPlanItemDTO `json:"items"`
}

type WeeklyPlanItemDTO struct {
	ID                int    `json:"id"`
	BudgetItemID      int    `json:"budgetItemId"`
	Name              string `json:"name"`
	WeeklyDuration    int    `json:"weeklyDuration"`
	WeeklyOccurrences int    `json:"weeklyOccurrences"`
	Icon              string `json:"icon"`
	Color             string `json:"color"`
	Notes             string `json:"notes"`
	Position          int    `json:"position"`
}

type UpdateWeeklyItemRequest struct {
	ID             int    `json:"id"`
	BudgetItemID   int    `json:"budgetItemId"`
	WeeklyDuration int    `json:"weeklyDuration"`
	Notes          string `json:"notes"`
}

type SetOffWeekRequest struct {
	IsOffWeek bool `json:"isOffWeek"`
}

// --- Current Event ---

type CurrentEventDTO struct {
	PlanItem  PlanItemDTO `json:"planItem"`
	StartTime string      `json:"startTime"`
}

type PlanItemDTO struct {
	BudgetItemID   int    `json:"budgetItemId"`
	Name           string `json:"name"`
	WeeklyDuration int    `json:"weeklyDuration"`
}

type StartEventRequest struct {
	BudgetItemID   int    `json:"budgetItemId"`
	Name           string `json:"name"`
	WeeklyDuration int    `json:"weeklyDuration"`
}

type AdjustStartRequest struct {
	StartTime string `json:"startTime"`
}

// --- Calendar Event ---

type CalendarEventDTO struct {
	UID          string `json:"uid"`
	Summary      string `json:"summary"`
	Start        string `json:"start"`
	End          string `json:"end"`
	BudgetItemID int    `json:"budgetItemId"`
}

// --- Stats ---

type WeeklyStatsSummaryDTO struct {
	StartDate      string             `json:"startDate"`
	EndDate        string             `json:"endDate"`
	PerDay         []DailyStatsDTO    `json:"perDay"`
	PerPlanItem    []PlanItemStatsDTO `json:"perPlanItem"`
	TotalPlanned   int                `json:"totalPlanned"`
	TotalTime      int                `json:"totalTime"`
	TotalRemaining int                `json:"totalRemaining"`
}

type DailyStatsDTO struct {
	Date        string             `json:"date"`
	PerPlanItem []PlanItemStatsDTO `json:"perPlanItem"`
	TotalTime   int                `json:"totalTime"`
}

type PlanItemStatsDTO struct {
	PlanItem  StatsPlanItemDTO `json:"planItem"`
	Duration  int              `json:"duration"`
	Remaining int              `json:"remaining"`
	StartDate string           `json:"startDate"`
	EndDate   string           `json:"endDate"`
}

type StatsPlanItemDTO struct {
	BudgetPlanID       int    `json:"budgetPlanId"`
	BudgetItemID       int    `json:"budgetItemId"`
	WeeklyItemID       int    `json:"weeklyItemId"`
	Name               string `json:"name"`
	Icon               string `json:"icon"`
	Color              string `json:"color"`
	Position           int    `json:"position"`
	WeeklyItemDuration int    `json:"weeklyItemDuration"`
	BudgetItemDuration int    `json:"budgetItemDuration"`
	WeeklyOccurrences  int    `json:"weeklyOccurrences"`
	Notes              string `json:"notes"`
}

type PlanItemHistoryStatsDTO struct {
	StartDate    string                      `json:"startDate"`
	EndDate      string                      `json:"endDate"`
	StatsPerWeek []WeeklyItemHistoryStatsDTO `json:"statsPerWeek"`
}

type WeeklyItemHistoryStatsDTO struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Duration  int    `json:"duration"`
	Planned   int    `json:"planned"`
}

// --- Reports ---

type ReportDTO struct {
	PlanID            int                    `json:"planId"`
	PlanName          string                 `json:"planName"`
	StartDate         string                 `json:"startDate"`
	EndDate           string                 `json:"endDate"`
	WeekCount         int                    `json:"weekCount"`
	ExcludedWeekCount int                    `json:"excludedWeekCount"`
	Weeks             []WeeklyReportEntryDTO `json:"weeks"`
	Totals            ReportTotalsDTO        `json:"totals"`
}

type WeeklyReportEntryDTO struct {
	WeekNumber          string          `json:"weekNumber"`
	StartDate           string          `json:"startDate"`
	EndDate             string          `json:"endDate"`
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

type ReportItemDTO struct {
	BudgetItemID   int    `json:"budgetItemId"`
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

type ItemDetailReportDTO struct {
	PlanID              int                     `json:"planId"`
	PlanName            string                  `json:"planName"`
	ItemID              int                     `json:"itemId"`
	ItemName            string                  `json:"itemName"`
	ItemIcon            string                  `json:"itemIcon"`
	ItemColor           string                  `json:"itemColor"`
	StartDate           string                  `json:"startDate"`
	EndDate             string                  `json:"endDate"`
	TotalActualTime     int                     `json:"totalActualTime"`
	TotalBudgetPlanTime int                     `json:"totalBudgetPlanTime"`
	TotalWeeklyPlanTime int                     `json:"totalWeeklyPlanTime"`
	CompletionPercent   float64                 `json:"completionPercent"`
	RemainingTime       int                     `json:"remainingTime"`
	OverBudgetTime      int                     `json:"overBudgetTime"`
	AveragePerDay       int                     `json:"averagePerDay"`
	AveragePerActiveDay int                     `json:"averagePerActiveDay"`
	AveragePerWeek      int                     `json:"averagePerWeek"`
	MedianPerDay        int                     `json:"medianPerDay"`
	MedianPerActiveDay  int                     `json:"medianPerActiveDay"`
	MedianPerWeek       int                     `json:"medianPerWeek"`
	ActiveDaysCount     int                     `json:"activeDaysCount"`
	TotalDaysCount      int                     `json:"totalDaysCount"`
	WeekCount           int                     `json:"weekCount"`
	ExcludedWeekCount   int                     `json:"excludedWeekCount"`
	Weeks               []ItemWeekEntryDTO      `json:"weeks"`
	Days                []ItemDayEntryDTO       `json:"days"`
	DayOfWeekAvg        []DayOfWeekEntryDTO     `json:"dayOfWeekAvg"`
	HourlyHeatmap       []HourlyHeatmapEntryDTO `json:"hourlyHeatmap"`
}

type ItemWeekEntryDTO struct {
	WeekNumber     string `json:"weekNumber"`
	StartDate      string `json:"startDate"`
	EndDate        string `json:"endDate"`
	BudgetPlanTime int    `json:"budgetPlanTime"`
	WeeklyPlanTime int    `json:"weeklyPlanTime"`
	ActualTime     int    `json:"actualTime"`
	IsOffWeek      bool   `json:"isOffWeek"`
}

type ItemDayEntryDTO struct {
	Date       string `json:"date"`
	ActualTime int    `json:"actualTime"`
	DayOfWeek  int    `json:"dayOfWeek"`
}

type DayOfWeekEntryDTO struct {
	DayOfWeek   int `json:"dayOfWeek"`
	AverageTime int `json:"averageTime"`
	TotalTime   int `json:"totalTime"`
}

type HourlyHeatmapEntryDTO struct {
	DayOfWeek int `json:"dayOfWeek"`
	Hour      int `json:"hour"`
	Count     int `json:"count"`
}

// --- Webhooks ---

type WebhookDTO struct {
	ID         int             `json:"id"`
	Type       string          `json:"type"`
	Token      string          `json:"token"`
	WebhookURL string          `json:"webhookUrl"`
	Data       json.RawMessage `json:"data"`
}

type CreateWebhookRequest struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type WebhookTriggerResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}
