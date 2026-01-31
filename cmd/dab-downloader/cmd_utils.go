package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"dab-downloader/internal/ui"
)

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Run various debugging utilities.",
}

var testApiAvailabilityCmd = &cobra.Command{
	Use:   "api-availability",
	Short: "Test basic DAB API connectivity.",
	Run: func(cmd *cobra.Command, args []string) {
		_, api, _ := initConfigAPIAndDownloader()
		api.TestAPIAvailability(context.Background())
	},
}

var testArtistEndpointsCmd = &cobra.Command{
	Use:   "artist-endpoints [artist_id]",
	Short: "Test different artist endpoint formats for a given artist ID.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		_, api, _ := initConfigAPIAndDownloader()
		artistID := args[0]
		api.TestArtistEndpoints(context.Background(), artistID)
	},
}

var testPlaylistEndpointsCmd = &cobra.Command{
	Use:   "playlist-endpoints [playlist_id]",
	Short: "Test different playlist endpoint formats for a given playlist ID.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		_, api, _ := initConfigAPIAndDownloader()
		playlistID := args[0]
		// Handle URL if provided
		if strings.Contains(playlistID, "/shared/library/") {
			parts := strings.Split(playlistID, "/")
			playlistID = parts[len(parts)-1]
			if strings.Contains(playlistID, "?") {
				playlistID = strings.Split(playlistID, "?")[0]
			}
		}
		api.TestPlaylistEndpoints(context.Background(), playlistID)
	},
}

var comprehensiveArtistDebugCmd = &cobra.Command{
	Use:   "comprehensive-artist-debug [artist_id]",
	Short: "Perform comprehensive debugging for an artist ID.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		_, api, _ := initConfigAPIAndDownloader()
		artistID := args[0]
		api.DebugArtistID(context.Background(), artistID)
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Interactively edit the configuration.",
	Run: func(cmd *cobra.Command, args []string) {
		conf, _, _ := initConfigAPIAndDownloader()
		if err := ui.RunConfigMenu(conf); err != nil {
			ui.Error.Printf("❌ Failed to run config menu: %v\n", err)
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of dab-downloader",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("dab-downloader %s\n", toolVersion)
	},
}

func printInstallInstructions() {
	fmt.Println("\n📦 Install FFmpeg:")
	fmt.Println("• Windows: choco install ffmpeg  or  winget install ffmpeg")
	fmt.Println("• macOS:   brew install ffmpeg")
	fmt.Println("• Ubuntu:  sudo apt install ffmpeg")
	fmt.Println("• Arch:    sudo pacman -S ffmpeg")
	fmt.Println("\n🔄 Restart the application after installation")
}
