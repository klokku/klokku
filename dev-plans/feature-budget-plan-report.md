# Budget Plan Report

## Overview

The Budget Plan Report provides a comprehensive view of time spent on each item within a budget plan over its entire lifetime. It allows users to compare their actual time usage against both the original budget plan allocations and the weekly plan adjustments, across weekly and monthly (4-week) periods.

The report is available for **any** budget plan, not just the currently active one.

## Core Concepts

### Report Lifetime

The report covers the period from the **first calendar event** that references any item in the given budget plan, up to and including the **current week**. If no calendar events exist for any of the plan's items, the report is empty.

### Time Attribution

- All calendar events whose `budget_item_id` matches an existing item in the requested budget plan are included, **regardless of which plan was active** at the time the event was recorded.
- Calendar events referencing items from other plans are **excluded**.
- Calendar events referencing deleted budget items (items no longer in the plan) are **excluded**.
- The currently running event is **excluded** from the report.

### Three Metrics Per Item Per Period

For each budget item in each time period (week or 4-week month), the report provides:

1. **Budget Plan Hours** — the sum of the item's `WeeklyDuration` from the original budget plan definition. This is a fixed value per week (the same value every week, straight from the budget item). For a 4-week period, this is the budget item's weekly duration multiplied by 4.
2. **Weekly Plan Hours** — the sum of the item's planned duration as recorded in the weekly plan for that specific week. This reflects any per-week overrides the user may have made. For a 4-week period, this is the sum of the 4 corresponding weekly plan values. If no weekly plan entry exists for a given week, the original budget plan item duration is used as the fallback.
3. **Actual Hours** — the sum of tracked time from calendar events for that item within the period.

## Endpoints

### 1. Summary Report

Returns the aggregated totals for the **entire lifetime** of the budget plan.

**Per item:**
- Total budget plan hours (weekly duration × number of weeks in lifetime)
- Total weekly plan hours (sum of all weekly plan values across lifetime)
- Total actual hours tracked

**Overall totals:**
- Sum of all items' budget plan hours
- Sum of all items' weekly plan hours
- Sum of all items' actual hours

### 2. Weekly Report

Returns a **per-week breakdown** for the entire lifetime of the budget plan.

Each entry represents one ISO week and contains data for **all budget items**:

- Week identifier (e.g., 2025-W03)
- Week start and end dates
- Per item:
  - Budget plan hours (the item's fixed weekly duration)
  - Weekly plan hours (from the weekly plan, or budget plan duration if no weekly plan entry exists)
  - Actual hours tracked in that week
- Week totals:
  - Sum of budget plan hours across all items
  - Sum of weekly plan hours across all items
  - Sum of actual hours across all items

Weeks where no time was tracked still appear in the report with zero actual hours (but non-zero planned values).

Data is sorted **oldest week first**.

### 3. Monthly (4-Week) Report

Returns a **per-4-week-period breakdown** for the entire lifetime of the budget plan.

The 4-week periods are **aligned with the plan's start** (the date of the first calendar event for the plan). The first period starts at the beginning of the week containing that first event, and each subsequent period starts exactly 4 weeks later. The final period may be incomplete if the current date falls within it.

Each entry represents one 4-week period and contains data for **all budget items**:

- Period start and end dates
- Period number (1, 2, 3, ...)
- Per item:
  - Budget plan hours (the item's weekly duration × number of weeks in the period, typically 4, but may be fewer for the last incomplete period)
  - Weekly plan hours (sum of weekly plan values for the weeks in the period)
  - Actual hours tracked in the period
- Period totals:
  - Sum of budget plan hours across all items
  - Sum of weekly plan hours across all items
  - Sum of actual hours across all items

Periods where no time was tracked still appear in the report with zero actual hours.

Data is sorted **oldest period first**.

## Item Metadata

Each budget item in the report includes its current metadata:

- Name (current name, even if it was different when time was tracked)
- Icon
- Color
- Position (for consistent ordering)

## Edge Cases

- **Deleted budget items**: Calendar events referencing items that have been deleted from the plan are excluded from all report endpoints.
- **No events**: If no calendar events exist for any item in the plan, all endpoints return empty data.
- **Current week**: The current (incomplete) week is included in both the weekly and monthly reports, with actual hours reflecting tracked time up to now (excluding the currently running event).
- **Incomplete final 4-week period**: The last monthly period may contain fewer than 4 weeks if the current date falls within it. The budget plan hours for that period are prorated to the number of weeks it spans.
- **Missing weekly plan entries**: If no weekly plan entry exists for an item in a given week, the original budget plan item's `WeeklyDuration` is used as the weekly plan value for that week.

---

# Implementation Plan

## Overview

The feature is implemented as a new `budget_plan_report` package under `pkg/budget_plan_report/`, following the existing project patterns (handler → service → repository, interface-based DI, DTO conversion). It introduces one new database query (find earliest event per plan) and reuses existing repositories/services for the rest.

## Step 1: New Database Query — Earliest Event for a Plan

**File:** `pkg/calendar/repository.go`

Add a new method to the `calendar.Repository` interface:

```go
GetEarliestEventTimeForBudgetItems(ctx context.Context, userId int, budgetItemIds []int) (time.Time, error)
```

This runs a single SQL query:

```sql
SELECT MIN(start_time) FROM calendar_event
WHERE user_id = $1 AND budget_item_id = ANY($2)
```

Returns `time.Time{}` (zero value) if no events exist, which the service interprets as "empty report".

**Why here:** The calendar repository already owns all calendar_event queries. This is a natural extension.

## Step 2: Domain Models

**New file:** `pkg/budget_plan_report/budget_plan_report.go`

Define the domain structs:

```go
// BudgetPlanReportItem — per-item stats within a period
type BudgetPlanReportItem struct {
    BudgetItemId     int
    Name             string
    Icon             string
    Color            string
    Position         int
    BudgetPlanHours  time.Duration  // from budget plan item WeeklyDuration
    WeeklyPlanHours  time.Duration  // from weekly plan (or fallback)
    ActualHours      time.Duration  // from calendar events
}

// WeeklyReportEntry — one week's data
type WeeklyReportEntry struct {
    WeekNumber          string        // e.g. "2025-W03"
    StartDate           time.Time
    EndDate             time.Time
    Items               []BudgetPlanReportItem
    TotalBudgetPlanHours time.Duration
    TotalWeeklyPlanHours time.Duration
    TotalActualHours     time.Duration
}

// MonthlyReportEntry — one 4-week period's data
type MonthlyReportEntry struct {
    PeriodNumber         int
    StartDate            time.Time
    EndDate              time.Time
    WeekCount            int           // typically 4, may be less for last period
    Items                []BudgetPlanReportItem
    TotalBudgetPlanHours time.Duration
    TotalWeeklyPlanHours time.Duration
    TotalActualHours     time.Duration
}

// SummaryReport — lifetime totals
type SummaryReport struct {
    PlanId               int
    PlanName             string
    StartDate            time.Time     // first event's week start
    EndDate              time.Time     // current week end
    WeekCount            int           // total weeks in lifetime
    Items                []BudgetPlanReportItem
    TotalBudgetPlanHours time.Duration
    TotalWeeklyPlanHours time.Duration
    TotalActualHours     time.Duration
}
```

## Step 3: Service Interface and Implementation

**New file:** `pkg/budget_plan_report/service.go`

### Interface

```go
type Service interface {
    GetSummaryReport(ctx context.Context, planId int) (SummaryReport, error)
    GetWeeklyReport(ctx context.Context, planId int) ([]WeeklyReportEntry, error)
    GetMonthlyReport(ctx context.Context, planId int) ([]MonthlyReportEntry, error)
}
```

### Dependencies (small interfaces, following the existing pattern)

```go
type budgetPlanReader interface {
    GetPlan(ctx context.Context, planId int) (budget_plan.BudgetPlan, error)
}

type calendarEventsReader interface {
    GetEvents(ctx context.Context, from time.Time, to time.Time) ([]calendar.Event, error)
}

type calendarEarliestEventFinder interface {
    GetEarliestEventTimeForBudgetItems(ctx context.Context, budgetItemIds []int) (time.Time, error)
}

type weeklyPlanItemsReader interface {
    GetItemsForWeek(ctx context.Context, date time.Time) ([]weekly_plan.WeeklyPlanItem, error)
}
```

### Constructor

```go
func NewService(
    budgetPlanReader budgetPlanReader,
    calendarReader calendarEventsReader,
    earliestEventFinder calendarEarliestEventFinder,
    weeklyPlanReader weeklyPlanItemsReader,
    clock utils.Clock,
) Service
```

### Core Algorithm (shared across all three endpoints)

All three endpoints share the same data gathering logic, differing only in how they aggregate:

1. **Load the budget plan** — get plan with items, collect budget item IDs
2. **Find earliest event** — call `GetEarliestEventTimeForBudgetItems` with the item IDs; return empty if none found
3. **Calculate lifetime boundaries** — from the week containing the earliest event to the current week (respecting user's `WeekFirstDay`)
4. **Fetch all calendar events** in the lifetime range in a single call to `GetEvents(from, to)`
5. **Filter events** — keep only events whose `budget_item_id` is in the plan's current item set
6. **For each week in the range:**
   - Call `GetItemsForWeek` to get the weekly plan data (handles fallback to budget plan automatically)
   - Aggregate event durations per budget item for that week
   - Build the three metrics per item
7. **Aggregate** into weekly entries, monthly entries (groups of 4 weeks), or summary totals depending on the endpoint

### Performance Consideration

The existing `GetItemsForWeek` makes one DB call per week. For long-lived plans (e.g., 2 years = ~104 weeks), this means ~104 DB calls. This is acceptable for a report endpoint (not called on every page load), but should be noted:

- Calendar events are fetched in a **single query** for the entire range
- Weekly plan items require **one query per week** (reusing the existing `weeklyPlanService.GetItemsForWeek` which also handles creating weekly plan entries from budget plan when they don't exist)

If performance becomes an issue later, a bulk weekly plan query could be added, but this is out of scope for the initial implementation.

## Step 4: Handler and DTOs

**New file:** `pkg/budget_plan_report/handler.go`

### DTOs

All durations are serialized as **integer seconds** (matching the existing convention):

```go
type BudgetPlanReportItemDTO struct {
    BudgetItemId    int    `json:"budgetItemId"`
    Name            string `json:"name"`
    Icon            string `json:"icon"`
    Color           string `json:"color"`
    Position        int    `json:"position"`
    BudgetPlanHours int    `json:"budgetPlanHours"`  // seconds
    WeeklyPlanHours int    `json:"weeklyPlanHours"`  // seconds
    ActualHours     int    `json:"actualHours"`       // seconds
}

type WeeklyReportEntryDTO struct {
    WeekNumber           string                    `json:"weekNumber"`
    StartDate            time.Time                 `json:"startDate"`
    EndDate              time.Time                 `json:"endDate"`
    Items                []BudgetPlanReportItemDTO `json:"items"`
    TotalBudgetPlanHours int                       `json:"totalBudgetPlanHours"`
    TotalWeeklyPlanHours int                       `json:"totalWeeklyPlanHours"`
    TotalActualHours     int                       `json:"totalActualHours"`
}

type MonthlyReportEntryDTO struct {
    PeriodNumber         int                       `json:"periodNumber"`
    StartDate            time.Time                 `json:"startDate"`
    EndDate              time.Time                 `json:"endDate"`
    WeekCount            int                       `json:"weekCount"`
    Items                []BudgetPlanReportItemDTO `json:"items"`
    TotalBudgetPlanHours int                       `json:"totalBudgetPlanHours"`
    TotalWeeklyPlanHours int                       `json:"totalWeeklyPlanHours"`
    TotalActualHours     int                       `json:"totalActualHours"`
}

type SummaryReportDTO struct {
    PlanId               int                       `json:"planId"`
    PlanName             string                    `json:"planName"`
    StartDate            time.Time                 `json:"startDate"`
    EndDate              time.Time                 `json:"endDate"`
    WeekCount            int                       `json:"weekCount"`
    Items                []BudgetPlanReportItemDTO `json:"items"`
    TotalBudgetPlanHours int                       `json:"totalBudgetPlanHours"`
    TotalWeeklyPlanHours int                       `json:"totalWeeklyPlanHours"`
    TotalActualHours     int                       `json:"totalActualHours"`
}
```

### Handler Methods

```go
type Handler struct {
    service Service
}

func NewHandler(service Service) *Handler

func (h *Handler) GetSummaryReport(w http.ResponseWriter, r *http.Request)
func (h *Handler) GetWeeklyReport(w http.ResponseWriter, r *http.Request)
func (h *Handler) GetMonthlyReport(w http.ResponseWriter, r *http.Request)
```

Each handler:
- Extracts `planId` from URL path parameter
- Calls the service
- Converts to DTO
- Returns JSON

Error handling follows the existing pattern: 400 for bad input, 404 if plan not found or no data, 500 for internal errors.

## Step 5: Routes

**File:** `internal/app/routes.go`

Add three new routes:

```go
// Budget Plan Report
r.HandleFunc("/api/budgetplan/{planId}/report/summary", deps.BudgetPlanReportHandler.GetSummaryReport).Methods("GET")
r.HandleFunc("/api/budgetplan/{planId}/report/weekly", deps.BudgetPlanReportHandler.GetWeeklyReport).Methods("GET")
r.HandleFunc("/api/budgetplan/{planId}/report/monthly", deps.BudgetPlanReportHandler.GetMonthlyReport).Methods("GET")
```

These nest under the existing `/api/budgetplan/{planId}` path, which is natural since the report is scoped to a specific plan.

## Step 6: Dependency Wiring

**File:** `internal/app/dependencies.go`

Add to the `Dependencies` struct:

```go
BudgetPlanReportService budget_plan_report.Service
BudgetPlanReportHandler *budget_plan_report.Handler
```

Wire in `BuildDependencies`:

```go
deps.BudgetPlanReportService = budget_plan_report.NewService(
    deps.BudgetPlanService,      // budgetPlanReader
    deps.CalendarProvider,       // calendarEventsReader
    deps.KlokkuCalendarService,  // calendarEarliestEventFinder (needs new method)
    deps.WeeklyPlanService,      // weeklyPlanItemsReader
    deps.Clock,
)
deps.BudgetPlanReportHandler = budget_plan_report.NewHandler(deps.BudgetPlanReportService)
```

Note: The `calendarEarliestEventFinder` interface needs to be satisfied. The `calendar.Service` will need to expose a method that delegates to the repository's new `GetEarliestEventTimeForBudgetItems`. Alternatively, this can be added directly to `CalendarProvider` with a new method.

## Step 7: Calendar Service Extension

**File:** `pkg/calendar/service.go`

Add a new method to `calendar.Service`:

```go
func (s *Service) GetEarliestEventTimeForBudgetItems(ctx context.Context, budgetItemIds []int) (time.Time, error)
```

This delegates to the repository method added in Step 1, passing the current user's ID from context.

Also add this method to the `calendar.Calendar` interface if it's needed through the `CalendarProvider` abstraction, or wire the `calendar.Service` directly to the report service via the small interface.

## Step 8: Tests

**New file:** `pkg/budget_plan_report/service_test.go`

Unit tests using mock implementations of the small interfaces (following the pattern from `pkg/stats/stats_service_test.go`):

- **Empty report**: Plan exists but no calendar events → empty response
- **Single week**: One week of data, verify all three metrics
- **Multiple weeks**: Verify week ordering (oldest first), gap weeks with zero actual hours
- **Monthly aggregation**: Verify 4-week grouping aligned to plan start, incomplete final period
- **Summary aggregation**: Verify lifetime totals
- **Deleted items filtering**: Events with budget_item_id not in plan's current items are excluded
- **Weekly plan fallback**: When no weekly plan entry exists, budget plan duration is used
- **Weekly plan override**: When weekly plan has a custom duration, it overrides budget plan

## Summary of Files Changed / Created

| File | Action | Description |
|---|---|---|
| `pkg/calendar/repository.go` | Modified | Add `GetEarliestEventTimeForBudgetItems` to interface and implementation |
| `pkg/calendar/service.go` | Modified | Add delegation method for earliest event query |
| `pkg/budget_plan_report/budget_plan_report.go` | **New** | Domain models |
| `pkg/budget_plan_report/service.go` | **New** | Service interface and implementation |
| `pkg/budget_plan_report/handler.go` | **New** | HTTP handler and DTOs |
| `pkg/budget_plan_report/service_test.go` | **New** | Unit tests |
| `internal/app/dependencies.go` | Modified | Wire new service and handler |
| `internal/app/routes.go` | Modified | Register three new routes |

## Implementation Order

1. Add `GetEarliestEventTimeForBudgetItems` to calendar repository + service (Step 1, 7)
2. Create domain models (Step 2)
3. Implement the service with core algorithm (Step 3)
4. Write unit tests (Step 8)
5. Create handler and DTOs (Step 4)
6. Wire dependencies and routes (Step 5, 6)
7. Manual integration test with existing database
