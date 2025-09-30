package downloader

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

	// Create a metadata processor for testing
	processor := NewMetadataProcessor()
	
	ctx := context.Background()
	
	// Test track search
	track, err := processor.mbClient.SearchTrack(ctx, "Queen", "A Night at the Opera", "Bohemian Rhapsody")
	if err != nil {
		t.Logf("Track search failed (this might be expected): %v", err)
	} else {
		t.Logf("Found track: ID=%s, Title=%s", track.ID, track.Title)
		if len(track.ArtistCredit) > 0 {
			t.Logf("Artist: ID=%s, Name=%s", track.ArtistCredit[0].Artist.ID, track.ArtistCredit[0].Artist.Name)
		}
	}

	// Test release search
	release, err := processor.mbClient.SearchRelease(ctx, "Queen", "A Night at the Opera")
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
	isrcTrack, err := processor.mbClient.SearchTrackByISRC(ctx, "GBUM71505078") // Bohemian Rhapsody ISRC
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

func TestFindReleaseIDFromISRC(t *testing.T) {
	t.Log("Testing ISRC-based release ID lookup...")
	
	// Clear cache before test
	ClearAlbumCache()
	
	// Create tracks with ISRC
	tracks := []shared.Track{
		{
			ID:     "track1",
			Title:  "High by the Beach",
			Artist: "Lana Del Rey",
			ISRC:   "GBUM71505078", // Known ISRC
		},
		{
			ID:     "track2", 
			Title:  "Music to Watch Boys To",
			Artist: "Lana Del Rey",
			// No ISRC for this track
		},
	}
	
	albumArtist := "Lana Del Rey"
	albumTitle := "Honeymoon"
	
	// Test the function
	FindReleaseIDFromISRC(tracks, albumArtist, albumTitle)
	
	// Check if release ID was cached
	processor := NewMetadataProcessor()
	cachedReleaseID := processor.cache.GetCachedReleaseID(albumArtist, albumTitle)
	if cachedReleaseID != "" {
		t.Logf("Successfully found and cached release ID: %s", cachedReleaseID)
	} else {
		t.Log("No release ID found (this might be expected if ISRC lookup fails)")
	}
	
	// Test with tracks that have no ISRC
	tracksNoISRC := []shared.Track{
		{
			ID:     "track3",
			Title:  "Some Song",
			Artist: "Some Artist",
			// No ISRC
		},
	}
	
	processor.FindReleaseIDFromISRC(tracksNoISRC, "Some Artist", "Some Album")
	cachedReleaseID2 := processor.cache.GetCachedReleaseID("Some Artist", "Some Album")
	if cachedReleaseID2 == "" {
		t.Log("Correctly handled tracks with no ISRC - no release ID cached")
	} else {
		t.Errorf("Unexpected release ID cached for tracks without ISRC: %s", cachedReleaseID2)
	}
}

func TestGetISRCMetadata(t *testing.T) {
	t.Log("Testing efficient ISRC metadata extraction...")
	
	// Test with a known ISRC
	isrc := "GBUM71505078" // High by the Beach - Lana Del Rey
	
	metadata, err := GetISRCMetadata(isrc)
	if err != nil {
		t.Logf("ISRC metadata lookup failed (this might be expected): %v", err)
		return
	}
	
	t.Logf("Successfully extracted metadata from ISRC %s:", isrc)
	t.Logf("  Track ID: %s", metadata.TrackID)
	t.Logf("  Track Artist ID: %s", metadata.TrackArtistID)
	t.Logf("  Release ID: %s", metadata.ReleaseID)
	t.Logf("  Release Artist ID: %s", metadata.ReleaseArtistID)
	t.Logf("  Release Group ID: %s", metadata.ReleaseGroupID)
	
	// Verify we got all the essential IDs
	if metadata.TrackID == "" {
		t.Error("Track ID should not be empty")
	}
	if metadata.ReleaseID == "" {
		t.Error("Release ID should not be empty")
	}
	if metadata.ReleaseGroupID == "" {
		t.Error("Release Group ID should not be empty")
	}
	
	// Test with invalid ISRC
	_, err = GetISRCMetadata("INVALID_ISRC")
	if err == nil {
		t.Error("Expected error for invalid ISRC")
	} else {
		t.Logf("Correctly handled invalid ISRC: %v", err)
	}
}

func TestISRCMetadataWithTrackCountMatching(t *testing.T) {
	t.Log("Testing ISRC metadata extraction with track count matching...")
	
	// Test with a known ISRC that might appear on multiple releases
	isrc := "GBUM71505078" // High by the Beach - Lana Del Rey
	
	// Test with different expected track counts
	testCases := []struct {
		name               string
		expectedTrackCount int
		description        string
	}{
		{"Single", 1, "Should prefer single release"},
		{"EP", 5, "Should prefer EP release"},
		{"Album", 13, "Should prefer full album release"},
		{"No preference", 0, "Should use default selection"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metadata, err := GetISRCMetadataWithTrackCount(isrc, tc.expectedTrackCount)
			if err != nil {
				t.Logf("ISRC metadata lookup failed for %s (this might be expected): %v", tc.description, err)
				return
			}
			
			t.Logf("%s - Expected tracks: %d", tc.description, tc.expectedTrackCount)
			t.Logf("  Selected Release ID: %s", metadata.ReleaseID)
			t.Logf("  Release Group ID: %s", metadata.ReleaseGroupID)
			
			// Verify we got essential metadata
			if metadata.TrackID == "" {
				t.Error("Track ID should not be empty")
			}
			if metadata.ReleaseID == "" {
				t.Error("Release ID should not be empty")
			}
		})
	}
}

func TestReleaseSelectionDebug(t *testing.T) {
	t.Log("Testing release selection with debug information...")
	
	// Test with a known ISRC
	isrc := "GBUM71505078" // High by the Beach - Lana Del Rey
	
	// Get the raw MusicBrainz track data to see what releases are available
	processor := NewMetadataProcessor()
	ctx := context.Background()
	mbTrack, err := processor.mbClient.SearchTrackByISRC(ctx, isrc)
	if err != nil {
		t.Logf("ISRC search failed: %v", err)
		return
	}
	
	t.Logf("Found %d releases for ISRC %s:", len(mbTrack.Releases), isrc)
	for i, release := range mbTrack.Releases {
		totalTracks := 0
		for _, media := range release.Media {
			totalTracks += len(media.Tracks)
		}
		t.Logf("  Release %d: ID=%s, Title=%s, Tracks=%d", i+1, release.ID, release.Title, totalTracks)
		t.Logf("    Release Group ID: %s", release.ReleaseGroup.ID)
		t.Logf("    Date: %s", release.Date)
	}
	
	// Test the selection logic with different track counts
	if len(mbTrack.Releases) > 1 {
		t.Log("Testing selection logic with multiple releases...")
		
		// Test with track count that should match a specific release
		for expectedCount := 1; expectedCount <= 15; expectedCount++ {
			selected := processor.selectBestTrackRelease(mbTrack.Releases, expectedCount)
			totalTracks := 0
			for _, media := range selected.Media {
				totalTracks += len(media.Tracks)
			}
			if totalTracks == expectedCount {
				t.Logf("  Expected %d tracks: Found exact match with %d tracks (Release: %s)", 
					expectedCount, totalTracks, selected.ID)
			}
		}
	} else {
		t.Log("Only one release found - track count matching not applicable")
	}
}

func TestIntelligentReleaseSelection(t *testing.T) {
	t.Log("Testing intelligent release selection...")
	
	// Test with different expected track counts to see which releases get selected
	testCases := []struct {
		name               string
		expectedTrackCount int
		description        string
	}{
		{"Single", 1, "Should prefer single release"},
		{"EP", 5, "Should prefer EP release"},  
		{"Album", 13, "Should prefer full album release"},
		{"Large Album", 20, "Should prefer full album over compilations"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metadata, err := GetISRCMetadataWithTrackCount("GBUM71505078", tc.expectedTrackCount)
			if err != nil {
				t.Logf("ISRC metadata lookup failed: %v", err)
				return
			}
			
			t.Logf("%s (expected %d tracks):", tc.description, tc.expectedTrackCount)
			t.Logf("  Selected Release ID: %s", metadata.ReleaseID)
			t.Logf("  Release Group ID: %s", metadata.ReleaseGroupID)
			
			// Get the release title for context
			processor := NewMetadataProcessor()
			ctx := context.Background()
			mbTrack, err := processor.mbClient.SearchTrackByISRC(ctx, "GBUM71505078")
			if err == nil {
				for _, release := range mbTrack.Releases {
					if release.ID == metadata.ReleaseID {
						t.Logf("  Selected Release Title: %s", release.Title)
						t.Logf("  Release Date: %s", release.Date)
						break
					}
				}
			}
		})
	}
}

func TestDigitalMediaPreference(t *testing.T) {
	t.Log("Testing Digital Media format preference...")
	
	// Test with a known ISRC to see if Digital Media releases are preferred
	isrc := "GBUM71505078" // High by the Beach - Lana Del Rey
	
	// Get the raw MusicBrainz track data to see what formats are available
	processor := NewMetadataProcessor()
	ctx := context.Background()
	mbTrack, err := processor.mbClient.SearchTrackByISRC(ctx, isrc)
	if err != nil {
		t.Logf("ISRC search failed: %v", err)
		return
	}
	
	t.Logf("Found %d releases for ISRC %s:", len(mbTrack.Releases), isrc)
	digitalMediaFound := false
	physicalMediaFound := false
	
	for i, release := range mbTrack.Releases {
		for _, media := range release.Media {
			format := strings.ToLower(media.Format)
			t.Logf("  Release %d: ID=%s, Title=%s, Format=%s", i+1, release.ID, release.Title, media.Format)
			
			if format == "digital media" {
				digitalMediaFound = true
			}
			if format == "cd" || format == "vinyl" {
				physicalMediaFound = true
			}
		}
	}
	
	if digitalMediaFound {
		t.Log("Digital Media releases found - testing preference...")
		
		// Test with album-sized track count to see if Digital Media is preferred
		metadata, err := GetISRCMetadataWithTrackCount(isrc, 13)
		if err != nil {
			t.Logf("ISRC metadata lookup failed: %v", err)
			return
		}
		
		// Check if the selected release has Digital Media format
		for _, release := range mbTrack.Releases {
			if release.ID == metadata.ReleaseID {
				for _, media := range release.Media {
					if strings.ToLower(media.Format) == "digital media" {
						t.Logf("✅ Successfully selected Digital Media release: %s", release.Title)
						return
					}
				}
				t.Logf("Selected release format is not Digital Media: %s", release.Title)
				for _, media := range release.Media {
					t.Logf("  Format: %s", media.Format)
				}
				break
			}
		}
	} else {
		t.Log("No Digital Media releases found for this ISRC - test not applicable")
	}
	
	if !digitalMediaFound && !physicalMediaFound {
		t.Log("No format information available in releases")
	}
}