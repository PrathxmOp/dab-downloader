package main

import (
	"os"

	"github.com/spf13/cobra"

	"dab-downloader/internal/ui"
)

var loginCmd = &cobra.Command{
	Use:   "login [email] [password]",
	Short: "Login to DAB API",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		_, api, _ := initConfigAPIAndDownloader()

		email := args[0]
		password := args[1]

		ui.Info.Println("🔐 Attempting to login...")
		if err := api.Login(email, password); err != nil {
			ui.Error.Printf("❌ Login failed: %v\n", err)
			os.Exit(1)
		}
		ui.Success.Println("✅ Login successful!")
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from DAB API",
	Run: func(cmd *cobra.Command, args []string) {
		_, api, _ := initConfigAPIAndDownloader()

		if err := api.Logout(); err != nil {
			ui.Error.Printf("❌ Logout failed: %v\n", err)
			os.Exit(1)
		}
		ui.Success.Println("✅ Logout successful! Session token removed.")
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check login status",
	Run: func(cmd *cobra.Command, args []string) {
		_, api, _ := initConfigAPIAndDownloader()

		status := api.GetLoginStatus()
		if status.IsLoggedIn {
			ui.Success.Println("✅ " + status.Message)
		} else {
			ui.Warning.Println("⚠️ " + status.Message)
		}
	},
}
