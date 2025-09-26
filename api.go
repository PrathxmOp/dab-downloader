package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time" // Add time import

	"golang.org/x/sync/semaphore"
)

const requestInterval = 500 * time.Millisecond // Define rate limit interval

// NewDabAPI creates a new API client
func NewDabAPI(endpoint, outputLocation string, client *http.Client) *DabAPI {
	return &DabAPI{
		endpoint:       strings.TrimSuffix(endpoint, "/"),
		outputLocation: outputLocation,
		client:         client,
		rateLimiter:    time.NewTicker(requestInterval), // Initialize rate limiter
	}
}

type DabAPI struct {
	endpoint       string
	outputLocation string
	client         *http.Client
	mu             sync.Mutex // Mutex to protect rate limiter
	rateLimiter    *time.Ticker // Rate limiter for API requests
}

// Request makes HTTP requests to the API
func (api *DabAPI) Request(ctx context.Context, path string, isPathOnly bool, params []QueryParam) (*http.Response, error) {
	api.mu.Lock()
	<-api.rateLimiter.C // Wait for the rate limiter
	api.mu.Unlock()

	var fullURL string

	if isPathOnly {
		fullURL = fmt.Sprintf("%s/%s", api.endpoint, strings.TrimPrefix(path, "/"))
	} else {
		fullURL = path
	}

	u, err := url.Parse(fullURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %w", err)
	}

	if len(params) > 0 {
		q := u.Query()
		for _, param := range params {
			q.Add(param.Name, param.Value)
		}
		u.RawQuery = q.Encode()
	}

	var resp *http.Response
	err = RetryWithBackoff(defaultMaxRetries, 1, func() error {
		req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
		if err != nil {
			return fmt.Errorf("error creating request: %w", err)
		}
		req.Header.Set("User-Agent", userAgent)

		resp, err = api.client.Do(req)
		if err != nil {
			return fmt.Errorf("error executing request: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			return fmt.Errorf("rate limit exceeded (429), retrying") // Return error to trigger retry
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return fmt.Errorf("request failed with status: %s", resp.Status)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetAlbum retrieves album information
func (api *DabAPI) GetAlbum(ctx context.Context, albumID string) (*Album, error) {
	resp, err := api.Request(ctx, "api/album", true, []QueryParam{
		{Name: "albumId", Value: albumID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get album: %w", err)
	}
	defer resp.Body.Close()

	var albumResp AlbumResponse
	if err := json.NewDecoder(resp.Body).Decode(&albumResp); err != nil {
		return nil, fmt.Errorf("failed to decode album response: %w", err)
	}

	// Process tracks to add missing metadata
	for i := range albumResp.Album.Tracks {
		track := &albumResp.Album.Tracks[i]

		// Set album information if missing
		if track.Album == "" {
			track.Album = albumResp.Album.Title
		}
		if track.AlbumArtist == "" {
			track.AlbumArtist = albumResp.Album.Artist
		}
		if track.Genre == "" {
			track.Genre = albumResp.Album.Genre
		}
		if track.ReleaseDate == "" {
			track.ReleaseDate = albumResp.Album.ReleaseDate
		}
		if track.Year == "" && len(albumResp.Album.ReleaseDate) >= 4 {
			track.Year = albumResp.Album.ReleaseDate[:4]
		}

		// Set track number if not provided
		if track.TrackNumber == 0 {
			track.TrackNumber = i + 1
		}
		if track.DiscNumber == 0 {
			track.DiscNumber = 1
		}
	}

	// Set album totals if not provided
	if albumResp.Album.TotalTracks == 0 {
		albumResp.Album.TotalTracks = len(albumResp.Album.Tracks)
	}
	if albumResp.Album.TotalDiscs == 0 {
		albumResp.Album.TotalDiscs = 1
	}
	if albumResp.Album.Year == "" && len(albumResp.Album.ReleaseDate) >= 4 {
		albumResp.Album.Year = albumResp.Album.ReleaseDate[:4]
	}

	// Prepend API endpoint to cover URL if it's a relative path
	if strings.HasPrefix(albumResp.Album.Cover, "/") {
		albumResp.Album.Cover = api.endpoint + albumResp.Album.Cover
	}

	return &albumResp.Album, nil
}

// GetArtist retrieves artist information and discography
func (api *DabAPI) GetArtist(ctx context.Context, artistID string, config *Config, debug bool) (*Artist, error) {
	if debug {
		fmt.Printf("DEBUG - GetArtist called with artistID: '%s'\n", artistID)
	}

	resp, err := api.Request(ctx, "api/discography", true, []QueryParam{
		{Name: "artistId", Value: artistID},
	})
	if err != nil {
		if debug {
			fmt.Printf("DEBUG - GetArtist API request failed: %v\n", err)
		}
		return nil, fmt.Errorf("failed to get artist: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if debug {
			fmt.Printf("DEBUG - GetArtist failed to read response body: %v\n", err)
		}
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if debug {
		// Debug: Print the raw JSON response
		fmt.Printf("DEBUG - Raw artist response body length: %d bytes\n", len(body))
		fmt.Printf("DEBUG - Raw artist response: %s\n", string(body))
	}

	// The discography endpoint returns a different structure
	var discographyResp struct {
		Artist Artist  `json:"artist"`
		Albums []Album `json:"albums"`
	}

	if err := json.Unmarshal(body, &discographyResp); err != nil {
		if debug {
			fmt.Printf("DEBUG - GetArtist JSON unmarshal failed: %v\n", err)
		}
		return nil, fmt.Errorf("failed to decode artist response: %w", err)
	}

	// Combine the artist info with the albums
	artist := discographyResp.Artist
	artist.Albums = discographyResp.Albums

	// Prioritize artist name from albums if the API returns "Unknown Artist"
	if artist.Name == "Unknown Artist" && len(artist.Albums) > 0 {
		artist.Name = artist.Albums[0].Artist
	} else if artist.Name == "" && len(artist.Albums) > 0 { // Keep existing logic for truly empty name
		artist.Name = artist.Albums[0].Artist
	}

	if debug {
		fmt.Printf("DEBUG - Successfully parsed artist: '%s' with %d albums\n", artist.Name, len(artist.Albums))
	}

	// Process albums to ensure proper categorization
	colorInfo.Println("🔍 Fetching detailed album information...")

	var wg sync.WaitGroup
	sem := semaphore.NewWeighted(int64(config.Parallelism)) // Use configured parallelism for fetching

	for i := range artist.Albums {
		wg.Add(1)
		album := &artist.Albums[i] // Capture album for goroutine

		go func(album *Album) {
			defer wg.Done()
			if err := sem.Acquire(ctx, 1); err != nil {
				colorError.Printf("Failed to acquire semaphore for album %s: %v\n", album.Title, err)
				return
			}
			defer sem.Release(1)

			// If album type is not provided by the discography endpoint, fetch full album details
			if album.Type == "" || len(album.Tracks) == 0 {
				colorInfo.Printf("  Fetching details for album: %s (ID: %s)\n", album.Title, album.ID)
				if debug {
					fmt.Printf("DEBUG - Fetching full album details for album ID: %s, Title: %s\n", album.ID, album.Title)
				}
				fullAlbum, err := api.GetAlbum(ctx, album.ID)
				if err != nil {
					if debug {
						fmt.Printf("DEBUG - Failed to fetch full album details for %s: %v\n", album.Title, err)
					}
					// Continue with heuristic if fetching full album fails
				} else {
					// Update album with full details
					album.Type = fullAlbum.Type
					album.Tracks = fullAlbum.Tracks
					album.TotalTracks = fullAlbum.TotalTracks
					album.TotalDiscs = fullAlbum.TotalDiscs
					album.Year = fullAlbum.Year
				}
			}

			// Auto-detect type if still not provided or tracks were empty
			if album.Type == "" {
				trackCount := len(album.Tracks)
				if trackCount == 0 {
					album.Type = "album" // Default assumption if no track info
				} else if trackCount == 1 {
					album.Type = "single"
				} else if trackCount <= 6 {
					album.Type = "ep"
				} else {
					album.Type = "album"
				}
			}

			// Normalize type to lowercase for consistency
			album.Type = strings.ToLower(album.Type)

			// Set year if missing
			if album.Year == "" && len(album.ReleaseDate) >= 4 {
				album.Year = album.ReleaseDate[:4]
			}
		}(album)
	}
	wg.Wait()

	return &artist, nil
}

// GetTrack retrieves track information
func (api *DabAPI) GetTrack(ctx context.Context, trackID string) (*Track, error) {
	resp, err := api.Request(ctx, "api/track", true, []QueryParam{
		{Name: "trackId", Value: trackID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get track: %w", err)
	}
	defer resp.Body.Close()

	var trackResp TrackResponse
	if err := json.NewDecoder(resp.Body).Decode(&trackResp); err != nil {
		return nil, fmt.Errorf("failed to decode track response: %w", err)
	}

	// Set missing metadata defaults
	track := &trackResp.Track
	if track.TrackNumber == 0 {
		track.TrackNumber = 1
	}
	if track.DiscNumber == 0 {
		track.DiscNumber = 1
	}
	if track.Year == "" && len(track.ReleaseDate) >= 4 {
		track.Year = track.ReleaseDate[:4]
	}

	return track, nil
}

// GetStreamURL retrieves the stream URL for a track
func (api *DabAPI) GetStreamURL(ctx context.Context, trackID string) (string, error) {
	var streamURL StreamURL
	err := RetryWithBackoff(defaultMaxRetries, 1, func() error {
		resp, err := api.Request(ctx, "api/stream", true, []QueryParam{
			{Name: "trackId", Value: trackID},
			{Name: "quality", Value: "27"}, // Highest quality FLAC
		})
		if err != nil {
			return fmt.Errorf("failed to get stream URL: %w", err)
		}
		defer resp.Body.Close()

		if err := json.NewDecoder(resp.Body).Decode(&streamURL); err != nil {
			return fmt.Errorf("failed to decode stream URL: %w", err)
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	return streamURL.URL, nil
}

// DownloadCover downloads cover art
func (api *DabAPI) DownloadCover(ctx context.Context, coverURL string) ([]byte, error) {
	var coverData []byte
	err := RetryWithBackoff(defaultMaxRetries, 1, func() error {
		resp, err := api.Request(ctx, coverURL, false, nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		coverData, err = io.ReadAll(resp.Body)
		return err
	})
	return coverData, err
}


// Search searches for artists, albums, or tracks.
func (api *DabAPI) Search(ctx context.Context, query string, searchType string, limit int, debug bool) (*SearchResults, error) {
	results := &SearchResults{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan error, 3)

	searchTypes := []string{}
	if searchType == "all" {
		searchTypes = []string{"artist", "album", "track"}
	} else {
		searchTypes = []string{searchType}
	}

	for _, t := range searchTypes {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			params := []QueryParam{
				{Name: "q", Value: query},
				{Name: "type", Value: t},
				{Name: "limit", Value: strconv.Itoa(limit)},
			}
			resp, err := api.Request(ctx, "api/search", true, params)
			if err != nil {
				errChan <- err
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				errChan <- err
				return
			}

			mu.Lock()
			defer mu.Unlock()

			if debug {
				fmt.Printf("DEBUG - Raw search response body: %s\n", string(body))
			}
			var data map[string]json.RawMessage
			if err := json.Unmarshal(body, &data); err != nil {
				errChan <- err
				return
			}

			switch t {
			case "artist":
				if res, ok := data["artists"]; ok {
					if err := json.Unmarshal(res, &results.Artists); err != nil {
						errChan <- err
					}
				} else if res, ok := data["tracks"]; ok {
					var tempTracks []Track
					if err := json.Unmarshal(res, &tempTracks); err != nil {
						errChan <- err
						return
					}
					uniqueArtists := make(map[string]Artist)
					for _, track := range tempTracks {
						artist := Artist{
							ID:   track.ArtistId,
							Name: track.Artist,
						}
						uniqueArtists[fmt.Sprintf("%v", artist.ID)] = artist // Use artist ID as key for uniqueness
					}
					for _, artist := range uniqueArtists {
						results.Artists = append(results.Artists, artist)
					}
				} else if res, ok := data["results"]; ok {
					if err := json.Unmarshal(res, &results.Artists); err != nil {
						errChan <- err
					}
				}
			case "album":
				if res, ok := data["albums"]; ok {
					if err := json.Unmarshal(res, &results.Albums); err != nil {
						errChan <- err
					}
				} else if res, ok := data["results"]; ok {
					if err := json.Unmarshal(res, &results.Albums); err != nil {
						errChan <- err
					}
				}
			case "track":
				if res, ok := data["tracks"]; ok {
					if err := json.Unmarshal(res, &results.Tracks); err != nil {
						errChan <- err
					}
				} else if res, ok := data["results"]; ok {
					if err := json.Unmarshal(res, &results.Tracks); err != nil {
						errChan <- err
					}
				}
			}
		}(t)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			// For now, just return the first error
			return nil, err
		}
	}

	return results, nil
}