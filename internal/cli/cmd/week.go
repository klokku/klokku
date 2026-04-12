package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/klokku/klokku/internal/cli/api"
	"github.com/klokku/klokku/internal/cli/output"
	"github.com/spf13/cobra"
)

func newWeekCmd() *cobra.Command {
	weekCmd := &cobra.Command{
		Use:   "week",
		Short: "Manage weekly plans",
	}

	weekCmd.AddCommand(newWeekGetCmd())
	weekCmd.AddCommand(newWeekResetCmd())
	weekCmd.AddCommand(newWeekOffCmd())
	weekCmd.AddCommand(newWeekItemCmd())

	return weekCmd
}

func defaultDateRFC3339() string {
	return time.Now().Format(time.RFC3339)
}

func newWeekGetCmd() *cobra.Command {
	var date string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get weekly plan for a date",
		RunE: func(cmd *cobra.Command, args []string) error {
			if date == "" {
				date = defaultDateRFC3339()
			}
			client, err := newAPIClient()
			if err != nil {
				return err
			}
			plan, err := client.GetWeeklyPlan(date)
			if err != nil {
				return err
			}
			return output.Print(outputFormat, plan, func() {
				printWeeklyPlanText(plan)
			})
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date in RFC3339 format, e.g. 2026-04-05T00:00:00Z (default: today)")
	return cmd
}

func newWeekResetCmd() *cobra.Command {
	var date string
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset weekly plan to budget defaults",
		RunE: func(cmd *cobra.Command, args []string) error {
			if date == "" {
				date = defaultDateRFC3339()
			}
			client, err := newAPIClient()
			if err != nil {
				return err
			}
			plan, err := client.ResetWeeklyPlan(date)
			if err != nil {
				return err
			}
			return output.Print(outputFormat, plan, func() {
				fmt.Println("Weekly plan reset to budget defaults.")
				printWeeklyPlanText(plan)
			})
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date in RFC3339 format, e.g. 2026-04-05T00:00:00Z (default: today)")
	return cmd
}

func newWeekOffCmd() *cobra.Command {
	var (
		date  string
		off   bool
		noOff bool
	)
	cmd := &cobra.Command{
		Use:   "off",
		Short: "Mark/unmark a week as off",
		RunE: func(cmd *cobra.Command, args []string) error {
			if date == "" {
				date = defaultDateRFC3339()
			}
			isOff := true
			if noOff {
				isOff = false
			}
			client, err := newAPIClient()
			if err != nil {
				return err
			}
			plan, err := client.SetOffWeek(date, isOff)
			if err != nil {
				return err
			}
			return output.Print(outputFormat, plan, func() {
				if plan.IsOffWeek {
					fmt.Println("Week marked as off.")
				} else {
					fmt.Println("Week unmarked as off.")
				}
			})
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date in RFC3339 format, e.g. 2026-04-05T00:00:00Z (required)")
	cmd.Flags().BoolVar(&off, "off", true, "Mark week as off (default)")
	cmd.Flags().BoolVar(&noOff, "no-off", false, "Unmark week as off")
	return cmd
}

func newWeekItemCmd() *cobra.Command {
	itemCmd := &cobra.Command{
		Use:   "item",
		Short: "Manage weekly plan items",
	}
	itemCmd.AddCommand(newWeekItemUpdateCmd())
	itemCmd.AddCommand(newWeekItemResetCmd())
	return itemCmd
}

func newWeekItemUpdateCmd() *cobra.Command {
	var (
		date         string
		budgetItemID int
		duration     string
		notes        string
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a weekly plan item",
		RunE: func(cmd *cobra.Command, args []string) error {
			if date == "" {
				return fmt.Errorf("--date is required")
			}
			if budgetItemID == 0 {
				return fmt.Errorf("--budget-item-id is required")
			}
			req := api.UpdateWeeklyItemRequest{
				BudgetItemID: budgetItemID,
			}
			if cmd.Flags().Changed("duration") {
				dur, err := parseDuration(duration)
				if err != nil {
					return err
				}
				req.WeeklyDuration = dur
			}
			if cmd.Flags().Changed("notes") {
				req.Notes = notes
			}
			client, err := newAPIClient()
			if err != nil {
				return err
			}
			item, err := client.UpdateWeeklyItem(date, req)
			if err != nil {
				return err
			}
			return output.Print(outputFormat, item, func() {
				fmt.Printf("Updated weekly item: %s (duration: %s)\n", item.Name, formatDuration(item.WeeklyDuration))
			})
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date in RFC3339 format, e.g. 2026-04-05T00:00:00Z (required)")
	cmd.Flags().IntVar(&budgetItemID, "budget-item-id", 0, "Budget item ID (required)")
	cmd.Flags().StringVar(&duration, "duration", "", "Weekly duration, e.g. 8h or 28800")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes for this week")
	return cmd
}

func newWeekItemResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset <itemId>",
		Short: "Reset a weekly plan item to budget defaults",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			itemID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid item ID: %s", args[0])
			}
			client, err := newAPIClient()
			if err != nil {
				return err
			}
			item, err := client.ResetWeeklyItem(itemID)
			if err != nil {
				return err
			}
			return output.Print(outputFormat, item, func() {
				fmt.Printf("Reset weekly item: %s\n", item.Name)
			})
		},
	}
}

func printWeeklyPlanText(plan *api.WeeklyPlanDTO) {
	if plan.IsOffWeek {
		fmt.Println("(Off week)")
	}
	fmt.Printf("Budget Plan ID: %d\n", plan.BudgetPlanID)
	if len(plan.Items) > 0 {
		fmt.Println()
		headers := []string{"ID", "NAME", "DURATION", "DAYS/WEEK", "NOTES"}
		rows := make([][]string, 0, len(plan.Items))
		for _, item := range plan.Items {
			rows = append(rows, []string{
				strconv.Itoa(item.ID),
				item.Name,
				formatDuration(item.WeeklyDuration),
				strconv.Itoa(item.WeeklyOccurrences),
				item.Notes,
			})
		}
		output.PrintText(headers, rows)
	}
}
