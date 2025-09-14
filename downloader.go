

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
func (api *DabAPI) DownloadTrack(ctx context.Context, track Track, album *Album, outputPath string, coverData []byte, bar *pb.ProgressBar, debug bool, format string, bitrate string) error {
	// Get stream URL
	streamURL, err := api.GetStreamURL(ctx, idToString(track.ID))
	if err != nil {
		return fmt.Errorf("failed to get stream URL: %w", err)
	}

	// Download the audio file
	err = RetryWithBackoff(maxRetries, 5, func() error {
		audioResp, err := api.Request(ctx, streamURL, false, nil)
		if err != nil {
			return fmt.Errorf("failed to download audio: %w", err)
		}
		defer audioResp.Body.Close()

		// Wrap the response body in the progress bar reader
		if bar != nil {
			if debug {
				fmt.Println("DEBUG: Starting progress bar for", track.Title)
			}
			bar.SetWriter(os.Stdout) // Ensure output to stdout
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

		if _, err := io.Copy(out, audioResp.Body); err != nil {
			// Clean up the file on error to prevent partial files
			os.Remove(outputPath)
			return fmt.Errorf("failed to write audio file: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Add metadata to the downloaded file
	err = AddMetadata(outputPath, track, album, coverData, len(album.Tracks))
	if err != nil {
		return fmt.Errorf("failed to add metadata: %w", err)
	}

	if format != "flac" {
		colorInfo.Printf("🎵 Compressing to %s with bitrate %s kbps...\n", format, bitrate)
		convertedFile, err := ConvertTrack(outputPath, format, bitrate)
		if err != nil {
			return fmt.Errorf("failed to convert track: %w", err)
		}
		// Conversion successful, remove original FLAC file
		if err := os.Remove(outputPath); err != nil {
			colorWarning.Printf("âš ï¸ Failed to remove original FLAC file: %v\n", err)
		}
		if debug {
			colorInfo.Printf("âœ… Successfully converted to %s: %s\n", format, convertedFile)
		}
	}

	return nil
}

// DownloadSingleTrack downloads a single track.
// It now accepts a full Track object, assuming it comes from search results.
func (api *DabAPI) DownloadSingleTrack(ctx context.Context, track Track, debug bool, format string, bitrate string) error {
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
	if isTTY() {
		bar = pb.New(0)
		bar.SetWriter(os.Stdout) // Ensure output to stdout
		bar.SetTemplateString(`{{ string . "prefix" }} {{ bar . }} {{ percent . }} | {{ speed . "%s/s" }} | ETA {{ rtime . "%s" }}`)
		bar.Set("prefix", fmt.Sprintf("Downloading %-40s: ", TruncateString(albumTrack.Title, 40)))
		if debug {
			fmt.Println("DEBUG: Creating single track progress bar for", albumTrack.Title)
		}
		bar.Start()
	}

	// Download the track
	if err := api.DownloadTrack(ctx, *albumTrack, album, trackPath, coverData, bar, debug, format, bitrate); err != nil {
		if bar != nil {
			bar.Finish()
		}
		return err
	}
	if bar != nil {
		bar.Finish()
	}

	colorSuccess.Printf("✅ Successfully downloaded: %s\n", trackPath)
	return nil
}


// DownloadAlbum downloads all tracks from an album
func (api *DabAPI) DownloadAlbum(ctx context.Context, albumID string, parallelism int, debug bool, pool *pb.Pool, format string, bitrate string) (*DownloadStats, error) {
	album, err := api.GetAlbum(ctx, albumID)
	if err != nil {
		return nil, fmt.Errorf("failed to get album info: %w", err)
	}

	artistDir := filepath.Join(api.outputLocation, SanitizeFileName(album.Artist))
	albumDir := filepath.Join(artistDir, SanitizeFileName(album.Title))

	// Download cover
	var coverData []byte
	if album.Cover != "" {
		coverData, err = api.DownloadCover(ctx, album.Cover)
		if err != nil {
			colorWarning.Printf("⚠️ Could not download cover art for album %s: %v\n", album.Title, err)
		}
	}

	// Setup for concurrent downloads
	var wg sync.WaitGroup
	sem := semaphore.NewWeighted(int64(parallelism))
	stats := &DownloadStats{}
	errorChan := make(chan trackError, len(album.Tracks))

	var localPool bool
	if pool == nil && isTTY() {
		pool, err = pb.StartPool()
		if err != nil {
			return nil, fmt.Errorf("failed to start progress bar pool: %w", err)
		}
		localPool = true
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
				stats.SkippedCount++
				return
			}

			// Create a new progress bar for each track
			var bar *pb.ProgressBar
			if pool != nil {
				bar = pb.New(0)
				bar.SetTemplateString(`{{ string . "prefix" }} {{ bar . }} {{ percent . }} | {{ speed . "%s/s" }} | ETA {{ rtime . "%s" }}`)
				bar.Set("prefix", fmt.Sprintf("Track %-2d: %-40s", trackNumber, TruncateString(track.Title, 40)))
				if debug {
					fmt.Printf("DEBUG: Created progress bar for track %s\n", track.Title)
				}
				pool.Add(bar)
				if debug {
					fmt.Printf("DEBUG: Added progress bar for track %s to the pool\n", track.Title)
				}
			}

			if err := api.DownloadTrack(ctx, track, album, trackPath, coverData, bar, debug, format, bitrate); err != nil {
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
