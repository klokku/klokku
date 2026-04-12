package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/klokku/klokku/internal/cli/api"
	"github.com/klokku/klokku/internal/cli/config"
	"github.com/klokku/klokku/internal/cli/output"
	"github.com/spf13/cobra"
)

func newWebhookCmd() *cobra.Command {
	webhookCmd := &cobra.Command{
		Use:   "webhook",
		Short: "Manage webhooks",
	}

	webhookCmd.AddCommand(newWebhookListCmd())
	webhookCmd.AddCommand(newWebhookCreateCmd())
	webhookCmd.AddCommand(newWebhookDeleteCmd())
	webhookCmd.AddCommand(newWebhookTriggerCmd())

	return webhookCmd
}

func newWebhookListCmd() *cobra.Command {
	var webhookType string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List webhooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if webhookType == "" {
				return fmt.Errorf("--type is required (e.g. START_CURRENT_EVENT)")
			}
			client, err := newAPIClient()
			if err != nil {
				return err
			}
			webhooks, err := client.ListWebhooks(webhookType)
			if err != nil {
				return err
			}
			return output.Print(outputFormat, webhooks, func() {
				headers := []string{"ID", "TYPE", "TOKEN", "URL"}
				rows := make([][]string, 0, len(webhooks))
				for _, w := range webhooks {
					rows = append(rows, []string{
						strconv.Itoa(w.ID), w.Type, w.Token, w.WebhookURL,
					})
				}
				output.PrintText(headers, rows)
			})
		},
	}
	cmd.Flags().StringVar(&webhookType, "type", "", "Filter by webhook type")
	return cmd
}

func newWebhookCreateCmd() *cobra.Command {
	var (
		webhookType  string
		budgetItemID int
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a webhook",
		RunE: func(cmd *cobra.Command, args []string) error {
			if webhookType == "" {
				return fmt.Errorf("--type is required")
			}
			req := api.CreateWebhookRequest{
				Type: webhookType,
			}
			if webhookType == "START_CURRENT_EVENT" && budgetItemID > 0 {
				data, _ := json.Marshal(map[string]int{"budgetItemId": budgetItemID})
				req.Data = data
			}
			client, err := newAPIClient()
			if err != nil {
				return err
			}
			webhook, err := client.CreateWebhook(req)
			if err != nil {
				return err
			}
			return output.Print(outputFormat, webhook, func() {
				fmt.Printf("Created webhook (ID: %d, token: %s)\n", webhook.ID, webhook.Token)
				fmt.Printf("URL: %s\n", webhook.WebhookURL)
			})
		},
	}
	cmd.Flags().StringVar(&webhookType, "type", "", "Webhook type, e.g. START_CURRENT_EVENT (required)")
	cmd.Flags().IntVar(&budgetItemID, "budget-item-id", 0, "Budget item ID (for START_CURRENT_EVENT)")
	return cmd
}

func newWebhookDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a webhook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid webhook ID: %s", args[0])
			}
			client, err := newAPIClient()
			if err != nil {
				return err
			}
			if err := client.DeleteWebhook(id); err != nil {
				return err
			}
			return output.Print(outputFormat, map[string]string{"status": "deleted"}, func() {
				fmt.Printf("Deleted webhook %d\n", id)
			})
		},
	}
}

func newWebhookTriggerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "trigger <token>",
		Short: "Trigger a webhook (no auth required)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Webhook trigger doesn't need auth, but needs server URL
			cfg, err := config.Load("")
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			resolved := config.Resolve(cfg, flagServer, flagUserID, flagToken)
			if resolved.Server == "" {
				return fmt.Errorf("server URL is required")
			}
			client := api.NewClientNoAuth(resolved.Server)
			resp, err := client.TriggerWebhook(args[0])
			if err != nil {
				return err
			}
			return output.Print(outputFormat, resp, func() {
				if resp.Success {
					fmt.Printf("Webhook triggered successfully: %s\n", resp.Message)
				} else {
					fmt.Printf("Webhook trigger failed: %s\n", resp.Message)
				}
			})
		},
	}
}
