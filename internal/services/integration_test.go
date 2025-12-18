package services

import (
	"context"
	"net/http"
	"testing"
	"time"

	"dab-downloader/internal/config"
	"dab-downloader/internal/interfaces"
)

func TestServiceIntegration(t *testing.T) {
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

	// Create service container
	container := NewServiceContainer(cfg, httpClient)

	// Test that services can interact with each other
	_ = context.Background()

	// Test config service
	defaultConfig := container.Config.GetDefaultConfig()
	err := container.Config.ValidateConfig(defaultConfig)
	if err != nil {
		t.Errorf("Default config validation failed: %v", err)
	}

	// Test file system service
	testPath := container.FileSystem.GetDownloadPath("TestArtist", "TestAlbum", "TestTrack", "flac", cfg)
	if testPath == "" {
		t.Error("File system service should generate valid paths")
	}

	// Test logger service
	container.Logger.SetDebugMode(true)
	container.Logger.Info("Test integration message")
	container.Logger.Debug("Test debug message")

	// Test warning collector
	if container.WarningCollector.HasWarnings() {
		t.Error("New warning collector should have no warnings")
	}

	// Test conversion service
	formats := container.Conversion.GetSupportedFormats()
	if len(formats) == 0 {
		t.Error("Conversion service should have supported formats")
	}

	// Test that services are properly wired
	if container.DownloadService == nil {
		t.Error("Download service should be initialized")
	}
	if container.SearchService == nil {
		t.Error("Search service should be initialized")
	}

	// Test API client interface (basic check)
	if container.APIClient == nil {
		t.Error("API client should be initialized")
	}

	// Test that all services implement their interfaces correctly
	// This is mostly a compile-time check, but we can verify at runtime too
	var _ interfaces.ConfigService = container.Config
	var _ interfaces.FileSystemService = container.FileSystem
	var _ interfaces.LoggerService = container.Logger
	var _ interfaces.WarningCollectorService = container.WarningCollector
	var _ interfaces.ConversionService = container.Conversion
	var _ interfaces.DownloadService = container.DownloadService
	var _ interfaces.SearchService = container.SearchService
	var _ interfaces.APIClient = container.APIClient
	var _ interfaces.SpotifyService = container.SpotifyService
	var _ interfaces.NavidromeService = container.NavidromeService
	var _ interfaces.UpdaterService = container.UpdaterService
	var _ interfaces.MetadataService = container.Metadata
}

func TestDependencyInjection(t *testing.T) {
	// Test that services can be created with different configurations
	cfg1 := &config.Config{
		APIURL:           "https://api1.test.com",
		DownloadLocation: "./downloads1",
		Parallelism:      2,
	}

	cfg2 := &config.Config{
		APIURL:           "https://api2.test.com",
		DownloadLocation: "./downloads2",
		Parallelism:      5,
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}

	container1 := NewServiceContainer(cfg1, httpClient)
	container2 := NewServiceContainer(cfg2, httpClient)

	// Verify that different containers have different configurations
	path1 := container1.FileSystem.GetDownloadPath("Artist", "Album", "Track", "flac", cfg1)
	path2 := container2.FileSystem.GetDownloadPath("Artist", "Album", "Track", "flac", cfg2)

	if path1 == path2 {
		t.Error("Different service containers should generate different paths based on their configs")
	}

	// Verify that services are independent
	container1.Logger.SetDebugMode(true)
	container2.Logger.SetDebugMode(false)

	// Both containers should work independently
	container1.Logger.Debug("Container 1 debug message")
	container2.Logger.Info("Container 2 info message")
}