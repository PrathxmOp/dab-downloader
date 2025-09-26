package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/cheggaaa/pb/v3"
	"golang.org/x/sync/semaphore"
)

// DownloadTrack downloads a single track with metadata
func (api *DabAPI) DownloadTrack(ctx context.Context, track Track, album *Album, outputPath string, coverData []byte, bar *pb.ProgressBar, debug bool, format string, bitrate string, config *Config) (string, error) {
	// Get stream URL
	streamURL, err := api.GetStreamURL(ctx, idToString(track.ID))
	if err != nil {
		return "", fmt.Errorf("failed to get stream URL: %w", err)
	}

	var expectedFileSize int64 // Store expected size for final verification

	// Determine retry attempts
	maxRetries := defaultMaxRetries
	if config != nil && config.MaxRetryAttempts > 0 {
		maxRetries = config.MaxRetryAttempts
	}

	// Download the audio file
	err = RetryWithBackoff(maxRetries, 5, func() error {
		audioResp, err := api.Request(ctx, streamURL, false, nil)
		if err != nil {
			return fmt.Errorf("failed to download audio: %w", err)
		}
		defer audioResp.Body.Close()

		expectedSize := audioResp.ContentLength
		expectedFileSize = expectedSize // Store for final verification
		if debug && expectedSize > 0 {
			fmt.Printf("DEBUG: Expected file size for %s: %d bytes\n", track.Title, expectedSize)
		}

		// Wrap the response body in the progress bar reader
		if bar != nil {
			if debug {
				fmt.Println("DEBUG: Starting progress bar for", track.Title)
			}
			if audioResp.ContentLength <= 0 {
				bar.Set("indeterminate", true) // Force spinner for unknown size
			} else {
				bar.SetTotal(audioResp.ContentLength)
			}
			audioResp.Body = bar.NewProxyReader(audioResp.Body)
		}

		// Create directory if needed
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Create and write to the output file
		out, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer out.Close()

		bytesWritten, err := io.Copy(out, audioResp.Body)
		if err != nil {
			// Clean up the file on error to prevent partial files
			os.Remove(outputPath)
			return fmt.Errorf("failed to write audio file: %w", err)
		}

		// Verify file size if ContentLength is available
		if expectedSize > 0 && bytesWritten != expectedSize {
			// Clean up the incomplete file
			os.Remove(outputPath)
			if debug {
				fmt.Printf("DEBUG: File size mismatch for %s - expected: %d, got: %d bytes\n", 
					track.Title, expectedSize, bytesWritten)
			}
			return fmt.Errorf("incomplete download: expected %d bytes, got %d bytes", expectedSize, bytesWritten)
		}

		if debug && expectedSize > 0 {
			fmt.Printf("DEBUG: Successfully downloaded %s - %d bytes verified\n", track.Title, bytesWritten)
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	// Final verification: check if the file exists and has the correct size
	// This catches any issues that might occur after the download completes
	if FileExists(outputPath) {
		// Only verify if verification is enabled (default true if not specified)
		verifyEnabled := config == nil || config.VerifyDownloads // Default to true
		if verifyEnabled && expectedFileSize > 0 {
			if verifyErr := VerifyFileIntegrity(outputPath, expectedFileSize, debug); verifyErr != nil {
				// Remove the corrupted file and return error
				os.Remove(outputPath)
				return "", fmt.Errorf("post-download verification failed: %w", verifyErr)
			}
		}
	} else {
		return "", fmt.Errorf("download completed but file not found on disk: %s", outputPath)
	}

	// Add metadata to the downloaded file
	err = AddMetadata(outputPath, track, album, coverData, len(album.Tracks))
	if err != nil {
		return "", fmt.Errorf("failed to add metadata: %w", err)
	}

	finalPath := outputPath
	if format != "flac" {
		colorInfo.Printf("🎵 Compressing to %s with bitrate %s kbps...\n", format, bitrate)
		convertedFile, err := ConvertTrack(outputPath, format, bitrate)
		if err != nil {
			return "", fmt.Errorf("failed to convert track: %w", err)
		}
		// Conversion successful, remove original FLAC file
		if err := os.Remove(outputPath); err != nil {
			colorWarning.Printf("⚠️ Failed to remove original FLAC file: %v\n", err)
		}
		finalPath = convertedFile
		if debug {
			colorInfo.Printf("✅ Successfully converted to %s: %s\n", format, convertedFile)
		}
	}

	return finalPath, nil
}

// DownloadSingleTrack downloads a single track.
// It now accepts a full Track object, assuming it comes from search results.
func (api *DabAPI) DownloadSingleTrack(ctx context.Context, track Track, debug bool, format string, bitrate string, pool *pb.Pool, config *Config) error {
	colorInfo.Printf("🎶 Preparing to download track: %s by %s (Album ID: %s)...\n", track.Title, track.Artist, track.AlbumID)

	// Fetch the album information using the track's AlbumID
	album, err := api.GetAlbum(ctx, track.AlbumID)
	if err != nil {
		colorWarning.Printf("⚠️ Could not fetch album info for track %s (ID: %s): %v. Attempting to proceed with limited album info.\n", track.Title, idToString(track.ID), err)
		// Create a minimal album object if fetching fails, to allow metadata to be added
		album = &Album{Title: track.Album, Artist: track.Artist, Tracks: []Track{track}}
	}

	// Find the specific track within the fetched album's tracks.
	// This is important because the 'track' object passed in might not have all details
	// that the full album fetch provides (e.g., full cover URL, stream URL details).
	var albumTrack *Track
	for i := range album.Tracks {
		if idToString(album.Tracks[i].ID) == idToString(track.ID) {
			albumTrack = &album.Tracks[i]
			break
		}
	}

	if albumTrack == nil {
		return fmt.Errorf("failed to find track %s (ID: %s) within its album %s (ID: %s)", track.Title, idToString(track.ID), album.Title, album.ID)
	}

	// Download cover
	var coverData []byte
	if album.Cover != "" {
		coverData, err = api.DownloadCover(ctx, album.Cover)
		if err != nil {
			colorWarning.Printf("⚠️ Could not download cover art for album %s: %v\n", album.Title, err)
		}
	}

	// Create track path
	artistDir := filepath.Join(api.outputLocation, SanitizeFileName(albumTrack.Artist))
	albumDir := filepath.Join(artistDir, SanitizeFileName(album.Title))
	trackFileName := GetTrackFilename(albumTrack.TrackNumber, albumTrack.Title)
	trackPath := filepath.Join(albumDir, trackFileName)

	// Skip if already exists
	if FileExists(trackPath) {
		colorWarning.Printf("⭐ Track already exists: %s\n", trackPath)
		return nil
	}

	// Create progress bar
	var bar *pb.ProgressBar
	if pool != nil { // Use pool if provided
		bar = pb.New(0)
		bar.SetTemplateString(`{{ string . "prefix" }} {{ bar . }} {{ percent . }} | {{ speed . "%s/s" }} | ETA {{ rtime . "%s" }}`)
		bar.Set("prefix", fmt.Sprintf("Downloading %-40s: ", TruncateString(albumTrack.Title, 40)))
		if debug {
			fmt.Println("DEBUG: Creating single track progress bar for", albumTrack.Title)
		}
		pool.Add(bar) // Add to pool
	} else if isTTY() { // Fallback to single bar if no pool and is TTY
		bar = pb.New(0)
		bar.SetWriter(os.Stdout)
		bar.SetTemplateString(`{{ string . "prefix" }} {{ bar . }} {{ percent . }} | {{ speed . "%s/s" }} | ETA {{ rtime . "%s" }}`)
		bar.Set("prefix", fmt.Sprintf("Downloading %-40s: ", TruncateString(albumTrack.Title, 40)))
		if debug {
			fmt.Println("DEBUG: Creating single track progress bar for", albumTrack.Title)
		}
		bar.Start()
	}

	// Download the track
	finalPath, err := api.DownloadTrack(ctx, *albumTrack, album, trackPath, coverData, bar, debug, format, bitrate, config)
	if err != nil {
		if bar != nil && pool == nil { // Only finish if it's a standalone bar
			bar.Finish()
		}
		return err
	}
	if bar != nil && pool == nil { // Only finish if it's a standalone bar
		bar.Finish()
	}

	colorSuccess.Printf("✅ Successfully downloaded: %s\n", finalPath)
	return nil
}


// DownloadAlbum downloads all tracks from an album
func (api *DabAPI) DownloadAlbum(ctx context.Context, albumID string, config *Config, debug bool, pool *pb.Pool) (*DownloadStats, error) {
	album, err := api.GetAlbum(ctx, albumID)
	if err != nil {
		return nil, fmt.Errorf("failed to get album info: %w", err)
	}

	artistDir := filepath.Join(api.outputLocation, SanitizeFileName(album.Artist))
	albumDir := filepath.Join(artistDir, SanitizeFileName(album.Title))

	if err := os.MkdirAll(albumDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create album directory: %w", err)
	}

	// Download cover
	var coverData []byte
	if album.Cover != "" {
		coverData, err = api.DownloadCover(ctx, album.Cover)
		if err != nil {
			colorWarning.Printf("⚠️ Could not download cover art for album %s: %v\n", album.Title, err)
		}
	}

	if config.SaveAlbumArt && coverData != nil {
		coverPath := filepath.Join(albumDir, "cover.jpg")
		if err := os.WriteFile(coverPath, coverData, 0644); err != nil {
			colorWarning.Printf("⚠️ Failed to save cover art for album %s: %v\n", album.Title, err)
		}
	}

	// Setup for concurrent downloads
	var wg sync.WaitGroup
	sem := semaphore.NewWeighted(int64(config.Parallelism))
	stats := &DownloadStats{}
	errorChan := make(chan trackError, len(album.Tracks))

	var localPool bool
	if pool == nil && isTTY() {
		var err error
		pool, err = pb.StartPool()
		if err != nil {
			colorError.Printf("❌ Failed to start progress bar pool: %v\n", err)
			// Continue without the pool
		} else {
			localPool = true
		}
	}

	// Create all progress bars first
	bars := make([]*pb.ProgressBar, len(album.Tracks))
	if pool != nil {
		for i, track := range album.Tracks {
			trackNumber := track.TrackNumber
			if trackNumber == 0 {
				trackNumber = i + 1
			}
			bar := pb.New(0)
			bar.SetTemplateString(`{{ string . "prefix" }} {{ bar . }} {{ percent . }} | {{ speed . "%s/s" }} | ETA {{ rtime . "%s" }}`)
			bar.Set("prefix", fmt.Sprintf("Track %-2d: %-40s", trackNumber, TruncateString(track.Title, 40)))
			bars[i] = bar
			pool.Add(bar)
		}
	}

	// Loop through tracks and start a goroutine for each download
	for idx, track := range album.Tracks {
		wg.Add(1)
		if err := sem.Acquire(ctx, 1); err != nil {
			colorError.Printf("Failed to acquire semaphore: %v\n", err)
			wg.Done()
			continue
		}

		go func(idx int, track Track) {
			defer wg.Done()
			defer sem.Release(1)

			trackNumber := track.TrackNumber
			if trackNumber == 0 {
				trackNumber = idx + 1
			}

			trackFileName := fmt.Sprintf("%02d - %s.flac", trackNumber, SanitizeFileName(track.Title))
			trackPath := filepath.Join(albumDir, trackFileName)

			// Skip if already exists
			if FileExists(trackPath) {
				colorWarning.Printf("⭐ Track already exists: %s\n", trackPath)
				stats.SkippedCount++
				return
			}

			var bar *pb.ProgressBar
			if pool != nil {
				bar = bars[idx]
			}

			if _, err := api.DownloadTrack(ctx, track, album, trackPath, coverData, bar, debug, config.Format, config.Bitrate, config); err != nil {
				errorChan <- trackError{track.Title, fmt.Errorf("track %s: %w", track.Title, err)}
				return
			}

			stats.SuccessCount++

		}(idx, track)
	}

	// Wait for all downloads to finish
	wg.Wait()
	if localPool && pool != nil {
		pool.Stop()
	}
	close(errorChan)

	// Collect errors
	for err := range errorChan {
		stats.FailedCount++
		stats.FailedItems = append(stats.FailedItems, fmt.Sprintf("%s: %v", err.Title, err.Err))
	}

	return stats, nil
}