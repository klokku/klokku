package cmd

import (
	"fmt"
	"os"

	"github.com/klokku/klokku/internal/cli/api"
	"github.com/klokku/klokku/internal/cli/config"
	"github.com/klokku/klokku/internal/cli/output"
	"github.com/spf13/cobra"
)

var (
	flagServer string
	flagUserID string
	flagToken  string
	flagOutput string

	outputFormat output.Format
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "klokku-cli",
		Short: "Klokku CLI - time planning and tracking",
		Long:  "Command-line interface for the Klokku time planning and tracking API.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			outputFormat = output.Detect(flagOutput)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().StringVar(&flagServer, "server", "", "Klokku server URL (env: KLOKKU_SERVER)")
	rootCmd.PersistentFlags().StringVar(&flagUserID, "user-id", "", "User UID for self-hosted mode (env: KLOKKU_USER_ID)")
	rootCmd.PersistentFlags().StringVar(&flagToken, "token", "", "Personal access token for managed mode (env: KLOKKU_TOKEN)")
	rootCmd.PersistentFlags().StringVarP(&flagOutput, "output", "o", "", "Output format: json, text (default: auto-detect)")

	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newUserCmd())
	rootCmd.AddCommand(newBudgetCmd())
	rootCmd.AddCommand(newWeekCmd())
	rootCmd.AddCommand(newEventCmd())
	rootCmd.AddCommand(newStatsCmd())
	rootCmd.AddCommand(newReportCmd())
	rootCmd.AddCommand(newWebhookCmd())

	return rootCmd
}

// newAPIClient loads config, resolves with flags/env, validates, and returns an API client.
func newAPIClient() (*api.Client, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	resolved := config.Resolve(cfg, flagServer, flagUserID, flagToken)

	if err := config.Validate(resolved); err != nil {
		return nil, err
	}

	return api.NewClient(resolved.Server, resolved.Token, resolved.UserID), nil
}

// Execute runs the root command.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
