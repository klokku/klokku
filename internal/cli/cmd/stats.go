package cmd

import (
	"fmt"

	"github.com/klokku/klokku/internal/cli/output"
	"github.com/spf13/cobra"
)

func newStatsCmd() *cobra.Command {
	statsCmd := &cobra.Command{
		Use:   "stats",
		Short: "View statistics",
	}

	statsCmd.AddCommand(newStatsWeeklyCmd())
	statsCmd.AddCommand(newStatsItemHistoryCmd())

	return statsCmd
}

func newStatsWeeklyCmd() *cobra.Command {
	var date string
	cmd := &cobra.Command{
		Use:   "weekly",
		Short: "Get weekly statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			if date == "" {
				date = defaultDateRFC3339()
			}
			client, err := newAPIClient()
			if err != nil {
				return err
			}
			stats, err := client.GetWeeklyStats(date)
			if err != nil {
				return err
			}
			return output.Print(outputFormat, stats, func() {
				fmt.Printf("Week: %s — %s\n", stats.StartDate, stats.EndDate)
				fmt.Printf("Total planned: %s  Total tracked: %s  Remaining: %s\n",
					formatDuration(stats.TotalPlanned),
					formatDuration(stats.TotalTime),
					formatDuration(stats.TotalRemaining))
				if len(stats.PerPlanItem) > 0 {
					fmt.Println()
					headers := []string{"NAME", "TRACKED", "REMAINING"}
					rows := make([][]string, 0, len(stats.PerPlanItem))
					for _, item := range stats.PerPlanItem {
						rows = append(rows, []string{
							item.PlanItem.Name,
							formatDuration(item.Duration),
							formatDuration(item.Remaining),
						})
					}
					output.PrintText(headers, rows)
				}
			})
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date in RFC3339 format, e.g. 2026-04-05T00:00:00Z (default: today)")
	return cmd
}

func newStatsItemHistoryCmd() *cobra.Command {
	var (
		from         string
		to           string
		budgetItemID int
	)
	cmd := &cobra.Command{
		Use:   "item-history",
		Short: "Get history stats for a budget item",
		RunE: func(cmd *cobra.Command, args []string) error {
			if from == "" || to == "" {
				return fmt.Errorf("--from and --to are required")
			}
			if budgetItemID == 0 {
				return fmt.Errorf("--budget-item-id is required")
			}
			client, err := newAPIClient()
			if err != nil {
				return err
			}
			stats, err := client.GetItemHistory(from, to, budgetItemID)
			if err != nil {
				return err
			}
			return output.Print(outputFormat, stats, func() {
				fmt.Printf("Item history: %s — %s\n", stats.StartDate, stats.EndDate)
				if len(stats.StatsPerWeek) > 0 {
					fmt.Println()
					headers := []string{"WEEK START", "WEEK END", "TRACKED", "PLANNED"}
					rows := make([][]string, 0, len(stats.StatsPerWeek))
					for _, w := range stats.StatsPerWeek {
						rows = append(rows, []string{
							w.StartDate, w.EndDate,
							formatDuration(w.Duration),
							formatDuration(w.Planned),
						})
					}
					output.PrintText(headers, rows)
				}
			})
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "Start date in RFC3339 format, e.g. 2026-04-01T00:00:00Z")
	cmd.Flags().StringVar(&to, "to", "", "End date in RFC3339 format, e.g. 2026-04-30T23:59:59Z")
	cmd.Flags().IntVar(&budgetItemID, "budget-item-id", 0, "Budget item ID (required)")
	return cmd
}
