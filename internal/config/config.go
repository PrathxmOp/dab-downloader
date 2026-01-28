package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"dab-downloader/internal/utils"
)

const (
	DefaultMaxRetries = 3
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
	WarningBehavior     string `json:"WarningBehavior"` // "immediate", "summary", or "silent"
}

// NamingOptions defines the configurable naming masks
type NamingOptions struct {
	AlbumFolderMask  string `json:"album_folder_mask"`
	EpFolderMask     string `json:"ep_folder_mask"`
	SingleFolderMask string `json:"single_folder_mask"`
	FileMask         string `json:"file_mask"`
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
	if err := utils.CreateDirIfNotExists(dir); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}
		return nil
	}
	
	// GetConfigFilePath returns the path to the configuration file.
	// Priority:
	// 1. Local config/config.json (if exists) - Backward compatibility / Portable
	// 2. Local config directory (if exists) - Portable mode / Docker volume
	// 3. Docker environment - Default to local config/config.json
	// 4. Standard user config directory (~/.config/dab-downloader/config.json)
	func GetConfigFilePath() string {
		localConfig := filepath.Join("config", "config.json")
		localConfigDir := "config"
	
		// 1. If local config file exists, use it
		if utils.FileExists(localConfig) {
			return localConfig
		}
	
		// 2. If local config directory exists, use it (implies portable mode or Docker volume)
		if info, err := os.Stat(localConfigDir); err == nil && info.IsDir() {
			return localConfig
		}
	
		// 3. If running in Docker, default to local config
		if _, err := os.Stat("/.dockerenv"); err == nil {
			return localConfig
		}
	
		// 4. Try standard user config directory
		userConfigDir, err := os.UserConfigDir()
		if err == nil {
			appConfigDir := filepath.Join(userConfigDir, "dab-downloader")
			return filepath.Join(appConfigDir, "config.json")
		}
	
		// Fallback to local config path
		return localConfig
	}
		// GetConfigDir returns the directory where configuration files are stored.
	func GetConfigDir() string {
		return filepath.Dir(GetConfigFilePath())
	}
	