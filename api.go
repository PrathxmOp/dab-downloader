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
)

// NewDabAPI creates a new API client
func NewDabAPI(endpoint, outputLocation string) *DabAPI {
	return &DabAPI{
		endpoint:       strings.TrimSuffix(endpoint, "/"),
		outputLocation: outputLocation,
		client: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

// Request makes HTTP requests to the API
func (api *DabAPI) Request(ctx context.Context, path string, isPathOnly bool, params []QueryParam) (*http.Response, error) {
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

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("request failed with status: %s", resp.Status)
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
func (api *DabAPI) GetArtist(ctx context.Context, artistID string, debug bool) (*Artist, error) {
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

	if artist.Name == "" && len(artist.Albums) > 0 {
		artist.Name = artist.Albums[0].Artist
	}

	if debug {
		fmt.Printf("DEBUG - Successfully parsed artist: '%s' with %d albums\n", artist.Name, len(artist.Albums))
	}

	// Process albums to ensure proper categorization
	for i := range artist.Albums {
		album := &artist.Albums[i]

		// Auto-detect type if not provided
		if album.Type == "" {
			trackCount := len(album.Tracks)
			if trackCount == 0 {
				// If we don't have track info, we'll need to fetch it later
				album.Type = "album" // Default assumption
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
	}

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
	err := RetryWithBackoff(maxRetries, 1, func() error {
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
	err := RetryWithBackoff(maxRetries, 1, func() error {
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
func (api *DabAPI) Search(ctx context.Context, query string, searchType string, limit int) (*SearchResults, error) {
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
