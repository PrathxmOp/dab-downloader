package downloader

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cheggaaa/pb/v3"
	"golang.org/x/sync/semaphore"

	"dab-downloader/internal/config"
	"dab-downloader/internal/models"
	"dab-downloader/internal/ui"
	"dab-downloader/internal/utils"
)

// DownloadArtistDiscography downloads an artist's complete discography
func (d *Downloader) DownloadArtistDiscography(ctx context.Context, artistID string, conf *config.Config, debug bool, filter string, noConfirm bool, pool *pb.Pool) error {
	warningCollector := utils.NewWarningCollector(conf.WarningBehavior != "silent")
	
	artist, err := d.api.GetArtist(ctx, artistID, conf, debug)
	if err != nil {
		return fmt.Errorf("failed to get artist info: %w", err)
	}

	ui.Info.Printf("🎤 Found artist: %s\n", artist.Name)

	if len(artist.Albums) == 0 {
		ui.Warning.Println("⚠️ No albums found for this artist")
		return nil
	}

	albums, eps, singles, other := d.categorizeAlbums(artist.Albums)

	totalItems := len(albums) + len(eps) + len(singles) + len(other)
	ui.Info.Printf("📊 Found %d items:\n", totalItems)

	if len(albums) > 0 {
		ui.Info.Printf("   🎵 Albums: %d\n", len(albums))
	}
	if len(eps) > 0 {
		ui.Info.Printf("   🎶 EPs: %d\n", len(eps))
	}
	if len(singles) > 0 {
		ui.Info.Printf("   🎤 Singles: %d\n", len(singles))
	}
	if len(other) > 0 {
		ui.Info.Printf("   ❓ Others: %d\n", len(other))
	}

	itemsToDownload := []models.Album{}
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
		ui.Info.Println("\nWhat would you like to download?")
		fmt.Println("1) Everything (albums + EPs + singles)")
		fmt.Println("2) Only albums")
		fmt.Println("3) Only EPs")
		fmt.Println("4) Only singles")
		fmt.Println("5) Custom selection")

		choice := utils.GetUserInput("Choose option (1-5, or q to quit)", "1")

		if strings.ToLower(choice) == "q" {
			ui.Warning.Println("⚠️ Download cancelled by user.")
			return models.ErrDownloadCancelled
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
			itemsToDownload = d.getCustomSelection(albums, eps, singles, other)
			if itemsToDownload == nil {
				ui.Warning.Println("⚠️ Download cancelled by user.")
				return models.ErrDownloadCancelled
			}
		default:
			ui.Error.Println("❌ Invalid option, please try again.")
			return fmt.Errorf("invalid selection option")
		}
	}

	if len(itemsToDownload) == 0 {
		ui.Warning.Println("⚠️ No items selected for download.")
		return models.ErrNoItemsSelected
	}

	ui.Info.Printf("\n📋 Items to download (%d):\n", len(itemsToDownload))
	for i, item := range itemsToDownload {
		fmt.Printf("%d. [%s] %s (%s)\n", i+1, strings.ToUpper(item.Type), item.Title, item.ReleaseDate)
	}

	if !noConfirm {
		confirm := utils.GetYesNoInput("Proceed with download? (y/N)", "n")
		if !confirm {
			ui.Warning.Println("⚠️ Download cancelled.")
			return nil
		}
	}

	artistDir := filepath.Join(d.outputLocation, utils.SanitizeFileName(artist.Name))
	if err := utils.CreateDirIfNotExists(artistDir); err != nil {
		return fmt.Errorf("failed to create artist directory: %w", err)
	}

	var wg sync.WaitGroup
	sem := semaphore.NewWeighted(int64(conf.Parallelism))
	stats := &models.DownloadStats{}
	errorChan := make(chan struct {
		Title string
		Err   error
	}, len(itemsToDownload))
	
	var localPool bool
	if pool == nil && utils.IsTTY() {
		var poolErr error
		pool, poolErr = pb.StartPool()
		if poolErr != nil {
			ui.Error.Printf("❌ Failed to start progress bar pool: %v\n", poolErr)
		} else {
			localPool = true
		}
	}

	for idx, item := range itemsToDownload {
		wg.Add(1)
		if err := sem.Acquire(ctx, 1); err != nil {
			ui.Error.Printf("Failed to acquire semaphore: %v\n", err)
			wg.Done()
			continue
		}

		go func(idx int, item models.Album) {
			defer wg.Done()
			defer sem.Release(1)

			ui.Info.Printf("🎵 Downloading %s %d/%d: %s\n", strings.ToUpper(item.Type), idx+1, len(itemsToDownload), item.Title)
			itemStats, err := d.DownloadAlbum(ctx, item.ID, conf, debug, pool, warningCollector)
			if err != nil {
				errorChan <- struct {
					Title string
					Err   error
				}{
					Title: item.Title,
					Err:   fmt.Errorf("item %s: %w", item.Title, err),
				}
			} else {
				stats.SuccessCount += itemStats.SuccessCount
				stats.SkippedCount += itemStats.SkippedCount
				stats.FailedCount += itemStats.FailedCount
				stats.FailedItems = append(stats.FailedItems, itemStats.FailedItems...)
			}
		}(idx, item)
	}

	wg.Wait()
	if localPool && pool != nil {
		pool.Stop()
	}
	close(errorChan)

	for err := range errorChan {
		stats.FailedCount++
		stats.FailedItems = append(stats.FailedItems, fmt.Sprintf("%s: %v", err.Title, err.Err))
	}

	if conf.WarningBehavior == "summary" {
		warningCollector.PrintSummary()
	}
	
	d.printDownloadStats(artist.Name, stats)
	
	return nil
}

func (d *Downloader) printDownloadStats(artistName string, stats *models.DownloadStats) {
	ui.Info.Printf("\n📊 Download Summary for %s:\n", artistName)
	ui.Success.Printf("✅ Successfully downloaded: %d items\n", stats.SuccessCount)

	if stats.SkippedCount > 0 {
		ui.Warning.Printf("⭐ Skipped (already exist): %d items\n", stats.SkippedCount)
	}

	if len(stats.FailedItems) > 0 {
		ui.Error.Printf("❌ Failed to download: %d items\n", len(stats.FailedItems))
		for _, msg := range stats.FailedItems {
			ui.Error.Printf("   - %s\n", msg)
		}
	}

	ui.Success.Printf("🎉 Artist discography downloaded to: %s\n", filepath.Join(d.outputLocation, utils.SanitizeFileName(artistName)))
}

func (d *Downloader) getCustomSelection(albums, eps, singles, other []models.Album) []models.Album {
	items := append(append(append([]models.Album{}, albums...), eps...), singles...)
	items = append(items, other...)

	fmt.Println("Available items:")
	for i, item := range items {
		fmt.Printf("%d. [%s] %s (%s)\n", i+1, strings.ToUpper(item.Type), item.Title, item.ReleaseDate)
	}

	for {
		input := utils.GetUserInput("Enter selection (e.g., 1-5 | 1,5 | 1, or q to quit)", "none")
		if strings.ToLower(input) == "none" || strings.ToLower(input) == "q" {
			return nil
		}

		selected := d.parseSelection(input, items)
		if len(selected) > 0 {
			return selected
		}
		ui.Error.Printf("❌ Invalid selection. Please enter numbers between 1 and %d (e.g., 1-5, 1,5, 1).\n", len(items))
	}
}

func (d *Downloader) categorizeAlbums(allAlbums []models.Album) ([]models.Album, []models.Album, []models.Album, []models.Album) {
	uniqueAlbums := make(map[string]models.Album)
	for _, album := range allAlbums {
		key := fmt.Sprintf("%s|%s|%s", album.ID, album.Title, album.ReleaseDate)
	
uniqueAlbums[key] = album
	}

	albums := []models.Album{}
	eps := []models.Album{}
	singles := []models.Album{}
	other := []models.Album{}

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

func (d *Downloader) parseSelection(input string, allItems []models.Album) []models.Album {
	selected := []models.Album{}
	parts := strings.Split(input, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			start, err1 := utils.ParseSelectionInput(strings.TrimSpace(rangeParts[0]), len(allItems))
			end, err2 := utils.ParseSelectionInput(strings.TrimSpace(rangeParts[1]), len(allItems))
			
			if err1 == nil && err2 == nil && len(start) > 0 && len(end) > 0 {
				s := start[0]
				e := end[0]
				if s > 0 && e > 0 && s <= e && s <= len(allItems) && e <= len(allItems) {
					selected = append(selected, allItems[s-1:e]...)
				}
			}
		} else {
			nums, err := utils.ParseSelectionInput(part, len(allItems))
			if err == nil {
				for _, idx := range nums {
					selected = append(selected, allItems[idx-1])
				}
			}
		}
	}
	return selected
}
