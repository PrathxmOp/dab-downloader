package downloader

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cheggaaa/pb/v3"
	"golang.org/x/sync/semaphore"

	"dab-downloader/internal/api"
	"dab-downloader/internal/config"
	"dab-downloader/internal/ffmpeg"
	"dab-downloader/internal/models"
	"dab-downloader/internal/ui"
	"dab-downloader/internal/utils"
)

type Downloader struct {
	api            *api.DabAPI
	outputLocation string
}

func NewDownloader(api *api.DabAPI, outputLocation string) *Downloader {
	return &Downloader{
		api:            api,
		outputLocation: outputLocation,
	}
}

// DownloadTrack downloads a single track with metadata
func (d *Downloader) DownloadTrack(ctx context.Context, track models.Track, album *models.Album, outputPath string, coverData []byte, bar *pb.ProgressBar, debug bool, format string, bitrate string, conf *config.Config, warningCollector *utils.WarningCollector) (string, error) {
	// Get stream URL
	streamURL, err := d.api.GetStreamURL(ctx, utils.IDToString(track.ID))
	if err != nil {
		return "", fmt.Errorf("failed to get stream URL: %w", err)
	}

	var expectedFileSize int64 // Store expected size for final verification

	// Determine retry attempts
	maxRetries := config.DefaultMaxRetries
	if conf != nil && conf.MaxRetryAttempts > 0 {
		maxRetries = conf.MaxRetryAttempts
	}

	// Download the audio file
	err = utils.RetryWithBackoff(maxRetries, 5, func() error {
		audioResp, err := d.api.Request(ctx, streamURL, false, nil)
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

		bytesWritten, err := io.Copy(out, audioResp.Body)
		out.Close() // Explicitly close after copying

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
	if utils.FileExists(outputPath) {
		// Only verify if verification is enabled (default true if not specified)
		verifyEnabled := conf == nil || conf.VerifyDownloads // Default to true
		if verifyEnabled && expectedFileSize > 0 {
			if verifyErr := utils.VerifyFileIntegrity(outputPath, expectedFileSize, debug); verifyErr != nil {
				// Remove the corrupted file and return error
				os.Remove(outputPath)
				return "", fmt.Errorf("post-download verification failed: %w", verifyErr)
			}
		}
	} else {
		return "", fmt.Errorf("download completed but file not found on disk: %s", outputPath)
	}

	// Add metadata to the downloaded file
	err = ffmpeg.AddMetadataWithDebug(outputPath, track, album, coverData, len(album.Tracks), warningCollector, debug)
	if err != nil {
		return "", fmt.Errorf("failed to add metadata: %w", err)
	}

	finalPath := outputPath
	if format != "flac" {
		ui.Info.Printf("🎵 Compressing to %s with bitrate %s kbps...\n", format, bitrate)
		convertedFile, err := ffmpeg.ConvertTrack(outputPath, format, bitrate)
		if err != nil {
			return "", fmt.Errorf("failed to convert track: %w", err)
		}
		// Conversion successful, remove original FLAC file
		if err := os.Remove(outputPath); err != nil {
			ui.Warning.Printf("⚠️ Failed to remove original FLAC file: %v\n", err)
		}
		finalPath = convertedFile
		if debug {
			ui.Info.Printf("✅ Successfully converted to %s: %s\n", format, convertedFile)
		}
	}

	return finalPath, nil
}

// DownloadSingleTrack downloads a single track.
func (d *Downloader) DownloadSingleTrack(ctx context.Context, track models.Track, debug bool, format string, bitrate string, pool *pb.Pool, conf *config.Config, warningCollector *utils.WarningCollector) error {
	// Create warning collector if not provided (standalone track download)
	var ownCollector bool
	if warningCollector == nil {
		warningCollector = utils.NewWarningCollector(conf.WarningBehavior != "silent")
		ownCollector = true
	}
	ui.Info.Printf("🎶 Preparing to download track: %s by %s (Album ID: %s)...\n", track.Title, track.Artist, track.AlbumID)

	// Fetch the album information using the track's AlbumID
	album, err := d.api.GetAlbum(ctx, track.AlbumID)
	if err != nil {
		if conf.WarningBehavior == "immediate" {
			ui.Warning.Printf("⚠️ Could not fetch album info for track %s (ID: %s): %v. Attempting to proceed with limited album info.\n", track.Title, utils.IDToString(track.ID), err)
		} else {
			warningCollector.AddAlbumFetchWarning(track.Title, utils.IDToString(track.ID), err.Error())
		}
		// Create a minimal album object if fetching fails, to allow metadata to be added
		album = &models.Album{Title: track.Album, Artist: track.Artist, Tracks: []models.Track{track}}
	}

	var albumTrack *models.Track
	for i := range album.Tracks {
		if utils.IDToString(album.Tracks[i].ID) == utils.IDToString(track.ID) {
			albumTrack = &album.Tracks[i]
			break
		}
	}

	if albumTrack == nil {
		return fmt.Errorf("failed to find track %s (ID: %s) within its album %s (ID: %s)", track.Title, utils.IDToString(track.ID), album.Title, album.ID)
	}

	// Download cover
	var coverData []byte
	if album.Cover != "" {
		coverData, err = d.api.DownloadCover(ctx, album.Cover)
		if err != nil {
			if conf.WarningBehavior == "immediate" {
				ui.Warning.Printf("⚠️ Could not download cover art for album %s: %v\n", album.Title, err)
			} else {
				warningCollector.AddCoverArtDownloadWarning(album.Title, err.Error())
			}
		}
	}

	// Create track path
	artistDir := filepath.Join(d.outputLocation, utils.SanitizeFileName(albumTrack.Artist))
	albumDir := filepath.Join(artistDir, utils.SanitizeFileName(album.Title))
	trackFileName := utils.GetTrackFilename(albumTrack.TrackNumber, albumTrack.Title)
	trackPath := filepath.Join(albumDir, trackFileName)

	// Format-aware duplicate detection
	targetPath := trackPath
	if format != "flac" {
		targetPath = strings.TrimSuffix(trackPath, ".flac") + "." + format
	}

	// Skip if already exists
	if utils.FileExists(targetPath) || (format != "flac" && utils.FileExists(trackPath)) {
		existingPath := targetPath
		if !utils.FileExists(targetPath) {
			existingPath = trackPath
		}
		
		if conf.WarningBehavior == "immediate" {
			ui.Warning.Printf("⭐ Track already exists: %s\n", existingPath)
		} else {
			warningCollector.AddTrackSkippedWarning(existingPath)
		}
		return nil
	}

	// Create progress bar
	var bar *pb.ProgressBar
	if pool != nil {
		bar = pb.New(0)
		bar.SetTemplateString(`{{ string . "prefix" }} {{ bar . }} {{ percent . }} | {{ speed . "%s/s" }} | ETA {{ rtime . "%s" }}`)
		bar.Set("prefix", fmt.Sprintf("Downloading %-40s: ", utils.TruncateString(albumTrack.Title, 40)))
		pool.Add(bar)
	} else if utils.IsTTY() {
		bar = pb.New(0)
		bar.SetWriter(os.Stdout)
		bar.SetTemplateString(`{{ string . "prefix" }} {{ bar . }} {{ percent . }} | {{ speed . "%s/s" }} | ETA {{ rtime . "%s" }}`)
		bar.Set("prefix", fmt.Sprintf("Downloading %-40s: ", utils.TruncateString(albumTrack.Title, 40)))
		bar.Start()
	}

	// Download the track
	finalPath, err := d.DownloadTrack(ctx, *albumTrack, album, trackPath, coverData, bar, debug, format, bitrate, conf, warningCollector)
	if err != nil {
		if bar != nil && pool == nil {
			bar.Finish()
		}
		return err
	}
	if bar != nil && pool == nil {
		bar.Finish()
	}

	ui.Success.Printf("✅ Successfully downloaded: %s\n", finalPath)
	
	if ownCollector && conf.WarningBehavior == "summary" {
		warningCollector.PrintSummary()
	}
	
	return nil
}

// DownloadAlbum downloads all tracks from an album
func (d *Downloader) DownloadAlbum(ctx context.Context, albumID string, conf *config.Config, debug bool, pool *pb.Pool, warningCollector *utils.WarningCollector) (*models.DownloadStats, error) {
	var ownCollector bool
	if warningCollector == nil {
		warningCollector = utils.NewWarningCollector(conf.WarningBehavior != "silent")
		ownCollector = true
	}
	
album, err := d.api.GetAlbum(ctx, albumID)
	if err != nil {
		return nil, fmt.Errorf("failed to get album info: %w", err)
	}

	artistDir := filepath.Join(d.outputLocation, utils.SanitizeFileName(album.Artist))
albumDir := filepath.Join(artistDir, utils.SanitizeFileName(album.Title))

	if err := os.MkdirAll(albumDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create album directory: %w", err)
	}

	// Download cover
	var coverData []byte
	if album.Cover != "" {
		coverData, err = d.api.DownloadCover(ctx, album.Cover)
		if err != nil {
			if conf.WarningBehavior == "immediate" {
				ui.Warning.Printf("⚠️ Could not download cover art for album %s: %v\n", album.Title, err)
			} else {
				warningCollector.AddCoverArtDownloadWarning(album.Title, err.Error())
			}
		}
	}

	if conf.SaveAlbumArt && coverData != nil {
		coverPath := filepath.Join(albumDir, "cover.jpg")
		if err := os.WriteFile(coverPath, coverData, 0644); err != nil {
			if conf.WarningBehavior == "immediate" {
				ui.Warning.Printf("⚠️ Failed to save cover art for album %s: %v\n", album.Title, err)
			} else {
				warningCollector.AddCoverArtDownloadWarning(album.Title, fmt.Sprintf("Failed to save: %v", err))
			}
		}
	}

	var wg sync.WaitGroup
	sem := semaphore.NewWeighted(int64(conf.Parallelism))
	stats := &models.DownloadStats{}
	errorChan := make(chan struct {
		Title string
		Err   error
	}, len(album.Tracks))

	var localPool bool
	if pool == nil && utils.IsTTY() {
		var err error
		pool, err = pb.StartPool()
		if err != nil {
			ui.Error.Printf("❌ Failed to start progress bar pool: %v\n", err)
		} else {
			localPool = true
		}
	}

	bars := make([]*pb.ProgressBar, len(album.Tracks))
	if pool != nil {
		for i, track := range album.Tracks {
			trackNumber := track.TrackNumber
			if trackNumber == 0 {
				trackNumber = i + 1
			}
			bar := pb.New(0)
			bar.SetTemplateString(`{{ string . "prefix" }} {{ bar . }} {{ percent . }} | {{ speed . "%s/s" }} | ETA {{ rtime . "%s" }}`)
			bar.Set("prefix", fmt.Sprintf("Track %-2d: %-40s", trackNumber, utils.TruncateString(track.Title, 40)))
			bars[i] = bar
			pool.Add(bar)
		}
	}

	for idx, track := range album.Tracks {
		wg.Add(1)
		if err := sem.Acquire(ctx, 1); err != nil {
			ui.Error.Printf("Failed to acquire semaphore: %v\n", err)
			wg.Done()
			continue
		}

		go func(idx int, track models.Track) {
			defer wg.Done()
			defer sem.Release(1)

			trackNumber := track.TrackNumber
			if trackNumber == 0 {
				trackNumber = idx + 1
			}

			trackFileName := fmt.Sprintf("%02d - %s.flac", trackNumber, utils.SanitizeFileName(track.Title))
			trackPath := filepath.Join(albumDir, trackFileName)

			targetPath := trackPath
			if conf.Format != "flac" {
				targetPath = strings.TrimSuffix(trackPath, ".flac") + "." + conf.Format
			}

			if utils.FileExists(targetPath) || (conf.Format != "flac" && utils.FileExists(trackPath)) {
				existingPath := targetPath
				if !utils.FileExists(targetPath) {
					existingPath = trackPath
				}

				if conf.WarningBehavior == "immediate" {
					ui.Warning.Printf("⭐ Track already exists: %s\n", existingPath)
				} else {
					warningCollector.AddTrackSkippedWarning(existingPath)
				}
				stats.SkippedCount++
				return
			}

			var bar *pb.ProgressBar
			if pool != nil {
				bar = bars[idx]
			}

			if _, err := d.DownloadTrack(ctx, track, album, trackPath, coverData, bar, debug, conf.Format, conf.Bitrate, conf, warningCollector); err != nil {
				errorChan <- struct {
					Title string
					Err   error
				}{track.Title, fmt.Errorf("track %s: %w", track.Title, err)}
				return
			}

			stats.SuccessCount++

		}(idx, track)
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

	if album != nil {
		d.updateFailedTracksWithReleaseMetadata(albumDir, album, warningCollector)
	}

	if ownCollector && conf.WarningBehavior == "summary" {
		warningCollector.PrintSummary()
	}

	return stats, nil
}

func (d *Downloader) updateFailedTracksWithReleaseMetadata(albumDir string, album *models.Album, warningCollector *utils.WarningCollector) {
	if album == nil {
		return
	}

	mbRelease := ffmpeg.GetCachedReleaseFromStore(album.Artist, album.Title)
	if mbRelease == nil {
		return
	}

	files, err := filepath.Glob(filepath.Join(albumDir, "*.flac"))
	if err != nil {
		return
	}

	updatedCount := 0
	for _, filePath := range files {
		if ffmpeg.UpdateTrackWithReleaseMetadata(filePath, mbRelease, warningCollector) {
			updatedCount++
		}
	}

	if updatedCount > 0 && warningCollector != nil {
		warningCollector.RemoveMusicBrainzReleaseWarning(album.Artist, album.Title)
	}
}
