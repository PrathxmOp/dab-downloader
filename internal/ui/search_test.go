package ui

import (
	"context"
	"dab-downloader/internal/models"
	"testing"
)

// MockSearcher implements the Searcher interface for testing
type MockSearcher struct {
	Results *models.SearchResults
	Error   error
}

func (m *MockSearcher) Search(ctx context.Context, query string, searchType string, limit int, debug bool) (*models.SearchResults, error) {
	return m.Results, m.Error
}

func TestHandleSearch_ISRC_Prioritization(t *testing.T) {
	// Setup mock results containing both an Artist and a Track
	mockResults := &models.SearchResults{
		Artists: []models.Artist{
			{Name: "Astrud Gilberto", ID: "artist_id_1"},
		},
		Tracks: []models.Track{
			{Title: "The Girl From Ipanema", Artist: "Astrud Gilberto", ID: "track_id_1"},
		},
	}

	mockSearcher := &MockSearcher{
		Results: mockResults,
		Error:   nil,
	}

	ctx := context.Background()
	query := "USPR36409921" // The ISRC from the issue

	// Test with auto=true
	selectedItems, itemTypes, err := HandleSearch(ctx, mockSearcher, query, "all", false, true)

	if err != nil {
		t.Fatalf("HandleSearch failed: %v", err)
	}

	if len(selectedItems) == 0 {
		t.Fatal("Expected results, got none")
	}

	// Logic check: The first item should be a Track, because the query looks like an ISRC
	firstType := itemTypes[0]
	if firstType != "track" {
		t.Errorf("Expected first result type to be 'track' for ISRC query, got '%s'", firstType)
	}

	track, ok := selectedItems[0].(models.Track)
	if !ok {
		t.Errorf("Expected item to be castable to models.Track")
	}
	if track.Title != "The Girl From Ipanema" {
		t.Errorf("Expected track title 'The Girl From Ipanema', got '%s'", track.Title)
	}
}

func TestHandleSearch_NormalQuery_Prioritization(t *testing.T) {
	// Setup mock results containing both an Artist and a Track
	mockResults := &models.SearchResults{
		Artists: []models.Artist{
			{Name: "Some Artist", ID: "artist_id_1"},
		},
		Tracks: []models.Track{
			{Title: "Some Song", Artist: "Some Artist", ID: "track_id_1"},
		},
	}

	mockSearcher := &MockSearcher{
		Results: mockResults,
		Error:   nil,
	}

	ctx := context.Background()
	query := "Some Artist" // Not an ISRC

	// Test with auto=true
	selectedItems, itemTypes, err := HandleSearch(ctx, mockSearcher, query, "all", false, true)

	if err != nil {
		t.Fatalf("HandleSearch failed: %v", err)
	}

	if len(selectedItems) == 0 {
		t.Fatal("Expected results, got none")
	}

	// Logic check: Standard logic prioritizes Artists > Albums > Tracks for general queries (as per existing code)
	// See ui/search.go lines 122+
	/*
		if len(results.Artists) > 0 {
			selectedItems = append(selectedItems, results.Artists[0])
			itemTypes = append(itemTypes, "artist")
		} else if ...
	*/

	firstType := itemTypes[0]
	if firstType != "artist" {
		t.Errorf("Expected first result type to be 'artist' for normal query, got '%s'", firstType)
	}
}
