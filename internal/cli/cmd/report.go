package cmd

import (
	"fmt"
	"strconv"

	"github.com/klokku/klokku/internal/cli/output"
	"github.com/spf13/cobra"
)

func newReportCmd() *cobra.Command {
	reportCmd := &cobra.Command{
		Use:   "report",
		Short: "View budget plan reports",
	}

	reportCmd.AddCommand(newReportGetCmd())
	reportCmd.AddCommand(newReportItemCmd())

	return reportCmd
}

func newReportGetCmd() *cobra.Command {
	var from, to string
	cmd := &cobra.Command{
		Use:   "get <planId>",
		Short: "Get an overall budget plan report",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			planID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid plan ID: %s", args[0])
			}
			client, err := newAPIClient()
			if err != nil {
				return err
			}
			report, err := client.GetReport(planID, from, to)
			if err != nil {
				return err
			}
			return output.Print(outputFormat, report, func() {
				fmt.Printf("Report: %s (ID: %d)\n", report.PlanName, report.PlanID)
				fmt.Printf("Period: %s — %s (%d weeks, %d excluded)\n",
					report.StartDate, report.EndDate, report.WeekCount, report.ExcludedWeekCount)
				fmt.Printf("Budget: %s  Weekly Plan: %s  Actual: %s\n",
					formatDuration(report.Totals.TotalBudgetPlanTime),
					formatDuration(report.Totals.TotalWeeklyPlanTime),
					formatDuration(report.Totals.TotalActualTime))
				if len(report.Totals.Items) > 0 {
					fmt.Println()
					headers := []string{"NAME", "BUDGET", "WEEKLY PLAN", "ACTUAL", "AVG/WEEK"}
					rows := make([][]string, 0, len(report.Totals.Items))
					for _, item := range report.Totals.Items {
						rows = append(rows, []string{
							item.Name,
							formatDuration(item.BudgetPlanTime),
							formatDuration(item.WeeklyPlanTime),
							formatDuration(item.ActualTime),
							formatDuration(item.AveragePerWeek),
						})
					}
					output.PrintText(headers, rows)
				}
			})
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "Start date in RFC3339 format, e.g. 2026-04-01T00:00:00Z")
	cmd.Flags().StringVar(&to, "to", "", "End date in RFC3339 format, e.g. 2026-04-30T23:59:59Z")
	return cmd
}

func newReportItemCmd() *cobra.Command {
	var from, to string
	cmd := &cobra.Command{
		Use:   "item <planId> <itemId>",
		Short: "Get a detailed report for a specific budget item",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			planID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid plan ID: %s", args[0])
			}
			itemID, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid item ID: %s", args[1])
			}
			client, err := newAPIClient()
			if err != nil {
				return err
			}
			report, err := client.GetItemReport(planID, itemID, from, to)
			if err != nil {
				return err
			}
			return output.Print(outputFormat, report, func() {
				fmt.Printf("Item Report: %s (Plan: %s)\n", report.ItemName, report.PlanName)
				fmt.Printf("Period: %s — %s\n", report.StartDate, report.EndDate)
				fmt.Printf("Completion: %.1f%%\n", report.CompletionPercent)
				fmt.Printf("Budget: %s  Weekly Plan: %s  Actual: %s\n",
					formatDuration(report.TotalBudgetPlanTime),
					formatDuration(report.TotalWeeklyPlanTime),
					formatDuration(report.TotalActualTime))
				fmt.Printf("Avg/day: %s  Avg/week: %s  Active days: %d/%d\n",
					formatDuration(report.AveragePerDay),
					formatDuration(report.AveragePerWeek),
					report.ActiveDaysCount, report.TotalDaysCount)
			})
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "Start date in RFC3339 format, e.g. 2026-04-01T00:00:00Z")
	cmd.Flags().StringVar(&to, "to", "", "End date in RFC3339 format, e.g. 2026-04-30T23:59:59Z")
	return cmd
}
