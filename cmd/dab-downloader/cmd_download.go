package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/spf13/cobra"

	"dab-downloader/internal/batch"
	"dab-downloader/internal/ffmpeg"
	"dab-downloader/internal/models"
	"dab-downloader/internal/ui"
	"dab-downloader/internal/utils"
)

var artistCmd = &cobra.Command{
	Use:   "artist [artist_id]",
	Short: "Download an artist's entire discography.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		conf, _, d := initConfigAPIAndDownloader()
		if conf.Format != "flac" && !ffmpeg.CheckFFmpeg() {
			printInstallInstructions()
			return
		}
		artistID := args[0]
		ui.Info.Println("🎵 Starting artist discography download for ID:", artistID)
		if err := d.DownloadArtistDiscography(context.Background(), artistID, conf, debug, filter, noConfirm, nil); err != nil {
			if errors.Is(err, models.ErrDownloadCancelled) {
				ui.Warning.Println("⚠️ Discography download cancelled by user.")
			} else if errors.Is(err, models.ErrNoItemsSelected) {
				ui.Warning.Println("⚠️ No items were selected for download.")
			} else {
				ui.Error.Printf("❌ Failed to download discography: %v\n", err)
			}
		} else {
			ui.Success.Println("✅ Discography download completed!")
		}
	},
}

var albumCmd = &cobra.Command{
	Use:   "album [album_id]",
	Short: "Download an album by its ID.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		conf, _, d := initConfigAPIAndDownloader()
		if conf.Format != "flac" && !ffmpeg.CheckFFmpeg() {
			printInstallInstructions()
			return
		}
		albumID := args[0]
		// Handle URL if provided
		if strings.Contains(albumID, "/shared/album/") {
			parts := strings.Split(albumID, "/")
			albumID = parts[len(parts)-1]
			if strings.Contains(albumID, "?") {
				albumID = strings.Split(albumID, "?")[0]
			}
		}
		ui.Info.Println("🎵 Starting album download for ID:", albumID)
		if _, err := d.DownloadAlbum(context.Background(), albumID, conf, debug, nil, nil); err != nil {
			ui.Error.Printf("❌ Failed to download album: %v\n", err)
		} else {
			ui.Success.Println("✅ Album download completed!")
		}
	},
}

var playlistCmd = &cobra.Command{
	Use:   "playlist [playlist_id_or_url]",
	Short: "Download a DAB playlist by its ID or URL.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		conf, _, d := initConfigAPIAndDownloader()
		if conf.Format != "flac" && !ffmpeg.CheckFFmpeg() {
			printInstallInstructions()
			return
		}
		
		playlistID := args[0]
		// Handle URL if provided
		if strings.Contains(playlistID, "/shared/library/") {
			parts := strings.Split(playlistID, "/")
			playlistID = parts[len(parts)-1]
			if strings.Contains(playlistID, "?") {
				playlistID = strings.Split(playlistID, "?")[0]
			}
		}

		ui.Info.Println("🎵 Starting playlist download for ID:", playlistID)
		if _, err := d.DownloadPlaylist(context.Background(), playlistID, conf, debug, nil, nil); err != nil {
			ui.Error.Printf("❌ Failed to download playlist: %v\n", err)
		} else {
			ui.Success.Println("✅ Playlist download completed!")
		}
	},
}

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for artists, albums, or tracks.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		conf, a, d := initConfigAPIAndDownloader()
		if conf.Format != "flac" && !ffmpeg.CheckFFmpeg() {
			ui.Error.Println("❌ ffmpeg is not installed or not in your PATH. Please install ffmpeg to use the format conversion feature.")
			return
		}
		query := args[0]
		selectedItems, itemTypes, err := ui.HandleSearch(context.Background(), a, query, searchType, debug, auto)
		if err != nil {
			ui.Error.Printf("❌ Search failed: %v\n", err)
			return
		}
		if len(selectedItems) == 0 {
			return
		}

		time.Sleep(100 * time.Millisecond)

		var pool *pb.Pool
		var localPool bool
		if utils.IsTTY() && len(selectedItems) > 1 {
			var err error
			pool, err = pb.StartPool()
			if err != nil {
				ui.Error.Printf("❌ Failed to start progress bar pool: %v\n", err)
			} else {
				localPool = true
			}
		}

		for i, selectedItem := range selectedItems {
			itemType := itemTypes[i]
			switch itemType {
			case "artist":
				artist := selectedItem.(models.Artist)
				ui.Info.Println("🎵 Starting artist discography download for:", artist.Name)
				artistIDStr := utils.IDToString(artist.ID)
				if err := d.DownloadArtistDiscography(context.Background(), artistIDStr, conf, debug, filter, noConfirm, pool); err != nil {
					ui.Error.Printf("❌ Failed to download discography for %s: %v\n", artist.Name, err)
				} else {
					ui.Success.Println("✅ Discography download completed for", artist.Name)
				}
			case "album":
				album := selectedItem.(models.Album)
				ui.Info.Println("🎵 Starting album download for:", album.Title, "by", album.Artist)
				if _, err := d.DownloadAlbum(context.Background(), album.ID, conf, debug, pool, nil); err != nil {
					ui.Error.Printf("❌ Failed to download album %s: %v\n", album.Title, err)
				} else {
					ui.Success.Println("✅ Album download completed for", album.Title)
				}
			case "playlist":
				playlist := selectedItem.(models.Playlist)
				ui.Info.Println("🎵 Starting playlist download for:", playlist.Title)
				if _, err := d.DownloadPlaylist(context.Background(), playlist.ID, conf, debug, pool, nil); err != nil {
					ui.Error.Printf("❌ Failed to download playlist %s: %v\n", playlist.Title, err)
				} else {
					ui.Success.Println("✅ Playlist download completed for", playlist.Title)
				}
			case "track":
				track := selectedItem.(models.Track)
				
				// Offer to download the full album if in interactive mode or if expand flag is set
				downloadAlbum := expandSearch
				if !downloadAlbum && !auto && utils.IsTTY() {
					prompt := fmt.Sprintf("Do you want to download the entire album '%s' instead of just the track '%s'?", track.Album, track.Title)
					if utils.GetYesNoInput(prompt, "n") {
						downloadAlbum = true
					}
				}

				if downloadAlbum {
					ui.Info.Println("🎵 Starting album download for:", track.Album, "by", track.Artist)
					if _, err := d.DownloadAlbum(context.Background(), track.AlbumID, conf, debug, pool, nil); err != nil {
						ui.Error.Printf("❌ Failed to download album %s: %v\n", track.Album, err)
					} else {
						ui.Success.Println("✅ Album download completed for", track.Album)
					}
				} else {
					ui.Info.Println("🎵 Starting track download for:", track.Title, "by", track.Artist)
					if err := d.DownloadSingleTrack(context.Background(), track, debug, conf.Format, conf.Bitrate, pool, conf, nil); err != nil {
						ui.Error.Printf("❌ Failed to download track %s: %v\n", track.Title, err)
					} else {
						ui.Success.Println("✅ Track download completed for", track.Title)
					}
				}
			default:
				ui.Error.Println("❌ Unknown item type selected.")
			}
		}

		if localPool && pool != nil {
			pool.Stop()
		}
	},
}

var batchCmd = &cobra.Command{
	Use:   "batch [file_path]",
	Short: "Download items listed in a text file.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		conf, a, d := initConfigAPIAndDownloader()
		if conf.Format != "flac" && !ffmpeg.CheckFFmpeg() {
			ui.Error.Println("❌ ffmpeg is not installed or not in your PATH. Please install ffmpeg to use the format conversion feature.")
			return
		}
		filePath := args[0]
		if err := batch.ProcessBatchFile(context.Background(), filePath, a, d, conf, debug, expandSearch); err != nil {
			ui.Error.Printf("❌ Batch processing failed: %v\n", err)
		} else {
			ui.Success.Println("✅ Batch processing completed!")
		}
	},
}
