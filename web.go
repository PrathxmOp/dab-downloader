package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

type WebRequest struct {
	URL      string  `json:"url,omitempty"`
	ArtistID string  `json:"artistId,omitempty"`
	AlbumID  string  `json:"albumId,omitempty"`
	Query    string  `json:"query,omitempty"`
	Tracks   []Track `json:"tracks,omitempty"`
	Config   Config  `json:"config,omitempty"`
}

type WebResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

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

	fs := http.FileServer(http.Dir("./web"))
	mux.Handle("/", fs)

	mux.HandleFunc("/api/search", searchHandler)
	mux.HandleFunc("/api/download-tracks", downloadTracksHandler)
	mux.HandleFunc("/api/download-artist", artistHandler)
	mux.HandleFunc("/api/download-album", albumHandler)
	mux.HandleFunc("/api/download-spotify", spotifyHandler)
	mux.HandleFunc("/api/copy-navidrome", navidromeHandler)
	mux.HandleFunc("/api/settings", settingsHandler)

	colorInfo.Println("🚀 Starting web server on http://localhost:6797")
	if err := http.ListenAndServe(":6797", mux); err != nil {
		colorError.Printf("❌ Failed to start web server: %v\n", err)
	}
}

func settingsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		config := &Config{}
		if err := LoadConfig("config/config.json", config); err != nil {
			// If the file doesn't exist, we can just return an empty config
			if !os.IsNotExist(err) {
				writeJSONResponse(w, false, fmt.Sprintf("Failed to load config: %v", err))
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(config); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	case http.MethodPost:
		var config Config
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := SaveConfig("config/config.json", &config); err != nil {
			writeJSONResponse(w, false, fmt.Sprintf("Failed to save config: %v", err))
			return
		}

		writeJSONResponse(w, true, "")
	default:
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req WebRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	api := NewDabAPI(req.Config.APIURL, req.Config.DownloadLocation)

	if strings.Contains(req.Query, "spotify.com") {
		spotifyClient := NewSpotifyClient(req.Config.SpotifyClientID, req.Config.SpotifyClientSecret)
		if err := spotifyClient.Authenticate(); err != nil {
			writeSpotifySearchResponse(w, nil, "Failed to authenticate with Spotify")
			return
		}

		var spotifyTracks []SpotifyTrack
		var err error

		if strings.Contains(req.Query, "/playlist/") {
			spotifyTracks, _, err = spotifyClient.GetPlaylistTracks(req.Query)
		} else if strings.Contains(req.Query, "/album/") {
			spotifyTracks, _, err = spotifyClient.GetAlbumTracks(req.Query)
		} else if strings.Contains(req.Query, "/track/") {
			var track *SpotifyTrack
			track, err = spotifyClient.GetTrack(req.Query)
			if track != nil {
				spotifyTracks = append(spotifyTracks, *track)
			}
		} else {
			writeSpotifySearchResponse(w, nil, "Invalid Spotify URL.")
			return
		}

		if err != nil {
			writeSpotifySearchResponse(w, nil, fmt.Sprintf("Failed to get tracks from Spotify: %v", err))
			return
		}

		var results []SpotifyTrackWithDABResults
		for _, spotifyTrack := range spotifyTracks {
			query := spotifyTrack.Name + " - " + spotifyTrack.Artist
			dabTracks, err := searchAndGetTracks(context.Background(), api, query, "track", req.Config.Debug, true)
			if err != nil {
				colorError.Printf("❌ Search failed for track %s: %v\n", query, err)
			}

			results = append(results, SpotifyTrackWithDABResults{
				SpotifyTrack: spotifyTrack,
				DABResults:   dabTracks,
			})
		}

		writeSpotifySearchResponse(w, results, "")
	} else {
		dabTracks, err := searchAndGetTracks(context.Background(), api, req.Query, "track", req.Config.Debug, false)
		if err != nil {
			writeSpotifySearchResponse(w, nil, fmt.Sprintf("Search failed: %v", err))
			return
		}
		var results []SpotifyTrackWithDABResults
		results = append(results, SpotifyTrackWithDABResults{
			DABResults: dabTracks,
		})
		writeSpotifySearchResponse(w, results, "")
	}
}

func downloadTracksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req WebRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	api := NewDabAPI(req.Config.APIURL, req.Config.DownloadLocation)

	for _, track := range req.Tracks {
		if err := api.DownloadSingleTrack(context.Background(), track, req.Config.Debug, req.Config.Format, req.Config.Bitrate); err != nil {
			colorError.Printf("❌ Failed to download track %s: %v\n", track.Title, err)
		} else {
			colorSuccess.Println("✅ Track download completed for", track.Title)
		}
	}

	writeJSONResponse(w, true, "")
}

func artistHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req WebRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	api := NewDabAPI(req.Config.APIURL, req.Config.DownloadLocation)

	if err := api.DownloadArtistDiscography(context.Background(), req.ArtistID, req.Config.Debug, "all", true, req.Config.Format, req.Config.Bitrate); err != nil {
		writeJSONResponse(w, false, fmt.Sprintf("Failed to download artist discography: %v", err))
		return
	}

	writeJSONResponse(w, true, "")
}

func albumHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req WebRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	api := NewDabAPI(req.Config.APIURL, req.Config.DownloadLocation)

	if _, err := api.DownloadAlbum(context.Background(), req.AlbumID, req.Config.Parallelism, req.Config.Debug, nil, req.Config.Format, req.Config.Bitrate); err != nil {
		writeJSONResponse(w, false, fmt.Sprintf("Failed to download album: %v", err))
		return
	}

	writeJSONResponse(w, true, "")
}

func spotifyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req WebRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	api := NewDabAPI(req.Config.APIURL, req.Config.DownloadLocation)
	spotifyClient := NewSpotifyClient(req.Config.SpotifyClientID, req.Config.SpotifyClientSecret)
	if err := spotifyClient.Authenticate(); err != nil {
		writeJSONResponse(w, false, "Failed to authenticate with Spotify")
		return
	}

	var spotifyTracks []SpotifyTrack
	var err error

	if strings.Contains(req.URL, "/playlist/") {
		spotifyTracks, _, err = spotifyClient.GetPlaylistTracks(req.URL)
	} else if strings.Contains(req.URL, "/album/") {
		spotifyTracks, _, err = spotifyClient.GetAlbumTracks(req.URL)
	} else {
		writeJSONResponse(w, false, "Invalid Spotify URL.")
		return
	}

	if err != nil {
		writeJSONResponse(w, false, fmt.Sprintf("Failed to get tracks from Spotify: %v", err))
		return
	}

	for _, spotifyTrack := range spotifyTracks {
		query := spotifyTrack.Name + " - " + spotifyTrack.Artist
		selectedItems, _, err := handleSearch(context.Background(), api, query, "track", req.Config.Debug, true)
		if err != nil {
			colorError.Printf("❌ Search failed for track %s: %v\n", query, err)
			continue
		}

		if len(selectedItems) > 0 {
			track := selectedItems[0].(Track)
			if err := api.DownloadSingleTrack(context.Background(), track, req.Config.Debug, req.Config.Format, req.Config.Bitrate); err != nil {
				colorError.Printf("❌ Failed to download track %s: %v\n", track.Title, err)
			}
		}
	}

	writeJSONResponse(w, true, "")
}

func navidromeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req WebRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	api := NewDabAPI(req.Config.APIURL, req.Config.DownloadLocation)
	spotifyClient := NewSpotifyClient(req.Config.SpotifyClientID, req.Config.SpotifyClientSecret)
	var err error
	if err = spotifyClient.Authenticate(); err != nil {
		writeJSONResponse(w, false, "Failed to authenticate with Spotify")
		return
	}

	navidromeClient := NewNavidromeClient(req.Config.NavidromeURL, req.Config.NavidromeUsername, req.Config.NavidromePassword)
	if err = navidromeClient.Authenticate(); err != nil {
		writeJSONResponse(w, false, "Failed to authenticate with Navidrome")
		return
	}

	var spotifyTracks []SpotifyTrack
	var spotifyName string

	if strings.Contains(req.URL, "/playlist/") {
		spotifyTracks, spotifyName, err = spotifyClient.GetPlaylistTracks(req.URL)
	} else if strings.Contains(req.URL, "/album/") {
		spotifyTracks, spotifyName, err = spotifyClient.GetAlbumTracks(req.URL)
	} else {
		writeJSONResponse(w, false, "Invalid Spotify URL.")
		return
	}

	if err != nil {
		writeJSONResponse(w, false, fmt.Sprintf("Failed to get tracks from Spotify: %v", err))
		return
	}

	if err = navidromeClient.CreatePlaylist(spotifyName); err != nil {
		writeJSONResponse(w, false, fmt.Sprintf("Failed to create Navidrome playlist: %v", err))
		return
	}

	playlistID, err := navidromeClient.SearchPlaylist(spotifyName)
	if err != nil {
		writeJSONResponse(w, false, fmt.Sprintf("Failed to find playlist: %v", err))
		return
	}

	var navidromeTrackIDs []string
	for _, spotifyTrack := range spotifyTracks {
		track, err := navidromeClient.SearchTrack(spotifyTrack.Name, spotifyTrack.Artist)
		if err != nil {
			colorWarning.Printf("⚠️ Error searching for track %s by %s on Navidrome: %v\n", spotifyTrack.Name, spotifyTrack.Artist, err)
			continue
		}

		if track != nil {
			navidromeTrackIDs = append(navidromeTrackIDs, track.ID)
		} else {
			selectedItems, _, err := handleSearch(context.Background(), api, spotifyTrack.Name+" - "+spotifyTrack.Artist, "track", req.Config.Debug, true)
			if err != nil {
				colorError.Printf("❌ Search failed for track %s: %v\n", spotifyTrack.Name, err)
				continue
			}

			if len(selectedItems) > 0 {
				dabTrack := selectedItems[0].(Track)
				if err := api.DownloadSingleTrack(context.Background(), dabTrack, req.Config.Debug, req.Config.Format, req.Config.Bitrate); err != nil {
					colorError.Printf("❌ Failed to download track %s: %v\n", dabTrack.Title, err)
				} else {
					// This is a bit of a hack, but we need to wait for Navidrome to scan the new file
					// time.Sleep(5 * time.Second)
					newTrack, err := navidromeClient.SearchTrack(dabTrack.Title, dabTrack.Artist)
					if err == nil && newTrack != nil {
						navidromeTrackIDs = append(navidromeTrackIDs, newTrack.ID)
					}
				}
			}
		}
	}

	if err = navidromeClient.AddTracksToPlaylist(playlistID, navidromeTrackIDs); err != nil {
		writeJSONResponse(w, false, fmt.Sprintf("Failed to add tracks to playlist: %v", err))
		return
	}

	writeJSONResponse(w, true, "")
}

func writeJSONResponse(w http.ResponseWriter, success bool, errorMsg string) {
	w.Header().Set("Content-Type", "application/json")
	response := WebResponse{
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