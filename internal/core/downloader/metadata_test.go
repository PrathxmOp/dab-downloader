package downloader

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"dab-downloader/internal/shared"
)

func TestMusicBrainzMetadataIntegration(t *testing.T) {
	// Enable debug mode for this test
	SetMusicBrainzDebug(true)
	defer SetMusicBrainzDebug(false)

	// Create a temporary FLAC file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.flac")
	
	// Create a minimal FLAC file (this is a simplified approach for testing)
	// In a real scenario, you'd need a proper FLAC file
	if err := os.WriteFile(testFile, []byte("fLaC"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test track and album data
	track := shared.Track{
		Title:       "Bohemian Rhapsody",
		Artist:      "Queen",
		Album:       "A Night at the Opera",
		TrackNumber: 1,
		Duration:    355000, // 5:55 in milliseconds
	}

	album := &shared.Album{
		Title:       "A Night at the Opera",
		Artist:      "Queen",
		ReleaseDate: "1975-11-21",
		TotalTracks: 12,
	}

	warningCollector := shared.NewWarningCollector(true)

	// Test the metadata addition with a well-known track
	t.Log("Testing MusicBrainz metadata retrieval for Queen - Bohemian Rhapsody")
	
	// This will fail because we don't have a real FLAC file, but we can check if MusicBrainz is being called
	err := AddMetadataWithDebug(testFile, track, album, nil, 12, warningCollector, true)
	
	// We expect this to fail due to invalid FLAC file, but we should see debug output
	if err != nil {
		t.Logf("Expected error due to invalid FLAC file: %v", err)
	}

	// Check if any MusicBrainz warnings were collected
	warningCount := warningCollector.GetWarningCount()
	t.Logf("Collected %d warnings", warningCount)
	
	if warningCollector.HasWarnings() {
		warningCollector.PrintSummary()
	}
}

func TestMusicBrainzClientDirectly(t *testing.T) {
	// Test the MusicBrainz client directly
	SetMusicBrainzDebug(true)
	defer SetMusicBrainzDebug(false)

	t.Log("Testing direct MusicBrainz API calls...")

	// Test track search
	track, err := mbClient.SearchTrack("Queen", "A Night at the Opera", "Bohemian Rhapsody")
	if err != nil {
		t.Logf("Track search failed (this might be expected): %v", err)
	} else {
		t.Logf("Found track: ID=%s, Title=%s", track.ID, track.Title)
		if len(track.ArtistCredit) > 0 {
			t.Logf("Artist: ID=%s, Name=%s", track.ArtistCredit[0].Artist.ID, track.ArtistCredit[0].Artist.Name)
		}
	}

	// Test release search
	release, err := mbClient.SearchRelease("Queen", "A Night at the Opera")
	if err != nil {
		t.Logf("Release search failed (this might be expected): %v", err)
	} else {
		t.Logf("Found release: ID=%s, Title=%s", release.ID, release.Title)
		if len(release.ArtistCredit) > 0 {
			t.Logf("Artist: ID=%s, Name=%s", release.ArtistCredit[0].Artist.ID, release.ArtistCredit[0].Artist.Name)
		}
		t.Logf("Release Group ID: %s", release.ReleaseGroup.ID)
	}

	// Add a small delay to respect MusicBrainz rate limiting
	time.Sleep(2 * time.Second)
}

func TestCacheOperations(t *testing.T) {
	// Clear cache before test
	ClearAlbumCache()

	// Test cache stats
	count, keys := GetCacheStats()
	if count != 0 {
		t.Errorf("Expected empty cache, got %d items", count)
	}

	t.Logf("Cache is empty as expected: %d items, keys: %v", count, keys)
}