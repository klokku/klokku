package cmd

import (
	"fmt"
	"strconv"

	"github.com/klokku/klokku/internal/cli/api"
	"github.com/klokku/klokku/internal/cli/output"
	"github.com/spf13/cobra"
)

func newBudgetCmd() *cobra.Command {
	budgetCmd := &cobra.Command{
		Use:   "budget",
		Short: "Manage budget plans and items",
	}

	budgetCmd.AddCommand(newBudgetListCmd())
	budgetCmd.AddCommand(newBudgetGetCmd())
	budgetCmd.AddCommand(newBudgetCreateCmd())
	budgetCmd.AddCommand(newBudgetUpdateCmd())
	budgetCmd.AddCommand(newBudgetDeleteCmd())
	budgetCmd.AddCommand(newBudgetItemCmd())

	return budgetCmd
}

func newBudgetListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all budget plans",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newAPIClient()
			if err != nil {
				return err
			}
			plans, err := client.ListBudgetPlans()
			if err != nil {
				return err
			}
			return output.Print(outputFormat, plans, func() {
				headers := []string{"ID", "NAME", "CURRENT"}
				rows := make([][]string, 0, len(plans))
				for _, p := range plans {
					current := ""
					if p.IsCurrent {
						current = "yes"
					}
					rows = append(rows, []string{strconv.Itoa(p.ID), p.Name, current})
				}
				output.PrintText(headers, rows)
			})
		},
	}
}

func newBudgetGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <planId>",
		Short: "Get a budget plan with items",
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
			plan, err := client.GetBudgetPlan(planID)
			if err != nil {
				return err
			}
			return output.Print(outputFormat, plan, func() {
				fmt.Printf("Plan: %s (ID: %d", plan.Name, plan.ID)
				if plan.IsCurrent {
					fmt.Print(", current")
				}
				fmt.Println(")")
				if len(plan.Items) > 0 {
					fmt.Println()
					headers := []string{"ID", "NAME", "DURATION", "DAYS/WEEK", "ICON", "COLOR"}
					rows := make([][]string, 0, len(plan.Items))
					for _, item := range plan.Items {
						rows = append(rows, []string{
							strconv.Itoa(item.ID),
							item.Name,
							formatDuration(item.WeeklyDuration),
							strconv.Itoa(item.WeeklyOccurrences),
							item.Icon,
							item.Color,
						})
					}
					output.PrintText(headers, rows)
				}
			})
		},
	}
}

func newBudgetCreateCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new budget plan",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			client, err := newAPIClient()
			if err != nil {
				return err
			}
			plan, err := client.CreateBudgetPlan(name)
			if err != nil {
				return err
			}
			return output.Print(outputFormat, plan, func() {
				fmt.Printf("Created budget plan: %s (ID: %d)\n", plan.Name, plan.ID)
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Plan name (required)")
	return cmd
}

func newBudgetUpdateCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "update <planId>",
		Short: "Update a budget plan",
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
			// Fetch current plan first to preserve fields
			current, err := client.GetBudgetPlan(planID)
			if err != nil {
				return err
			}
			if name != "" {
				current.Name = name
			}
			updated, err := client.UpdateBudgetPlan(*current)
			if err != nil {
				return err
			}
			return output.Print(outputFormat, updated, func() {
				fmt.Printf("Updated budget plan: %s (ID: %d)\n", updated.Name, updated.ID)
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New plan name")
	return cmd
}

func newBudgetDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <planId>",
		Short: "Delete a budget plan",
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
			if err := client.DeleteBudgetPlan(planID); err != nil {
				return err
			}
			return output.Print(outputFormat, map[string]string{"status": "deleted"}, func() {
				fmt.Printf("Deleted budget plan %d\n", planID)
			})
		},
	}
}

// --- Budget Item subcommands ---

func newBudgetItemCmd() *cobra.Command {
	itemCmd := &cobra.Command{
		Use:   "item",
		Short: "Manage budget items within a plan",
	}

	itemCmd.AddCommand(newBudgetItemCreateCmd())
	itemCmd.AddCommand(newBudgetItemUpdateCmd())
	itemCmd.AddCommand(newBudgetItemDeleteCmd())
	itemCmd.AddCommand(newBudgetItemReorderCmd())

	return itemCmd
}

func newBudgetItemCreateCmd() *cobra.Command {
	var (
		name        string
		duration    string
		occurrences int
		icon        string
		color       string
	)
	cmd := &cobra.Command{
		Use:   "create <planId>",
		Short: "Create a budget item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			planID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid plan ID: %s", args[0])
			}
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if duration == "" {
				return fmt.Errorf("--duration is required")
			}
			dur, err := parseDuration(duration)
			if err != nil {
				return err
			}
			client, err := newAPIClient()
			if err != nil {
				return err
			}
			item, err := client.CreateBudgetItem(planID, api.BudgetItemDTO{
				Name:              name,
				WeeklyDuration:    dur,
				WeeklyOccurrences: occurrences,
				Icon:              icon,
				Color:             color,
			})
			if err != nil {
				return err
			}
			return output.Print(outputFormat, item, func() {
				fmt.Printf("Created item: %s (ID: %d) in plan %d\n", item.Name, item.ID, planID)
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Item name (required)")
	cmd.Flags().StringVar(&duration, "duration", "", "Weekly duration, e.g. 8h or 28800 (required)")
	cmd.Flags().IntVar(&occurrences, "occurrences", 0, "Days per week")
	cmd.Flags().StringVar(&icon, "icon", "", "Icon")
	cmd.Flags().StringVar(&color, "color", "", "Color hex code")
	return cmd
}

func newBudgetItemUpdateCmd() *cobra.Command {
	var (
		name        string
		duration    string
		occurrences int
		icon        string
		color       string
	)
	cmd := &cobra.Command{
		Use:   "update <planId> <itemId>",
		Short: "Update a budget item",
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
			// Fetch current plan to get item's current values
			plan, err := client.GetBudgetPlan(planID)
			if err != nil {
				return err
			}
			var current *api.BudgetItemDTO
			for _, item := range plan.Items {
				if item.ID == itemID {
					current = &item
					break
				}
			}
			if current == nil {
				return fmt.Errorf("item %d not found in plan %d", itemID, planID)
			}
			if name != "" {
				current.Name = name
			}
			if cmd.Flags().Changed("duration") {
				dur, err := parseDuration(duration)
				if err != nil {
					return err
				}
				current.WeeklyDuration = dur
			}
			if cmd.Flags().Changed("occurrences") {
				current.WeeklyOccurrences = occurrences
			}
			if cmd.Flags().Changed("icon") {
				current.Icon = icon
			}
			if cmd.Flags().Changed("color") {
				current.Color = color
			}
			updated, err := client.UpdateBudgetItem(planID, *current)
			if err != nil {
				return err
			}
			return output.Print(outputFormat, updated, func() {
				fmt.Printf("Updated item: %s (ID: %d)\n", updated.Name, updated.ID)
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Item name")
	cmd.Flags().StringVar(&duration, "duration", "", "Weekly duration, e.g. 8h or 28800")
	cmd.Flags().IntVar(&occurrences, "occurrences", 0, "Days per week")
	cmd.Flags().StringVar(&icon, "icon", "", "Icon")
	cmd.Flags().StringVar(&color, "color", "", "Color hex code")
	return cmd
}

func newBudgetItemDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <planId> <itemId>",
		Short: "Delete a budget item",
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
			if err := client.DeleteBudgetItem(planID, itemID); err != nil {
				return err
			}
			return output.Print(outputFormat, map[string]string{"status": "deleted"}, func() {
				fmt.Printf("Deleted item %d from plan %d\n", itemID, planID)
			})
		},
	}
}

func newBudgetItemReorderCmd() *cobra.Command {
	var afterID int
	cmd := &cobra.Command{
		Use:   "reorder <planId> <itemId>",
		Short: "Move a budget item after another item",
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
			if err := client.ReorderBudgetItem(planID, itemID, afterID); err != nil {
				return err
			}
			return output.Print(outputFormat, map[string]string{"status": "reordered"}, func() {
				fmt.Printf("Reordered item %d in plan %d\n", itemID, planID)
			})
		},
	}
	cmd.Flags().IntVar(&afterID, "after", 0, "ID of the item to place after (0 for first position)")
	return cmd
}
