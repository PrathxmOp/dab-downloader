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

// ============================================================================
// 1. Constants and Types
// ============================================================================

const (
	DefaultMaxRetries = 3
	DefaultRetryDelay = 5
	DefaultFileMode   = 0755
)

// DownloadOptions holds configuration for track downloads
type DownloadOptions struct {
	OutputPath       string
	Format           string
	Bitrate          string
	Debug            bool
	MaxRetries       int
	VerifyDownloads  bool
}

// DownloadResult contains information about a completed download
type DownloadResult struct {
	FilePath     string
	BytesWritten int64
	Format       string
	Converted    bool
}

// ============================================================================
// 2. Constructor and Configuration
// ============================================================================

// TrackDownloader handles track download operations
type TrackDownloader struct {
	api               *dab.DabAPI
	metadataProcessor *MetadataProcessor
	config            *config.Config
	debug             bool
}

// NewTrackDownloader creates a new track downloader with the given API client
func NewTrackDownloader(api *dab.DabAPI, cfg *config.Config) *TrackDownloader {
	return &TrackDownloader{
		api:               api,
		metadataProcessor: NewMetadataProcessor(),
		config:            cfg,
		debug:             false,
	}
}

// SetDebugMode enables or disables debug logging
func (td *TrackDownloader) SetDebugMode(debug bool) {
	td.debug = debug
	td.metadataProcessor.SetDebugMode(debug)
}

// ============================================================================
// 3. Public API Methods
// ============================================================================

// DownloadTrack downloads a single track with metadata and optional conversion
func (td *TrackDownloader) DownloadTrack(ctx context.Context, track shared.Track, album *shared.Album, options DownloadOptions, coverData []byte, progressBar *pb.ProgressBar, warningCollector *shared.WarningCollector) (*DownloadResult, error) {
	// Validate inputs
	if err := td.validateDownloadInputs(track, album, options); err != nil {
		return nil, fmt.Errorf("invalid download parameters: %w", err)
	}

	// Get stream URL
	streamURL, err := td.getStreamURL(ctx, track)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream URL: %w", err)
	}

	// Download the audio file
	downloadResult, err := td.downloadAudioFile(ctx, streamURL, options, progressBar)
	if err != nil {
		return nil, fmt.Errorf("failed to download audio: %w", err)
	}

	// Add metadata
	if err := td.addMetadata(downloadResult.FilePath, track, album, coverData, warningCollector); err != nil {
		td.cleanup(downloadResult.FilePath)
		return nil, fmt.Errorf("failed to add metadata: %w", err)
	}

	// Convert format if needed
	finalResult, err := td.convertIfNeeded(downloadResult, options)
	if err != nil {
		td.cleanup(downloadResult.FilePath)
		return nil, fmt.Errorf("failed to convert track: %w", err)
	}

	return finalResult, nil
}

// DownloadTrackWithDefaults downloads a track using default options
func (td *TrackDownloader) DownloadTrackWithDefaults(ctx context.Context, track shared.Track, album *shared.Album, outputPath string, coverData []byte, warningCollector *shared.WarningCollector) (*DownloadResult, error) {
	options := DownloadOptions{
		OutputPath:      outputPath,
		Format:          "flac",
		Bitrate:         "320",
		Debug:           td.debug,
		MaxRetries:      td.getMaxRetries(),
		VerifyDownloads: td.getVerifyDownloads(),
	}

	return td.DownloadTrack(ctx, track, album, options, coverData, nil, warningCollector)
}

// ============================================================================
// 4. Private Core Methods
// ============================================================================

// validateDownloadInputs validates the download parameters
func (td *TrackDownloader) validateDownloadInputs(track shared.Track, album *shared.Album, options DownloadOptions) error {
	if track.ID == nil {
		return fmt.Errorf("track ID cannot be nil")
	}
	if options.OutputPath == "" {
		return fmt.Errorf("output path cannot be empty")
	}
	if options.Format == "" {
		return fmt.Errorf("format cannot be empty")
	}
	return nil
}

// getStreamURL retrieves the stream URL for a track
func (td *TrackDownloader) getStreamURL(ctx context.Context, track shared.Track) (string, error) {
	streamURL, err := td.api.GetStreamURL(ctx, shared.IdToString(track.ID))
	if err != nil {
		return "", fmt.Errorf("failed to get stream URL for track %s: %w", track.Title, err)
	}
	return streamURL, nil
}

// downloadAudioFile handles the actual file download with retry logic
func (td *TrackDownloader) downloadAudioFile(ctx context.Context, streamURL string, options DownloadOptions, progressBar *pb.ProgressBar) (*DownloadResult, error) {
	var result *DownloadResult
	var expectedFileSize int64

	maxRetries := options.MaxRetries
	if maxRetries <= 0 {
		maxRetries = td.getMaxRetries()
	}

	err := shared.RetryWithBackoff(maxRetries, DefaultRetryDelay, func() error {
		downloadResult, expectedSize, err := td.performDownload(ctx, streamURL, options, progressBar)
		if err != nil {
			return err
		}
		
		result = downloadResult
		expectedFileSize = expectedSize
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Post-download verification
	if err := td.verifyDownload(result.FilePath, expectedFileSize, options); err != nil {
		td.cleanup(result.FilePath)
		return nil, fmt.Errorf("download verification failed: %w", err)
	}

	return result, nil
}

// performDownload executes a single download attempt
func (td *TrackDownloader) performDownload(ctx context.Context, streamURL string, options DownloadOptions, progressBar *pb.ProgressBar) (*DownloadResult, int64, error) {
	// Make the request
	audioResp, err := td.api.Request(ctx, streamURL, false, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to request audio stream: %w", err)
	}
	defer audioResp.Body.Close()

	expectedSize := audioResp.ContentLength
	if td.debug && expectedSize > 0 {
		fmt.Printf("DEBUG: Expected file size: %d bytes\n", expectedSize)
	}

	// Setup progress tracking
	reader := td.setupProgressTracking(audioResp.Body, audioResp.ContentLength, progressBar)

	// Create output directory
	if err := td.createOutputDirectory(options.OutputPath); err != nil {
		return nil, 0, err
	}

	// Write file
	bytesWritten, err := td.writeAudioFile(options.OutputPath, reader)
	if err != nil {
		td.cleanup(options.OutputPath)
		return nil, 0, err
	}

	// Verify size during download
	if err := td.verifySizeDuringDownload(expectedSize, bytesWritten, options.OutputPath); err != nil {
		td.cleanup(options.OutputPath)
		return nil, 0, err
	}

	result := &DownloadResult{
		FilePath:     options.OutputPath,
		BytesWritten: bytesWritten,
		Format:       "flac", // Initial format is always FLAC
		Converted:    false,
	}

	return result, expectedSize, nil
}

// setupProgressTracking configures progress bar for the download
func (td *TrackDownloader) setupProgressTracking(body io.ReadCloser, contentLength int64, progressBar *pb.ProgressBar) io.Reader {
	if progressBar == nil {
		return body
	}

	if td.debug {
		fmt.Println("DEBUG: Setting up progress tracking")
	}

	if contentLength <= 0 {
		progressBar.Set("indeterminate", true)
	} else {
		progressBar.SetTotal(contentLength)
	}

	return progressBar.NewProxyReader(body)
}

// createOutputDirectory ensures the output directory exists
func (td *TrackDownloader) createOutputDirectory(outputPath string) error {
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, DefaultFileMode); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	return nil
}

// writeAudioFile writes the audio data to the output file
func (td *TrackDownloader) writeAudioFile(outputPath string, reader io.Reader) (int64, error) {
	out, err := os.Create(outputPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create output file %s: %w", outputPath, err)
	}
	defer out.Close()

	bytesWritten, err := io.Copy(out, reader)
	if err != nil {
		return 0, fmt.Errorf("failed to write audio data: %w", err)
	}

	return bytesWritten, nil
}

// verifySizeDuringDownload checks if the downloaded size matches expected size
func (td *TrackDownloader) verifySizeDuringDownload(expectedSize, actualSize int64, outputPath string) error {
	if expectedSize > 0 && actualSize != expectedSize {
		if td.debug {
			fmt.Printf("DEBUG: File size mismatch - expected: %d, got: %d bytes\n", expectedSize, actualSize)
		}
		return fmt.Errorf("incomplete download: expected %d bytes, got %d bytes", expectedSize, actualSize)
	}

	if td.debug && expectedSize > 0 {
		fmt.Printf("DEBUG: Successfully downloaded %d bytes verified\n", actualSize)
	}

	return nil
}

// verifyDownload performs post-download verification
func (td *TrackDownloader) verifyDownload(filePath string, expectedSize int64, options DownloadOptions) error {
	if !shared.FileExists(filePath) {
		return fmt.Errorf("download completed but file not found: %s", filePath)
	}

	// Only verify if verification is enabled
	verifyEnabled := options.VerifyDownloads
	if !verifyEnabled {
		return nil
	}

	if expectedSize > 0 {
		if err := shared.VerifyFileIntegrity(filePath, expectedSize, options.Debug); err != nil {
			return fmt.Errorf("post-download verification failed: %w", err)
		}
	}

	return nil
}

// addMetadata adds metadata to the downloaded file
func (td *TrackDownloader) addMetadata(filePath string, track shared.Track, album *shared.Album, coverData []byte, warningCollector *shared.WarningCollector) error {
	totalTracks := 0
	if album != nil {
		totalTracks = len(album.Tracks)
	}

	err := td.metadataProcessor.AddMetadataWithDebug(filePath, track, album, coverData, totalTracks, warningCollector, td.debug)
	if err != nil {
		return fmt.Errorf("failed to add metadata to %s: %w", filePath, err)
	}

	return nil
}

// convertIfNeeded converts the file to the target format if needed
func (td *TrackDownloader) convertIfNeeded(result *DownloadResult, options DownloadOptions) (*DownloadResult, error) {
	if options.Format == "flac" {
		return result, nil // No conversion needed
	}

	shared.ColorInfo.Printf("🎵 Converting to %s with bitrate %s kbps...\n", options.Format, options.Bitrate)
	
	convertedFile, err := ConvertTrack(result.FilePath, options.Format, options.Bitrate)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to %s: %w", options.Format, err)
	}

	// Remove original FLAC file after successful conversion
	if err := td.cleanup(result.FilePath); err != nil {
		shared.ColorWarning.Printf("⚠️ Failed to remove original FLAC file: %v\n", err)
	}

	if td.debug {
		shared.ColorInfo.Printf("✅ Successfully converted to %s: %s\n", options.Format, convertedFile)
	}

	// Update result
	result.FilePath = convertedFile
	result.Format = options.Format
	result.Converted = true

	return result, nil
}

// ============================================================================
// 5. Helper/Utility Functions
// ============================================================================

// getMaxRetries returns the configured max retries or default
func (td *TrackDownloader) getMaxRetries() int {
	if td.config != nil && td.config.MaxRetryAttempts > 0 {
		return td.config.MaxRetryAttempts
	}
	return shared.DefaultMaxRetries
}

// getVerifyDownloads returns the configured verification setting or default
func (td *TrackDownloader) getVerifyDownloads() bool {
	if td.config != nil {
		return td.config.VerifyDownloads
	}
	return true // Default to true
}

// cleanup removes a file and logs any errors
func (td *TrackDownloader) cleanup(filePath string) error {
	if err := os.Remove(filePath); err != nil {
		if td.debug {
			fmt.Printf("DEBUG: Failed to cleanup file %s: %v\n", filePath, err)
		}
		return err
	}
	return nil
}

// ============================================================================
// 6. Global Compatibility Layer (for backward compatibility)
// ============================================================================

var (
	// Global instance for backward compatibility
	globalDownloader *TrackDownloader
)

// initGlobalDownloader initializes the global downloader if needed
func initGlobalDownloader(api *dab.DabAPI, cfg *config.Config) {
	if globalDownloader == nil {
		globalDownloader = NewTrackDownloader(api, cfg)
	}
}

// DownloadTrack downloads a single track with metadata (global function for compatibility)
func DownloadTrack(ctx context.Context, api *dab.DabAPI, track shared.Track, album *shared.Album, outputPath string, coverData []byte, bar *pb.ProgressBar, debug bool, format string, bitrate string, config *config.Config, warningCollector *shared.WarningCollector) (string, error) {
	initGlobalDownloader(api, config)
	globalDownloader.SetDebugMode(debug)

	options := DownloadOptions{
		OutputPath:      outputPath,
		Format:          format,
		Bitrate:         bitrate,
		Debug:           debug,
		MaxRetries:      0, // Use config default
		VerifyDownloads: true,
	}

	result, err := globalDownloader.DownloadTrack(ctx, track, album, options, coverData, bar, warningCollector)
	if err != nil {
		return "", err
	}

	return result.FilePath, nil
}