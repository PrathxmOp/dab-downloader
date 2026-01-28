package main

import (
	"context"
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"dab-downloader/internal/api"
	"dab-downloader/internal/config"
	"dab-downloader/internal/downloader"
	"dab-downloader/internal/ffmpeg"
	"dab-downloader/internal/models"
	"dab-downloader/internal/ui"
	"dab-downloader/internal/updater"
	"dab-downloader/internal/utils"
	"dab-downloader/internal/version"
	"dab-downloader/pkg/navidrome"
	"dab-downloader/pkg/spotify"
	"dab-downloader/pkg/listenbrainz"
)

var toolVersion string

const authorName = "Prathxm"

//go:embed version.json
var versionJSON []byte

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
	expandNavidrome     bool
	navidromeURL        string
	navidromeUsername   string
	navidromePassword   string
	format              string = "flac"
	bitrate             string = "320"
	ignoreSuffix        string
	insecure            bool
	warningBehavior     string = "summary"
)

var rootCmd = &cobra.Command{
	Use:   "dab-downloader",
	Short: "A high-quality FLAC music downloader for the DAB API.",
}

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

var artistCmd = &cobra.Command{
	Use:   "artist [artist_id]",
	Short: "Download an artist's entire discography.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		conf, _, d := initConfigAPIAndDownloader()
		if conf.Format != "flac" && !ffmpeg.CheckFFmpeg() {
			printInstallInstructions()
			return
		}
		artistID := args[0]
		ui.Info.Println("🎵 Starting artist discography download for ID:", artistID)
		if err := d.DownloadArtistDiscography(context.Background(), artistID, conf, debug, filter, noConfirm, nil); err != nil {
			if errors.Is(err, models.ErrDownloadCancelled) {
				ui.Warning.Println("⚠️ Discography download cancelled by user.")
			} else if errors.Is(err, models.ErrNoItemsSelected) {
				ui.Warning.Println("⚠️ No items were selected for download.")
			} else {
				ui.Error.Printf("❌ Failed to download discography: %v\n", err)
			}
		} else {
			ui.Success.Println("✅ Discography download completed!")
		}
	},
}

var albumCmd = &cobra.Command{
	Use:   "album [album_id]",
	Short: "Download an album by its ID.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		conf, _, d := initConfigAPIAndDownloader()
		if conf.Format != "flac" && !ffmpeg.CheckFFmpeg() {
			printInstallInstructions()
			return
		}
		albumID := args[0]
		ui.Info.Println("🎵 Starting album download for ID:", albumID)
		if _, err := d.DownloadAlbum(context.Background(), albumID, conf, debug, nil, nil); err != nil {
			ui.Error.Printf("❌ Failed to download album: %v\n", err)
		} else {
			ui.Success.Println("✅ Album download completed!")
		}
	},
}

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for artists, albums, or tracks.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		conf, a, d := initConfigAPIAndDownloader()
		if conf.Format != "flac" && !ffmpeg.CheckFFmpeg() {
			ui.Error.Println("❌ ffmpeg is not installed or not in your PATH. Please install ffmpeg to use the format conversion feature.")
			return
		}
		query := args[0]
		selectedItems, itemTypes, err := ui.HandleSearch(context.Background(), a, query, searchType, debug, auto)
		if err != nil {
			ui.Error.Printf("❌ Search failed: %v\n", err)
			return
		}
		if len(selectedItems) == 0 {
			return
		}

		time.Sleep(100 * time.Millisecond)

		var pool *pb.Pool
		var localPool bool
		if utils.IsTTY() && len(selectedItems) > 1 {
			var err error
			pool, err = pb.StartPool()
			if err != nil {
				ui.Error.Printf("❌ Failed to start progress bar pool: %v\n", err)
			} else {
				localPool = true
			}
		}

		for i, selectedItem := range selectedItems {
			itemType := itemTypes[i]
			switch itemType {
			case "artist":
				artist := selectedItem.(models.Artist)
				ui.Info.Println("🎵 Starting artist discography download for:", artist.Name)
				artistIDStr := utils.IDToString(artist.ID)
				if err := d.DownloadArtistDiscography(context.Background(), artistIDStr, conf, debug, filter, noConfirm, pool); err != nil {
					ui.Error.Printf("❌ Failed to download discography for %s: %v\n", artist.Name, err)
				} else {
					ui.Success.Println("✅ Discography download completed for", artist.Name)
				}
			case "album":
				album := selectedItem.(models.Album)
				ui.Info.Println("🎵 Starting album download for:", album.Title, "by", album.Artist)
				if _, err := d.DownloadAlbum(context.Background(), album.ID, conf, debug, pool, nil); err != nil {
					ui.Error.Printf("❌ Failed to download album %s: %v\n", album.Title, err)
				} else {
					ui.Success.Println("✅ Album download completed for", album.Title)
				}
			case "track":
				track := selectedItem.(models.Track)
				ui.Info.Println("🎵 Starting track download for:", track.Title, "by", track.Artist)
				if err := d.DownloadSingleTrack(context.Background(), track, debug, conf.Format, conf.Bitrate, pool, conf, nil); err != nil {
					ui.Error.Printf("❌ Failed to download track %s: %v\n", track.Title, err)
				} else {
					ui.Success.Println("✅ Track download completed for", track.Title)
				}
			default:
				ui.Error.Println("❌ Unknown item type selected.")
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
		conf, a, d := initConfigAPIAndDownloader()
		if conf.Format != "flac" && !ffmpeg.CheckFFmpeg() {
			ui.Error.Println("❌ ffmpeg is not installed or not in your PATH. Please install ffmpeg to use the format conversion feature.")
			return
		}
		url := args[0]

		spotifyClient := spotify.NewSpotifyClient(conf.SpotifyClientID, conf.SpotifyClientSecret)
		if err := spotifyClient.Authenticate(); err != nil {
			ui.Error.Printf("❌ Failed to authenticate with Spotify: %v\n", err)
			return
		}

		var spotifyTracks []spotify.SpotifyTrack
		var err error

		if strings.Contains(url, "/playlist/") {
			spotifyTracks, _, err = spotifyClient.GetPlaylistTracks(url)
		} else if strings.Contains(url, "/album/") {
			spotifyTracks, _, err = spotifyClient.GetAlbumTracks(url)
		} else {
			ui.Error.Println("❌ Invalid Spotify URL. Please provide a playlist or album URL.")
			return
		}

		if err != nil {
			ui.Error.Printf("❌ Failed to get tracks from Spotify: %v\n", err)
			return
		}

		if expandPlaylist {
			ui.Info.Println("Expanding playlist to download full albums...")
			uniqueAlbums := make(map[string]spotify.SpotifyTrack)
			for _, track := range spotifyTracks {
				albumKey := strings.ToLower(track.AlbumName + " - " + track.AlbumArtist)
				if _, exists := uniqueAlbums[albumKey]; !exists {
					uniqueAlbums[albumKey] = track
				}
			}

			ui.Info.Printf("Found %d unique albums in the playlist.\n", len(uniqueAlbums))

			for _, track := range uniqueAlbums {
				albumSearchQuery := track.AlbumName + " - " + track.AlbumArtist
				ui.Info.Printf("Searching for album: %s\n", albumSearchQuery)

				selectedItems, itemTypes, err := ui.HandleSearch(context.Background(), a, albumSearchQuery, "album", debug, auto)
				if err != nil {
					ui.Error.Printf("❌ Search failed for album '%s': %v\n", albumSearchQuery, err)
					continue
				}

				if len(selectedItems) == 0 {
					ui.Warning.Printf("⚠️ No results found for album: %s\n", albumSearchQuery)
					continue
				}

				time.Sleep(500 * time.Millisecond)

				for i, selectedItem := range selectedItems {
					if itemTypes[i] == "album" {
						album := selectedItem.(models.Album)
						ui.Info.Println("🎵 Starting album download for:", album.Title, "by", album.Artist)
						if _, err := d.DownloadAlbum(context.Background(), album.ID, conf, debug, nil, nil); err != nil {
							ui.Error.Printf("❌ Failed to download album %s: %v\n", album.Title, err)
						} else {
							ui.Success.Println("✅ Album download completed for", album.Title)
						}
						break
					}
				}
			}
			return
		}

		var pool *pb.Pool
		var localPool bool
		if utils.IsTTY() && len(spotifyTracks) > 1 {
			var err error
			pool, err = pb.StartPool()
			if err != nil {
				ui.Error.Printf("❌ Failed to start progress bar pool: %v\n", err)
			} else {
				localPool = true
			}
		}

		for _, spotifyTrack := range spotifyTracks {
			trackName := spotifyTrack.Name + " - " + spotifyTrack.Artist
			selectedItems, itemTypes, err := ui.HandleSearch(context.Background(), a, trackName, "track", debug, auto)
			if err != nil {
				ui.Error.Printf("❌ Search failed for track %s: %v\n", trackName, err)
				if pool != nil {
					pool.Stop()
				}
				return
			}

			if len(selectedItems) == 0 {
				ui.Warning.Printf("⚠️ No results found for track: %s\n", trackName)
				continue
			}

			time.Sleep(500 * time.Millisecond)

			for i, selectedItem := range selectedItems {
				itemType := itemTypes[i]
				if itemType == "track" {
					track := selectedItem.(models.Track)
					ui.Info.Println("🎵 Starting track download for:", track.Title, "by", track.Artist)
					if err := d.DownloadSingleTrack(context.Background(), track, debug, conf.Format, conf.Bitrate, pool, conf, nil); err != nil {
						ui.Error.Printf("❌ Failed to download track %s: %v\n", track.Title, err)
					} else {
						ui.Success.Println("✅ Track download completed for", track.Title)
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
	Use:   "navidrome [playlist_url]",
	Short: "Copy a Spotify or ListenBrainz playlist to Navidrome.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		conf, a, d := initConfigAPIAndDownloader()
		playlistURL := args[0]

		// Define a unified track structure to handle both sources
		type UnifiedTrack struct {
			Name      string
			Artist    string
			AlbumName string
		}

		var tracks []UnifiedTrack
		var playlistName string

		// Determine source and fetch tracks
		if strings.Contains(playlistURL, "spotify.com") {
			spotifyClient := spotify.NewSpotifyClient(conf.SpotifyClientID, conf.SpotifyClientSecret)
			if err := spotifyClient.Authenticate(); err != nil {
				ui.Error.Printf("❌ Failed to authenticate with Spotify: %v\n", err)
				return
			}

			var spotifyTracks []spotify.SpotifyTrack
			var err error

			if strings.Contains(playlistURL, "/playlist/") {
				spotifyTracks, playlistName, err = spotifyClient.GetPlaylistTracks(playlistURL)
			} else if strings.Contains(playlistURL, "/album/") {
				spotifyTracks, playlistName, err = spotifyClient.GetAlbumTracks(playlistURL)
			} else {
				ui.Error.Println("❌ Invalid Spotify URL. Please provide a playlist or album URL.")
				return
			}

			if err != nil {
				ui.Error.Printf("❌ Failed to get tracks from Spotify: %v\n", err)
				return
			}
			
			// Convert to UnifiedTrack
			for _, t := range spotifyTracks {
				tracks = append(tracks, UnifiedTrack{
					Name:      t.Name,
					Artist:    t.Artist,
					AlbumName: t.AlbumName, // Might be empty for some Spotify tracks
				})
			}

		} else if strings.Contains(playlistURL, "listenbrainz.org") {
			lbClient := listenbrainz.NewListenBrainzClient()
			lbTracks, lbName, err := lbClient.GetPlaylistTracks(playlistURL)
			if err != nil {
				ui.Error.Printf("❌ Failed to get tracks from ListenBrainz: %v\n", err)
				return
			}
			playlistName = lbName
			
			// Convert to UnifiedTrack
			for _, t := range lbTracks {
				tracks = append(tracks, UnifiedTrack{
					Name:      t.Name,
					Artist:    t.Artist,
					AlbumName: t.AlbumName,
				})
			}
		} else {
			ui.Error.Println("❌ Unsupported URL. Please provide a valid Spotify or ListenBrainz playlist URL.")
			return
		}

		navidromeClient := navidrome.NewNavidromeClient(conf.NavidromeURL, conf.NavidromeUsername, conf.NavidromePassword)
		if err := navidromeClient.Authenticate(); err != nil {
			ui.Error.Printf("❌ Failed to authenticate with Navidrome: %v\n", err)
			return
		}

		if expandNavidrome {
			ui.Info.Println("Expanding playlist to download full albums...")
			uniqueAlbums := make(map[string]UnifiedTrack)
			for _, track := range tracks {
				if track.AlbumName == "" {
					continue
				}
				albumKey := strings.ToLower(track.AlbumName + " - " + track.Artist)
				if _, exists := uniqueAlbums[albumKey]; !exists {
					uniqueAlbums[albumKey] = track
				}
			}

			ui.Info.Printf("Found %d unique albums in the playlist.\n", len(uniqueAlbums))

			for _, track := range uniqueAlbums {
				albumSearchQuery := track.AlbumName + " - " + track.Artist

				ui.Info.Printf("Searching for album '%s' by '%s' in Navidrome...", track.AlbumName, track.Artist)
				navidromeAlbum, err := navidromeClient.SearchAlbum(track.AlbumName, track.Artist)
				if err != nil {
					ui.Warning.Printf("⚠️ Error searching for album %s in Navidrome: %v\n", albumSearchQuery, err)
				} else if navidromeAlbum != nil {
					ui.Success.Printf("✅ Album '%s' already exists in Navidrome, skipping download.\n", albumSearchQuery)
					continue
				}

				selectedItems, itemTypes, err := ui.HandleSearch(context.Background(), a, albumSearchQuery, "album", debug, auto)
				if err != nil {
					ui.Error.Printf("❌ Search failed for album '%s': %v\n", albumSearchQuery, err)
					continue
				}

				if len(selectedItems) == 0 {
					ui.Warning.Printf("⚠️ No results found for album: %s\n", albumSearchQuery)
					continue
				}

				time.Sleep(100 * time.Millisecond)

				for i, selectedItem := range selectedItems {
					if itemTypes[i] == "album" {
						album := selectedItem.(models.Album)
						ui.Info.Println("🎵 Starting album download for:", album.Title, "by", album.Artist)
						if _, err := d.DownloadAlbum(context.Background(), album.ID, conf, debug, nil, nil); err != nil {
							ui.Error.Printf("❌ Failed to download album %s: %v\n", album.Title, err)
						} else {
							ui.Success.Println("✅ Album download completed for", album.Title)
						}
						break
					}
				}
			}
		}
		
		finalPlaylistName := utils.GetUserInput("Enter a name for the new Navidrome playlist", playlistName)
		if err := navidromeClient.CreatePlaylist(finalPlaylistName); err != nil {
			ui.Error.Printf("❌ Failed to create Navidrome playlist: %v\n", err)
			return
		}

		playlistID, err := navidromeClient.SearchPlaylist(finalPlaylistName)
		if err != nil {
			ui.Error.Printf("❌ Failed to find newly created playlist '%s': %v\n", finalPlaylistName, err)
			return
		}

		var navidromeTrackIDs []string

		for _, track := range tracks {
			trackName := track.Name
			if ignoreSuffix != "" {
				trackName = utils.RemoveSuffix(trackName, ignoreSuffix)
			}
			foundTrack, err := navidromeClient.SearchTrack(trackName, track.Artist, track.AlbumName)
			if err != nil {
				ui.Warning.Printf("⚠️ Error searching for track %s by %s on Navidrome: %v\n", track.Name, track.Artist, err)
				continue
			}

			if foundTrack != nil {
				navidromeTrackIDs = append(navidromeTrackIDs, foundTrack.ID)
				ui.Success.Println("track is already found skipping")
			} else {
				ui.Warning.Printf("⚠️ Track %s by %s not found on Navidrome. Searching DAB...\n", track.Name, track.Artist)

				dabSearchQuery := track.Name + " - " + track.Artist
				if ignoreSuffix != "" {
					dabSearchQuery = trackName + " - " + track.Artist
				}
				dabSearchResults, dabItemTypes, err := ui.HandleSearch(context.Background(), a, dabSearchQuery, "track", debug, auto)
				if err != nil {
					ui.Error.Printf("❌ Failed to search DAB for %s: %v\n", track.Name, err)
					continue
				}

				if len(dabSearchResults) > 0 {
					time.Sleep(500 * time.Millisecond)
					selectedDabItem := dabSearchResults[0]
					selectedDabItemType := dabItemTypes[0]
					if selectedDabItemType == "track" {
						dabTrack := selectedDabItem.(models.Track)
						ui.Info.Printf("🎵 Downloading %s by %s from DAB...\n", dabTrack.Title, dabTrack.Artist)
						if err := d.DownloadSingleTrack(context.Background(), dabTrack, debug, conf.Format, conf.Bitrate, nil, conf, nil); err != nil {
							ui.Error.Printf("❌ Failed to download track %s from DAB: %v\n", dabTrack.Title, err)
						} else {
							ui.Success.Printf("✅ Downloaded %s by %s from DAB. It should appear in Navidrome soon.\n", dabTrack.Title, dabTrack.Artist)
							time.Sleep(5 * time.Second)
							reScannedTrack, err := navidromeClient.SearchTrack(dabTrack.Title, dabTrack.Artist, dabTrack.Album)
							if err != nil {
								ui.Warning.Printf("⚠️ Failed to re-search for downloaded track %s in Navidrome: %v\n", dabTrack.Title, err)
							} else if reScannedTrack != nil {
								navidromeTrackIDs = append(navidromeTrackIDs, reScannedTrack.ID)
								ui.Success.Printf("✅ Found newly downloaded track %s in Navidrome (ID: %s) and added to list for playlist.\n", reScannedTrack.Title, reScannedTrack.ID)
							} else {
								ui.Warning.Printf("⚠️ Downloaded track %s not found in Navidrome after re-scan. It might be added later manually.\n", dabTrack.Title)
							}
						}
					}
				}
			}
		}

		if len(navidromeTrackIDs) > 0 {
			if err := navidromeClient.AddTracksToPlaylist(playlistID, navidromeTrackIDs); err != nil {
				ui.Error.Printf("❌ Failed to add tracks to Navidrome playlist: %v\n", err)
			} else {
				ui.Success.Printf("✅ Successfully added %d tracks to Navidrome playlist '%s'\n", len(navidromeTrackIDs), finalPlaylistName)
			}
		}
	},
}

var addToPlaylistCmd = &cobra.Command{
	Use:   "add-to-playlist [playlist_id] [song_id...]",
	Short: "Add one or more songs to a Navidrome playlist.",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		conf, _, _ := initConfigAPIAndDownloader()

		navidromeClient := navidrome.NewNavidromeClient(conf.NavidromeURL, conf.NavidromeUsername, conf.NavidromePassword)
		if err := navidromeClient.Authenticate(); err != nil {
			ui.Error.Printf("❌ Failed to authenticate with Navidrome: %v\n", err)
			return
		}

		playlistID := args[0]
		songIDs := args[1:]

		if err := navidromeClient.AddTracksToPlaylist(playlistID, songIDs); err != nil {
			ui.Error.Printf("❌ Failed to add tracks to playlist %s: %v\n", playlistID, err)
		} else {
			ui.Success.Printf("✅ Successfully added %d tracks to playlist %s\n", len(songIDs), playlistID)
		}
	},
}

var listenbrainzCmd = &cobra.Command{
	Use:   "listenbrainz [url]",
	Short: "Download a ListenBrainz playlist.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		conf, a, d := initConfigAPIAndDownloader()
		if conf.Format != "flac" && !ffmpeg.CheckFFmpeg() {
			ui.Error.Println("❌ ffmpeg is not installed or not in your PATH. Please install ffmpeg to use the format conversion feature.")
			return
		}
		url := args[0]

		lbClient := listenbrainz.NewListenBrainzClient()
		lbTracks, lbName, err := lbClient.GetPlaylistTracks(url)
		if err != nil {
			ui.Error.Printf("❌ Failed to get tracks from ListenBrainz: %v\n", err)
			return
		}

		ui.Info.Printf("🎵 Found %d tracks in ListenBrainz playlist: %s\n", len(lbTracks), lbName)

		var pool *pb.Pool
		var localPool bool
		if utils.IsTTY() && len(lbTracks) > 1 {
			var err error
			pool, err = pb.StartPool()
			if err != nil {
				ui.Error.Printf("❌ Failed to start progress bar pool: %v\n", err)
			} else {
				localPool = true
			}
		}

		for _, lbTrack := range lbTracks {
			trackName := lbTrack.Name + " - " + lbTrack.Artist
			selectedItems, itemTypes, err := ui.HandleSearch(context.Background(), a, trackName, "track", debug, auto)
			if err != nil {
				ui.Error.Printf("❌ Search failed for track %s: %v\n", trackName, err)
				continue
			}

			if len(selectedItems) == 0 {
				ui.Warning.Printf("⚠️ No results found for track: %s\n", trackName)
				continue
			}

			time.Sleep(500 * time.Millisecond)

			for i, selectedItem := range selectedItems {
				itemType := itemTypes[i]
				if itemType == "track" {
					track := selectedItem.(models.Track)
					ui.Info.Println("🎵 Starting track download for:", track.Title, "by", track.Artist)
					if err := d.DownloadSingleTrack(context.Background(), track, debug, conf.Format, conf.Bitrate, pool, conf, nil); err != nil {
						ui.Error.Printf("❌ Failed to download track %s: %v\n", track.Title, err)
					} else {
						ui.Success.Println("✅ Track download completed for", track.Title)
					}
				}
			}
		}

		if localPool && pool != nil {
			pool.Stop()
		}
	},
}

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

func initConfigAPIAndDownloader() (*config.Config, *api.DabAPI, *downloader.Downloader) {
	color.NoColor = !utils.IsTTY()
	homeDir, err := os.UserHomeDir()
	if err != nil {
		ui.Warning.Println("⚠️ Could not determine home directory, will use current directory for downloads.")
		homeDir = "."
	}

	conf := &config.Config{
		APIURL:           "https://dabmusic.xyz",
		DownloadLocation: filepath.Join(homeDir, "Music"),
		Parallelism:      5,
		UpdateRepo:       "PrathxmOp/dab-downloader",
		VerifyDownloads:  true,
		MaxRetryAttempts: config.DefaultMaxRetries,
		WarningBehavior:  "summary",
	}

	configFile := config.GetConfigFilePath()

	if !utils.FileExists(configFile) {
		ui.Info.Println("✨ Welcome to DAB Downloader! Let's set up your configuration.")

		conf.APIURL = utils.GetUserInput(fmt.Sprintf("Enter DAB API URL (e.g., %s)", conf.APIURL), conf.APIURL)
		conf.DownloadLocation = utils.GetUserInput(fmt.Sprintf("Enter download location (e.g., %s)", conf.DownloadLocation), conf.DownloadLocation)

		defaultParallelism := strconv.Itoa(conf.Parallelism)
		parallelismStr := utils.GetUserInput(fmt.Sprintf("Enter number of parallel downloads (default: %s)", defaultParallelism), defaultParallelism)
		if p, err := strconv.Atoi(parallelismStr); err == nil && p > 0 {
			conf.Parallelism = p
		}

		conf.SpotifyClientID = utils.GetUserInput("Enter your Spotify Client ID", "")
		conf.SpotifyClientSecret = utils.GetUserInput("Enter your Spotify Client Secret", "")
		conf.NavidromeURL = utils.GetUserInput("Enter your Navidrome URL", "")
		conf.NavidromeUsername = utils.GetUserInput("Enter your Navidrome Username", "")
		conf.NavidromePassword = utils.GetUserInput("Enter your Navidrome Password", "")
		conf.Format = utils.GetUserInput("Enter default output format (e.g., flac, mp3, ogg, opus)", "flac")
		conf.Bitrate = utils.GetUserInput("Enter default bitrate for lossy formats (e.g., 320)", "320")
		conf.UpdateRepo = utils.GetUserInput("Enter GitHub repository for updates", "PrathxmOp/dab-downloader")

		if err := config.SaveConfig(configFile, conf); err != nil {
			ui.Error.Printf("❌ Failed to save initial config: %v\n", err)
		} else {
			ui.Success.Println("✅ Configuration saved to", configFile)
		}
	} else {
		if err := config.LoadConfig(configFile, conf); err != nil {
			ui.Error.Printf("❌ Failed to load config from %s: %v\n", configFile, err)
		} else {
			ui.Info.Println("✅ Loaded configuration from", configFile)
			if conf.Format == "" {
				conf.Format = "flac"
			}
			if conf.Bitrate == "" {
				conf.Bitrate = "320"
			}
		}
	}

	if apiURL != "" {
		conf.APIURL = apiURL
	}
	if downloadLocation != "" {
		conf.DownloadLocation = downloadLocation
	}
	if spotifyClientID != "" {
		conf.SpotifyClientID = spotifyClientID
	}
	if spotifyClientSecret != "" {
		conf.SpotifyClientSecret = spotifyClientSecret
	}
	if navidromeURL != "" {
		conf.NavidromeURL = navidromeURL
	}
	if navidromeUsername != "" {
		conf.NavidromeUsername = navidromeUsername
	}
	if navidromePassword != "" {
		conf.NavidromePassword = navidromePassword
	}
	if format != "flac" {
		conf.Format = format
	}
	if bitrate != "320" {
		conf.Bitrate = bitrate
	}
	if warningBehavior != "summary" {
		conf.WarningBehavior = warningBehavior
	}

	httpClient := &http.Client{
		Timeout: 10 * time.Minute,
	}

	if insecure {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	a := api.NewDabAPI(conf.APIURL, conf.DownloadLocation, httpClient)
	d := downloader.NewDownloader(a, conf.DownloadLocation)
	return conf, a, d
}

func init() {
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "", "DAB API URL")
	rootCmd.PersistentFlags().StringVar(&downloadLocation, "download-location", "", "Directory to save downloads")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&insecure, "insecure", false, "Skip TLS certificate verification")
	rootCmd.PersistentFlags().StringVar(&warningBehavior, "warnings", "summary", "Warning behavior")

	albumCmd.Flags().StringVar(&format, "format", "flac", "Format to convert to")
	albumCmd.Flags().StringVar(&bitrate, "bitrate", "320", "Bitrate for lossy formats")

	artistCmd.Flags().StringVar(&filter, "filter", "all", "Filter by item type")
	artistCmd.Flags().BoolVar(&noConfirm, "no-confirm", false, "Skip confirmation prompt")
	artistCmd.Flags().StringVar(&format, "format", "flac", "Format to convert to")
	artistCmd.Flags().StringVar(&bitrate, "bitrate", "320", "Bitrate for lossy formats")

	searchCmd.Flags().StringVar(&searchType, "type", "all", "Type of content to search for")
	searchCmd.Flags().BoolVar(&auto, "auto", false, "Automatically download first result")
	searchCmd.Flags().StringVar(&format, "format", "flac", "Format to convert to")
	searchCmd.Flags().StringVar(&bitrate, "bitrate", "320", "Bitrate for lossy formats")

	spotifyCmd.Flags().StringVar(&spotifyPlaylist, "spotify", "", "Spotify playlist URL")
	spotifyCmd.Flags().BoolVar(&auto, "auto", false, "Automatically download first result")
	spotifyCmd.Flags().BoolVar(&expandPlaylist, "expand", false, "Expand playlist to full albums")
	spotifyCmd.Flags().StringVar(&format, "format", "flac", "Format to convert to")
	spotifyCmd.Flags().StringVar(&bitrate, "bitrate", "320", "Bitrate for lossy formats")

	rootCmd.PersistentFlags().StringVar(&spotifyClientID, "spotify-client-id", "", "Spotify Client ID")
	rootCmd.PersistentFlags().StringVar(&spotifyClientSecret, "spotify-client-secret", "", "Spotify Client Secret")

	rootCmd.PersistentFlags().StringVar(&navidromeURL, "navidrome-url", "", "Navidrome URL")
	rootCmd.PersistentFlags().StringVar(&navidromeUsername, "navidrome-username", "", "Navidrome Username")
	rootCmd.PersistentFlags().StringVar(&navidromePassword, "navidrome-password", "", "Navidrome Password")
	navidromeCmd.Flags().StringVar(&ignoreSuffix, "ignore-suffix", "", "Ignore suffix in search")
	navidromeCmd.Flags().BoolVar(&expandNavidrome, "expand", false, "Expand to full albums")
	navidromeCmd.Flags().BoolVar(&auto, "auto", false, "Automatically download first result")

	listenbrainzCmd.Flags().BoolVar(&auto, "auto", false, "Automatically download first result")
	listenbrainzCmd.Flags().StringVar(&format, "format", "flac", "Format to convert to")
	listenbrainzCmd.Flags().StringVar(&bitrate, "bitrate", "320", "Bitrate for lossy formats")

	rootCmd.AddCommand(artistCmd)
	rootCmd.AddCommand(albumCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(spotifyCmd)
	rootCmd.AddCommand(navidromeCmd)
	rootCmd.AddCommand(listenbrainzCmd)
	rootCmd.AddCommand(addToPlaylistCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(debugCmd)

	debugCmd.AddCommand(testApiAvailabilityCmd)
	debugCmd.AddCommand(testArtistEndpointsCmd)
	debugCmd.AddCommand(comprehensiveArtistDebugCmd)

	rootCmd.AddCommand(versionCmd)
}

func main() {
	var vInfo version.VersionInfo
	if err := json.Unmarshal(versionJSON, &vInfo); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading embedded version.json: %v\n", err)
		os.Exit(1)
	}

toolVersion = vInfo.Version
	rootCmd.Version = toolVersion

	conf, _, _ := initConfigAPIAndDownloader()

	if _, err := os.Stat("/.dockerenv"); err == nil {
		conf.IsDockerContainer = true
	}

	updater.CheckForUpdates(conf, toolVersion)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}