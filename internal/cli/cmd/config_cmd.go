package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/klokku/klokku/internal/cli/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
	}

	configCmd.AddCommand(newConfigInitCmd())
	configCmd.AddCommand(newConfigShowCmd())

	return configCmd
}

func newConfigInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize CLI configuration interactively",
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)

			fmt.Println("Klokku CLI Configuration Setup")
			fmt.Println()

			fmt.Print("Server URL (e.g., https://app.klokku.com or http://localhost:8181): ")
			server, _ := reader.ReadString('\n')
			server = strings.TrimSpace(server)
			if server == "" {
				return fmt.Errorf("server URL is required")
			}

			fmt.Println()
			fmt.Println("Authentication mode:")
			fmt.Println("  1) Managed (app.klokku.com) - uses personal access token")
			fmt.Println("  2) Self-hosted - uses user ID")
			fmt.Print("Choose [1/2]: ")
			choice, _ := reader.ReadString('\n')
			choice = strings.TrimSpace(choice)

			cfg := config.Config{Server: server}

			switch choice {
			case "1":
				fmt.Println()
				fmt.Println("Generate a personal access token at: https://app.klokku.com/auth/api-keys")
				fmt.Print("Token: ")
				token, _ := reader.ReadString('\n')
				token = strings.TrimSpace(token)
				if token == "" {
					return fmt.Errorf("token is required for managed mode")
				}
				cfg.Token = token
			case "2":
				fmt.Print("User ID (UID): ")
				userID, _ := reader.ReadString('\n')
				userID = strings.TrimSpace(userID)
				if userID == "" {
					return fmt.Errorf("user ID is required for self-hosted mode")
				}
				cfg.UserID = userID
			default:
				return fmt.Errorf("invalid choice: %s", choice)
			}

			path := config.DefaultPath()
			if err := config.Save(path, cfg); err != nil {
				return err
			}

			fmt.Printf("\nConfiguration saved to %s\n", path)
			return nil
		},
	}
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load("")
			if err != nil {
				return err
			}

			resolved := config.Resolve(cfg, flagServer, flagUserID, flagToken)

			fmt.Printf("Config file: %s\n", config.DefaultPath())
			fmt.Printf("Server:      %s\n", resolved.Server)
			if resolved.Token != "" {
				// Mask the token, showing only first 8 chars
				masked := resolved.Token
				if len(masked) > 8 {
					masked = masked[:8] + "..."
				}
				fmt.Printf("Auth:        token (%s)\n", masked)
			} else if resolved.UserID != "" {
				fmt.Printf("Auth:        user-id (%s)\n", resolved.UserID)
			} else {
				fmt.Printf("Auth:        not configured\n")
			}

			return nil
		},
	}
}
