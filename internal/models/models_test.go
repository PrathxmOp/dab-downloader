package models

import (
	"encoding/json"
	"testing"
)

func TestTrackUnmarshalling(t *testing.T) {
	jsonData := `{
		"id": 12345,
		"title": "Test Track",
		"artist": "Test Artist",
		"album": "Test Album",
		"duration": 300,
		"trackNumber": 1
	}`

	var track Track
	err := json.Unmarshal([]byte(jsonData), &track)
	if err != nil {
		t.Fatalf("Failed to unmarshal Track: %v", err)
	}

	if track.Title != "Test Track" {
		t.Errorf("Title mismatch: got %s, want %s", track.Title, "Test Track")
	}
	// JSON numbers are floats by default in interface{}, but we expect int in struct
	// if ID is interface{}. Wait, in models.go Track.ID is interface{}.
	// Let's check how it comes out.
	// Actually, json.Unmarshal into interface{} makes numbers float64.
	// But here we unmarshal into Track struct.
	// Track.ID is interface{}.
	
	// Check other fields
	if track.Artist != "Test Artist" {
		t.Errorf("Artist mismatch: got %s, want %s", track.Artist, "Test Artist")
	}
	if track.Duration != 300 {
		t.Errorf("Duration mismatch: got %d, want %d", track.Duration, 300)
	}
}

func TestArtistUnmarshalling(t *testing.T) {
	jsonData := `{
		"id": "artist_123",
		"name": "The Testers",
		"albums": [
			{"id": "album_1", "title": "First Album"}
		]
	}`

	var artist Artist
	err := json.Unmarshal([]byte(jsonData), &artist)
	if err != nil {
		t.Fatalf("Failed to unmarshal Artist: %v", err)
	}

	if artist.Name != "The Testers" {
		t.Errorf("Name mismatch: got %s, want %s", artist.Name, "The Testers")
	}
	if len(artist.Albums) != 1 {
		t.Errorf("Albums count mismatch: got %d, want 1", len(artist.Albums))
	}
	if artist.Albums[0].Title != "First Album" {
		t.Errorf("Album title mismatch: got %s, want %s", artist.Albums[0].Title, "First Album")
	}
}
