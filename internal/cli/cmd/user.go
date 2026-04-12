package cmd

import (
	"fmt"

	"github.com/klokku/klokku/internal/cli/api"
	"github.com/klokku/klokku/internal/cli/output"
	"github.com/spf13/cobra"
)

func newUserCmd() *cobra.Command {
	userCmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
	}

	userCmd.AddCommand(newUserCurrentCmd())
	userCmd.AddCommand(newUserListCmd())

	return userCmd
}

func newUserCurrentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Get current user",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newAPIClient()
			if err != nil {
				return err
			}

			user, err := client.GetCurrentUser()
			if err != nil {
				return err
			}

			return output.Print(outputFormat, user, func() {
				printUserText(user)
			})
		},
	}
}

func newUserListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all users",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newAPIClient()
			if err != nil {
				return err
			}

			users, err := client.ListUsers()
			if err != nil {
				return err
			}

			return output.Print(outputFormat, users, func() {
				headers := []string{"UID", "USERNAME", "DISPLAY NAME", "TIMEZONE"}
				rows := make([][]string, 0, len(users))
				for _, u := range users {
					rows = append(rows, []string{u.UID, u.Username, u.DisplayName, u.Settings.Timezone})
				}
				output.PrintText(headers, rows)
			})
		},
	}
}

func printUserText(u *api.UserDTO) {
	fmt.Printf("UID:          %s\n", u.UID)
	fmt.Printf("Username:     %s\n", u.Username)
	fmt.Printf("Display Name: %s\n", u.DisplayName)
	fmt.Printf("Timezone:     %s\n", u.Settings.Timezone)
	fmt.Printf("Week Start:   %s\n", u.Settings.WeekStartDay)
	fmt.Printf("Calendar:     %s\n", u.Settings.EventCalendarType)
}
