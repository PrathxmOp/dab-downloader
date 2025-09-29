package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Add these constants to types.go or create constants.go
const (
	RequestTimeout    = 10 * time.Minute
	UserAgent         = "DAB-Downloader/2.0"
	DefaultMaxRetries = 3
)

// NamingOptions defines the configurable naming masks
type NamingOptions struct {
	AlbumFolderMask  string `json:"album_folder_mask"`
	EpFolderMask     string `json:"ep_folder_mask"`
	SingleFolderMask string `json:"single_folder_mask"`
	FileMask         string `json:"file_mask"`
}

// GetDefaultNamingMasks returns the default naming masks
func GetDefaultNamingMasks() NamingOptions {
	return NamingOptions{
		AlbumFolderMask:  "{artist}/{artist} - {album} ({year})",
		EpFolderMask:     "{artist}/EPs/{artist} - {album} ({year})",
		SingleFolderMask: "{artist}/Singles/{artist} - {album} ({year})",
		FileMask:         "{track_number} - {artist} - {title}",
	}
}

// ApplyDefaultNamingMasks applies default naming masks to empty fields
func (cfg *Config) ApplyDefaultNamingMasks() {
	defaults := GetDefaultNamingMasks()
	
	if cfg.NamingMasks.AlbumFolderMask == "" {
		cfg.NamingMasks.AlbumFolderMask = defaults.AlbumFolderMask
	}
	if cfg.NamingMasks.EpFolderMask == "" {
		cfg.NamingMasks.EpFolderMask = defaults.EpFolderMask
	}
	if cfg.NamingMasks.SingleFolderMask == "" {
		cfg.NamingMasks.SingleFolderMask = defaults.SingleFolderMask
	}
	if cfg.NamingMasks.FileMask == "" {
		cfg.NamingMasks.FileMask = defaults.FileMask
	}
}

// Configuration structure
type Config struct {
	APIURL              string        `json:"APIURL"`
	DownloadLocation    string        `json:"DownloadLocation"`
	Parallelism         int           `json:"Parallelism"`
	SpotifyClientID     string        `json:"SpotifyClientID"`
	SpotifyClientSecret string        `json:"SpotifyClientSecret"`
	NavidromeURL        string        `json:"NavidromeURL"`
	NavidromeUsername   string        `json:"NavidromeUsername"`
	NavidromePassword   string        `json:"NavidromePassword"`
	Format              string        `json:"Format"`
	Bitrate             string        `json:"Bitrate"`
	SaveAlbumArt        bool          `json:"SaveAlbumArt"`
	DisableUpdateCheck  bool          `json:"DisableUpdateCheck"`
	IsDockerContainer   bool          `json:"-"` // Not saved to config.json
	UpdateRepo          string        `json:"UpdateRepo"`
	NamingMasks         NamingOptions `json:"naming"`
	VerifyDownloads     bool          `json:"VerifyDownloads"`     // Enable/disable download verification
	MaxRetryAttempts    int           `json:"MaxRetryAttempts"`    // Configurable retry attempts
	WarningBehavior     string        `json:"WarningBehavior"`     // "immediate", "summary", or "silent"
}

// CreateDirIfNotExists creates a directory if it does not exist
func CreateDirIfNotExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(filePath string, config *Config) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	if err := json.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return nil
}

// SaveConfig saves configuration to a JSON file
func SaveConfig(filePath string, config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	dir := filepath.Dir(filePath)
	if err := CreateDirIfNotExists(dir); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}