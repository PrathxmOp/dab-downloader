package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndSaveConfig(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configFile := filepath.Join(tempDir, "config.json")

	// Create a sample config
	expectedConfig := &Config{
		APIURL:           "https://example.com",
		DownloadLocation: "/tmp/downloads",
		Parallelism:      3,
		Format:           "mp3",
		Bitrate:          "192",
		MaxRetryAttempts: 5,
		WarningBehavior:  "silent",
	}

	// Test SaveConfig
	err = SaveConfig(configFile, expectedConfig)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Errorf("Config file was not created")
	}

	// Test LoadConfig
	loadedConfig := &Config{}
	err = LoadConfig(configFile, loadedConfig)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Compare loaded config with expected config
	if loadedConfig.APIURL != expectedConfig.APIURL {
		t.Errorf("APIURL mismatch: got %s, want %s", loadedConfig.APIURL, expectedConfig.APIURL)
	}
	if loadedConfig.DownloadLocation != expectedConfig.DownloadLocation {
		t.Errorf("DownloadLocation mismatch: got %s, want %s", loadedConfig.DownloadLocation, expectedConfig.DownloadLocation)
	}
	if loadedConfig.Parallelism != expectedConfig.Parallelism {
		t.Errorf("Parallelism mismatch: got %d, want %d", loadedConfig.Parallelism, expectedConfig.Parallelism)
	}
	if loadedConfig.Format != expectedConfig.Format {
		t.Errorf("Format mismatch: got %s, want %s", loadedConfig.Format, expectedConfig.Format)
	}
	if loadedConfig.Bitrate != expectedConfig.Bitrate {
		t.Errorf("Bitrate mismatch: got %s, want %s", loadedConfig.Bitrate, expectedConfig.Bitrate)
	}
	if loadedConfig.MaxRetryAttempts != expectedConfig.MaxRetryAttempts {
		t.Errorf("MaxRetryAttempts mismatch: got %d, want %d", loadedConfig.MaxRetryAttempts, expectedConfig.MaxRetryAttempts)
	}
	if loadedConfig.WarningBehavior != expectedConfig.WarningBehavior {
		t.Errorf("WarningBehavior mismatch: got %s, want %s", loadedConfig.WarningBehavior, expectedConfig.WarningBehavior)
	}
}
