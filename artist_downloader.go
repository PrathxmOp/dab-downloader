package main

import (
	"context"
	"fmt"
	
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/cheggaaa/pb/v3"
	"golang.org/x/sync/semaphore"
)

// DownloadArtistDiscography downloads an artist's complete discography
func (api *DabAPI) DownloadArtistDiscography(ctx context.Context, artistID string, config *Config, debug bool, filter string, noConfirm bool) error {
	artist, err := api.GetArtist(ctx, artistID, config, debug)
	if err != nil {
		return fmt.Errorf("failed to get artist info: %w", err)
	}

	colorInfo.Printf("🎤 Found artist: %s\n", artist.Name)

	if len(artist.Albums) == 0 {
		colorWarning.Println("⚠️ No albums found for this artist")
		return nil
	}

	// Categorize albums by type
	albums, eps, singles, other := api.categorizeAlbums(artist.Albums)

	// Show categorized content
	totalItems := len(albums) + len(eps) + len(singles) + len(other)
	colorInfo.Printf("📊 Found %d items:\n", totalItems)

	if len(albums) > 0 {
		colorInfo.Printf("   🎵 Albums: %d\n", len(albums))
	}
	if len(eps) > 0 {
		colorInfo.Printf("   🎶 EPs: %d\n", len(eps))
	}
	if len(singles) > 0 {
		colorInfo.Printf("   🎤 Singles: %d\n", len(singles))
	}
	if len(other) > 0 {
		colorInfo.Printf("   ❓ Others: %d\n", len(other))
	}

	itemsToDownload := []Album{}
	if filter != "all" {
		filterParts := strings.Split(filter, ",")
		for _, part := range filterParts {
			switch strings.TrimSpace(part) {
			case "albums":
				itemsToDownload = append(itemsToDownload, albums...)
			case "eps":
				itemsToDownload = append(itemsToDownload, eps...)
			case "singles":
				itemsToDownload = append(itemsToDownload, singles...)
			}
		}
	} else {
		// Menu for download selection
		colorInfo.Println("\nWhat would you like to download?")
		fmt.Println("1) Everything (albums + EPs + singles)")
		fmt.Println("2) Only albums")
		fmt.Println("3) Only EPs")
		fmt.Println("4) Only singles")
		fmt.Println("5) Custom selection")

		choice := GetUserInput("Choose option (1-5, or q to quit)", "1")

		if strings.ToLower(choice) == "q" {
			colorWarning.Println("⚠️ Download cancelled by user.")
			return ErrDownloadCancelled
		}

		switch choice {
		case "1":
			itemsToDownload = append(itemsToDownload, albums...)
			itemsToDownload = append(itemsToDownload, eps...)
			itemsToDownload = append(itemsToDownload, singles...)
			itemsToDownload = append(itemsToDownload, other...)
		case "2":
			itemsToDownload = albums
		case "3":
			itemsToDownload = eps
		case "4":
			itemsToDownload = singles
		case "5":
			itemsToDownload = api.getCustomSelection(albums, eps, singles, other)
			if itemsToDownload == nil {
				colorWarning.Println("⚠️ Download cancelled by user.")
				return ErrDownloadCancelled
			}
		default:
			colorError.Println("❌ Invalid option, please try again.")
			return fmt.Errorf("invalid selection option")
		}
	}

	if len(itemsToDownload) == 0 {
		colorWarning.Println("⚠️ No items selected for download.")
		return ErrNoItemsSelected
	}

	colorInfo.Printf("\n📋 Items to download (%d):\n", len(itemsToDownload))
	for i, item := range itemsToDownload {
		fmt.Printf("%d. [%s] %s (%s)\n", i+1, strings.ToUpper(item.Type), item.Title, item.ReleaseDate)
	}

	// Confirm download
	if !noConfirm {
		confirm := GetYesNoInput("Proceed with download? (y/N)", "n")
		if !confirm {
			colorWarning.Println("⚠️ Download cancelled.")
			return nil
		}
	}

	// Setup for download
	artistDir := filepath.Join(api.outputLocation, SanitizeFileName(artist.Name))
	if err := CreateDirIfNotExists(artistDir); err != nil {
		return fmt.Errorf("failed to create artist directory: %w", err)
	}

	var wg sync.WaitGroup
	sem := semaphore.NewWeighted(int64(config.Parallelism))
	stats := &DownloadStats{}
	errorChan := make(chan trackError, len(itemsToDownload))
	var pool *pb.Pool
	if isTTY() {
		var poolErr error
		pool, poolErr = pb.StartPool()
		if poolErr != nil {
			colorError.Printf("❌ Failed to start progress bar pool: %v\n", poolErr)
			// Continue without the pool
		}
	} else {
		if debug {
			colorInfo.Println("DEBUG: isTTY() is false. Progress bars will not be displayed.")
		}
	}

	// Download each item

	for idx, item := range itemsToDownload {
		wg.Add(1)
		if err := sem.Acquire(ctx, 1); err != nil {
			colorError.Printf("Failed to acquire semaphore: %v\n", err)
			wg.Done()
			continue
		}

		go func(idx int, item Album) {
			defer wg.Done()
			defer sem.Release(1)

			colorInfo.Printf("🎵 Downloading %s %d/%d: %s\n", strings.ToUpper(item.Type), idx+1, len(itemsToDownload), item.Title)
			itemStats, err := api.DownloadAlbum(ctx, item.ID, config, debug, pool)
			if err != nil {
				errorChan <- trackError{item.Title, fmt.Errorf("item %s: %w", item.Title, err)}
			} else {
				stats.SuccessCount += itemStats.SuccessCount
				stats.SkippedCount += itemStats.SkippedCount
				stats.FailedCount += itemStats.FailedCount
				stats.FailedItems = append(stats.FailedItems, itemStats.FailedItems...)
			}
		}(idx, item)
	}

	// Wait for all downloads to finish
	wg.Wait()
	if pool != nil {
		pool.Stop()
	}
	close(errorChan)

	// Collect errors
	for err := range errorChan {
		stats.FailedCount++
		stats.FailedItems = append(stats.FailedItems, fmt.Sprintf("%s: %v", err.Title, err.Err))
	}

	// Print summary
	api.printDownloadStats(artist.Name, stats)
	return nil
}

// printDownloadStats prints the download statistics
func (api *DabAPI) printDownloadStats(artistName string, stats *DownloadStats) {
	colorInfo.Printf("\n📊 Download Summary for %s:\n", artistName)
	colorSuccess.Printf("✅ Successfully downloaded: %d items\n", stats.SuccessCount)

	if stats.SkippedCount > 0 {
		colorWarning.Printf("⭐ Skipped (already exist): %d items\n", stats.SkippedCount)
	}

	if len(stats.FailedItems) > 0 {
		colorError.Printf("❌ Failed to download: %d items\n", len(stats.FailedItems))
		for _, msg := range stats.FailedItems {
			colorError.Printf("   - %s\n", msg)
		}
	}

	colorSuccess.Printf("🎉 Artist discography downloaded to: %s\n", filepath.Join(api.outputLocation, SanitizeFileName(artistName)))
}

// getCustomSelection handles user's custom selection of albums/EPs/singles
func (api *DabAPI) getCustomSelection(albums, eps, singles, other []Album) []Album {
	items := append(append(append([]Album{}, albums...), eps...), singles...)
	items = append(items, other...)

	fmt.Println("Available items:")
	for i, item := range items {
		fmt.Printf("%d. [%s] %s (%s)\n", i+1, strings.ToUpper(item.Type), item.Title, item.ReleaseDate)
	}

	for {
		input := GetUserInput("Enter selection (e.g., 1-5 | 1,5 | 1, or q to quit)", "none")
		if strings.ToLower(input) == "none" || strings.ToLower(input) == "q" {
			return nil
		}

		selected := api.parseSelection(input, items)
		if len(selected) > 0 {
			return selected
		}
		colorError.Printf("❌ Invalid selection. Please enter numbers between 1 and %d (e.g., 1-5, 1,5, 1).\n", len(items))
	}
}

// categorizeAlbums categorizes albums by type and removes duplicates
func (api *DabAPI) categorizeAlbums(allAlbums []Album) ([]Album, []Album, []Album, []Album) {
	// Deduplicate albums based on ID, Title, and ReleaseDate
	uniqueAlbums := make(map[string]Album)
	for _, album := range allAlbums {
		key := fmt.Sprintf("%s|%s|%s", album.ID, album.Title, album.ReleaseDate)
		uniqueAlbums[key] = album
	}

	albums := []Album{}
	eps := []Album{}
	singles := []Album{}
	other := []Album{}

	for _, album := range uniqueAlbums {
		switch strings.ToLower(album.Type) {
			case "album":
				albums = append(albums, album)
			case "ep":
				eps = append(eps, album)
			case "single":
				singles = append(singles, album)
			default:
				other = append(other, album)
		}
	}
	return albums, eps, singles, other
}

// parseSelection parses user input for album selection
func (api *DabAPI) parseSelection(input string, allItems []Album) []Album {
	selected := []Album{}
	parts := strings.Split(input, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			start, err1 := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			end, err2 := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err1 == nil && err2 == nil && start > 0 && end > 0 && start <= end && start <= len(allItems) && end <= len(allItems) {
				selected = append(selected, allItems[start-1:end]...)
			}
		} else {
			idx, err := strconv.Atoi(part)
			if err == nil && idx > 0 && idx <= len(allItems) {
				selected = append(selected, allItems[idx-1])
			}
		}
	}
	return selected
}