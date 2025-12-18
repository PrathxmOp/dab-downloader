package musicbrainz

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/time/rate"

	"dab-downloader/internal/shared"
)

// 1. Constants and types
const (
	defaultBaseURL       = "https://musicbrainz.org/ws/2/"
	defaultUserAgent     = "dab-downloader/2.0 ( prathxm.in@gmail.com )"
	defaultTimeout       = 30 * time.Second
	defaultRateLimit     = 333 * time.Millisecond // MusicBrainz allows ~3 requests per second
	defaultBurstLimit    = 6
	defaultMaxRetries    = 5
	defaultInitialDelay  = 2 * time.Second
	defaultMaxDelay      = 60 * time.Second
)

// Config holds configuration for MusicBrainz API client
type Config struct {
	BaseURL       string        `json:"base_url"`
	UserAgent     string        `json:"user_agent"`
	Timeout       time.Duration `json:"timeout"`
	MaxRetries    int           `json:"max_retries"`
	InitialDelay  time.Duration `json:"initial_delay"`
	MaxDelay      time.Duration `json:"max_delay"`
	RateLimit     time.Duration `json:"rate_limit"`
	BurstLimit    int           `json:"burst_limit"`
	Debug         bool          `json:"debug"`
}

// Client represents a MusicBrainz API client
type Client struct {
	httpClient  *http.Client
	config      Config
	rateLimiter *rate.Limiter
}

// 2. Constructor and configuration

// DefaultConfig returns sensible defaults for MusicBrainz API client
func DefaultConfig() Config {
	return Config{
		BaseURL:      defaultBaseURL,
		UserAgent:    defaultUserAgent,
		Timeout:      defaultTimeout,
		MaxRetries:   defaultMaxRetries,
		InitialDelay: defaultInitialDelay,
		MaxDelay:     defaultMaxDelay,
		RateLimit:    defaultRateLimit,
		BurstLimit:   defaultBurstLimit,
		Debug:        false,
	}
}

// NewClient creates a new MusicBrainz API client with default configuration
func NewClient() *Client {
	return NewClientWithConfig(DefaultConfig())
}

// NewClientWithConfig creates a new MusicBrainz API client with custom configuration
func NewClientWithConfig(config Config) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		config:      config,
		rateLimiter: rate.NewLimiter(rate.Every(config.RateLimit), config.BurstLimit),
	}
}

// NewClientWithDebug creates a new MusicBrainz API client with debug mode enabled
func NewClientWithDebug(debug bool) *Client {
	config := DefaultConfig()
	config.Debug = debug
	return NewClientWithConfig(config)
}

// UpdateConfig updates the client configuration
func (c *Client) UpdateConfig(config Config) {
	c.config = config
	c.httpClient.Timeout = config.Timeout
	c.rateLimiter = rate.NewLimiter(rate.Every(config.RateLimit), config.BurstLimit)
}

// GetConfig returns the current client configuration
func (c *Client) GetConfig() Config {
	return c.config
}

// SetDebug enables or disables debug logging for the client
func (c *Client) SetDebug(debug bool) {
	c.config.Debug = debug
}

// 3. Core HTTP methods (private)

// makeRequest creates and executes an HTTP request with proper headers
func (c *Client) makeRequest(ctx context.Context, path string) (*http.Response, error) {
	reqURL, err := url.Parse(c.config.BaseURL + path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.config.UserAgent)
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

// get makes a single GET request to the MusicBrainz API
func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	resp, err := c.makeRequest(ctx, path)
	if err != nil {
		// Handle network timeouts
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, &shared.HTTPError{
				StatusCode: http.StatusGatewayTimeout,
				Status:     "Gateway Timeout",
				Message:    err.Error(),
			}
		}
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		message := string(body)
		if len(message) > 200 {
			message = message[:200] + "..."
		}
		return nil, &shared.HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Message:    message,
		}
	}

	return body, nil
}

// getWithRetry makes a GET request with retry logic
func (c *Client) getWithRetry(ctx context.Context, path string) ([]byte, error) {
	var result []byte
	var err error

	retryErr := shared.RetryWithBackoffForHTTPWithDebug(
		c.config.MaxRetries,
		c.config.InitialDelay,
		c.config.MaxDelay,
		func() error {
			result, err = c.get(ctx, path)
			return err
		},
		c.config.Debug,
	)

	if retryErr != nil {
		return nil, retryErr
	}
	return result, nil
}

// 4. Public API methods (grouped by functionality)

// Metadata retrieval methods

// GetTrackMetadata fetches track metadata from MusicBrainz by MBID
func (c *Client) GetTrackMetadata(ctx context.Context, mbid string) (*Track, error) {
	if mbid == "" {
		return nil, fmt.Errorf("MBID cannot be empty")
	}

	path := fmt.Sprintf("recording/%s?inc=artists+releases+url-rels", mbid)
	body, err := c.getWithRetry(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch track metadata for MBID %s: %w", mbid, err)
	}

	var track Track
	if err := json.Unmarshal(body, &track); err != nil {
		return nil, fmt.Errorf("failed to unmarshal track metadata: %w", err)
	}
	return &track, nil
}

// GetReleaseMetadata fetches release (album) metadata from MusicBrainz by MBID
func (c *Client) GetReleaseMetadata(ctx context.Context, mbid string) (*Release, error) {
	if mbid == "" {
		return nil, fmt.Errorf("MBID cannot be empty")
	}

	path := fmt.Sprintf("release/%s?inc=artists+labels+recordings+url-rels+release-groups", mbid)
	body, err := c.getWithRetry(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release metadata for MBID %s: %w", mbid, err)
	}

	var release Release
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("failed to unmarshal release metadata: %w", err)
	}
	return &release, nil
}

// Search methods

// SearchTrackByISRC searches for a track on MusicBrainz using ISRC
func (c *Client) SearchTrackByISRC(ctx context.Context, isrc string) (*Track, error) {
	if isrc == "" {
		return nil, fmt.Errorf("ISRC cannot be empty")
	}

	query := fmt.Sprintf("isrc:\"%s\"", isrc)
	path := fmt.Sprintf("recording?query=%s&inc=artists+releases+release-groups+recordings&limit=1", url.QueryEscape(query))
	
	body, err := c.getWithRetry(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to search track by ISRC %s: %w", isrc, err)
	}

	var searchResult struct {
		Recordings []Track `json:"recordings"`
	}
	if err := json.Unmarshal(body, &searchResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ISRC search result: %w", err)
	}

	if len(searchResult.Recordings) == 0 {
		return nil, fmt.Errorf("no track found for ISRC: %s", isrc)
	}

	return &searchResult.Recordings[0], nil
}

// SearchTrack searches for a track on MusicBrainz by artist, album, and title
func (c *Client) SearchTrack(ctx context.Context, artist, album, title string) (*Track, error) {
	if artist == "" || title == "" {
		return nil, fmt.Errorf("artist and title cannot be empty")
	}

	query := buildTrackSearchQuery(artist, album, title)
	path := fmt.Sprintf("recording?query=%s&limit=1", url.QueryEscape(query))
	
	body, err := c.getWithRetry(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to search track: %w", err)
	}

	var searchResult struct {
		Recordings []Track `json:"recordings"`
	}
	if err := json.Unmarshal(body, &searchResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal track search result: %w", err)
	}

	if len(searchResult.Recordings) == 0 {
		return nil, fmt.Errorf("no track found for: %s - %s - %s", artist, album, title)
	}

	return &searchResult.Recordings[0], nil
}

// SearchRelease searches for a release on MusicBrainz by artist and album
func (c *Client) SearchRelease(ctx context.Context, artist, album string) (*Release, error) {
	if artist == "" || album == "" {
		return nil, fmt.Errorf("artist and album cannot be empty")
	}

	query := fmt.Sprintf("artist:\"%s\" AND release:\"%s\"", artist, album)
	path := fmt.Sprintf("release?query=%s&limit=1", url.QueryEscape(query))
	
	body, err := c.getWithRetry(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to search release: %w", err)
	}

	var searchResult struct {
		Releases []Release `json:"releases"`
	}
	if err := json.Unmarshal(body, &searchResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal release search result: %w", err)
	}

	if len(searchResult.Releases) == 0 {
		return nil, fmt.Errorf("no release found for: %s - %s", artist, album)
	}

	return &searchResult.Releases[0], nil
}

// 5. Helper/utility functions

// buildTrackSearchQuery constructs a search query for track searches
func buildTrackSearchQuery(artist, album, title string) string {
	if album == "" {
		return fmt.Sprintf("artist:\"%s\" AND recording:\"%s\"", artist, title)
	}
	return fmt.Sprintf("artist:\"%s\" AND release:\"%s\" AND recording:\"%s\"", artist, album, title)
}

// Data types

// Artist represents a MusicBrainz artist
type Artist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ArtistCredit represents artist credit information
type ArtistCredit struct {
	Artist Artist `json:"artist"`
}

// MediaTrack represents a track within media
type MediaTrack struct {
	ID     string `json:"id"`
	Number string `json:"number"`
	Title  string `json:"title"`
	Length int    `json:"length"`
}

// Disc represents a disc within media
type Disc struct {
	ID string `json:"id"`
}

// Media represents media information
type Media struct {
	Format string       `json:"format"`
	Discs  []Disc       `json:"discs"`
	Tracks []MediaTrack `json:"tracks"`
}

// ReleaseGroup represents a MusicBrainz release group
type ReleaseGroup struct {
	ID string `json:"id"`
}

// TrackRelease represents release information within a track
type TrackRelease struct {
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	Date         string         `json:"date"`
	ArtistCredit []ArtistCredit `json:"artist-credit"`
	ReleaseGroup ReleaseGroup   `json:"release-group"`
	Media        []Media        `json:"media"`
}

// Track represents a MusicBrainz recording (track)
type Track struct {
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	ArtistCredit []ArtistCredit `json:"artist-credit"`
	Releases     []TrackRelease `json:"releases"`
	Length       int            `json:"length"` // Duration in milliseconds
}

// Label represents a MusicBrainz label
type Label struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// LabelInfo represents label information
type LabelInfo struct {
	CatalogNumber string `json:"catalog-number"`
	Label         Label  `json:"label"`
}

// TextRepresentation represents text representation information
type TextRepresentation struct {
	Language string `json:"language"`
	Script   string `json:"script"`
}

// Release represents a MusicBrainz release (album)
type Release struct {
	ID                 string               `json:"id"`
	Title              string               `json:"title"`
	Status             string               `json:"status"`
	Date               string               `json:"date"`
	Country            string               `json:"country"`
	ArtistCredit       []ArtistCredit       `json:"artist-credit"`
	LabelInfo          []LabelInfo          `json:"label-info"`
	Media              []Media              `json:"media"`
	TextRepresentation TextRepresentation   `json:"text-representation"`
	Packaging          string               `json:"packaging"`
	Barcode            string               `json:"barcode"`
	ReleaseGroup       ReleaseGroup         `json:"release-group"`
}