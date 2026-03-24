# Budget Plan Report — UI Implementation Instructions

## Context

Klokku is a time-tracking application. Users create **budget plans** with items (e.g., "Exercise 5h/week", "Reading 3h/week"), then track time against those items. A new backend feature provides **lifetime reports** for any budget plan — showing how much time was actually spent vs. planned, broken down by week and by 4-week periods.

Your task is to build the UI for this report feature.

---

## Tech Stack

- **React 19** with TypeScript
- **Vite** bundler (dev server on port 3000, proxies `/api` to `localhost:8181`)
- **TanStack React Query** for data fetching and caching
- **Tailwind CSS 4** for styling (utility-first, dark mode support)
- **Radix UI** component primitives (dialog, tabs, select, tooltip, etc.)
- **Recharts** for charts and data visualization
- **date-fns** for date formatting
- **Lucide React** for icons
- **React Router 7** for routing

---

## Project Structure

```
src/
├── api/              # API hooks (useQuery/useMutation wrappers) + types.ts
├── components/
│   ├── ui/           # Reusable primitives (table, card, chart, tabs, button, etc.)
│   ├── dashboard/    # Dashboard-specific components
│   ├── statistics/   # WeekChooser and related
│   └── ...
├── pages/
│   ├── AppRoutes.tsx # Route definitions
│   ├── links.ts      # Route path constants
│   ├── history/      # Existing weekly/daily stats pages
│   ├── budgetPlan/   # Budget plan management pages
│   └── ...
├── hooks/            # Context providers
└── lib/              # dateUtils.ts, utils.ts
```

---

## API Endpoints

All endpoints require the `X-User-Id` header (handled by `useFetchWithProfileUid()` hook). All durations are in **seconds** (integers).

### 1. Summary Report

```
GET /api/budgetplan/{planId}/report/summary
```

**Response: `SummaryReportDTO`**

```typescript
interface SummaryReportDTO {
  planId: number;
  planName: string;
  startDate: string;          // ISO 8601 (RFC3339)
  endDate: string;
  weekCount: number;          // total weeks in lifetime
  items: ReportItemDTO[];
  totalBudgetPlanHours: number;  // seconds
  totalWeeklyPlanHours: number;  // seconds
  totalActualHours: number;      // seconds
}
```

### 2. Weekly Report

```
GET /api/budgetplan/{planId}/report/weekly
```

**Response: `WeeklyReportEntryDTO[]`**

```typescript
interface WeeklyReportEntryDTO {
  weekNumber: string;            // e.g. "2025-W10"
  startDate: string;
  endDate: string;
  items: ReportItemDTO[];
  totalBudgetPlanHours: number;  // seconds
  totalWeeklyPlanHours: number;  // seconds
  totalActualHours: number;      // seconds
}
```

### 3. Monthly (4-Week) Report

```
GET /api/budgetplan/{planId}/report/monthly
```

**Response: `MonthlyReportEntryDTO[]`**

```typescript
interface MonthlyReportEntryDTO {
  periodNumber: number;          // 1, 2, 3, ...
  startDate: string;
  endDate: string;
  weekCount: number;             // typically 4, may be less for last period
  items: ReportItemDTO[];
  totalBudgetPlanHours: number;  // seconds
  totalWeeklyPlanHours: number;  // seconds
  totalActualHours: number;      // seconds
}
```

### Shared Item Structure

```typescript
interface ReportItemDTO {
  budgetItemId: number;
  name: string;
  icon: string;
  color: string;                 // hex color, e.g. "#ff0000"
  position: number;              // for ordering
  budgetPlanHours: number;       // seconds — from original budget plan
  weeklyPlanHours: number;       // seconds — from weekly plan (or fallback)
  actualHours: number;           // seconds — from tracked time
}
```

### How to Get `planId`

Use the existing `GET /api/budgetplan` endpoint (already available via `useBudgetPlan` hook) to list all plans. Each `BudgetPlan` has `id`, `name`, and `isCurrent`. The user should be able to select which plan to view the report for.

### Empty State

When a plan has no tracked events, the summary returns `weekCount: 0` and empty `items[]`. The weekly and monthly endpoints return empty arrays `[]`.

---

## Three Metrics Explained (for display labels)

For every budget item, in every time period, there are three numbers:

| Metric | Field | Meaning | Display Label Suggestion |
|---|---|---|---|
| Budget Plan Hours | `budgetPlanHours` | What the original budget plan allocates for this period | "Budget" or "Planned (original)" |
| Weekly Plan Hours | `weeklyPlanHours` | What the user actually planned (may differ from budget if they customized a week) | "Planned" or "Planned (adjusted)" |
| Actual Hours | `actualHours` | How much time was actually tracked | "Actual" or "Tracked" |

- If `weeklyPlanHours === budgetPlanHours`, the user never customized that week — you may choose to show only one "Planned" column.
- If they differ, it means the user overrode the plan for specific weeks.

---

## What to Build

### Navigation & Access

Add a new route for the report. Suggested path: `/budget-plans/:planId/report`.

The report should be accessible from:
- The budget plans list page — e.g., a "Report" button/link per plan row
- Alternatively, a dedicated "Reports" entry in the sidebar

Add the route to `src/pages/links.ts` and `src/pages/AppRoutes.tsx`.

### Page Layout

The report page should have:

1. **Header area**: Plan name, lifetime date range (e.g., "Jan 6, 2025 — Mar 21, 2025"), total weeks count
2. **Tab navigation**: Three tabs — "Summary", "Weekly", "Monthly"
3. **Content area**: The selected tab's data

Use the existing `Tabs` component (`src/components/ui/tabs.tsx`) for tab navigation — same pattern as `HistoryPage.tsx`.

### Summary Tab

Shows the **lifetime totals** for the entire plan.

**Table** with columns:
- Item name (with color dot and/or icon)
- Budget Plan Hours (total across all weeks)
- Weekly Plan Hours (total across all weeks)
- Actual Hours (total tracked)
- Completion % (`actualHours / weeklyPlanHours * 100`)

**Totals row** at the bottom.

**Optional chart**: A bar chart or grouped bar chart (Recharts) showing per-item comparison of the three metrics.

### Weekly Tab

Shows a **table with one row per week**, with sub-rows or expandable sections for per-item data.

Two possible layouts (choose one):

**Option A — Compact table (recommended for many weeks)**:
- Rows: one per week
- Columns: Week number, Budget Plan Hours (total), Weekly Plan Hours (total), Actual Hours (total), Completion %
- Clicking/expanding a week row shows per-item breakdown

**Option B — Full table**:
- Rows: one per week × one per item
- Grouped by week with week header rows showing totals

For the Actual Hours column, use the `ProgressCell` component pattern (colored progress bar background) — same as in `WeeklyHistory.tsx`.

### Monthly Tab

Same structure as Weekly tab but with 4-week periods instead of individual weeks.

- Each row represents a 4-week period
- Show period number, date range, week count (usually 4, last may be less)
- Same three metrics + completion %
- Same expand-to-see-items pattern

### Plan Selector

At the top of the report page (or as a dropdown), allow the user to switch between budget plans. Pre-select the current plan (where `isCurrent === true`).

You can fetch the plan list using the existing `useBudgetPlan()` hook.

---

## Formatting & Display Conventions

### Duration Formatting

Use `formatSecondsToDuration(seconds)` from `src/lib/dateUtils.ts`. It returns strings like `"5h 30m"`, `"0h 0m"`, or `"-2h 15m"` for negative values.

All API values are in **seconds** — pass them directly to this function.

### Date Formatting

Use `date-fns` `formatDate()` for display dates. The API returns ISO 8601 strings.

Week numbers come as strings like `"2025-W10"` — display as-is or format to "Week 10, 2025".

### Colors

Each `ReportItemDTO` has a `color` field (hex string). Use it for:
- Color dots next to item names
- Chart segment colors
- Optionally, light background tint on item rows

### Completion Percentage

Calculate client-side: `(actualHours / weeklyPlanHours) * 100`.

Color coding (matching existing `ProgressCell` pattern):
- `> 100%`: red background (`bg-red-100`)
- `> 90%`: dark green (`bg-green-200`)
- `<= 90%`: light green (`bg-green-100`)

---

## API Hook Pattern

Create a new file `src/api/useBudgetPlanReport.ts` following the existing pattern in `useStats.ts`:

```typescript
import { useQuery } from "@tanstack/react-query";
import { useFetchWithProfileUid } from "@/api/fetchWithProfileUid.ts";

// Define the response types (see TypeScript interfaces above)

export const useBudgetPlanReportSummary = (planId: number) => {
  const fetchWithAuth = useFetchWithProfileUid();
  const { isLoading, data } = useQuery({
    queryKey: ["budgetPlanReport", "summary", planId],
    queryFn: async () => {
      const response = await fetchWithAuth(`/api/budgetplan/${planId}/report/summary`);
      if (!response.ok) throw new Error("Failed to fetch report summary");
      return await response.json() as SummaryReportDTO;
    },
    enabled: !!planId,
  });
  return { isLoading, summary: data };
};

export const useBudgetPlanReportWeekly = (planId: number) => {
  const fetchWithAuth = useFetchWithProfileUid();
  const { isLoading, data } = useQuery({
    queryKey: ["budgetPlanReport", "weekly", planId],
    queryFn: async () => {
      const response = await fetchWithAuth(`/api/budgetplan/${planId}/report/weekly`);
      if (!response.ok) throw new Error("Failed to fetch weekly report");
      return await response.json() as WeeklyReportEntryDTO[];
    },
    enabled: !!planId,
  });
  return { isLoading, weeklyEntries: data };
};

export const useBudgetPlanReportMonthly = (planId: number) => {
  const fetchWithAuth = useFetchWithProfileUid();
  const { isLoading, data } = useQuery({
    queryKey: ["budgetPlanReport", "monthly", planId],
    queryFn: async () => {
      const response = await fetchWithAuth(`/api/budgetplan/${planId}/report/monthly`);
      if (!response.ok) throw new Error("Failed to fetch monthly report");
      return await response.json() as MonthlyReportEntryDTO[];
    },
    enabled: !!planId,
  });
  return { isLoading, monthlyEntries: data };
};
```

---

## Existing Components to Reuse

| Component | Location | Use For |
|---|---|---|
| `Table, TableHeader, TableBody, TableRow, TableCell, TableHead` | `src/components/ui/table.tsx` | All data tables |
| `Tabs, TabsList, TabsTrigger, TabsContent` | `src/components/ui/tabs.tsx` | Summary/Weekly/Monthly tab switching |
| `Card, CardHeader, CardTitle, CardContent` | `src/components/ui/card.tsx` | Wrapping the report sections |
| `ProgressCell` | `src/pages/history/ProgressCell.tsx` | Actual hours bar with color coding |
| `Spinner` | `src/components/ui/spinner.tsx` | Loading state |
| `Empty, EmptyTitle, EmptyDescription` | `src/components/ui/empty.tsx` | Empty state when no data |
| `Select, SelectTrigger, SelectContent, SelectItem` | `src/components/ui/select.tsx` | Plan selector dropdown |
| `Tooltip, TooltipTrigger, TooltipContent` | `src/components/ui/tooltip.tsx` | Hover info (e.g., show budget plan hours when they differ from weekly plan) |
| `BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip` | `recharts` (via `src/components/ui/chart.tsx`) | Optional charts |
| `formatSecondsToDuration` | `src/lib/dateUtils.ts` | All duration formatting |

---

## Files to Create / Modify

### New Files

| File | Purpose |
|---|---|
| `src/api/useBudgetPlanReport.ts` | API hooks for the three report endpoints |
| `src/api/types.ts` | Add new TypeScript interfaces (append to existing file) |
| `src/pages/budgetPlanReport/BudgetPlanReportPage.tsx` | Main report page with tabs |
| `src/pages/budgetPlanReport/SummaryTab.tsx` | Summary view component |
| `src/pages/budgetPlanReport/WeeklyTab.tsx` | Weekly breakdown component |
| `src/pages/budgetPlanReport/MonthlyTab.tsx` | Monthly breakdown component |

### Modified Files

| File | Change |
|---|---|
| `src/pages/links.ts` | Add `budgetPlanReport` path entry |
| `src/pages/AppRoutes.tsx` | Add route for the report page |
| `src/pages/budgetPlan/BudgetPlansPage.tsx` | Add "Report" link/button per plan row |

---

## Design Guidelines

- Follow the visual style of the existing History page (`src/pages/history/`)
- Use the same table styling, progress bars, and color patterns
- Keep it data-dense but readable — this is a reporting view, not a dashboard
- The report can contain many weeks of data — ensure the tables scroll well vertically
- Use the item's `color` field for visual distinction (color dots, chart colors)
- Current week should be visually distinguishable (e.g., bold or highlighted row)
- Consider using sticky table headers for long tables
- The report is read-only — no edits, just viewing
