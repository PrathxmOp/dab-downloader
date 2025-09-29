package shared

import (
	"fmt"
	"net/http"
	"time"
)

// VersionInfo represents the structure of our version.json file
type VersionInfo struct {
	Version string `json:"version"`
}

// Music data structures
type Track struct {
	ID            interface{} `json:"id"`
	Title         string      `json:"title"`
	Artist        string      `json:"artist"`
	ArtistId      interface{} `json:"artistId"` // Added ArtistId field
	Cover         string      `json:"albumCover"`
	ReleaseDate   string      `json:"releaseDate"`
	Duration      int         `json:"duration"`
	Album         string      `json:"album,omitempty"`
	AlbumTitle    string      `json:"albumTitle,omitempty"`  // For search results
	AlbumArtist   string      `json:"albumArtist,omitempty"`
	Genre         string      `json:"genre,omitempty"`
	TrackNumber   int         `json:"trackNumber,omitempty"`
	DiscNumber    int         `json:"discNumber,omitempty"`
	Composer      string      `json:"composer,omitempty"`
	Producer      string      `json:"producer,omitempty"`
	Year          string      `json:"year,omitempty"`
	ISRC          string      `json:"isrc,omitempty"`
	Copyright     string      `json:"copyright,omitempty"`
	AlbumID       string      `json:"albumId"`                   // Added AlbumID field
	MusicBrainzID string      `json:"musicbrainzId,omitempty"`   // MusicBrainz ID for the track
}

type Artist struct {
	ID        interface{} `json:"id"` // Changed to interface{} to handle both string and number
	Name      string      `json:"name"`
	Picture   string      `json:"picture"`
	Albums    []Album     `json:"albums,omitempty"`
	Tracks    []Track     `json:"tracks,omitempty"`
	Bio       string      `json:"bio,omitempty"`
	Country   string      `json:"country,omitempty"`
	Followers int         `json:"followers,omitempty"`
}

type Album struct {
	ID            string      `json:"id"`
	Title         string      `json:"title"`
	Artist        string      `json:"artist"`
	Cover         string      `json:"cover"`
	ReleaseDate   string      `json:"releaseDate"`
	Tracks        []Track     `json:"tracks"`
	Genre         string      `json:"genre,omitempty"`
	Type          string      `json:"type,omitempty"` // "album", "ep", "single", etc.
	Label         interface{} `json:"label,omitempty"`
	UPC           string      `json:"upc,omitempty"`
	Copyright     string      `json:"copyright,omitempty"`
	Year          string      `json:"year,omitempty"`
	TotalTracks   int         `json:"totalTracks,omitempty"`
	TotalDiscs    int         `json:"totalDiscs,omitempty"`
	MusicBrainzID string      `json:"musicbrainzId,omitempty"` // MusicBrainz ID for the album
}

// API response structures
type ArtistResponse struct {
	Artist Artist `json:"artist"`
}

type AlbumResponse struct {
	Album Album `json:"album"`
}

type TrackResponse struct {
	Track Track `json:"track"`
}

type StreamURL struct {
	URL string `json:"url"`
}

type ArtistSearchResponse struct {
	Results []Artist `json:"results"`
}

type AlbumSearchResponse struct {
	Results []Album `json:"results"`
}

type TrackSearchResponse struct {
	Results []Track `json:"results"`
}

type SearchResults struct {
	Artists []Artist `json:"artists"`
	Albums  []Album  `json:"albums"`
	Tracks  []Track  `json:"tracks"`
}

// Query parameter structure
type QueryParam struct {
	Name  string
	Value string
}

// Download statistics
type DownloadStats struct {
	SuccessCount int
	SkippedCount int
	FailedCount  int
	FailedItems  []string
}

// Spotify types
type SpotifyTrack struct {
	Name        string
	Artist      string
	AlbumName   string
	AlbumArtist string
}

// Navidrome types
type NavidromePlaylist struct {
	ID   string
	Name string
}

// Update types
type UpdateInfo struct {
	Version     string
	DownloadURL string
	ReleaseDate string
	Notes       string
}

// trackError holds information about a failed track download
type TrackError struct {
	Title string
	Err   error
}

// ErrDownloadCancelled is returned when the user explicitly cancels a download operation.
var ErrDownloadCancelled = fmt.Errorf("download cancelled by user")

// ErrNoItemsSelected is returned when no items are selected for download.
var ErrNoItemsSelected = fmt.Errorf("no items selected for download")

// MusicBrainz types
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
	Packaging    string       `json:"packaging"`
	Barcode      string       `json:"barcode"`
	ReleaseGroup ReleaseGroup `json:"release-group"`
}

type ReleaseGroup struct {
	ID string `json:"id"`
}

type MusicBrainzTrack struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Length       int    `json:"length"`
	ArtistCredit []struct {
		Artist struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"artist"`
	} `json:"artist-credit"`
}

// MusicBrainzClient represents a client for the MusicBrainz API
type MusicBrainzClient struct {
	client *http.Client
	debug  bool
}

// NewMusicBrainzClientWithDebug creates a new MusicBrainz API client with debug mode
func NewMusicBrainzClientWithDebug(debug bool) *MusicBrainzClient {
	return &MusicBrainzClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		debug: debug,
	}
}

// SetDebug sets debug mode for the MusicBrainz client
func (mb *MusicBrainzClient) SetDebug(debug bool) {
	mb.debug = debug
}

// SearchTrack searches for a track on MusicBrainz (placeholder implementation)
func (mb *MusicBrainzClient) SearchTrack(artist, album, title string) (*MusicBrainzTrack, error) {
	// This is a placeholder - the actual implementation would be more complex
	return nil, fmt.Errorf("MusicBrainz track search not implemented")
}

// SearchRelease searches for a release on MusicBrainz (placeholder implementation)
func (mb *MusicBrainzClient) SearchRelease(artist, album string) (*MusicBrainzRelease, error) {
	// This is a placeholder - the actual implementation would be more complex
	return nil, fmt.Errorf("MusicBrainz release search not implemented")
}