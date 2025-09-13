package main

import (
	"context"
	"fmt"
)

func handleSearch(ctx context.Context, api *DabAPI, query string, searchType string, debug bool, auto bool) ([]interface{}, []string, error) {
	colorInfo.Printf("🔎 Searching for '%s' (type: %s)...", query, searchType)

	results, err := api.Search(ctx, query, searchType, 10)
	if err != nil {
		return nil, nil, err
	}

	totalResults := len(results.Artists) + len(results.Albums) + len(results.Tracks)
	if totalResults == 0 {
		colorWarning.Println("No results found.")
		return nil, nil, nil
	}

	if auto {
		var selectedItems []interface{}
		var itemTypes []string
		if len(results.Artists) > 0 {
			selectedItems = append(selectedItems, results.Artists[0])
			itemTypes = append(itemTypes, "artist")
		} else if len(results.Albums) > 0 {
			selectedItems = append(selectedItems, results.Albums[0])
			itemTypes = append(itemTypes, "album")
		} else if len(results.Tracks) > 0 {
			selectedItems = append(selectedItems, results.Tracks[0])
			itemTypes = append(itemTypes, "track")
		}
		return selectedItems, itemTypes, nil
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
	selectionStr := GetUserInput("\nEnter numbers to download (e.g., '1,3,5-7' or 'q' to quit)", "")
	if selectionStr == "q" || selectionStr == "" {
		return nil, nil, nil
	}

	selectedIndices, err := ParseSelectionInput(selectionStr, totalResults)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid selection: %w", err)
	}

	var selectedItems []interface{}
	var itemTypes []string

	for _, selectedIndex := range selectedIndices {
		index := selectedIndex - 1
		if index < len(results.Artists) {
			selectedItems = append(selectedItems, results.Artists[index])
			itemTypes = append(itemTypes, "artist")
		} else {
			index -= len(results.Artists)
			if index < len(results.Albums) {
				selectedItems = append(selectedItems, results.Albums[index])
				itemTypes = append(itemTypes, "album")
			} else {
				index -= len(results.Albums)
				if index < len(results.Tracks) {
					selectedItems = append(selectedItems, results.Tracks[index])
					itemTypes = append(itemTypes, "track")
				} else {
					return nil, nil, fmt.Errorf("invalid index %d after parsing", selectedIndex)
				}
			}
		}
	}

	return selectedItems, itemTypes, nil
}