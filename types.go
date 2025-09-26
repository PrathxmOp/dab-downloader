package main

import (
	"fmt"
	
	"time"
)

// Add these constants to types.go or create constants.go
const (
	requestTimeout    = 10 * time.Minute
	userAgent         = "DAB-Downloader/2.0"
	defaultMaxRetries = 3
)

// Configuration structure
type Config struct {
	APIURL              string
	DownloadLocation    string
	Parallelism         int
	SpotifyClientID     string
	SpotifyClientSecret string
	NavidromeURL        string
	NavidromeUsername   string
	NavidromePassword   string
	Format              string
	Bitrate             string
	SaveAlbumArt        bool
	DisableUpdateCheck  bool `json:"DisableUpdateCheck"`
	IsDockerContainer   bool `json:"-"` // Not saved to config.json
	UpdateRepo          string `json:"UpdateRepo"`
	NamingMasks         NamingOptions `json:"naming"`
	VerifyDownloads     bool `json:"VerifyDownloads"` // Enable/disable download verification
	MaxRetryAttempts    int  `json:"MaxRetryAttempts"` // Configurable retry attempts
}

// NamingOptions defines the configurable naming masks
type NamingOptions struct {
	AlbumFolderMask  string `json:"album_folder_mask"`
	EpFolderMask     string `json:"ep_folder_mask"`
	SingleFolderMask string `json:"single_folder_mask"`
	FileMask         string `json:"file_mask"`
}

// VersionInfo represents the structure of our version.json file
type VersionInfo struct {
	Version string `json:"version"`
}



// Music data structures
type Track struct {
	ID          interface{} `json:"id"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	ArtistId    interface{} `json:"artistId"` // Added ArtistId field
	Cover       string `json:"albumCover"`
	ReleaseDate string `json:"releaseDate"`
	Duration    int    `json:"duration"`
	Album       string `json:"album,omitempty"`
	AlbumArtist string `json:"albumArtist,omitempty"`
	Genre       string `json:"genre,omitempty"`
	TrackNumber int    `json:"trackNumber,omitempty"`
	DiscNumber  int    `json:"discNumber,omitempty"`
	Composer    string `json:"composer,omitempty"`
	Producer    string `json:"producer,omitempty"`
	Year        string `json:"year,omitempty"`
	ISRC        string `json:"isrc,omitempty"`
	Copyright   string `json:"copyright,omitempty"`
	AlbumID     string `json:"albumId"` // Added AlbumID field
	MusicBrainzID string `json:"musicbrainzId,omitempty"` // MusicBrainz ID for the track
}

type Artist struct {
	ID          interface{} `json:"id"` // Changed to interface{} to handle both string and number
	Name        string  `json:"name"`
	Picture     string  `json:"picture"`
	Albums      []Album `json:"albums,omitempty"`
	Tracks      []Track `json:"tracks,omitempty"`
	Bio         string  `json:"bio,omitempty"`
	Country     string  `json:"country,omitempty"`
	Followers   int     `json:"followers,omitempty"`
}

type Album struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Artist      string      `json:"artist"`
	Cover       string      `json:"cover"`
	ReleaseDate string      `json:"releaseDate"`
	Tracks      []Track     `json:"tracks"`
	Genre       string      `json:"genre,omitempty"`
	Type        string      `json:"type,omitempty"` // "album", "ep", "single", etc.
	Label       interface{} `json:"label,omitempty"`
	UPC         string      `json:"upc,omitempty"`
	Copyright   string      `json:"copyright,omitempty"`
	Year        string      `json:"year,omitempty"`
	TotalTracks int         `json:"totalTracks,omitempty"`
	TotalDiscs  int         `json:"totalDiscs,omitempty"`
	MusicBrainzID string `json:"musicbrainzId,omitempty"` // MusicBrainz ID for the album
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

// trackError holds information about a failed track download
type trackError struct {
	Title string
	Err   error
}

// ErrDownloadCancelled is returned when the user explicitly cancels a download operation.
var ErrDownloadCancelled = fmt.Errorf("download cancelled by user")

// ErrNoItemsSelected is returned when no items are selected for download.
var ErrNoItemsSelected = fmt.Errorf("no items selected for download")