package musicbrainz

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ExampleUsage demonstrates how to use the optimized MusicBrainz client
func ExampleUsage() {
	// Create a client with default configuration
	client := NewClient()
	
	// Or create with custom configuration
	customConfig := Config{
		BaseURL:      defaultBaseURL,
		UserAgent:    "my-app/1.0 (contact@example.com)",
		Timeout:      15 * time.Second,
		MaxRetries:   3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		RateLimit:    500 * time.Millisecond,
		BurstLimit:   5,
		Debug:        true,
	}
	customClient := NewClientWithConfig(customConfig)
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Search for a track by ISRC
	track, err := client.SearchTrackByISRC(ctx, "GBUM71505078")
	if err != nil {
		log.Printf("ISRC search failed: %v", err)
	} else {
		fmt.Printf("Found track: %s by %s\n", track.Title, track.ArtistCredit[0].Artist.Name)
	}
	
	// Search for a track by metadata
	track, err = client.SearchTrack(ctx, "Queen", "A Night at the Opera", "Bohemian Rhapsody")
	if err != nil {
		log.Printf("Track search failed: %v", err)
	} else {
		fmt.Printf("Found track: %s (ID: %s)\n", track.Title, track.ID)
	}
	
	// Search for a release
	release, err := client.SearchRelease(ctx, "Queen", "A Night at the Opera")
	if err != nil {
		log.Printf("Release search failed: %v", err)
	} else {
		fmt.Printf("Found release: %s (ID: %s)\n", release.Title, release.ID)
	}
	
	// Get detailed metadata by MBID
	if track != nil {
		detailedTrack, err := client.GetTrackMetadata(ctx, track.ID)
		if err != nil {
			log.Printf("Failed to get detailed track metadata: %v", err)
		} else {
			fmt.Printf("Detailed track: %s, Duration: %d ms\n", detailedTrack.Title, detailedTrack.Length)
		}
	}
	
	// Update client configuration
	newConfig := customClient.GetConfig()
	newConfig.Debug = false
	customClient.UpdateConfig(newConfig)
	
	// Enable debug mode
	client.SetDebug(true)
}