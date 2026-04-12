package cmd

import (
	"fmt"
	"strconv"

	"github.com/klokku/klokku/internal/cli/api"
	"github.com/klokku/klokku/internal/cli/output"
	"github.com/spf13/cobra"
)

func newEventCmd() *cobra.Command {
	eventCmd := &cobra.Command{
		Use:   "event",
		Short: "Manage events and time tracking",
	}

	eventCmd.AddCommand(newEventStartCmd())
	eventCmd.AddCommand(newEventCurrentCmd())
	eventCmd.AddCommand(newEventListCmd())
	eventCmd.AddCommand(newEventRecentCmd())
	eventCmd.AddCommand(newEventCreateCmd())
	eventCmd.AddCommand(newEventUpdateCmd())
	eventCmd.AddCommand(newEventDeleteCmd())

	return eventCmd
}

func newEventStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <budgetItemId>",
		Short: "Start tracking time for a budget item",
		Long: `Start tracking time for a budget item. The budget item's name and weekly duration
are automatically looked up from the current budget plan.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			budgetItemID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid budget item ID: %s", args[0])
			}
			client, err := newAPIClient()
			if err != nil {
				return err
			}

			// Look up the budget item to get name and weeklyDuration
			// (the server stores these on the current event)
			plans, err := client.ListBudgetPlans()
			if err != nil {
				return fmt.Errorf("failed to look up budget plans: %w", err)
			}
			var itemName string
			var weeklyDuration int
			for _, plan := range plans {
				if !plan.IsCurrent {
					continue
				}
				fullPlan, err := client.GetBudgetPlan(plan.ID)
				if err != nil {
					return fmt.Errorf("failed to look up budget plan: %w", err)
				}
				for _, item := range fullPlan.Items {
					if item.ID == budgetItemID {
						itemName = item.Name
						weeklyDuration = item.WeeklyDuration
						break
					}
				}
				break
			}

			event, err := client.StartEvent(budgetItemID, itemName, weeklyDuration)
			if err != nil {
				return err
			}
			return output.Print(outputFormat, event, func() {
				fmt.Printf("Started: %s (budget item %d) at %s\n",
					event.PlanItem.Name, event.PlanItem.BudgetItemID, event.StartTime)
			})
		},
	}
}

func newEventCurrentCmd() *cobra.Command {
	currentCmd := &cobra.Command{
		Use:   "current",
		Short: "Get or adjust the current event",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newAPIClient()
			if err != nil {
				return err
			}
			event, err := client.GetCurrentEvent()
			if err != nil {
				return err
			}
			return output.Print(outputFormat, event, func() {
				fmt.Printf("Current: %s (budget item %d)\n", event.PlanItem.Name, event.PlanItem.BudgetItemID)
				fmt.Printf("Started: %s\n", event.StartTime)
			})
		},
	}

	currentCmd.AddCommand(newEventCurrentAdjustStartCmd())

	return currentCmd
}

func newEventCurrentAdjustStartCmd() *cobra.Command {
	var startTime string
	cmd := &cobra.Command{
		Use:   "adjust-start",
		Short: "Adjust the start time of the current event",
		RunE: func(cmd *cobra.Command, args []string) error {
			if startTime == "" {
				return fmt.Errorf("--time is required")
			}
			client, err := newAPIClient()
			if err != nil {
				return err
			}
			event, err := client.AdjustCurrentEventStart(startTime)
			if err != nil {
				return err
			}
			return output.Print(outputFormat, event, func() {
				fmt.Printf("Adjusted start time to: %s\n", event.StartTime)
			})
		},
	}
	cmd.Flags().StringVar(&startTime, "time", "", "New start time in RFC3339 format, e.g. 2026-04-05T09:00:00Z (required)")
	return cmd
}

func newEventListCmd() *cobra.Command {
	var from, to string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List calendar events in a date range",
		RunE: func(cmd *cobra.Command, args []string) error {
			if from == "" || to == "" {
				return fmt.Errorf("--from and --to are required")
			}
			client, err := newAPIClient()
			if err != nil {
				return err
			}
			events, err := client.ListCalendarEvents(from, to)
			if err != nil {
				return err
			}
			return output.Print(outputFormat, events, func() {
				printCalendarEventsText(events)
			})
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "Start date in RFC3339 format (e.g. 2026-04-01T00:00:00Z)")
	cmd.Flags().StringVar(&to, "to", "", "End date in RFC3339 format (e.g. 2026-04-07T23:59:59Z)")
	return cmd
}

func newEventRecentCmd() *cobra.Command {
	var last int
	cmd := &cobra.Command{
		Use:   "recent",
		Short: "Get recent calendar events",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newAPIClient()
			if err != nil {
				return err
			}
			events, err := client.GetRecentEvents(last)
			if err != nil {
				return err
			}
			return output.Print(outputFormat, events, func() {
				printCalendarEventsText(events)
			})
		},
	}
	cmd.Flags().IntVar(&last, "last", 5, "Number of recent events to fetch")
	return cmd
}

func newEventCreateCmd() *cobra.Command {
	var (
		summary      string
		start        string
		end          string
		budgetItemID int
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a calendar event",
		RunE: func(cmd *cobra.Command, args []string) error {
			if summary == "" {
				return fmt.Errorf("--summary is required")
			}
			if start == "" || end == "" {
				return fmt.Errorf("--start and --end are required")
			}
			client, err := newAPIClient()
			if err != nil {
				return err
			}
			events, err := client.CreateCalendarEvent(api.CalendarEventDTO{
				Summary:      summary,
				Start:        start,
				End:          end,
				BudgetItemID: budgetItemID,
			})
			if err != nil {
				return err
			}
			return output.Print(outputFormat, events, func() {
				fmt.Printf("Created %d event(s)\n", len(events))
				printCalendarEventsText(events)
			})
		},
	}
	cmd.Flags().StringVar(&summary, "summary", "", "Event summary (required)")
	cmd.Flags().StringVar(&start, "start", "", "Start time in RFC3339 format, e.g. 2026-04-05T09:00:00Z (required)")
	cmd.Flags().StringVar(&end, "end", "", "End time in RFC3339 format, e.g. 2026-04-05T10:00:00Z (required)")
	cmd.Flags().IntVar(&budgetItemID, "budget-item-id", 0, "Budget item ID")
	return cmd
}

func newEventUpdateCmd() *cobra.Command {
	var (
		summary      string
		start        string
		end          string
		budgetItemID int
	)
	cmd := &cobra.Command{
		Use:   "update <eventUid>",
		Short: "Update a calendar event",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			eventUID := args[0]
			client, err := newAPIClient()
			if err != nil {
				return err
			}
			event := api.CalendarEventDTO{UID: eventUID}
			if summary != "" {
				event.Summary = summary
			}
			if start != "" {
				event.Start = start
			}
			if end != "" {
				event.End = end
			}
			if cmd.Flags().Changed("budget-item-id") {
				event.BudgetItemID = budgetItemID
			}
			events, err := client.UpdateCalendarEvent(eventUID, event)
			if err != nil {
				return err
			}
			return output.Print(outputFormat, events, func() {
				fmt.Printf("Updated %d event(s)\n", len(events))
				printCalendarEventsText(events)
			})
		},
	}
	cmd.Flags().StringVar(&summary, "summary", "", "Event summary")
	cmd.Flags().StringVar(&start, "start", "", "Start time in RFC3339 format, e.g. 2026-04-05T09:00:00Z")
	cmd.Flags().StringVar(&end, "end", "", "End time in RFC3339 format, e.g. 2026-04-05T10:00:00Z")
	cmd.Flags().IntVar(&budgetItemID, "budget-item-id", 0, "Budget item ID")
	return cmd
}

func newEventDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <eventUid>",
		Short: "Delete a calendar event",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newAPIClient()
			if err != nil {
				return err
			}
			if err := client.DeleteCalendarEvent(args[0]); err != nil {
				return err
			}
			return output.Print(outputFormat, map[string]string{"status": "deleted"}, func() {
				fmt.Printf("Deleted event %s\n", args[0])
			})
		},
	}
}

func printCalendarEventsText(events []api.CalendarEventDTO) {
	headers := []string{"UID", "SUMMARY", "START", "END", "BUDGET ITEM"}
	rows := make([][]string, 0, len(events))
	for _, e := range events {
		rows = append(rows, []string{
			e.UID, e.Summary, e.Start, e.End, strconv.Itoa(e.BudgetItemID),
		})
	}
	output.PrintText(headers, rows)
}
