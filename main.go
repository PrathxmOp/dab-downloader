package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
)

const toolVersion = "2.0.0" // Example version, can be updated
const authorName = "Prathxm"

var (
	apiURL           string
	downloadLocation string
	debug            bool
	filter           string
	noConfirm        bool
	searchType       string
)

var rootCmd = &cobra.Command{
	Use:     "dab-downloader",
	Version: toolVersion, // Set the version here
	Short:   "A high-quality FLAC music downloader for the DAB API.",
	Long: fmt.Sprintf(`DAB Downloader (v%s) by %s

A modular, high-quality FLAC music downloader with comprehensive metadata support for the DAB API.
It allows you to:
- Download entire artist discographies.
- Download full albums.
- Download individual tracks (by fetching their respective album first).

All downloads feature smart categorization, duplicate detection, and embedded cover art.`, toolVersion, authorName),
}

var artistCmd = &cobra.Command{
	Use:   "artist [artist_id]",
	Short: "Download an artist's entire discography.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		_, api := initConfigAndAPI()
		artistID := args[0]
		colorInfo.Println("🎵 Starting artist discography download for ID:", artistID)
		if err := api.DownloadArtistDiscography(context.Background(), artistID, debug, filter, noConfirm); err != nil {
			colorError.Printf("❌ Failed to download discography: %v\n", err)
		} else {
			colorSuccess.Println("✅ Discography download completed!")
		}
	},
}

var albumCmd = &cobra.Command{
	Use:   "album [album_id]",
	Short: "Download an album by its ID.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config, api := initConfigAndAPI()
		albumID := args[0]
		colorInfo.Println("🎵 Starting album download for ID:", albumID)
		if _, err := api.DownloadAlbum(context.Background(), albumID, config.Parallelism, debug, nil); err != nil {
			colorError.Printf("❌ Failed to download album: %v\n", err)
		} else {
			colorSuccess.Println("✅ Album download completed!")
		}
	},
}



var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for artists, albums, or tracks.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config, api := initConfigAndAPI() // Get config for parallelism
		query := args[0]
		selectedItem, itemType, err := handleSearch(context.Background(), api, query, searchType, debug)
		if err != nil {
			colorError.Printf("❌ Search failed: %v\n", err)
			return
		}
		if selectedItem == nil { // User quit or no results
			return
		}

		switch itemType {
		case "artist":
			artist := selectedItem.(Artist)
			colorInfo.Println("🎵 Starting artist discography download for:", artist.Name)
			if err := api.DownloadArtistDiscography(context.Background(), artist.ID, debug, filter, noConfirm); err != nil {
				colorError.Printf("❌ Failed to download discography for %s: %v\n", artist.Name, err)
			} else {
				colorSuccess.Println("✅ Discography download completed for", artist.Name)
			}
		case "album":
			album := selectedItem.(Album)
			colorInfo.Println("🎵 Starting album download for:", album.Title, "by", album.Artist)
			if _, err := api.DownloadAlbum(context.Background(), album.ID, config.Parallelism, debug, nil); err != nil {
				colorError.Printf("❌ Failed to download album %s: %v\n", album.Title, err)
			} else {
				colorSuccess.Println("✅ Album download completed for", album.Title)
			}
		case "track":
			track := selectedItem.(Track)
			colorInfo.Println("🎵 Starting track download for:", track.Title, "by", track.Artist)
			// Now call the modified DownloadSingleTrack which expects a Track object
			if err := api.DownloadSingleTrack(context.Background(), track, debug); err != nil {
				colorError.Printf("❌ Failed to download track %s: %v\n", track.Title, err)
			} else {
				colorSuccess.Println("✅ Track download completed for", track.Title)
			}
		default:
			colorError.Println("❌ Unknown item type selected.")
		}
	},
}

func initConfigAndAPI() (*Config, *DabAPI) {
	config := &Config{
		APIURL:           "https://dab.yeet.su",
		DownloadLocation: filepath.Join(os.Getenv("HOME"), "Music"),
		Parallelism:      5,
	}

	// Define the config file path in the current directory
	configFile := "config.json" // Changed to current directory

	// Check if config file exists
	if !FileExists(configFile) {
		colorInfo.Println("✨ Welcome to DAB Downloader! Let's set up your configuration.")

		// Prompt for API URL
		defaultAPIURL := config.APIURL
		config.APIURL = GetUserInput(fmt.Sprintf("Enter DAB API URL (e.g., %s)", defaultAPIURL), defaultAPIURL)

		// Prompt for Download Location
		defaultDownloadLocation := config.DownloadLocation
		config.DownloadLocation = GetUserInput(fmt.Sprintf("Enter download location (e.g., %s)", defaultDownloadLocation), defaultDownloadLocation)

		// Prompt for Parallelism
		defaultParallelism := strconv.Itoa(config.Parallelism)
		parallelismStr := GetUserInput(fmt.Sprintf("Enter number of parallel downloads (default: %s)", defaultParallelism), defaultParallelism)
		if p, err := strconv.Atoi(parallelismStr); err == nil && p > 0 {
			config.Parallelism = p
		} else {
			colorWarning.Printf("⚠️ Invalid parallelism value '%s', using default %d.\n", parallelismStr, config.Parallelism)
		}

		// Save the new config
		if err := SaveConfig(configFile, config); err != nil {
			colorError.Printf("❌ Failed to save initial config: %v\n", err)
		} else {
			colorSuccess.Println("✅ Configuration saved to", configFile)
		}
	} else {
		// Load existing config
		if err := LoadConfig(configFile, config); err != nil {
			colorError.Printf("❌ Failed to load config from %s: %v\n", configFile, err)
		} else {
			colorInfo.Println("✅ Loaded configuration from", configFile)
		}
	}

	// Command-line flags override config file
	if apiURL != "" {
		config.APIURL = apiURL
	}
	if downloadLocation != "" {
		config.DownloadLocation = downloadLocation
	}

	api := NewDabAPI(config.APIURL, config.DownloadLocation)
	return config, api
}

func init() {
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "", "DAB API URL")
	rootCmd.PersistentFlags().StringVar(&downloadLocation, "download-location", "", "Directory to save downloads")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")

	artistCmd.Flags().StringVar(&filter, "filter", "all", "Filter by item type (albums, eps, singles), comma-separated")
	artistCmd.Flags().BoolVar(&noConfirm, "no-confirm", false, "Skip confirmation prompt")

	searchCmd.Flags().StringVar(&searchType, "type", "all", "Type of content to search for (artist, album, track, all)")

	rootCmd.AddCommand(artistCmd)
	rootCmd.AddCommand(albumCmd)
	
	rootCmd.AddCommand(searchCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}