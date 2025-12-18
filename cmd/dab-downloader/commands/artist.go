package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"dab-downloader/internal/shared"
	"github.com/spf13/cobra"
)

// NewArtistCommand creates the artist download command
func NewArtistCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "artist [artist_id]",
		Short: "Download an artist's entire discography.",
		Args:  cobra.ExactArgs(1),
		RunE:  runArtistCommand,
	}

	// Add flags
	cmd.Flags().String("filter", "all", "Filter by item type (albums, eps, singles), comma-separated")
	cmd.Flags().Bool("no-confirm", false, "Skip confirmation prompt")
	cmd.Flags().String("format", "flac", "Format to convert to after downloading (e.g., mp3, ogg, opus)")
	cmd.Flags().String("bitrate", "320", "Bitrate for lossy formats (in kbps, e.g., 192, 256, 320)")

	return cmd
}

func runArtistCommand(cmd *cobra.Command, args []string) error {
	// Get configuration and services
	config, serviceContainer := initConfigAndServices(cmd)
	
	// Check FFmpeg if format is not FLAC
	if config.Format != "flac" && !shared.CheckFFmpeg() {
		printInstallInstructions()
		return nil
	}
	
	// Get command flags
	filter, _ := cmd.Flags().GetString("filter")
	noConfirm, _ := cmd.Flags().GetBool("no-confirm")
	format, _ := cmd.Flags().GetString("format")
	bitrate, _ := cmd.Flags().GetString("bitrate")
	debug, _ := cmd.Flags().GetBool("debug")
	
	// Check if filter flag was explicitly set by user
	filterChanged := cmd.Flags().Changed("filter")
	if !filterChanged {
		filter = "" // Use empty string to trigger menu
	}
	
	// Override config with command flags if provided
	if format != "flac" {
		config.Format = format
	}
	if bitrate != "320" {
		config.Bitrate = bitrate
	}
	
	artistID := args[0]
	serviceContainer.Logger.Info("🎵 Starting artist discography download for ID: %s", artistID)
	
	// Download artist discography
	stats, err := serviceContainer.DownloadService.DownloadArtist(context.Background(), artistID, config, debug, config.Format, config.Bitrate, filter, noConfirm)
	
	// Handle errors but don't return early - we still want to show summaries
	var hasError bool
	if err != nil {
		hasError = true
		if errors.Is(err, shared.ErrDownloadCancelled) {
			serviceContainer.Logger.Warning("⚠️ Discography download cancelled by user.")
		} else if errors.Is(err, shared.ErrNoItemsSelected) {
			serviceContainer.Logger.Warning("⚠️ No items were selected for download.")
		} else {
			serviceContainer.Logger.Error("❌ Failed to download discography: %v", err)
		}
	} else {
		serviceContainer.Logger.Success("✅ Discography download completed!")
	}
	
	// Warnings are now displayed by the calling command (search.go) before the summary
	
	if debug {
		serviceContainer.Logger.Debug("DEBUG: After warnings, about to check summary condition")
	}
	
	// Display download summary AFTER warnings (even if there were errors)
	if debug {
		if stats == nil {
			serviceContainer.Logger.Debug("DEBUG: stats is nil")
		} else {
			serviceContainer.Logger.Debug("DEBUG: stats - Success: %d, Failed: %d, Skipped: %d", stats.SuccessCount, stats.FailedCount, stats.SkippedCount)
		}
	}
	
	// Show summary if we have any stats at all (success, failed, or skipped)
	if stats != nil && (stats.SuccessCount > 0 || stats.FailedCount > 0 || stats.SkippedCount > 0) {
		if debug {
			serviceContainer.Logger.Debug("DEBUG: About to display download summary")
		}
		
		// Get artist name for the summary
		artistName := "Unknown Artist"
		if artist, err := serviceContainer.DownloadService.GetArtistInfo(context.Background(), artistID, config, false); err == nil && artist != nil {
			artistName = artist.Name
		}
		
		fmt.Printf("\n")
		shared.ColorInfo.Printf("📊 Download Summary for %s:\n", artistName)
		
		if stats.SuccessCount > 0 {
			shared.ColorSuccess.Printf("✅ Successfully downloaded: %d items\n", stats.SuccessCount)
		}
		
		if stats.SkippedCount > 0 {
			shared.ColorWarning.Printf("⏭️  Skipped (already exists): %d items\n", stats.SkippedCount)
		}
		
		if stats.FailedCount > 0 {
			shared.ColorError.Printf("❌ Failed downloads: %d items\n", stats.FailedCount)
			if len(stats.FailedItems) > 0 {
				shared.ColorError.Printf("   Failed items: %s\n", strings.Join(stats.FailedItems, ", "))
			}
		}
		
		// Show download location
		shared.ColorSuccess.Printf("📁 Artist discography downloaded to: %s\n", config.DownloadLocation)
		shared.ColorSuccess.Printf("🎉 Discography download completed for %s\n", artistName)
	} else {
		if debug {
			serviceContainer.Logger.Debug("DEBUG: Not displaying summary - condition not met")
		}
	}
	
	// Return error after showing summaries
	if hasError {
		return err
	}
	
	return nil
}

