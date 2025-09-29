package downloader

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	
	"github.com/cheggaaa/pb/v3"
	
	"dab-downloader/internal/shared"
	"dab-downloader/internal/config"
	"dab-downloader/internal/api/dab"
)

// DownloadTrack downloads a single track with metadata
func DownloadTrack(ctx context.Context, api *dab.DabAPI, track shared.Track, album *shared.Album, outputPath string, coverData []byte, bar *pb.ProgressBar, debug bool, format string, bitrate string, config *config.Config, warningCollector *shared.WarningCollector) (string, error) {
	// Get stream URL
	streamURL, err := api.GetStreamURL(ctx, shared.IdToString(track.ID))
	if err != nil {
		return "", fmt.Errorf("failed to get stream URL: %w", err)
	}

	var expectedFileSize int64 // Store expected size for final verification

	// Determine retry attempts
	maxRetries := shared.DefaultMaxRetries
	if config != nil && config.MaxRetryAttempts > 0 {
		maxRetries = config.MaxRetryAttempts
	}

	// Download the audio file
	err = shared.RetryWithBackoff(maxRetries, 5, func() error {
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
	if shared.FileExists(outputPath) {
		// Only verify if verification is enabled (default true if not specified)
		verifyEnabled := config == nil || config.VerifyDownloads // Default to true
		if verifyEnabled && expectedFileSize > 0 {
			if verifyErr := shared.VerifyFileIntegrity(outputPath, expectedFileSize, debug); verifyErr != nil {
				// Remove the corrupted file and return error
				os.Remove(outputPath)
				return "", fmt.Errorf("post-download verification failed: %w", verifyErr)
			}
		}
	} else {
		return "", fmt.Errorf("download completed but file not found on disk: %s", outputPath)
	}

	// Add metadata to the downloaded file
	err = AddMetadataWithDebug(outputPath, track, album, coverData, len(album.Tracks), warningCollector, debug)
	if err != nil {
		return "", fmt.Errorf("failed to add metadata: %w", err)
	}

	finalPath := outputPath
	if format != "flac" {
		shared.ColorInfo.Printf("🎵 Compressing to %s with bitrate %s kbps...\n", format, bitrate)
		convertedFile, err := ConvertTrack(outputPath, format, bitrate)
		if err != nil {
			return "", fmt.Errorf("failed to convert track: %w", err)
		}
		// Conversion successful, remove original FLAC file
		if err := os.Remove(outputPath); err != nil {
			shared.ColorWarning.Printf("⚠️ Failed to remove original FLAC file: %v\n", err)
		}
		finalPath = convertedFile
		if debug {
			shared.ColorInfo.Printf("✅ Successfully converted to %s: %s\n", format, convertedFile)
		}
	}

	return finalPath, nil
}