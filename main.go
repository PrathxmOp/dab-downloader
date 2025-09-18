package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	

	"github.com/cheggaaa/pb/v3"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var toolVersion string
const authorName = "Prathxm"

var (
	apiURL              string
	downloadLocation    string
	debug               bool
	filter              string
	noConfirm           bool
	searchType          string
	spotifyPlaylist     string
	spotifyClientID     string
	spotifyClientSecret string
	auto                bool
	expandPlaylist      bool
	navidromeURL        string
	navidromeUsername   string
	navidromePassword   string
	format              string = "flac"
	bitrate             string = "320"
	ignoreSuffix      string
)

var rootCmd = &cobra.Command{
	Use:     "dab-downloader",
	Short:   "A high-quality FLAC music downloader for the DAB API.",
}

var artistCmd = &cobra.Command{
	Use:   "artist [artist_id]",
	Short: "Download an artist's entire discography.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
			config, api := initConfigAndAPI()
			if config.Format != "flac" && !CheckFFmpeg() {
				printInstallInstructions()
				return
			}
			artistID := args[0]
			colorInfo.Println("🎵 Starting artist discography download for ID:", artistID)
			if err := api.DownloadArtistDiscography(context.Background(), artistID, config, debug, filter, noConfirm); err != nil {
				if errors.Is(err, ErrDownloadCancelled) {
					colorWarning.Println("⚠️ Discography download cancelled by user.")
				} else if errors.Is(err, ErrNoItemsSelected) {
                    colorWarning.Println("⚠️ No items were selected for download.")
                } else {
                    colorError.Printf("❌ Failed to download discography: %v\n", err)
                }
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
			if config.Format != "flac" && !CheckFFmpeg() {
				printInstallInstructions()
				return
			}
			albumID := args[0]
			colorInfo.Println("🎵 Starting album download for ID:", albumID)
			if _, err := api.DownloadAlbum(context.Background(), albumID, config, debug, nil); err != nil {
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
	Example: `  # Search for albums containing \"parat 3\"\n  dab-downloader search \"parat 3\" --type album\n\n  # Search for artists named \"coldplay\"\n  dab-downloader search \"coldplay\" --type artist\n\n  # Search for tracks named \"paradise\" and automatically download the first result\n  dab-downloader search \"paradise\" --type track --auto`,
	Run: func(cmd *cobra.Command, args []string) {
		config, api := initConfigAndAPI() // Get config for parallelism
			if config.Format != "flac" && !CheckFFmpeg() {
				colorError.Println("❌ ffmpeg is not installed or not in your PATH. Please install ffmpeg to use the format conversion feature.")
				return
			}
			query := args[0]
			selectedItems, itemTypes, err := handleSearch(context.Background(), api, query, searchType, debug, auto)
			if err != nil {
				colorError.Printf("❌ Search failed: %v\n", err)
				return
			}
			if len(selectedItems) == 0 { // User quit or no results
				return
			}


			// Initialize pool for multiple track downloads
			var pool *pb.Pool
			var localPool bool
			if isTTY() && len(selectedItems) > 1 { // Only create pool if multiple items and TTY
				var err error
				pool, err = pb.StartPool()
				if err != nil {
					colorError.Printf("❌ Failed to start progress bar pool: %v\n", err)
					// Continue without the pool
				} else {
					localPool = true
				}
			}

			for i, selectedItem := range selectedItems {
				itemType := itemTypes[i]
				switch itemType {
				case "artist":
					artist := selectedItem.(Artist)
					colorInfo.Println("🎵 Starting artist discography download for:", artist.Name)
					artistIDStr := idToString(artist.ID) // Convert ID to string using idToString
					if debug { // Add this debug print
						colorInfo.Printf("DEBUG - Passing artistIDStr to DownloadArtistDiscography: '%s'\n", artistIDStr)
					}
					if err := api.DownloadArtistDiscography(context.Background(), artistIDStr, config, debug, filter, noConfirm); err != nil {
						colorError.Printf("❌ Failed to download discography for %s: %v\n", artist.Name, err)
					} else {
						colorSuccess.Println("✅ Discography download completed for", artist.Name)
					}
				case "album":
					album := selectedItem.(Album)
					colorInfo.Println("🎵 Starting album download for:", album.Title, "by", album.Artist)
					if _, err := api.DownloadAlbum(context.Background(), album.ID, config, debug, nil); err != nil {
						colorError.Printf("❌ Failed to download album %s: %v\n", album.Title, err)
					} else {
						colorSuccess.Println("✅ Album download completed for", album.Title)
					}
				case "track":
					track := selectedItem.(Track)
					colorInfo.Println("🎵 Starting track download for:", track.Title, "by", track.Artist)
					// Now call the modified DownloadSingleTrack which expects a Track object and potentially a pool
					if err := api.DownloadSingleTrack(context.Background(), track, debug, config.Format, config.Bitrate, pool); err != nil {
						colorError.Printf("❌ Failed to download track %s: %v\n", track.Title, err)
					} else {
						colorSuccess.Println("✅ Track download completed for", track.Title)
					}
				default:
					colorError.Println("❌ Unknown item type selected.")
				}
			}

			if localPool && pool != nil {
				pool.Stop()
			}
		},
}

var spotifyCmd = &cobra.Command{
	Use:   "spotify [url]",
	Short: "Download a Spotify playlist or album.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config, api := initConfigAndAPI()
			if config.Format != "flac" && !CheckFFmpeg() {
				colorError.Println("❌ ffmpeg is not installed or not in your PATH. Please install ffmpeg to use the format conversion feature.")
				return
			}
			url := args[0]

			spotifyClient := NewSpotifyClient(config.SpotifyClientID, config.SpotifyClientSecret)
			if err := spotifyClient.Authenticate(); err != nil {
				colorError.Printf("❌ Failed to authenticate with Spotify: %v\n", err)
				return
			}

			var spotifyTracks []SpotifyTrack
			var err error

			if strings.Contains(url, "/playlist/") {
				spotifyTracks, _, err = spotifyClient.GetPlaylistTracks(url)
			} else if strings.Contains(url, "/album/") {
				spotifyTracks, _, err = spotifyClient.GetAlbumTracks(url) // I need to implement this
			} else {
				colorError.Println("❌ Invalid Spotify URL. Please provide a playlist or album URL.")
				return
			}

			if err != nil {
				colorError.Printf("❌ Failed to get tracks from Spotify: %v\n", err)
				return
			}

			if expandPlaylist {
				colorInfo.Println("Expanding playlist to download full albums...")

				// --- Logic for --expand flag ---
			
uniqueAlbums := make(map[string]SpotifyTrack)
				for _, track := range spotifyTracks {
					// Use a consistent key for the map
					albumKey := strings.ToLower(track.AlbumName + " - " + track.AlbumArtist)
					if _, exists := uniqueAlbums[albumKey]; !exists {
					
uniqueAlbums[albumKey] = track
					}
				}

				colorInfo.Printf("Found %d unique albums in the playlist.\n", len(uniqueAlbums))

				for _, track := range uniqueAlbums {
					albumSearchQuery := track.AlbumName + " - " + track.AlbumArtist
					colorInfo.Printf("Searching for album: %s\n", albumSearchQuery)

					// Use handleSearch to find the album on DAB
					selectedItems, itemTypes, err := handleSearch(context.Background(), api, albumSearchQuery, "album", debug, auto)
					if err != nil {
						colorError.Printf("❌ Search failed for album '%s': %v\n", albumSearchQuery, err)
						continue // Move to the next album
					}

					if len(selectedItems) == 0 {
						colorWarning.Printf("⚠️ No results found for album: %s\n", albumSearchQuery)
						continue
					}

					// Download the first result (or the one selected by the user)
					for i, selectedItem := range selectedItems {
						if itemTypes[i] == "album" {
							album := selectedItem.(Album)
							colorInfo.Println("🎵 Starting album download for:", album.Title, "by", album.Artist)
							if _, err := api.DownloadAlbum(context.Background(), album.ID, config, debug, nil); err != nil {
								colorError.Printf("❌ Failed to download album %s: %v\n", album.Title, err)
							} else {
								colorSuccess.Println("✅ Album download completed for", album.Title)
							}
						break // Only download the first album result for this search
						}
					}
				}
				// --- End of logic for --expand flag ---
				return // Exit after album downloads are done
			}

			var tracks []string
			for _, spotifyTrack := range spotifyTracks {
				tracks = append(tracks, spotifyTrack.Name+" - "+spotifyTrack.Artist)
			}

			// Initialize pool for multiple track downloads
			var pool *pb.Pool
			var localPool bool
			if isTTY() && len(spotifyTracks) > 1 { // Only create pool if multiple items and TTY
				var err error
				pool, err = pb.StartPool()
				if err != nil {
					colorError.Printf("❌ Failed to start progress bar pool: %v\n", err)
					// Continue without the pool
				} else {
					localPool = true
				}
			}

			for _, spotifyTrack := range spotifyTracks {
				trackName := spotifyTrack.Name + " - " + spotifyTrack.Artist // Construct search query
				selectedItems, itemTypes, err := handleSearch(context.Background(), api, trackName, "track", debug, auto)
				if err != nil {
					colorError.Printf("❌ Search failed for track %s: %v\n", trackName, err)
					if pool != nil {
						pool.Stop() // Stop pool on error
					}
					return // Exit on search error
				}

				if len(selectedItems) == 0 {
					colorWarning.Printf("⚠️ No results found for track: %s\n", trackName)
					continue
				}

				for i, selectedItem := range selectedItems {
					itemType := itemTypes[i]
					if itemType == "track" {
						track := selectedItem.(Track)
						colorInfo.Println("🎵 Starting track download for:", track.Title, "by", track.Artist)
						if err := api.DownloadSingleTrack(context.Background(), track, debug, config.Format, config.Bitrate, pool); err != nil {
							colorError.Printf("❌ Failed to download track %s: %v\n", track.Title, err)
						} else {
							colorSuccess.Println("✅ Track download completed for", track.Title)
						}
					}
				}
			}

			if localPool && pool != nil {
				pool.Stop()
			}
		},
}

var navidromeCmd = &cobra.Command{
	Use:   "navidrome [spotify_url]",
	Short: "Copy a Spotify playlist or album to Navidrome.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config, api := initConfigAndAPI()
		spotifyURL := args[0]

		spotifyClient := NewSpotifyClient(config.SpotifyClientID, config.SpotifyClientSecret)
		if err := spotifyClient.Authenticate(); err != nil {
			colorError.Printf("❌ Failed to authenticate with Spotify: %v\n", err)
			return
		}

		var spotifyTracks []SpotifyTrack
		var spotifyName string
		var err error

		if strings.Contains(spotifyURL, "/playlist/") {
			spotifyTracks, spotifyName, err = spotifyClient.GetPlaylistTracks(spotifyURL)
		} else if strings.Contains(spotifyURL, "/album/") {
			spotifyTracks, spotifyName, err = spotifyClient.GetAlbumTracks(spotifyURL)
		} else {
			colorError.Println("❌ Invalid Spotify URL. Please provide a playlist or album URL.")
			return
		}

		if err != nil {
			colorError.Printf("❌ Failed to get tracks from Spotify: %v\n", err)
			return
		}


		navidromeClient := NewNavidromeClient(config.NavidromeURL, config.NavidromeUsername, config.NavidromePassword)
		if err := navidromeClient.Authenticate(); err != nil {
			colorError.Printf("❌ Failed to authenticate with Navidrome: %v\n", err)
			return
		}

		playlistName := GetUserInput("Enter a name for the new Navidrome playlist", spotifyName) // MODIFIED
		if err := navidromeClient.CreatePlaylist(playlistName); err != nil {
			colorError.Printf("❌ Failed to create Navidrome playlist: %v\n", err)
			return
		}

		playlistID, err := navidromeClient.SearchPlaylist(playlistName)
		if err != nil {
			colorError.Printf("❌ Failed to find newly created playlist '%s': %v\n", playlistName, err)
			return
		}

		var navidromeTrackIDs []string // New slice to store Navidrome track IDs

		for _, spotifyTrack := range spotifyTracks { // Iterate over SpotifyTrack
			trackName := spotifyTrack.Name
			if ignoreSuffix != "" {
				trackName = removeSuffix(trackName, ignoreSuffix)
			}
			track, err := navidromeClient.SearchTrack(trackName, spotifyTrack.Artist) // Pass name and artist separately
			if err != nil {
				colorWarning.Printf("⚠️ Error searching for track %s by %s on Navidrome: %v\n", spotifyTrack.Name, spotifyTrack.Artist, err)
				continue
			}

			if track != nil {
				navidromeTrackIDs = append(navidromeTrackIDs, track.ID) // Collect track IDs
				colorSuccess.Printf("✅ Found track %s by %s on Navidrome (ID: %s)\n", spotifyTrack.Name, spotifyTrack.Artist, track.ID)
			} else {
				colorWarning.Printf("⚠️ Track %s by %s not found on Navidrome. Searching DAB...\n", spotifyTrack.Name, spotifyTrack.Artist)

				// Search DAB for the track
				dabSearchQuery := spotifyTrack.Name + " - " + spotifyTrack.Artist
				if ignoreSuffix != "" {
					dabSearchQuery = trackName + " - " + spotifyTrack.Artist
				}
				dabSearchResults, dabItemTypes, err := handleSearch(context.Background(), api, dabSearchQuery, "track", debug, auto)
				if err != nil {
					colorError.Printf("❌ Failed to search DAB for %s: %v\n", spotifyTrack.Name, err)
					continue
				}

				if len(dabSearchResults) > 0 {
					// Assuming the first result is the desired one if auto is true, or user selected one
					selectedDabItem := dabSearchResults[0]
					selectedDabItemType := dabItemTypes[0]

					if selectedDabItemType == "track" {
						dabTrack := selectedDabItem.(Track)
					colorInfo.Printf("🎵 Downloading %s by %s from DAB...\n", dabTrack.Title, dabTrack.Artist)
						if err := api.DownloadSingleTrack(context.Background(), dabTrack, debug, config.Format, config.Bitrate, nil); err != nil {
							colorError.Printf("❌ Failed to download track %s from DAB: %v\n", dabTrack.Title, err)
						} else {
							colorSuccess.Printf("✅ Downloaded %s by %s from DAB. It should appear in Navidrome soon.\n", dabTrack.Title, dabTrack.Artist)
							// After downloading, try to search for it in Navidrome again and add to playlist
							// This might require a small delay for Navidrome to scan the new file
							time.Sleep(5 * time.Second) // Give Navidrome some time to scan
							reScannedTrack, err := navidromeClient.SearchTrack(dabTrack.Title, dabTrack.Artist)
							if err != nil {
								colorWarning.Printf("⚠️ Failed to re-search for downloaded track %s in Navidrome: %v\n", dabTrack.Title, err)
							} else if reScannedTrack != nil {
								navidromeTrackIDs = append(navidromeTrackIDs, reScannedTrack.ID)
								colorSuccess.Printf("✅ Found newly downloaded track %s in Navidrome (ID: %s) and added to list for playlist.\n", reScannedTrack.Title, reScannedTrack.ID)
							} else {
								colorWarning.Printf("⚠️ Downloaded track %s not found in Navidrome after re-scan. It might be added later manually.\n", dabTrack.Title)
							}
						}
					} else {
						colorWarning.Printf("⚠️ DAB search for %s returned a non-track item type: %s. Skipping download.\n", spotifyTrack.Name, selectedDabItemType)
					}
				} else {
					colorWarning.Printf("⚠️ No results found on DAB for %s.\n", spotifyTrack.Name)
				}
			}
		}

		// Add all collected tracks to the playlist in a single call
		if len(navidromeTrackIDs) > 0 {
			if err := navidromeClient.AddTracksToPlaylist(playlistID, navidromeTrackIDs); err != nil { // New method call
				colorError.Printf("❌ Failed to add tracks to Navidrome playlist: %v\n", err)
			} else {
				colorSuccess.Printf("✅ Successfully added %d tracks to Navidrome playlist '%s'\n", len(navidromeTrackIDs), playlistName)

				// Verify that the tracks were added
				time.Sleep(2 * time.Second) // Add a small delay
				playlistTracks, err := navidromeClient.GetPlaylistTracks(playlistID)
				if err != nil {
					colorWarning.Printf("⚠️ Failed to verify playlist tracks: %v\n", err)
				} else {
					colorInfo.Printf("🔍 Found %d tracks in playlist '%s'\n", len(playlistTracks), playlistName)
					if len(playlistTracks) == len(navidromeTrackIDs) {
						colorSuccess.Println("✅ All tracks successfully added to the playlist.")
					} else {
						colorWarning.Printf("⚠️ Expected %d tracks, but found %d.\n", len(navidromeTrackIDs), len(playlistTracks))
					}
				}
			}
		} else {
			colorWarning.Printf("⚠️ No tracks found to add to Navidrome playlist '%s'\n", playlistName)
		}
	},
}

var addToPlaylistCmd = &cobra.Command{
	Use:   "add-to-playlist [playlist_id] [song_id...]",
	Short: "Add one or more songs to a Navidrome playlist.",
	Args:  cobra.MinimumNArgs(2), // At least playlist ID and one song ID
	Run: func(cmd *cobra.Command, args []string) {
		config, _ := initConfigAndAPI()

		navidromeClient := NewNavidromeClient(config.NavidromeURL, config.NavidromeUsername, config.NavidromePassword)
		if err := navidromeClient.Authenticate(); err != nil {
			colorError.Printf("❌ Failed to authenticate with Navidrome: %v\n", err)
			return
		}

		playlistID := args[0]
		songIDs := args[1:]

		if err := navidromeClient.AddTracksToPlaylist(playlistID, songIDs); err != nil {
			colorError.Printf("❌ Failed to add tracks to playlist %s: %v\n", playlistID, err)
		} else {
			colorSuccess.Printf("✅ Successfully added %d tracks to playlist %s\n", len(songIDs), playlistID)
		}
	},
}

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Run various debugging utilities.",
	Long:  "Provides commands to test API connectivity and artist endpoint formats.",
}

var testApiAvailabilityCmd = &cobra.Command{
	Use:   "api-availability",
	Short: "Test basic DAB API connectivity.",
	Run: func(cmd *cobra.Command, args []string) {
		_, api := initConfigAndAPI()
		api.TestAPIAvailability(context.Background())
	},
}

var testArtistEndpointsCmd = &cobra.Command{
	Use:   "artist-endpoints [artist_id]",
	Short: "Test different artist endpoint formats for a given artist ID.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		_, api := initConfigAndAPI()
		artistID := args[0]
		api.TestArtistEndpoints(context.Background(), artistID)
	},
}

var comprehensiveArtistDebugCmd = &cobra.Command{
	Use:   "comprehensive-artist-debug [artist_id]",
	Short: "Perform comprehensive debugging for an artist ID (API connectivity, endpoint formats, and ID type checks).",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		_, api := initConfigAndAPI()
		artistID := args[0]
		api.DebugArtistID(context.Background(), artistID)
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

    fmt.Println("\nðŸ“¦ Install FFmpeg:")
    fmt.Println("â€¢ Windows: choco install ffmpeg  or  winget install ffmpeg")
    fmt.Println("â€¢ macOS:   brew install ffmpeg")
    fmt.Println("â€¢ Ubuntu:  sudo apt install ffmpeg")
    fmt.Println("â€¢ Arch:    sudo pacman -S ffmpeg")
    fmt.Println("\nðŸ”„ Restart the application after installation")
}

func initConfigAndAPI() (*Config, *DabAPI) {
	color.NoColor = !isTTY() // Initialize color output
	homeDir, err := os.UserHomeDir()
	if err != nil {
		colorWarning.Println("⚠️ Could not determine home directory, will use current directory for downloads.")
		homeDir = "." // or some other sensible default
	}

	config := &Config{
		APIURL:           "https://dab.yeet.su",
		DownloadLocation: filepath.Join(homeDir, "Music"),
		Parallelism:      5,
		UpdateRepo:       "PrathxmOp/dab-downloader", // Default value
	}

	// Define the config file path in the current directory
	configFile := filepath.Join("config", "config.json")

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

		// Prompt for Spotify Credentials
		config.SpotifyClientID = GetUserInput("Enter your Spotify Client ID", "")
		config.SpotifyClientSecret = GetUserInput("Enter your Spotify Client Secret", "")

		// Prompt for Navidrome Credentials
		config.NavidromeURL = GetUserInput("Enter your Navidrome URL", "")
		config.NavidromeUsername = GetUserInput("Enter your Navidrome Username", "")
		config.NavidromePassword = GetUserInput("Enter your Navidrome Password", "")

		// Prompt for Format and Bitrate
		config.Format = GetUserInput("Enter default output format (e.g., flac, mp3, ogg, opus)", "flac")
		config.Bitrate = GetUserInput("Enter default bitrate for lossy formats (e.g., 320)", "320")

		// Prompt for Update Repository
		config.UpdateRepo = GetUserInput("Enter GitHub repository for updates (e.g., PrathxmOp/dab-downloader)", "PrathxmOp/dab-downloader")

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
			// Set defaults if not present in config file
			if config.Format == "" {
				config.Format = "flac"
			}
			if config.Bitrate == "" {
				config.Bitrate = "320"
			}
			
		}
	}

	// Command-line flags override config file
	if apiURL != "" {
		config.APIURL = apiURL
	}
	if downloadLocation != "" {
		config.DownloadLocation = downloadLocation
	}
	if spotifyClientID != "" {
		config.SpotifyClientID = spotifyClientID
	}
	if spotifyClientSecret != "" {
		config.SpotifyClientSecret = spotifyClientSecret
	}
	if navidromeURL != "" {
		config.NavidromeURL = navidromeURL
	}
	if navidromeUsername != "" {
		config.NavidromeUsername = navidromeUsername
	}
	if navidromePassword != "" {
		config.NavidromePassword = navidromePassword
	}

	// Override config with command-line flags if provided
	if format != "flac" { // Check if format flag was explicitly set
		config.Format = format
	}
	if bitrate != "320" { // Check if bitrate flag was explicitly set
		config.Bitrate = bitrate
	}

	api := NewDabAPI(config.APIURL, config.DownloadLocation)
	return config, api
}

func init() {
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "", "DAB API URL")
	rootCmd.PersistentFlags().StringVar(&downloadLocation, "download-location", "", "Directory to save downloads")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")

	albumCmd.Flags().StringVar(&format, "format", "flac", "Format to convert to after downloading (e.g., mp3, ogg, opus)")
	albumCmd.Flags().StringVar(&bitrate, "bitrate", "320", "Bitrate for lossy formats (in kbps, e.g., 192, 256, 320)")

	artistCmd.Flags().StringVar(&filter, "filter", "all", "Filter by item type (albums, eps, singles), comma-separated")
	artistCmd.Flags().BoolVar(&noConfirm, "no-confirm", false, "Skip confirmation prompt")
	artistCmd.Flags().StringVar(&format, "format", "flac", "Format to convert to after downloading (e.g., mp3, ogg, opus)")
	artistCmd.Flags().StringVar(&bitrate, "bitrate", "320", "Bitrate for lossy formats (in kbps, e.g., 192, 256, 320)")

	searchCmd.Flags().StringVar(&searchType, "type", "all", "Type of content to search for (artist, album, track, all)")
	searchCmd.Flags().BoolVar(&auto, "auto", false, "Automatically download the first result")
	searchCmd.Flags().StringVar(&format, "format", "flac", "Format to convert to after downloading (e.g., mp3, ogg, opus)")
	searchCmd.Flags().StringVar(&bitrate, "bitrate", "320", "Bitrate for lossy formats (in kbps, e.g., 192, 256, 320)")

	spotifyCmd.Flags().StringVar(&spotifyPlaylist, "spotify", "", "Spotify playlist URL to download")
	spotifyCmd.Flags().BoolVar(&auto, "auto", false, "Automatically download the first result")
	spotifyCmd.Flags().BoolVar(&expandPlaylist, "expand", false, "Expand playlist tracks to download the full albums")
	spotifyCmd.Flags().StringVar(&format, "format", "flac", "Format to convert to after downloading (e.g., mp3, ogg, opus)")
	spotifyCmd.Flags().StringVar(&bitrate, "bitrate", "320", "Bitrate for lossy formats (in kbps, e.g., 192, 256, 320)")
	rootCmd.PersistentFlags().StringVar(&spotifyClientID, "spotify-client-id", "", "Spotify Client ID")
	rootCmd.PersistentFlags().StringVar(&spotifyClientSecret, "spotify-client-secret", "", "Spotify Client Secret")

	rootCmd.PersistentFlags().StringVar(&navidromeURL, "navidrome-url", "", "Navidrome URL")
	rootCmd.PersistentFlags().StringVar(&navidromeUsername, "navidrome-username", "", "Navidrome Username")
	rootCmd.PersistentFlags().StringVar(&navidromePassword, "navidrome-password", "", "Navidrome Password")
	navidromeCmd.Flags().StringVar(&ignoreSuffix, "ignore-suffix", "", "Ignore suffix when searching for tracks")

	rootCmd.AddCommand(artistCmd)
	rootCmd.AddCommand(albumCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(spotifyCmd)
	rootCmd.AddCommand(navidromeCmd)
	rootCmd.AddCommand(addToPlaylistCmd)
	rootCmd.AddCommand(debugCmd)

	debugCmd.AddCommand(testApiAvailabilityCmd)
	debugCmd.AddCommand(testArtistEndpointsCmd)
	debugCmd.AddCommand(comprehensiveArtistDebugCmd)

	rootCmd.AddCommand(versionCmd)
}

func main() {
	var versionInfo VersionInfo
	if err := json.Unmarshal(versionJSON, &versionInfo); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading embedded version.json: %v\n", err)
		os.Exit(1)
	}
	toolVersion = versionInfo.Version

	// Set rootCmd.Version after toolVersion is populated
	rootCmd.Version = toolVersion

	// Set rootCmd.Long after toolVersion is populated
	rootCmd.Long = fmt.Sprintf("DAB Downloader (v%s) by %s\n\nA modular, high-quality FLAC music downloader with comprehensive metadata support for the DAB API.\nIt allows you to:\n- Download entire artist discographies.\n- Download full albums.\n- Download individual tracks (by fetching their respective album first).\n- Import and download Spotify playlists and albums.\n- Convert downloaded files to various formats (e.g., MP3, OGG, Opus) with specified bitrates.\n\nAll downloads feature smart categorization, duplicate detection, and embedded cover art.", toolVersion, authorName)

	// Now call CheckForUpdates with the config
	config, _ := initConfigAndAPI() // Temporarily load config here to get IsDockerContainer

	// Check if running in Docker
	if _, err := os.Stat("/.dockerenv"); err == nil {
		config.IsDockerContainer = true
	}

	CheckForUpdates(config, toolVersion)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
