package dab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"

	"dab-downloader/internal/config"
	"dab-downloader/internal/shared"
)

// NewDabAPI creates a new API client
func NewDabAPI(endpoint, outputLocation string, client *http.Client) *DabAPI {
	// Create a conservative rate limiter: 4 req/sec with burst of 8
	limiter := rate.NewLimiter(rate.Every(250*time.Millisecond), 8)
	
	return &DabAPI{
		endpoint:       strings.TrimSuffix(endpoint, "/"),
		outputLocation: outputLocation,
		client:         client,
		rateLimiter:    limiter,
	}
}

type DabAPI struct {
	endpoint       string
	outputLocation string
	client         *http.Client
	rateLimiter    *rate.Limiter // Rate limiter for API requests
	rateLimitHits  int           // Counter for consecutive rate limit hits
	mu             sync.Mutex    // Mutex to protect rate limit adjustments
}

// Request makes HTTP requests to the API with specialized 429 retry handling
func (api *DabAPI) Request(ctx context.Context, path string, isPathOnly bool, params []shared.QueryParam) (*http.Response, error) {
	// Wait for rate limiter permission - this allows bursts while maintaining overall rate limits
	if err := api.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait failed: %w", err)
	}

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

	// Use specialized 429 retry logic
	return api.requestWithRateLimitRetry(ctx, u.String())
}

// fibonacciDelay calculates delay using Fibonacci sequence: 1, 1, 2, 3, 5, 8, 13...
func fibonacciDelay(attempt int, baseDelay time.Duration) time.Duration {
	if attempt <= 0 {
		return baseDelay
	}
	
	// Fibonacci sequence: F(0)=1, F(1)=1, F(n)=F(n-1)+F(n-2)
	fib := []int{1, 2, 3, 5, 8, 13, 21, 34}
	
	if attempt >= len(fib) {
		// For attempts beyond our precomputed sequence, use the last value
		attempt = len(fib) - 1
	}
	
	return baseDelay * time.Duration(fib[attempt])
}

// requestWithRateLimitRetry implements Fibonacci backoff specifically for 429 errors
func (api *DabAPI) requestWithRateLimitRetry(ctx context.Context, url string) (*http.Response, error) {
	const maxRetries = 5
	const baseDelay = 1 * time.Second
	const maxDelay = 30 * time.Second
	
	var lastResp *http.Response
	var lastErr error
	var consecutiveRateLimits int
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}
		req.Header.Set("User-Agent", shared.UserAgent)

		resp, err := api.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("error executing request: %w", err)
			// For network errors, use Fibonacci retry logic
			if attempt < maxRetries-1 {
				delay := fibonacciDelay(attempt, baseDelay)
				if delay > maxDelay {
					delay = maxDelay
				}
				time.Sleep(delay)
				continue
			}
			return nil, lastErr
		}

		// Handle successful responses
		if resp.StatusCode == http.StatusOK {
			// Reset consecutive rate limit counter on success
			consecutiveRateLimits = 0
			api.mu.Lock()
			api.rateLimitHits = 0 // Reset global counter on success
			api.mu.Unlock()
			return resp, nil
		}

		// Handle 429 rate limit errors with incremental backoff
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			lastResp = resp
			consecutiveRateLimits++
			
			// Track global rate limit hits
			api.mu.Lock()
			api.rateLimitHits++
			shouldAdjust := api.rateLimitHits > 10 // Adjust after 10 consecutive hits
			api.mu.Unlock()
			
			// Automatically adjust rate limiter if we're hitting limits too often
			if shouldAdjust && attempt == 0 { // Only adjust on first attempt to avoid multiple adjustments
				api.AdjustRateLimitForOverload()
			}
			
			if attempt < maxRetries-1 {
				// Fibonacci backoff for 429 errors: 1s, 1s, 2s, 3s, 5s (capped at 30s)
				delay := fibonacciDelay(attempt, baseDelay)
				if delay > maxDelay {
					delay = maxDelay
				}
				
				// Add extra delay if we're hitting rate limits repeatedly
				if consecutiveRateLimits > 2 {
					delay = delay * time.Duration(consecutiveRateLimits)
					if delay > maxDelay {
						delay = maxDelay
					}
				}
				
				// Add some jitter to avoid thundering herd
				jitter := time.Duration(rand.Int63n(int64(delay/4)))
				finalDelay := delay + jitter
				
				// Log the retry attempt (visible to user for transparency)
				shared.ColorWarning.Printf("⚠️ Rate limit hit (429), retrying in %v (attempt %d/%d)\n", 
					finalDelay, attempt+1, maxRetries)
				
				// Wait before retrying
				select {
				case <-time.After(finalDelay):
					continue
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
		} else {
			// Handle other HTTP errors (don't retry)
			resp.Body.Close()
			return nil, fmt.Errorf("request failed with status: %s", resp.Status)
		}
	}
	
	// All retries exhausted
	if lastResp != nil && lastResp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limit exceeded (429) after %d attempts, server is overloaded - try reducing parallelism", maxRetries)
	}
	
	return nil, fmt.Errorf("request failed after %d attempts: %w", maxRetries, lastErr)
}

// AdjustRateLimitForOverload reduces the rate limit when server is consistently overloaded
func (api *DabAPI) AdjustRateLimitForOverload() {
	// Reduce rate limit to be more conservative: 2 req/sec with burst of 4
	api.rateLimiter = rate.NewLimiter(rate.Every(500*time.Millisecond), 4)
	shared.ColorWarning.Println("⚠️ Adjusted rate limit to be more conservative due to server overload")
}

// ResetRateLimit resets the rate limit to default values
func (api *DabAPI) ResetRateLimit() {
	// Reset to default: 4 req/sec with burst of 8
	api.rateLimiter = rate.NewLimiter(rate.Every(250*time.Millisecond), 8)
}

// GetAlbum retrieves album information
func (api *DabAPI) GetAlbum(ctx context.Context, albumID string) (*shared.Album, error) {
	resp, err := api.Request(ctx, "api/album", true, []shared.QueryParam{
		{Name: "albumId", Value: albumID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get album: %w", err)
	}
	defer resp.Body.Close()

	var albumResp shared.AlbumResponse
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
func (api *DabAPI) GetArtist(ctx context.Context, artistID string, config *config.Config, debug bool) (*shared.Artist, error) {
	if debug {
		fmt.Printf("DEBUG - GetArtist called with artistID: '%s'\n", artistID)
	}

	resp, err := api.Request(ctx, "api/discography", true, []shared.QueryParam{
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
		Artist shared.Artist  `json:"artist"`
		Albums []shared.Album `json:"albums"`
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
	shared.ColorInfo.Println("🔍 Fetching detailed album information...")

	// Determine parallelism for album detail fetching
	parallelism := 5 // Conservative default parallelism
	if config != nil && config.Parallelism > 0 {
		parallelism = config.Parallelism // Use configured parallelism
		// Cap at reasonable maximum
		if parallelism > 10 {
			parallelism = 10
		}
	}
	
	if debug {
		fmt.Printf("DEBUG - Using parallelism: %d workers for fetching %d album details\n", parallelism, len(artist.Albums))
	}

	var wg sync.WaitGroup
	sem := semaphore.NewWeighted(int64(parallelism)) // Use validated parallelism for fetching

	for i := range artist.Albums {
		wg.Add(1)
		album := &artist.Albums[i] // Capture album for goroutine

		go func(album *shared.Album, workerID int) {
			defer wg.Done()
			if err := sem.Acquire(ctx, 1); err != nil {
				shared.ColorError.Printf("Failed to acquire semaphore for album %s: %v\n", album.Title, err)
				return
			}
			defer sem.Release(1)

			// If album type is not provided by the discography endpoint, fetch full album details
			if album.Type == "" || len(album.Tracks) == 0 {
				if debug {
					fmt.Printf("DEBUG - Worker %d: Fetching full album details for album ID: %s, Title: %s\n", workerID, album.ID, album.Title)
				} else {
					shared.ColorInfo.Printf("  Fetching details for album: %s (ID: %s)\n", album.Title, album.ID)
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
		}(album, i)
	}
	wg.Wait()
	
	if debug {
		fmt.Printf("DEBUG - Completed parallel fetching of %d album details\n", len(artist.Albums))
	}

	return &artist, nil
}

// GetTrack retrieves track information
func (api *DabAPI) GetTrack(ctx context.Context, trackID string) (*shared.Track, error) {
	fmt.Printf("DEBUG - GetTrack called with trackID: '%s'\n", trackID)
	resp, err := api.Request(ctx, "api/track", true, []shared.QueryParam{
		{Name: "trackId", Value: trackID},
	})
	if err != nil {
		fmt.Printf("DEBUG - GetTrack API request failed: %v\n", err)
		return nil, fmt.Errorf("failed to get track: %w", err)
	}
	defer resp.Body.Close()

	var trackResp shared.TrackResponse
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
	var streamURL shared.StreamURL
	err := shared.RetryWithBackoff(shared.DefaultMaxRetries, 1, func() error {
		resp, err := api.Request(ctx, "api/stream", true, []shared.QueryParam{
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
	err := shared.RetryWithBackoff(shared.DefaultMaxRetries, 1, func() error {
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
func (api *DabAPI) Search(ctx context.Context, query string, searchType string, limit int, debug bool) (*shared.SearchResults, error) {
	results := &shared.SearchResults{}
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
			params := []shared.QueryParam{
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
				fmt.Printf("ERROR: Failed to unmarshal JSON. Raw response body: %s\n", string(body))
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
					var tempTracks []shared.Track
					if err := json.Unmarshal(res, &tempTracks); err != nil {
						errChan <- err
						return
					}
					uniqueArtists := make(map[string]shared.Artist)
					for _, track := range tempTracks {
						artist := shared.Artist{
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

// TestArtistEndpoints tests different possible artist endpoint formats
func (api *DabAPI) TestArtistEndpoints(ctx context.Context, artistID string) {
	shared.ColorInfo.Printf("🔍 Testing different artist endpoint formats for ID: %s\n", artistID)

	// Test different endpoint variations
	endpoints := []struct {
		path        string
		params      []shared.QueryParam
		description string
	}{
		{"discography", []shared.QueryParam{{Name: "artistId", Value: artistID}}, "Correct endpoint (discography?artistId=)"},
		{"api/discography", []shared.QueryParam{{Name: "artistId", Value: artistID}}, "With api prefix (api/discography?artistId=)"},
		{"discography", []shared.QueryParam{{Name: "id", Value: artistID}}, "Alternative param (discography?id=)"},
		{"api/artist", []shared.QueryParam{{Name: "artistId", Value: artistID}}, "Old format (api/artist?artistId=)"},
		{"api/artist", []shared.QueryParam{{Name: "id", Value: artistID}}, "Alternative param (api/artist?id=)"},
		{"api/artists", []shared.QueryParam{{Name: "artistId", Value: artistID}}, "Plural endpoint (api/artists?artistId=)"},
	}

	for i, endpoint := range endpoints {
		fmt.Printf("\n🧪 Test %d: %s\n", i+1, endpoint.description)

		resp, err := api.Request(ctx, endpoint.path, true, endpoint.params)
		if err != nil {
			shared.ColorError.Printf("   ❌ Failed: %v\n", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			shared.ColorError.Printf("   ❌ Failed to read body: %v\n", err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			shared.ColorSuccess.Printf("   ✅ SUCCESS! Status: %d, Body length: %d bytes\n", resp.StatusCode, len(body))
			shared.ColorSuccess.Printf("   Response preview: %.200s...\n", string(body))
		} else {
			shared.ColorWarning.Printf("   ⚠️  Status: %d, Body: %s\n", resp.StatusCode, string(body))
		}
	}
}

// TestAPIAvailability tests basic API connectivity
func (api *DabAPI) TestAPIAvailability(ctx context.Context) {
	shared.ColorInfo.Println("🌐 Testing basic API connectivity...")

	// Try a simple request to the base API
	resp, err := api.Request(ctx, "", true, nil)
	if err != nil {
		shared.ColorError.Printf("❌ Base API test failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	shared.ColorSuccess.Printf("✅ Base API accessible. Status: %d, Response: %.200s\n", resp.StatusCode, string(body))
}

// DebugArtistID performs comprehensive debugging for an artist ID
func (api *DabAPI) DebugArtistID(ctx context.Context, artistID string) {
	shared.ColorInfo.Printf("🐛 Starting comprehensive debug for artist ID: %s\n", artistID)

	// Test basic connectivity
	api.TestAPIAvailability(ctx)

	// Test different endpoint formats
	api.TestArtistEndpoints(ctx, artistID)

	// Check if it might be an album or track ID instead
	shared.ColorInfo.Printf("\n🔄 Testing if ID might be for album or track instead...\n")

	// Test as album ID
	resp, err := api.Request(ctx, "api/album", true, []shared.QueryParam{{Name: "albumId", Value: artistID}})
	if err == nil {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			shared.ColorWarning.Printf("⚠️  ID works as ALBUM ID! You might have provided an album ID instead of artist ID\n")
			shared.ColorWarning.Printf("   Album response preview: %.200s...\n", string(body))
		}
	}

	// Test as track ID
	resp, err = api.Request(ctx, "api/track", true, []shared.QueryParam{{Name: "trackId", Value: artistID}})
	if err == nil {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			shared.ColorWarning.Printf("⚠️  ID works as TRACK ID! You might have provided a track ID instead of artist ID\n")
			shared.ColorWarning.Printf("   Track response preview: %.200s...\n", string(body))
		}
	}
}