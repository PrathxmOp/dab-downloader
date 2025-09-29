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

	// Test ISRC search with a known ISRC
	t.Log("Testing ISRC-based search...")
	isrcTrack, err := mbClient.SearchTrackByISRC("GBUM71505078") // Bohemian Rhapsody ISRC
	if err != nil {
		t.Logf("ISRC search failed (this might be expected): %v", err)
	} else {
		t.Logf("Found track by ISRC: ID=%s, Title=%s", isrcTrack.ID, isrcTrack.Title)
		if len(isrcTrack.ArtistCredit) > 0 {
			t.Logf("Artist: ID=%s, Name=%s", isrcTrack.ArtistCredit[0].Artist.ID, isrcTrack.ArtistCredit[0].Artist.Name)
		}
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

func TestISRCPrioritySearch(t *testing.T) {
	// Enable debug mode for this test
	SetMusicBrainzDebug(true)
	defer SetMusicBrainzDebug(false)

	t.Log("Testing ISRC priority in metadata search...")

	// Test track with ISRC - should use ISRC search first
	trackWithISRC := shared.Track{
		Title:       "Bohemian Rhapsody",
		Artist:      "Queen",
		Album:       "A Night at the Opera",
		ISRC:        "GBUM71505078", // Known ISRC for Bohemian Rhapsody
		TrackNumber: 1,
		Duration:    355000,
	}

	// Test track without ISRC - should use traditional search
	trackWithoutISRC := shared.Track{
		Title:       "Bohemian Rhapsody",
		Artist:      "Queen",
		Album:       "A Night at the Opera",
		TrackNumber: 1,
		Duration:    355000,
	}

	album := &shared.Album{
		Title:       "A Night at the Opera",
		Artist:      "Queen",
		ReleaseDate: "1975-11-21",
		TotalTracks: 12,
	}

	warningCollector := shared.NewWarningCollector(true)

	// Create temporary FLAC files for testing
	tempDir := t.TempDir()
	
	testFileWithISRC := filepath.Join(tempDir, "test_with_isrc.flac")
	testFileWithoutISRC := filepath.Join(tempDir, "test_without_isrc.flac")
	
	// Create minimal FLAC files
	if err := os.WriteFile(testFileWithISRC, []byte("fLaC"), 0644); err != nil {
		t.Fatalf("Failed to create test file with ISRC: %v", err)
	}
	if err := os.WriteFile(testFileWithoutISRC, []byte("fLaC"), 0644); err != nil {
		t.Fatalf("Failed to create test file without ISRC: %v", err)
	}

	// Test with ISRC
	t.Log("Testing metadata addition with ISRC...")
	err := AddMetadataWithDebug(testFileWithISRC, trackWithISRC, album, nil, 12, warningCollector, true)
	if err != nil {
		t.Logf("Expected error due to invalid FLAC file (with ISRC): %v", err)
	}

	// Test without ISRC
	t.Log("Testing metadata addition without ISRC...")
	err = AddMetadataWithDebug(testFileWithoutISRC, trackWithoutISRC, album, nil, 12, warningCollector, true)
	if err != nil {
		t.Logf("Expected error due to invalid FLAC file (without ISRC): %v", err)
	}

	// Check warnings
	warningCount := warningCollector.GetWarningCount()
	t.Logf("Total warnings collected: %d", warningCount)
	
	if warningCollector.HasWarnings() {
		warningCollector.PrintSummary()
	}
}