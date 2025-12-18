package musicbrainz

import (
	"context"
	"testing"
	"time"
)

// 6. Debug/test functions (separate file)

// CreateTestClient creates a client configured for testing
func CreateTestClient() *Client {
	config := DefaultConfig()
	config.Debug = true
	config.Timeout = 10 * time.Second
	config.MaxRetries = 2
	return NewClientWithConfig(config)
}

// CreateMockClient creates a client with mock configuration for unit tests
func CreateMockClient() *Client {
	config := Config{
		BaseURL:      "http://localhost:8080/ws/2/",
		UserAgent:    "test-client/1.0",
		Timeout:      5 * time.Second,
		MaxRetries:   1,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		RateLimit:    10 * time.Millisecond,
		BurstLimit:   10,
		Debug:        true,
	}
	return NewClientWithConfig(config)
}

// Example test functions
func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	
	config := client.GetConfig()
	if config.BaseURL != defaultBaseURL {
		t.Errorf("Expected BaseURL %s, got %s", defaultBaseURL, config.BaseURL)
	}
}

func TestClientConfiguration(t *testing.T) {
	customConfig := Config{
		BaseURL:      "https://test.musicbrainz.org/ws/2/",
		UserAgent:    "test-agent/1.0",
		Timeout:      15 * time.Second,
		MaxRetries:   3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		RateLimit:    500 * time.Millisecond,
		BurstLimit:   3,
		Debug:        true,
	}
	
	client := NewClientWithConfig(customConfig)
	retrievedConfig := client.GetConfig()
	
	if retrievedConfig.BaseURL != customConfig.BaseURL {
		t.Errorf("Expected BaseURL %s, got %s", customConfig.BaseURL, retrievedConfig.BaseURL)
	}
	
	if retrievedConfig.Debug != customConfig.Debug {
		t.Errorf("Expected Debug %v, got %v", customConfig.Debug, retrievedConfig.Debug)
	}
}

// Benchmark functions for performance testing
func BenchmarkClientCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewClient()
	}
}

// Integration test helper
func TestIntegrationSearchTrack(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	client := CreateTestClient()
	ctx := context.Background()
	
	// Test with a well-known track
	track, err := client.SearchTrack(ctx, "The Beatles", "Abbey Road", "Come Together")
	if err != nil {
		t.Fatalf("SearchTrack failed: %v", err)
	}
	
	if track.Title == "" {
		t.Error("Expected track title to be non-empty")
	}
	
	if len(track.ArtistCredit) == 0 {
		t.Error("Expected at least one artist credit")
	}
}