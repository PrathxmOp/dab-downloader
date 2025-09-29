package interfaces

import (
	"context"
	"net/http"

	"dab-downloader/internal/config"
	"dab-downloader/internal/shared"
)

// APIClient defines the interface for DAB API interactions
type APIClient interface {
	// Search performs a search query and returns results
	Search(ctx context.Context, query, searchType string, limit int, debug bool) (*shared.SearchResults, error)
	
	// GetAlbum retrieves detailed album information by ID
	GetAlbum(ctx context.Context, albumID string) (*shared.Album, error)
	
	// GetArtist retrieves artist information and discography by ID
	GetArtist(ctx context.Context, artistID string, config *config.Config, debug bool) (*shared.Artist, error)
	
	// GetTrack retrieves track information by ID
	GetTrack(ctx context.Context, trackID string) (*shared.Track, error)
	
	// GetStreamURL retrieves the streaming URL for a track
	GetStreamURL(ctx context.Context, trackID string) (string, error)
	
	// DownloadCover downloads cover art and returns the image data
	DownloadCover(ctx context.Context, coverURL string) ([]byte, error)
	
	// Request makes HTTP requests to the API
	Request(ctx context.Context, path string, isPathOnly bool, params []shared.QueryParam) (*http.Response, error)
}

// DownloadService defines the interface for download operations
type DownloadService interface {
	// GetArtistInfo retrieves artist information and discography by ID
	GetArtistInfo(ctx context.Context, artistID string, config *config.Config, debug bool) (*shared.Artist, error)
	
	// GetAlbumInfo retrieves album information by ID
	GetAlbumInfo(ctx context.Context, albumID string, config *config.Config, debug bool) (*shared.Album, error)
	
	// DownloadAlbum downloads an entire album by ID
	DownloadAlbum(ctx context.Context, albumID string, config *config.Config, debug bool, format string, bitrate string) (*shared.DownloadStats, error)
	
	// DownloadArtist downloads an artist's discography by ID
	DownloadArtist(ctx context.Context, artistID string, config *config.Config, debug bool, format string, bitrate string, filter string, noConfirm bool) (*shared.DownloadStats, error)
	
	// DownloadTrack downloads a single track by ID
	DownloadTrack(ctx context.Context, trackID string, config *config.Config, debug bool, format string, bitrate string) (*shared.DownloadStats, error)
	
	// DownloadTrackDirect downloads a track using the track data directly (bypassing GetTrack API call)
	DownloadTrackDirect(ctx context.Context, track shared.Track, config *config.Config, debug bool, format string, bitrate string) (*shared.DownloadStats, error)
	
	// DownloadTracks downloads multiple tracks
	DownloadTracks(ctx context.Context, tracks []shared.Track, album *shared.Album, config *config.Config, debug bool, format string, bitrate string) (*shared.DownloadStats, error)
}

// SearchService defines the interface for search operations
type SearchService interface {
	// HandleSearch performs a search and handles user interaction for selection
	HandleSearch(ctx context.Context, query string, searchType string, debug bool, auto bool, config *config.Config) ([]interface{}, []string, error)
	
	// Search performs a raw search without user interaction
	Search(ctx context.Context, query string, searchType string, limit int, debug bool) (*shared.SearchResults, error)
}

// ConfigService defines the interface for configuration management
type ConfigService interface {
	// LoadConfig loads configuration from file
	LoadConfig(configFile string) (*config.Config, error)
	
	// SaveConfig saves configuration to file
	SaveConfig(configFile string, config *config.Config) error
	
	// ValidateConfig validates configuration settings
	ValidateConfig(config *config.Config) error
	
	// GetDefaultConfig returns a default configuration
	GetDefaultConfig() *config.Config
	
	// EnsureConfigExists creates a default config file if it doesn't exist
	EnsureConfigExists(configFile string) error
}

// SpotifyService defines the interface for Spotify integration
type SpotifyService interface {
	// Authenticate authenticates with Spotify API
	Authenticate() error
	
	// GetPlaylistTracks retrieves tracks from a Spotify playlist
	GetPlaylistTracks(playlistURL string) ([]shared.SpotifyTrack, string, error)
	
	// GetAlbumTracks retrieves tracks from a Spotify album
	GetAlbumTracks(albumURL string) ([]shared.SpotifyTrack, string, error)
}

// NavidromeService defines the interface for Navidrome integration
type NavidromeService interface {
	// Authenticate authenticates with Navidrome server
	Authenticate() error
	
	// CreatePlaylist creates a playlist in Navidrome
	CreatePlaylist(name string, tracks []shared.Track) error
	
	// GetPlaylists retrieves all playlists from Navidrome
	GetPlaylists() ([]shared.NavidromePlaylist, error)
	
	// AddTracksToPlaylist adds tracks to an existing playlist
	AddTracksToPlaylist(playlistID string, tracks []shared.Track) error
}

// UpdaterService defines the interface for application updates
type UpdaterService interface {
	// CheckForUpdates checks if a new version is available
	CheckForUpdates(ctx context.Context, currentVersion string, updateRepo string) (*shared.UpdateInfo, error)
	
	// DownloadUpdate downloads the latest version
	DownloadUpdate(ctx context.Context, updateInfo *shared.UpdateInfo, outputPath string) error
	
	// ApplyUpdate applies the downloaded update
	ApplyUpdate(updatePath string, currentBinaryPath string) error
}

// FileSystemService defines the interface for file system operations
type FileSystemService interface {
	// EnsureDirectoryExists creates a directory if it doesn't exist
	EnsureDirectoryExists(path string) error
	
	// GetDownloadPath constructs the full download path for a track
	GetDownloadPath(artist, album, track string, format string, config *config.Config) string
	
	// GetDownloadPathWithTrack constructs the full download path using naming masks and track metadata
	GetDownloadPathWithTrack(track shared.Track, album *shared.Album, format string, config *config.Config) string
	
	// FileExists checks if a file exists
	FileExists(path string) bool
	
	// GetFileSize returns the size of a file
	GetFileSize(path string) (int64, error)
	
	// ValidateDownloadLocation checks if the download location is accessible
	ValidateDownloadLocation(path string) error
	
	// SanitizeFileName sanitizes a filename for the file system
	SanitizeFileName(filename string) string
}

// LoggerService defines the interface for logging operations
type LoggerService interface {
	// Info logs an informational message
	Info(message string, args ...interface{})
	
	// Warning logs a warning message
	Warning(message string, args ...interface{})
	
	// Error logs an error message
	Error(message string, args ...interface{})
	
	// Debug logs a debug message
	Debug(message string, args ...interface{})
	
	// Success logs a success message
	Success(message string, args ...interface{})
	
	// SetDebugMode enables or disables debug logging
	SetDebugMode(enabled bool)
}

// WarningCollectorService defines the interface for warning collection
type WarningCollectorService interface {
	// AddWarning adds a warning to the collection
	AddWarning(warningType shared.WarningType, context, message, details string)
	
	// AddMusicBrainzTrackWarning adds a MusicBrainz track lookup warning
	AddMusicBrainzTrackWarning(artist, title, details string)
	
	// AddMusicBrainzReleaseWarning adds a MusicBrainz release lookup warning
	AddMusicBrainzReleaseWarning(artist, album, details string)
	
	// HasWarnings returns true if there are any warnings
	HasWarnings() bool
	
	// GetWarningCount returns the total number of warnings
	GetWarningCount() int
	
	// PrintSummary prints a formatted summary of all warnings
	PrintSummary()
}

// MetadataService defines the interface for metadata operations
type MetadataService interface {
	// AddMetadata adds metadata to an audio file
	AddMetadata(filePath string, track shared.Track, album *shared.Album, coverData []byte, totalTracks int) error
	
	// ExtractMetadata extracts metadata from an audio file
	ExtractMetadata(filePath string) (*shared.Track, error)
	
	// ValidateMetadata validates that metadata was correctly applied
	ValidateMetadata(filePath string, expectedTrack shared.Track) error
}

// ConversionService defines the interface for audio format conversion
type ConversionService interface {
	// ConvertTrack converts an audio file to a different format
	ConvertTrack(inputPath string, format string, bitrate string) (string, error)
	
	// GetSupportedFormats returns a list of supported output formats
	GetSupportedFormats() []string
	
	// ValidateFormat validates that a format is supported
	ValidateFormat(format string) error
	
	// ValidateBitrate validates that a bitrate is valid for the format
	ValidateBitrate(format string, bitrate string) error
}