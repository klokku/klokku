# Budget Plan Report — UI Implementation Reference

## API

### Single Endpoint

```
GET /api/budgetplan/{planId}/report
GET /api/budgetplan/{planId}/report?from={RFC3339}&to={RFC3339}
```

Without `from`/`to`: returns full lifetime data. With both: returns data for that period only. All time values are in **seconds**.

### Response Type

```typescript
interface BudgetPlanReport {
    planId: number;
    planName: string;
    startDate: string;       // ISO 8601
    endDate: string;
    weekCount: number;
    weeks: WeeklyReportEntry[];
    totals: ReportTotals;
}

interface ReportTotals {
    items: ReportItem[];
    totalBudgetPlanTime: number;
    totalWeeklyPlanTime: number;
    totalActualTime: number;
}

interface WeeklyReportEntry {
    weekNumber: string;      // e.g. "2025-W10"
    startDate: string;
    endDate: string;
    items: ReportItem[];
    totalBudgetPlanTime: number;
    totalWeeklyPlanTime: number;
    totalActualTime: number;
}

interface ReportItem {
    budgetItemId: number;
    name: string;
    icon: string;
    color: string;
    position: number;
    budgetPlanTime: number;  // seconds
    weeklyPlanTime: number;  // seconds
    actualTime: number;      // seconds
}
```

## Page Layout

Route: `/history/budget-report` (under History in sidebar)

### Controls
1. **BudgetPlanSelect** — dropdown to choose which plan to view
2. **Mode toggle** — "All time" | "Custom period" buttons
3. **PeriodSelector** (visible in custom mode) — from/to date pickers with prev/next navigation and quick presets (4w, 8w, 12w, 26w)

### Content (single page, no tabs)
1. **Period info** — date range and week count
2. **Period Totals** — table with per-item totals, completion bars on Budget Plan and Planned columns
3. **Weekly Breakdown** — expandable table with one row per week, click to see per-item details

### Key UI Components
- `CompletionCell` — progress bar background showing actual/planned ratio, tooltip with details
- `PlannedDiffBadge` — Badge showing % difference between weekly plan and budget plan (e.g., "+12%")
- `ReportItemName` — color dot + icon + name

## Files

| File | Purpose |
|---|---|
| `src/api/useBudgetPlanReport.ts` | Single hook: `useBudgetPlanReport(planId?, from?, to?)` |
| `src/api/types.ts` | `BudgetPlanReport`, `ReportTotals`, `WeeklyReportEntry`, `ReportItem` |
| `src/pages/budgetPlanReport/BudgetPlanReportPage.tsx` | Main page |
| `src/pages/budgetPlanReport/ReportTotals.tsx` | Period totals table |
| `src/pages/budgetPlanReport/WeeklyBreakdown.tsx` | Expandable weekly table |
| `src/pages/budgetPlanReport/PeriodSelector.tsx` | Date range controls |
| `src/pages/budgetPlanReport/CompletionCell.tsx` | Progress bar with tooltip |
| `src/pages/budgetPlanReport/PlannedDiffBadge.tsx` | Badge showing plan difference |
| `src/pages/budgetPlanReport/ReportItemName.tsx` | Item name with color/icon |
