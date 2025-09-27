package main

import (
	"context" // Add this import
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/time/rate"
)

const (
	musicBrainzAPI       = "https://musicbrainz.org/ws/2/"
	musicBrainzUserAgent = "dab-downloader/2.0 ( prathxm.in@gmail.com )" // Replace with your actual email or project contact
)

// MusicBrainzConfig holds retry configuration for MusicBrainz API calls
type MusicBrainzConfig struct {
	MaxRetries    int           `json:"max_retries"`
	InitialDelay  time.Duration `json:"initial_delay"`
	MaxDelay      time.Duration `json:"max_delay"`
}

// DefaultMusicBrainzConfig returns sensible defaults for MusicBrainz API retry behavior
func DefaultMusicBrainzConfig() MusicBrainzConfig {
	return MusicBrainzConfig{
		MaxRetries:   5,
		InitialDelay: 2 * time.Second,
		MaxDelay:     60 * time.Second,
	}
}

// MusicBrainzClient for making requests to the MusicBrainz API
type MusicBrainzClient struct {
	client      *http.Client
	config      MusicBrainzConfig
	debug       bool
	rateLimiter *rate.Limiter
	baseURL     string // Add this field
}




// NewMusicBrainzClientWithConfig creates a new MusicBrainz API client with custom retry configuration
func NewMusicBrainzClientWithConfig(config MusicBrainzConfig) *MusicBrainzClient {
	return &MusicBrainzClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: config,
		debug:  false,
		rateLimiter: rate.NewLimiter(rate.Every(time.Second), 1),
		baseURL: musicBrainzAPI,
	}
}

// NewMusicBrainzClientWithDebug creates a new MusicBrainz API client with debug mode
func NewMusicBrainzClientWithDebug(debug bool) *MusicBrainzClient {
	return &MusicBrainzClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: DefaultMusicBrainzConfig(),
		debug:  debug,
		rateLimiter: rate.NewLimiter(rate.Every(time.Second), 1),
		baseURL: musicBrainzAPI,
	}
}

// UpdateRetryConfig updates the retry configuration for the client
func (mb *MusicBrainzClient) UpdateRetryConfig(config MusicBrainzConfig) {
	mb.config = config
}

// GetRetryConfig returns the current retry configuration
func (mb *MusicBrainzClient) GetRetryConfig() MusicBrainzConfig {
	return mb.config
}

// SetDebug enables or disables debug logging for the client
func (mb *MusicBrainzClient) SetDebug(debug bool) {
	mb.debug = debug
}

// get makes a GET request to the MusicBrainz API (internal method without retry)
func (mb *MusicBrainzClient) get(path string) ([]byte, error) {
	var finalResp *http.Response // This will hold the successful response
	var err error

	// Construct the full URL for the MusicBrainz API request
	reqURL, err := url.Parse(mb.baseURL + path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MusicBrainz API URL: %w", err)
	}

	// Wait for the rate limiter
	mb.rateLimiter.Wait(context.Background())

	err = RetryWithBackoffForHTTPWithDebug(
		mb.config.MaxRetries,    // maxRetries from client config
		mb.config.InitialDelay,  // initialDelay from client config
		mb.config.MaxDelay,      // maxDelay from client config
		func() error {
			req, err := http.NewRequest("GET", reqURL.String(), nil)
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("User-Agent", musicBrainzUserAgent)
			req.Header.Set("Accept", "application/json")

			resp, err := mb.client.Do(req) // Use mb.client
			if err != nil {
				// Check for network-related errors that are not HTTP errors
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					return &HTTPError{StatusCode: http.StatusGatewayTimeout, Status: "Gateway Timeout", Message: err.Error()}
				}
				return err // Non-retryable network error or other error
			}

			// If the status code is retryable, return an HTTPError
			if IsRetryableHTTPError(&HTTPError{StatusCode: resp.StatusCode}) {
				// Close the body of the retryable response to prevent resource leaks
				resp.Body.Close()
				return &HTTPError{StatusCode: resp.StatusCode, Status: resp.Status, Message: "Retryable HTTP error"}
			}

			finalResp = resp // Assign the successful response to the outer variable
			return nil       // Success or non-retryable HTTP error
		},
		mb.debug, // Use client's debug setting
	)

	if err != nil {
		return nil, fmt.Errorf("failed to perform request after retries: %w", err)
	}

	defer finalResp.Body.Close()

	if finalResp.StatusCode != http.StatusOK {
		// Create structured HTTP error for retry logic
		body, _ := ioutil.ReadAll(finalResp.Body)
		message := string(body)
		if len(message) > 200 {
			message = message[:200] + "..."
		}
		return nil, &HTTPError{
			StatusCode: finalResp.StatusCode,
			Status:     finalResp.Status,
			Message:    message,
		}
	}

	body, err := ioutil.ReadAll(finalResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return body, nil
}

// getWithRetry makes a GET request to the MusicBrainz API with retry logic for retryable errors
func (mb *MusicBrainzClient) getWithRetry(path string) ([]byte, error) {
	var result []byte
	var err error

	retryErr := RetryWithBackoffForHTTPWithDebug(
		mb.config.MaxRetries,
		mb.config.InitialDelay,
		mb.config.MaxDelay,
		func() error {
			result, err = mb.get(path)
			return err
		},
		mb.debug,
	)

	if retryErr != nil {
		return nil, retryErr
	}
	return result, nil
}

// GetTrackMetadata fetches track metadata from MusicBrainz by MBID
func (mb *MusicBrainzClient) GetTrackMetadata(mbid string) (*MusicBrainzTrack, error) {
	path := fmt.Sprintf("recording/%s?inc=artists+releases+url-rels", mbid)
	body, err := mb.getWithRetry(path)
	if err != nil {
		return nil, err
	}

	var track MusicBrainzTrack
	if err := json.Unmarshal(body, &track); err != nil {
		return nil, fmt.Errorf("failed to unmarshal MusicBrainz track metadata: %w", err)
	}
	return &track, nil
}

// GetReleaseMetadata fetches release (album) metadata from MusicBrainz by MBID
func (mb *MusicBrainzClient) GetReleaseMetadata(mbid string) (*MusicBrainzRelease, error) {
	path := fmt.Sprintf("release/%s?inc=artists+labels+recordings+url-rels+release-groups", mbid)
	body, err := mb.getWithRetry(path)
	if err != nil {
		return nil, err
	}

	var release MusicBrainzRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("failed to unmarshal MusicBrainz release metadata: %w", err)
	}
	return &release, nil
}

// SearchTrack searches for a track on MusicBrainz
func (mb *MusicBrainzClient) SearchTrack(artist, album, title string) (*MusicBrainzTrack, error) {
	query := fmt.Sprintf("artist:\"%s\" AND release:\"%s\" AND recording:\"%s\"", artist, album, title)
	path := fmt.Sprintf("recording?query=%s&limit=1", url.QueryEscape(query))
	body, err := mb.getWithRetry(path)
	if err != nil {
		return nil, err
	}

	var searchResult struct {
		Recordings []MusicBrainzTrack `json:"recordings"`
	}
	if err := json.Unmarshal(body, &searchResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal MusicBrainz track search result: %w", err)
	}

	if len(searchResult.Recordings) > 0 {
		return &searchResult.Recordings[0], nil
	}

	return nil, fmt.Errorf("no track found on MusicBrainz for: %s - %s - %s", artist, album, title)
}

// SearchRelease searches for a release on MusicBrainz
func (mb *MusicBrainzClient) SearchRelease(artist, album string) (*MusicBrainzRelease, error) {
	query := fmt.Sprintf("artist:\"%s\" AND release:\"%s\"", artist, album)
	path := fmt.Sprintf("release?query=%s&limit=1", url.QueryEscape(query))
	body, err := mb.getWithRetry(path)
	if err != nil {
		return nil, err
	}

	var searchResult struct {
		Releases []MusicBrainzRelease `json:"releases"`
	}
	if err := json.Unmarshal(body, &searchResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal MusicBrainz release search result: %w", err)
	}

	if len(searchResult.Releases) > 0 {
		return &searchResult.Releases[0], nil
	}

	return nil, fmt.Errorf("no release found on MusicBrainz for: %s - %s", artist, album)
}

// MusicBrainzTrack represents a simplified MusicBrainz recording (track)
type MusicBrainzTrack struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	ArtistCredit []struct {
		Artist struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"artist"`
	} `json:"artist-credit"`
	Releases []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Date  string `json:"date"`
		Media []struct {
			Format string `json:"format"`
			Discs  []struct {
				ID string `json:"id"`
			} `json:"discs"`
			Tracks []struct {
				ID     string `json:"id"`
				Number string `json:"number"`
				Title  string `json:"title"`
				Length int    `json:"length"`
			} `json:"tracks"`
		} `json:"media"`
	} `json:"releases"`
	Length int `json:"length"` // Duration in milliseconds
}

// MusicBrainzRelease represents a simplified MusicBrainz release (album)
type MusicBrainzRelease struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Status       string `json:"status"`
	Date         string `json:"date"`
	Country      string `json:"country"`
	ArtistCredit []struct {
		Artist struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"artist"`
	} `json:"artist-credit"`
	LabelInfo []struct {
		CatalogNumber string `json:"catalog-number"`
		Label         struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"label"`
	} `json:"label-info"`
	Media []struct {
		Format string `json:"format"`
		Discs  []struct {
			ID string `json:"id"`
		} `json:"discs"`
		Tracks []struct {
			ID     string `json:"id"`
			Number string `json:"number"`
			Title  string `json:"title"`
			Length int    `json:"length"`
		} `json:"tracks"`
	} `json:"media"`
	TextRepresentation struct {
		Language string `json:"language"`
		Script   string `json:"script"`
	} `json:"text-representation"`
	Packaging    string         `json:"packaging"`
	Barcode      string         `json:"barcode"`
	ReleaseGroup ReleaseGroup `json:"release-group"`
	// Add other fields as needed
}

type ReleaseGroup struct {
	ID string `json:"id"`
}