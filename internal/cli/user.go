package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/ahoylog/kvik-tasks/internal/core"
	"github.com/ahoylog/kvik-tasks/internal/db"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
}

var userAddCmd = &cobra.Command{
	Use:   "add <username>",
	Short: "Add a new user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]

		fmt.Printf("Password for %s: ", username)
		passBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return fmt.Errorf("reading password: %w", err)
		}
		password := string(passBytes)
		if password == "" {
			return fmt.Errorf("password cannot be empty")
		}

		manager, err := db.NewManager()
		if err != nil {
			return fmt.Errorf("initializing database: %w", err)
		}
		defer manager.Close()

		userService := core.NewUserService(manager.RegistryDB())
		user, err := userService.Create(context.Background(), username, password)
		if err != nil {
			return err
		}

		fmt.Printf("User %q created (ID: %d)\n", user.Username, user.ID)
		return nil
	},
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered users",
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := db.NewManager()
		if err != nil {
			return fmt.Errorf("initializing database: %w", err)
		}
		defer manager.Close()

		userService := core.NewUserService(manager.RegistryDB())
		users, err := userService.List(context.Background())
		if err != nil {
			return err
		}

		if len(users) == 0 {
			fmt.Println("No users registered. Use 'kvt user add <username>' to create one.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tUsername\tCreated")
		for _, u := range users {
			fmt.Fprintf(w, "%d\t%s\t%s\n", u.ID, u.Username, u.CreatedAt.Format("2006-01-02"))
		}
		w.Flush()
		return nil
	},
}

func init() {
	userCmd.AddCommand(userAddCmd)
	userCmd.AddCommand(userListCmd)
	rootCmd.AddCommand(userCmd)
}
