---
name: klokku-cli
description: Manage Klokku time tracking via CLI. Use when the user wants to track time, switch activities, check what they're working on, view time statistics or reports, manage budget plans or weekly plans, create or manage calendar events, or interact with Klokku in any way. Also use when the user mentions Klokku, budget items, time budgets, or weekly planning.
---

# Klokku CLI — AI Agent Skill

## What is Klokku?

Klokku is a personal time tracking application that helps users understand how they spend their time and make realistic plans. Unlike productivity tools that try to maximize output, Klokku focuses on **balance** — helping users set boundaries, establish expectations, and keep priorities clear.

### Core concepts

- **Budget Plan**: A weekly time allocation framework. Users distribute their 168 available hours per week across different activities. A user has one "current" (active) budget plan at a time.
- **Budget Item**: An individual activity category within a budget plan (e.g., "Work", "Sleep", "Exercise", "Family"). Each has a name, weekly duration target, days-per-week occurrence, icon, and color. Recommended: 10–12 items per plan.
- **Weekly Plan**: A per-week copy of the budget plan that can be customized. Users can adjust durations and add notes for specific weeks without changing the underlying budget. Weeks can be marked as "off" (excluded from reports).
- **Current Event**: The activity being tracked right now. When a user starts a new event, the previous one is saved to the calendar.
- **Calendar Event**: A stored time entry with a summary, start/end time, and associated budget item. Events can be stored in Klokku's internal calendar or Google Calendar.
- **Off Week**: A week marked as non-working/vacation, excluded from budget plan reports and statistics.

### How time tracking works

1. User creates a **budget plan** with items representing their typical week
2. Each week, the budget is copied into a **weekly plan** that can be fine-tuned
3. Throughout the day, the user switches between **budget items** to track time (via the UI, CLI, webhooks, or integrations)
4. Starting a new event automatically saves the previous one as a **calendar event**
5. **Statistics and reports** show actual time vs. planned time, helping the user adjust

### Time format

All durations in the API and CLI are in **seconds** (integers). The CLI accepts human-friendly formats like `2h30m` for input and displays them accordingly in text mode. All dates must be in **RFC3339 format** (e.g., `2026-04-05T00:00:00Z`).

---

## Installing the CLI

### From source (requires Go 1.26+)

```sh
go install github.com/klokku/klokku/cmd/klokku-cli@latest
```

### From GitHub releases

Download the binary for your platform from [GitHub Releases](https://github.com/klokku/klokku/releases) and place it in your PATH.

### Verify installation

```sh
klokku-cli --help
```

---

## Configuring the CLI

### Interactive setup

```sh
klokku-cli config init
```

This creates `~/.config/klokku/config.yaml` with server URL and authentication.

### Authentication modes

**Managed (app.klokku.com)** — uses a personal access token:
```yaml
server: https://app.klokku.com
token: <your-token>
```
Generate a token at: https://app.klokku.com/auth/api-keys

**Self-hosted** — uses user ID:
```yaml
server: http://localhost:8181
user-id: <your-user-uid>
```

### Configuration priority

Flags (`--server`, `--token`, `--user-id`) override environment variables (`KLOKKU_SERVER`, `KLOKKU_TOKEN`, `KLOKKU_USER_ID`), which override the config file.

### Output format

- `--output json` — structured JSON (default when stdout is not a terminal)
- `--output text` — human-readable tables (default in a terminal)

When used by an AI agent, always use `--output json` for reliable parsing.

---

## CLI Command Reference

### User

```sh
klokku-cli user current                 # Get the authenticated user's info
klokku-cli user list                    # List all users (only self-hosted)
```

### Budget Plans

```sh
klokku-cli budget list                  # List all budget plans
klokku-cli budget get <planId>          # Get a plan with all its items
klokku-cli budget create --name "Plan"  # Create a new plan
klokku-cli budget update <planId> --name "New Name"
klokku-cli budget delete <planId>
```

### Budget Items

```sh
klokku-cli budget item create <planId> --name "Work" --duration 8h --occurrences 5 --color "#3B82F6"
klokku-cli budget item update <planId> <itemId> --name "Deep Work" --duration 6h
klokku-cli budget item delete <planId> <itemId>
klokku-cli budget item reorder <planId> <itemId> --after <precedingItemId>
```

The `--duration` flag accepts human-friendly formats (`8h`, `2h30m`, `45m`) or raw seconds (`28800`).

### Weekly Plans

```sh
klokku-cli week get                                # Current week
klokku-cli week get --date 2026-04-05T00:00:00Z    # Specific week
klokku-cli week reset --date 2026-04-05T00:00:00Z  # Reset to budget defaults
klokku-cli week off --date 2026-04-05T00:00:00Z    # Mark week as off
klokku-cli week off --date 2026-04-05T00:00:00Z --no-off  # Unmark

klokku-cli week item update --date 2026-04-05T00:00:00Z --budget-item-id 1 --duration 6h --notes "Short week"
klokku-cli week item reset <weeklyItemId>          # Reset single item to budget default
```

### Time Tracking (Events)

```sh
klokku-cli event start <budgetItemId>              # Start tracking a budget item
klokku-cli event current                           # See what's currently being tracked
klokku-cli event current adjust-start --time 2026-04-05T09:00:00Z  # Fix start time
```

### Calendar Events (History)

```sh
klokku-cli event list --from 2026-04-01T00:00:00Z --to 2026-04-07T23:59:59Z
klokku-cli event recent --last 5
klokku-cli event create --summary "Sport" --start 2026-04-05T09:00:00Z --end 2026-04-05T10:00:00Z --budget-item-id 1
klokku-cli event update <eventUid> --summary "Sport" --start ... --end ...
klokku-cli event delete <eventUid>
```

### Statistics

```sh
klokku-cli stats weekly                            # Current week stats
klokku-cli stats weekly --date 2026-04-05T00:00:00Z
klokku-cli stats item-history --from 2026-04-01T00:00:00Z --to 2026-04-30T23:59:59Z --budget-item-id 1
```

### Reports

```sh
klokku-cli report get <planId>                     # Lifetime report
klokku-cli report get <planId> --from 2026-04-01T00:00:00Z --to 2026-04-30T23:59:59Z
klokku-cli report item <planId> <itemId>           # Detailed item report
klokku-cli report item <planId> <itemId> --from ... --to ...
```

### Webhooks

```sh
klokku-cli webhook list --type START_CURRENT_EVENT
klokku-cli webhook create --type START_CURRENT_EVENT --budget-item-id 1
klokku-cli webhook delete <webhookId>
klokku-cli webhook trigger <token>                 # No auth required
```

---

## Common Workflows for AI Agents

### Check what the user is working on

```sh
klokku-cli event current --output json
```

### Switch to a different activity

First, find the budget item ID:
```sh
klokku-cli budget list --output json
klokku-cli budget get <planId> --output json
```

Then start tracking:
```sh
klokku-cli event start <budgetItemId>
```

### See how the week is going

```sh
klokku-cli stats weekly --output json
```

This shows time tracked vs. planned for each budget item.

### Log a past event

If the user forgot to track something (e.g. "Sport"):
```sh
klokku-cli event create --summary "Sport" --start 2026-04-05T14:00:00Z --end 2026-04-05T15:30:00Z --budget-item-id 3
```

### Adjust the current week's plan

If the user mentions they'll have less time for something this week:
```sh
klokku-cli week item update --date 2026-04-05T00:00:00Z --budget-item-id 1 --duration 4h --notes "Short week - traveling"
```

### Get a progress report

```sh
klokku-cli report get <planId> --output json
```

### Mark a vacation week

```sh
klokku-cli week off --date 2026-04-05T00:00:00Z
```

Value of "date" can be any time during the week.

---

## Tips for AI Agents

1. **Always use `--output json`** for structured, parseable responses.
2. **Budget item IDs are stable** — cache them from `budget get` to avoid repeated lookups.
3. **The current budget plan** is the one with `isCurrent: true` in `budget list`.
4. **Starting an event automatically stops the previous one** — there's no explicit "stop" command.
5. **Dates must be RFC3339** — always include the timezone offset (e.g., `T00:00:00Z` for UTC).
6. **Durations in JSON are seconds** — convert for display (3600 = 1 hour).
7. **Weekly plans are per-ISO-week** — any date within a week selects that week's plan.
8. **`event start` looks up the item name automatically** from the current budget plan.
9. **Webhook trigger does not require authentication** — useful for external automations.
10. **Off weeks are excluded from reports** — use this for vacations/holidays.
