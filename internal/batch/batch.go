package batch

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"dab-downloader/internal/api"
	"dab-downloader/internal/config"
	"dab-downloader/internal/downloader"
	"dab-downloader/internal/models"
	"dab-downloader/internal/ui"
	"dab-downloader/internal/utils"
	"dab-downloader/pkg/spotify"
)

// ProcessBatchFile reads a file and processes each line as a download task
func ProcessBatchFile(ctx context.Context, filePath string, api *api.DabAPI, d *downloader.Downloader, conf *config.Config, debug bool) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open batch file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "//") {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading batch file: %w", err)
	}

	ui.Info.Printf("📋 Found %d items in batch file\n", len(lines))

	for i, line := range lines {
		ui.Info.Printf("\n🔄 Processing item %d/%d: %s\n", i+1, len(lines), line)
		err := processLine(ctx, line, api, d, conf, debug)
		if err != nil {
			ui.Error.Printf("❌ Failed to process '%s': %v\n", line, err)
		} else {
			ui.Success.Printf("✅ Successfully processed '%s'\n", line)
		}
		// Small delay between items to be nice to the API
		time.Sleep(1 * time.Second) 
	}

	return nil
}

func processLine(ctx context.Context, line string, api *api.DabAPI, d *downloader.Downloader, conf *config.Config, debug bool) error {
	// 1. Spotify URL
	if strings.Contains(line, "spotify.com") {
		return processSpotify(ctx, line, api, d, conf, debug)
	}

	// 2. DAB Album URL
	if strings.Contains(line, "/shared/album/") {
		parts := strings.Split(line, "/")
		albumID := parts[len(parts)-1]
		if strings.Contains(albumID, "?") {
			albumID = strings.Split(albumID, "?")[0]
		}
		ui.Info.Println("🎵 Detected DAB Album URL. Downloading album:", albumID)
		_, err := d.DownloadAlbum(ctx, albumID, conf, debug, nil, nil)
		return err
	}

	// 3. DAB Playlist URL
	if strings.Contains(line, "/shared/library/") {
		parts := strings.Split(line, "/")
		playlistID := parts[len(parts)-1]
		if strings.Contains(playlistID, "?") {
			playlistID = strings.Split(playlistID, "?")[0]
		}
		ui.Info.Println("🎵 Detected DAB Playlist URL. Downloading playlist:", playlistID)
		_, err := d.DownloadPlaylist(ctx, playlistID, conf, debug, nil, nil)
		return err
	}

	// 4. Fallback: Search (assume it's a query or ISRC)
	ui.Info.Printf("🔎 Treating '%s' as search query...\n", line)
	// Auto-select first result
	return processSearch(ctx, line, api, d, conf, debug)
}

func processSpotify(ctx context.Context, url string, api *api.DabAPI, d *downloader.Downloader, conf *config.Config, debug bool) error {
	if conf.SpotifyClientID == "" || conf.SpotifyClientSecret == "" {
		return fmt.Errorf("spotify credentials not configured")
	}

	spotifyClient := spotify.NewSpotifyClient(conf.SpotifyClientID, conf.SpotifyClientSecret)
	if err := spotifyClient.Authenticate(); err != nil {
		return fmt.Errorf("failed to authenticate with Spotify: %w", err)
	}

	var spotifyTracks []spotify.SpotifyTrack
	var err error

	if strings.Contains(url, "/playlist/") {
		spotifyTracks, _, err = spotifyClient.GetPlaylistTracks(url)
	} else if strings.Contains(url, "/album/") {
		spotifyTracks, _, err = spotifyClient.GetAlbumTracks(url)
	} else {
		return fmt.Errorf("invalid Spotify URL")
	}

	if err != nil {
		return fmt.Errorf("failed to get tracks from Spotify: %w", err)
	}

	ui.Info.Printf("🎵 Found %d tracks on Spotify. Starting processing...\n", len(spotifyTracks))

	for _, spotifyTrack := range spotifyTracks {
		trackName := spotifyTrack.Name + " - " + spotifyTrack.Artist
		// Reuse search processing logic
		if err := processSearch(ctx, trackName, api, d, conf, debug); err != nil {
			ui.Error.Printf("❌ Failed to download '%s': %v\n", trackName, err)
		}
	}
	return nil
}

func processSearch(ctx context.Context, query string, api *api.DabAPI, d *downloader.Downloader, conf *config.Config, debug bool) error {
	// Search for track first, then album, then artist?
	// Usually users put song names.
	// Let's use "all" type and prefer track -> album -> artist order or just pick first result like HandleSearch(auto=true)
	
	// We can't reuse ui.HandleSearch easily because it has UI logic (bubbletea) which might interfere with batch mode 
	// or we can reuse it with auto=true.
	// ui.HandleSearch(..., auto=true) returns the first item.
	
	selectedItems, itemTypes, err := ui.HandleSearch(ctx, api, query, "all", debug, true) // auto=true
	if err != nil {
		return err
	}
	
	if len(selectedItems) == 0 {
		return fmt.Errorf("no results found for '%s'", query)
	}

	// Handle the first item found
	item := selectedItems[0]
	itemType := itemTypes[0]

	switch itemType {
	case "track":
		track := item.(models.Track)
		ui.Info.Println("🎵 Downloading track:", track.Title)
		return d.DownloadSingleTrack(ctx, track, debug, conf.Format, conf.Bitrate, nil, conf, nil)
	case "album":
		album := item.(models.Album)
		ui.Info.Println("🎵 Downloading album:", album.Title)
		_, err := d.DownloadAlbum(ctx, album.ID, conf, debug, nil, nil)
		return err
	case "artist":
		artist := item.(models.Artist)
		ui.Info.Println("🎵 Downloading artist discography:", artist.Name)
		// For artist, batch mode shouldn't ask for confirmation or filter by default, maybe download all?
		// DownloadArtistDiscography usually asks for filter.
		// Passing filter="all" and noConfirm=true
		return d.DownloadArtistDiscography(ctx, utils.IDToString(artist.ID), conf, debug, "all", true, nil)
	case "playlist":
		playlist := item.(models.Playlist)
		ui.Info.Println("🎵 Downloading playlist:", playlist.Title)
		_, err := d.DownloadPlaylist(ctx, playlist.ID, conf, debug, nil, nil)
		return err
	default:
		return fmt.Errorf("unknown item type: %s", itemType)
	}
}
