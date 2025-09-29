package services

import (
	"net/http"
	"testing"
	"time"

	"dab-downloader/internal/config"
	"dab-downloader/internal/shared"
)

func TestNewServiceContainer(t *testing.T) {
	// Create a test configuration
	cfg := &config.Config{
		APIURL:           "https://api.test.com",
		DownloadLocation: "./test-downloads",
		Parallelism:      3,
		Format:           "flac",
		Bitrate:          "320",
		VerifyDownloads:  true,
		MaxRetryAttempts: 3,
		WarningBehavior:  "display",
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Test service container creation
	container := NewServiceContainer(cfg, httpClient)

	// Verify all services are initialized
	if container.Config == nil {
		t.Error("Config service not initialized")
	}
	if container.APIClient == nil {
		t.Error("APIClient service not initialized")
	}
	if container.DownloadService == nil {
		t.Error("DownloadService not initialized")
	}
	if container.SearchService == nil {
		t.Error("SearchService not initialized")
	}
	if container.SpotifyService == nil {
		t.Error("SpotifyService not initialized")
	}
	if container.NavidromeService == nil {
		t.Error("NavidromeService not initialized")
	}
	if container.UpdaterService == nil {
		t.Error("UpdaterService not initialized")
	}
	if container.FileSystem == nil {
		t.Error("FileSystem service not initialized")
	}
	if container.Logger == nil {
		t.Error("Logger service not initialized")
	}
	if container.WarningCollector == nil {
		t.Error("WarningCollector service not initialized")
	}
	if container.Metadata == nil {
		t.Error("Metadata service not initialized")
	}
	if container.Conversion == nil {
		t.Error("Conversion service not initialized")
	}
}

func TestConfigService(t *testing.T) {
	cs := NewConfigService()

	// Test default config creation
	defaultConfig := cs.GetDefaultConfig()
	if defaultConfig.APIURL == "" {
		t.Error("Default config should have API URL")
	}
	if defaultConfig.DownloadLocation == "" {
		t.Error("Default config should have download location")
	}

	// Test config validation
	err := cs.ValidateConfig(defaultConfig)
	if err != nil {
		t.Errorf("Default config should be valid: %v", err)
	}

	// Test invalid config
	invalidConfig := &config.Config{}
	err = cs.ValidateConfig(invalidConfig)
	if err == nil {
		t.Error("Invalid config should fail validation")
	}
}

func TestFileSystemService(t *testing.T) {
	cfg := &config.Config{
		DownloadLocation: "./test-downloads",
	}

	fss := NewFileSystemService(cfg)

	// Test filename sanitization
	sanitized := fss.SanitizeFileName("Test/File\\Name:With*Invalid?Chars")
	if sanitized == "Test/File\\Name:With*Invalid?Chars" {
		t.Error("Filename should be sanitized")
	}

	// Test path generation
	path := fss.GetDownloadPath("Artist", "Album", "Track", "flac", cfg)
	if path == "" {
		t.Error("Download path should not be empty")
	}
}

func TestConsoleLogger(t *testing.T) {
	logger := NewConsoleLogger()

	// Test debug mode
	logger.SetDebugMode(true)
	// These won't fail but will test the interface
	logger.Info("Test info message")
	logger.Warning("Test warning message")
	logger.Error("Test error message")
	logger.Debug("Test debug message")
	logger.Success("Test success message")
}

func TestWarningCollector(t *testing.T) {
	wc := shared.NewWarningCollector(true)

	// Test initial state
	if wc.HasWarnings() {
		t.Error("New warning collector should have no warnings")
	}

	// Test adding warnings
	wc.AddMusicBrainzTrackWarning("Artist", "Track", "Test details")
	wc.AddMusicBrainzReleaseWarning("Artist", "Album", "Test details")

	if !wc.HasWarnings() {
		t.Error("Warning collector should have warnings after adding")
	}

	count := wc.GetWarningCount()
	if count != 2 {
		t.Errorf("Expected 2 warnings, got %d", count)
	}
}

func TestConversionService(t *testing.T) {
	cs := NewConversionService()

	// Test supported formats
	formats := cs.GetSupportedFormats()
	if len(formats) == 0 {
		t.Error("Should have supported formats")
	}

	// Test format validation
	err := cs.ValidateFormat("flac")
	if err != nil {
		t.Errorf("FLAC should be a valid format: %v", err)
	}

	err = cs.ValidateFormat("invalid")
	if err == nil {
		t.Error("Invalid format should fail validation")
	}

	// Test bitrate validation
	err = cs.ValidateBitrate("mp3", "320")
	if err != nil {
		t.Errorf("320 should be a valid bitrate for MP3: %v", err)
	}

	err = cs.ValidateBitrate("mp3", "999")
	if err == nil {
		t.Error("999 should be an invalid bitrate")
	}
}