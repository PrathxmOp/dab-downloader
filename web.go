package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Run the web UI for dab-downloader",
	Run: func(cmd *cobra.Command, args []string) {
		startWebServer()
	},
}

func init() {
	rootCmd.AddCommand(webCmd)
}

type DownloadRequest struct {
	URL string `json:"url"`
}

type DownloadTracksRequest struct {
	Tracks []Track `json:"tracks"`
}

type DownloadResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// Structures for the new search response
type SpotifyTrackWithDABResults struct {
	SpotifyTrack SpotifyTrack `json:"spotify_track"`
	DABResults   []Track      `json:"dab_results"`
}

type SpotifySearchResponse struct {
	Tracks []SpotifyTrackWithDABResults `json:"tracks"`
	Error  string                       `json:"error,omitempty"`
}

func startWebServer() {
	mux := http.NewServeMux()

	// Serve static files
	fs := http.FileServer(http.Dir("./web"))
	mux.Handle("/", fs)

	// API endpoints
	mux.HandleFunc("/api/download", downloadHandler) // for downloading all tracks from a url
	mux.HandleFunc("/api/search", searchHandler)
	mux.HandleFunc("/api/download-tracks", downloadTracksHandler)

	colorInfo.Println("🚀 Starting web server on http://localhost:6797")
	if err := http.ListenAndServe(":6797", mux); err != nil {
		colorError.Printf("❌ Failed to start web server: %v\n", err)
	}
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	config, api := initConfigAndAPI()

	if strings.Contains(req.URL, "spotify.com") {
		spotifyClient := NewSpotifyClient(config.SpotifyClientID, config.SpotifyClientSecret)
		if err := spotifyClient.Authenticate(); err != nil {
			colorError.Printf("❌ Failed to authenticate with Spotify: %v\n", err)
			writeSpotifySearchResponse(w, nil, "Failed to authenticate with Spotify")
			return
		}

		var spotifyTracks []SpotifyTrack
		var err error

		if strings.Contains(req.URL, "/playlist/") {
			spotifyTracks, _, err = spotifyClient.GetPlaylistTracks(req.URL)
		} else if strings.Contains(req.URL, "/album/") {
			spotifyTracks, _, err = spotifyClient.GetAlbumTracks(req.URL)
		} else if strings.Contains(req.URL, "/track/") {
			var track *SpotifyTrack
			track, err = spotifyClient.GetTrack(req.URL)
			if track != nil {
				spotifyTracks = append(spotifyTracks, *track)
			}
		} else {
			writeSpotifySearchResponse(w, nil, "Invalid Spotify URL. Please provide a playlist, album, or track URL.")
			return
		}

		if err != nil {
			writeSpotifySearchResponse(w, nil, fmt.Sprintf("Failed to get tracks from Spotify: %v", err))
			return
		}

		var results []SpotifyTrackWithDABResults
		for _, spotifyTrack := range spotifyTracks {
			query := spotifyTrack.Name + " - " + spotifyTrack.Artist
			dabTracks, err := searchAndGetTracks(context.Background(), api, query, "track", debug, true)
			if err != nil {
				colorError.Printf("❌ Search failed for track %s: %v\n", query, err)
				// even if search fails, we still want to show the spotify track in the UI
			}

			results = append(results, SpotifyTrackWithDABResults{
				SpotifyTrack: spotifyTrack,
				DABResults:   dabTracks,
			})
		}

		writeSpotifySearchResponse(w, results, "")
	} else {
		writeSpotifySearchResponse(w, nil, "Only Spotify URLs are currently supported.")
	}
}

func downloadTracksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req DownloadTracksRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	config, api := initConfigAndAPI()

	for _, track := range req.Tracks {
		colorInfo.Println("🎵 Starting track download for:", track.Title, "by", track.Artist)
		if err := api.DownloadSingleTrack(context.Background(), track, debug, config.Format, config.Bitrate); err != nil {
			colorError.Printf("❌ Failed to download track %s: %v\n", track.Title, err)
		} else {
			colorSuccess.Println("✅ Track download completed for", track.Title)
		}
	}

	writeJSONResponse(w, true, "")
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	config, api := initConfigAndAPI()

	// For now, we'll just handle spotify URLs
	if strings.Contains(req.URL, "spotify.com") {
		spotifyClient := NewSpotifyClient(config.SpotifyClientID, config.SpotifyClientSecret)
		if err := spotifyClient.Authenticate(); err != nil {
			colorError.Printf("❌ Failed to authenticate with Spotify: %v\n", err)
			writeJSONResponse(w, false, "Failed to authenticate with Spotify")
			return
		}

		var spotifyTracks []SpotifyTrack
		var err error

		if strings.Contains(req.URL, "/playlist/") {
			spotifyTracks, _, err = spotifyClient.GetPlaylistTracks(req.URL)
		} else if strings.Contains(req.URL, "/album/") {
			spotifyTracks, _, err = spotifyClient.GetAlbumTracks(req.URL)
		} else if strings.Contains(req.URL, "/track/") {
			var track *SpotifyTrack
			track, err = spotifyClient.GetTrack(req.URL)
			if track != nil {
				spotifyTracks = append(spotifyTracks, *track)
			}
		} else {
			writeJSONResponse(w, false, "Invalid Spotify URL. Please provide a playlist, album, or track URL.")
			return
		}

		if err != nil {
			writeJSONResponse(w, false, fmt.Sprintf("Failed to get tracks from Spotify: %v", err))
			return
		}

		for _, spotifyTrack := range spotifyTracks {
			query := spotifyTrack.Name + " - " + spotifyTrack.Artist
			selectedItems, itemTypes, err := handleSearch(context.Background(), api, query, "track", debug, true) // auto=true to select first result
			if err != nil {
				colorError.Printf("❌ Search failed for track %s: %v\n", query, err)
				continue
			}

			if len(selectedItems) == 0 {
				colorWarning.Printf("⚠️ No results found for track: %s\n", query)
				continue
			}

			for i, selectedItem := range selectedItems {
				itemType := itemTypes[i]
				if itemType == "track" {
					track := selectedItem.(Track)
					colorInfo.Println("🎵 Starting track download for:", track.Title, "by", track.Artist)
					if err := api.DownloadSingleTrack(context.Background(), track, debug, config.Format, config.Bitrate); err != nil {
						colorError.Printf("❌ Failed to download track %s: %v\n", track.Title, err)
					} else {
						colorSuccess.Println("✅ Track download completed for", track.Title)
					}
				}
			}
		}

		writeJSONResponse(w, true, "")
	} else {
		writeJSONResponse(w, false, "Only Spotify URLs are currently supported.")
	}
}

func writeJSONResponse(w http.ResponseWriter, success bool, errorMsg string) {
	w.Header().Set("Content-Type", "application/json")
	response := DownloadResponse{
		Success: success,
		Error:   errorMsg,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func writeSpotifySearchResponse(w http.ResponseWriter, tracks []SpotifyTrackWithDABResults, errorMsg string) {
	w.Header().Set("Content-Type", "application/json")
	response := SpotifySearchResponse{
		Tracks: tracks,
		Error:  errorMsg,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
