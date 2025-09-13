package main

import (
	"context"
	"fmt"
	"strconv"
)

func handleSearch(ctx context.Context, api *DabAPI, query string, searchType string, debug bool) (interface{}, string, error) {
	colorInfo.Printf("🔎 Searching for '%s' (type: %s)...", query, searchType)

	results, err := api.Search(ctx, query, searchType, 10)
	if err != nil {
		return nil, "", err
	}

	totalResults := len(results.Artists) + len(results.Albums) + len(results.Tracks)
	if totalResults == 0 {
		colorWarning.Println("No results found.")
		return nil, "", nil
	}

	colorInfo.Printf("Found %d results:\n", totalResults)

	// Display results
	counter := 1
	if len(results.Artists) > 0 {
		colorInfo.Println("\n--- Artists ---")
		for _, artist := range results.Artists {
			fmt.Printf("%d. %s\n", counter, artist.Name)
			counter++
		}
	}
	if len(results.Albums) > 0 {
		colorInfo.Println("\n--- Albums ---")
		for _, album := range results.Albums {
			fmt.Printf("%d. %s - %s\n", counter, album.Title, album.Artist)
			counter++
		}
	}
	if len(results.Tracks) > 0 {
		colorInfo.Println("\n--- Tracks ---")
		for _, track := range results.Tracks {
			fmt.Printf("%d. %s - %s (%s)\n", counter, track.Title, track.Artist, track.Album)
			counter++
		}
	}

	// Prompt for selection
	selectionStr := GetUserInput("\nEnter number to download (or 'q' to quit)", "")
	if selectionStr == "q" || selectionStr == "" {
		return nil, "", nil
	}

	selection, err := strconv.Atoi(selectionStr)
	if err != nil || selection < 1 || selection >= counter {
		return nil, "", fmt.Errorf("invalid selection")
	}

	// Return selected item
	selectedIndex := selection - 1
	if selectedIndex < len(results.Artists) {
		artist := results.Artists[selectedIndex]
		return artist, "artist", nil
	} else {
		selectedIndex -= len(results.Artists)
	}

	if selectedIndex < len(results.Albums) {
		album := results.Albums[selectedIndex]
		return album, "album", nil
	} else {
		selectedIndex -= len(results.Albums)
	}

	if selectedIndex < len(results.Tracks) {
		track := results.Tracks[selectedIndex]
		return track, "track", nil
	}

	return nil, "", fmt.Errorf("invalid selection")
}
