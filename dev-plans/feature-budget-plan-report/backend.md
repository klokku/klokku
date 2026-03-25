# Budget Plan Report

## Overview

The Budget Plan Report provides a comprehensive view of time spent on each item within a budget plan over a chosen time period. It allows users to compare their actual time usage against both the original budget plan allocations and the weekly plan adjustments, with a per-week breakdown and aggregated totals.

The report is available for **any** budget plan, not just the currently active one. Users can view the entire lifetime of a plan or select a custom week-aligned period.

## Core Concepts

### Report Period

By default, the report covers the period from the **first calendar event** that references any item in the given budget plan, up to and including the **current week**. If no calendar events exist for any of the plan's items, the report is empty.

Users can optionally specify a custom date range (from/to), which must align to week boundaries. The server snaps provided dates to the nearest week start/end using the user's `WeekFirstDay` setting.

### Time Attribution

- All calendar events whose `budget_item_id` matches an existing item in the requested budget plan are included, **regardless of which plan was active** at the time the event was recorded.
- Calendar events referencing items from other plans are **excluded**.
- Calendar events referencing deleted budget items (items no longer in the plan) are **excluded**.
- The currently running event is **excluded** from the report.

### Three Metrics Per Item Per Period

For each budget item in each week, the report provides:

1. **Budget Plan Time** — the item's `WeeklyDuration` from the original budget plan definition. This is a fixed value per week.
2. **Weekly Plan Time** — the item's planned duration as recorded in the weekly plan for that specific week. This reflects any per-week overrides the user may have made. If no weekly plan entry exists for a given week, the original budget plan item duration is used as the fallback.
3. **Actual Time** — the tracked time from calendar events for that item within the week.

## Endpoint

### `GET /api/budgetplan/{planId}/report`

**Query parameters (optional):**
- `from` — RFC3339 date string. Must be provided together with `to`.
- `to` — RFC3339 date string. Must be provided together with `from`.

**Behavior:**
- If neither `from` nor `to` provided: returns data for the entire plan lifetime (earliest event week to current week).
- If both provided: returns data for the specified period only (snapped to week boundaries).
- If only one provided: returns 400 error.

**Response:**

The response contains both the per-week breakdown and the aggregated totals for the period:

```json
{
  "planId": 1,
  "planName": "My Plan",
  "startDate": "2025-01-06T00:00:00+01:00",
  "endDate": "2025-03-23T23:59:59.999999999+01:00",
  "weekCount": 11,
  "weeks": [
    {
      "weekNumber": "2025-W02",
      "startDate": "...",
      "endDate": "...",
      "items": [
        {
          "budgetItemId": 10,
          "name": "Exercise",
          "icon": "run",
          "color": "#ff0000",
          "position": 100,
          "budgetPlanTime": 18000,
          "weeklyPlanTime": 18000,
          "actualTime": 14400
        }
      ],
      "totalBudgetPlanTime": 604800,
      "totalWeeklyPlanTime": 604800,
      "totalActualTime": 590000
    }
  ],
  "totals": {
    "items": [ ... ],
    "totalBudgetPlanTime": 6652800,
    "totalWeeklyPlanTime": 6652800,
    "totalActualTime": 6100000
  }
}
```

All time values are in **seconds**.

Weeks where no time was tracked still appear in the report with zero actual time (but non-zero planned values). Data is sorted **oldest week first**.

## Item Metadata

Each budget item in the report includes its current metadata:

- Name (current name, even if it was different when time was tracked)
- Icon
- Color
- Position (for consistent ordering)

## Edge Cases

- **Deleted budget items**: Calendar events referencing items that have been deleted from the plan are excluded.
- **No events**: If no calendar events exist for any item in the plan, the report returns empty data (weekCount=0).
- **Current week**: The current (incomplete) week is included, with actual time reflecting tracked time up to now (excluding the currently running event).
- **Missing weekly plan entries**: If no weekly plan entry exists for an item in a given week, the original budget plan item's `WeeklyDuration` is used as the weekly plan value for that week.
- **Timezone handling**: All week boundaries are calculated in the user's timezone to ensure events near midnight are attributed to the correct week.
