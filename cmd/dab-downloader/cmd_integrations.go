package main

import (
	"context"
	"strings"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/spf13/cobra"

	"dab-downloader/internal/ffmpeg"
	"dab-downloader/internal/models"
	"dab-downloader/internal/ui"
	"dab-downloader/internal/utils"
	"dab-downloader/pkg/navidrome"
	"dab-downloader/pkg/spotify"
	"dab-downloader/pkg/listenbrainz"
)

var spotifyCmd = &cobra.Command{
	Use:   "spotify [url]",
	Short: "Download a Spotify playlist or album.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		conf, a, d := initConfigAPIAndDownloader()
		if conf.Format != "flac" && !ffmpeg.CheckFFmpeg() {
			ui.Error.Println("❌ ffmpeg is not installed or not in your PATH. Please install ffmpeg to use the format conversion feature.")
			return
		}
		url := args[0]

		spotifyClient := spotify.NewSpotifyClient(conf.SpotifyClientID, conf.SpotifyClientSecret)
		if err := spotifyClient.Authenticate(); err != nil {
			ui.Error.Printf("❌ Failed to authenticate with Spotify: %v\n", err)
			return
		}

		var spotifyTracks []spotify.SpotifyTrack
		var err error

		if strings.Contains(url, "/playlist/") {
			spotifyTracks, _, err = spotifyClient.GetPlaylistTracks(url)
		} else if strings.Contains(url, "/album/") {
			spotifyTracks, _, err = spotifyClient.GetAlbumTracks(url)
		} else {
			ui.Error.Println("❌ Invalid Spotify URL. Please provide a playlist or album URL.")
			return
		}

		if err != nil {
			ui.Error.Printf("❌ Failed to get tracks from Spotify: %v\n", err)
			return
		}

		if expandPlaylist {
			ui.Info.Println("Expanding playlist to download full albums...")
		
uniqueAlbums := make(map[string]spotify.SpotifyTrack)
			for _, track := range spotifyTracks {
				albumKey := strings.ToLower(track.AlbumName + " - " + track.AlbumArtist)
				if _, exists := uniqueAlbums[albumKey]; !exists {
					uniqueAlbums[albumKey] = track
				}
			}

			ui.Info.Printf("Found %d unique albums in the playlist.\n", len(uniqueAlbums))

			for _, track := range uniqueAlbums {
				albumSearchQuery := track.AlbumName + " - " + track.AlbumArtist
				ui.Info.Printf("Searching for album: %s\n", albumSearchQuery)

				selectedItems, itemTypes, err := ui.HandleSearch(context.Background(), a, albumSearchQuery, "album", debug, auto)
				if err != nil {
					ui.Error.Printf("❌ Search failed for album '%s': %v\n", albumSearchQuery, err)
					continue
				}

				if len(selectedItems) == 0 {
					ui.Warning.Printf("⚠️ No results found for album: %s\n", albumSearchQuery)
					continue
				}

				time.Sleep(500 * time.Millisecond)

				for i, selectedItem := range selectedItems {
					if itemTypes[i] == "album" {
						album := selectedItem.(models.Album)
						ui.Info.Println("🎵 Starting album download for:", album.Title, "by", album.Artist)
						if _, err := d.DownloadAlbum(context.Background(), album.ID, conf, debug, nil, nil); err != nil {
							ui.Error.Printf("❌ Failed to download album %s: %v\n", album.Title, err)
						} else {
							ui.Success.Println("✅ Album download completed for", album.Title)
						}
						break
					}
				}
			}
			return
		}

		var pool *pb.Pool
		var localPool bool
		if utils.IsTTY() && len(spotifyTracks) > 1 {
			var err error
			pool, err = pb.StartPool()
			if err != nil {
				ui.Error.Printf("❌ Failed to start progress bar pool: %v\n", err)
			} else {
				localPool = true
			}
		}

		for _, spotifyTrack := range spotifyTracks {
			trackName := spotifyTrack.Name + " - " + spotifyTrack.Artist
			selectedItems, itemTypes, err := ui.HandleSearch(context.Background(), a, trackName, "track", debug, auto)
			if err != nil {
				ui.Error.Printf("❌ Search failed for track %s: %v\n", trackName, err)
				if pool != nil {
					pool.Stop()
				}
				return
			}

			if len(selectedItems) == 0 {
				ui.Warning.Printf("⚠️ No results found for track: %s\n", trackName)
				continue
			}

			time.Sleep(500 * time.Millisecond)

			for i, selectedItem := range selectedItems {
				itemType := itemTypes[i]
				if itemType == "track" {
					track := selectedItem.(models.Track)
					ui.Info.Println("🎵 Starting track download for:", track.Title, "by", track.Artist)
					if err := d.DownloadSingleTrack(context.Background(), track, debug, conf.Format, conf.Bitrate, pool, conf, nil); err != nil {
						ui.Error.Printf("❌ Failed to download track %s: %v\n", track.Title, err)
					} else {
						ui.Success.Println("✅ Track download completed for", track.Title)
					}
				}
			}
		}

		if localPool && pool != nil {
			pool.Stop()
		}
	},
}

var navidromeCmd = &cobra.Command{
	Use:   "navidrome [playlist_url]",
	Short: "Copy a Spotify or ListenBrainz playlist to Navidrome.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		conf, a, d := initConfigAPIAndDownloader()
		playlistURL := args[0]

		// Define a unified track structure to handle both sources
		type UnifiedTrack struct {
			Name      string
			Artist    string
			AlbumName string
		}

		var tracks []UnifiedTrack
		var playlistName string

		// Determine source and fetch tracks
		if strings.Contains(playlistURL, "spotify.com") {
			spotifyClient := spotify.NewSpotifyClient(conf.SpotifyClientID, conf.SpotifyClientSecret)
			if err := spotifyClient.Authenticate(); err != nil {
				ui.Error.Printf("❌ Failed to authenticate with Spotify: %v\n", err)
				return
			}

			var spotifyTracks []spotify.SpotifyTrack
			var err error

			if strings.Contains(playlistURL, "/playlist/") {
				spotifyTracks, playlistName, err = spotifyClient.GetPlaylistTracks(playlistURL)
			} else if strings.Contains(playlistURL, "/album/") {
				spotifyTracks, playlistName, err = spotifyClient.GetAlbumTracks(playlistURL)
			} else {
				ui.Error.Println("❌ Invalid Spotify URL. Please provide a playlist or album URL.")
				return
			}

			if err != nil {
				ui.Error.Printf("❌ Failed to get tracks from Spotify: %v\n", err)
				return
			}
			
			// Convert to UnifiedTrack
			for _, t := range spotifyTracks {
				tracks = append(tracks, UnifiedTrack{
					Name:      t.Name,
					Artist:    t.Artist,
					AlbumName: t.AlbumName, // Might be empty for some Spotify tracks
				})
			}

		} else if strings.Contains(playlistURL, "listenbrainz.org") {
			lbClient := listenbrainz.NewListenBrainzClient()
			lbTracks, lbName, err := lbClient.GetPlaylistTracks(playlistURL)
			if err != nil {
				ui.Error.Printf("❌ Failed to get tracks from ListenBrainz: %v\n", err)
				return
			}
			playlistName = lbName
			
			// Convert to UnifiedTrack
			for _, t := range lbTracks {
				tracks = append(tracks, UnifiedTrack{
					Name:      t.Name,
					Artist:    t.Artist,
					AlbumName: t.AlbumName,
				})
			}
		} else {
			ui.Error.Println("❌ Unsupported URL. Please provide a valid Spotify or ListenBrainz playlist URL.")
			return
		}

		navidromeClient := navidrome.NewNavidromeClient(conf.NavidromeURL, conf.NavidromeUsername, conf.NavidromePassword)
		if err := navidromeClient.Authenticate(); err != nil {
			ui.Error.Printf("❌ Failed to authenticate with Navidrome: %v\n", err)
			return
		}

		if expandNavidrome {
			ui.Info.Println("Expanding playlist to download full albums...")
		
uniqueAlbums := make(map[string]UnifiedTrack)
			for _, track := range tracks {
				if track.AlbumName == "" {
					continue
				}
				albumKey := strings.ToLower(track.AlbumName + " - " + track.Artist)
				if _, exists := uniqueAlbums[albumKey]; !exists {
					uniqueAlbums[albumKey] = track
				}
			}

			ui.Info.Printf("Found %d unique albums in the playlist.\n", len(uniqueAlbums))

			for _, track := range uniqueAlbums {
				albumSearchQuery := track.AlbumName + " - " + track.Artist

				ui.Info.Printf("Searching for album '%s' by '%s' in Navidrome...", track.AlbumName, track.Artist)
				navidromeAlbum, err := navidromeClient.SearchAlbum(track.AlbumName, track.Artist)
				if err != nil {
					ui.Warning.Printf("⚠️ Error searching for album %s in Navidrome: %v\n", albumSearchQuery, err)
				} else if navidromeAlbum != nil {
					ui.Success.Printf("✅ Album '%s' already exists in Navidrome, skipping download.\n", albumSearchQuery)
					continue
				}

				selectedItems, itemTypes, err := ui.HandleSearch(context.Background(), a, albumSearchQuery, "album", debug, auto)
				if err != nil {
					ui.Error.Printf("❌ Search failed for album '%s': %v\n", albumSearchQuery, err)
					continue
				}

				if len(selectedItems) == 0 {
					ui.Warning.Printf("⚠️ No results found for album: %s\n", albumSearchQuery)
					continue
				}

				time.Sleep(100 * time.Millisecond)

				for i, selectedItem := range selectedItems {
					if itemTypes[i] == "album" {
						album := selectedItem.(models.Album)
						ui.Info.Println("🎵 Starting album download for:", album.Title, "by", album.Artist)
						if _, err := d.DownloadAlbum(context.Background(), album.ID, conf, debug, nil, nil); err != nil {
							ui.Error.Printf("❌ Failed to download album %s: %v\n", album.Title, err)
						} else {
							ui.Success.Println("✅ Album download completed for", album.Title)
						}
						break
					}
				}
			}
		}
		
		finalPlaylistName := utils.GetUserInput("Enter a name for the new Navidrome playlist", playlistName)
		if err := navidromeClient.CreatePlaylist(finalPlaylistName); err != nil {
			ui.Error.Printf("❌ Failed to create Navidrome playlist: %v\n", err)
			return
		}

		playlistID, err := navidromeClient.SearchPlaylist(finalPlaylistName)
		if err != nil {
			ui.Error.Printf("❌ Failed to find newly created playlist '%s': %v\n", finalPlaylistName, err)
			return
		}

		var navidromeTrackIDs []string

		for _, track := range tracks {
			trackName := track.Name
			if ignoreSuffix != "" {
				trackName = utils.RemoveSuffix(trackName, ignoreSuffix)
			}
			foundTrack, err := navidromeClient.SearchTrack(trackName, track.Artist, track.AlbumName)
			if err != nil {
				ui.Warning.Printf("⚠️ Error searching for track %s by %s on Navidrome: %v\n", track.Name, track.Artist, err)
				continue
			}

			if foundTrack != nil {
				navidromeTrackIDs = append(navidromeTrackIDs, foundTrack.ID)
				ui.Success.Println("track is already found skipping")
			} else {
				ui.Warning.Printf("⚠️ Track %s by %s not found on Navidrome. Searching DAB...\n", track.Name, track.Artist)

				dabSearchQuery := track.Name + " - " + track.Artist
				if ignoreSuffix != "" {
					dabSearchQuery = trackName + " - " + track.Artist
				}
				dabSearchResults, dabItemTypes, err := ui.HandleSearch(context.Background(), a, dabSearchQuery, "track", debug, auto)
				if err != nil {
					ui.Error.Printf("❌ Failed to search DAB for %s: %v\n", track.Name, err)
					continue
				}

				if len(dabSearchResults) > 0 {
					time.Sleep(500 * time.Millisecond)
					selectedDabItem := dabSearchResults[0]
					selectedDabItemType := dabItemTypes[0]
					if selectedDabItemType == "track" {
						dabTrack := selectedDabItem.(models.Track)
						ui.Info.Printf("🎵 Downloading %s by %s from DAB...\n", dabTrack.Title, dabTrack.Artist)
						if err := d.DownloadSingleTrack(context.Background(), dabTrack, debug, conf.Format, conf.Bitrate, nil, conf, nil); err != nil {
							ui.Error.Printf("❌ Failed to download track %s from DAB: %v\n", dabTrack.Title, err)
						} else {
							ui.Success.Printf("✅ Downloaded %s by %s from DAB. It should appear in Navidrome soon.\n", dabTrack.Title, dabTrack.Artist)
							time.Sleep(5 * time.Second)
							reScannedTrack, err := navidromeClient.SearchTrack(dabTrack.Title, dabTrack.Artist, dabTrack.Album)
							if err != nil {
								ui.Warning.Printf("⚠️ Failed to re-search for downloaded track %s in Navidrome: %v\n", dabTrack.Title, err)
							} else if reScannedTrack != nil {
								navidromeTrackIDs = append(navidromeTrackIDs, reScannedTrack.ID)
								ui.Success.Printf("✅ Found newly downloaded track %s in Navidrome (ID: %s) and added to list for playlist.\n", reScannedTrack.Title, reScannedTrack.ID)
							} else {
								ui.Warning.Printf("⚠️ Downloaded track %s not found in Navidrome after re-scan. It might be added later manually.\n", dabTrack.Title)
							}
						}
					}
				}
			}
		}

		if len(navidromeTrackIDs) > 0 {
			if err := navidromeClient.AddTracksToPlaylist(playlistID, navidromeTrackIDs); err != nil {
				ui.Error.Printf("❌ Failed to add tracks to Navidrome playlist: %v\n", err)
			} else {
				ui.Success.Printf("✅ Successfully added %d tracks to Navidrome playlist '%s'\n", len(navidromeTrackIDs), finalPlaylistName)
			}
		}
	},
}

var addToPlaylistCmd = &cobra.Command{
	Use:   "add-to-playlist [playlist_id] [song_id...]",
	Short: "Add one or more songs to a Navidrome playlist.",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		conf, _, _ := initConfigAPIAndDownloader()

		navidromeClient := navidrome.NewNavidromeClient(conf.NavidromeURL, conf.NavidromeUsername, conf.NavidromePassword)
		if err := navidromeClient.Authenticate(); err != nil {
			ui.Error.Printf("❌ Failed to authenticate with Navidrome: %v\n", err)
			return
		}

		playlistID := args[0]
		songIDs := args[1:]

		if err := navidromeClient.AddTracksToPlaylist(playlistID, songIDs); err != nil {
			ui.Error.Printf("❌ Failed to add tracks to playlist %s: %v\n", playlistID, err)
		} else {
			ui.Success.Printf("✅ Successfully added %d tracks to playlist %s\n", len(songIDs), playlistID)
		}
	},
}

var listenbrainzCmd = &cobra.Command{
	Use:   "listenbrainz [url]",
	Short: "Download a ListenBrainz playlist.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		conf, a, d := initConfigAPIAndDownloader()
		if conf.Format != "flac" && !ffmpeg.CheckFFmpeg() {
			ui.Error.Println("❌ ffmpeg is not installed or not in your PATH. Please install ffmpeg to use the format conversion feature.")
			return
		}
		url := args[0]

		lbClient := listenbrainz.NewListenBrainzClient()
		lbTracks, lbName, err := lbClient.GetPlaylistTracks(url)
		if err != nil {
			ui.Error.Printf("❌ Failed to get tracks from ListenBrainz: %v\n", err)
			return
		}

		ui.Info.Printf("🎵 Found %d tracks in ListenBrainz playlist: %s\n", len(lbTracks), lbName)

		var pool *pb.Pool
		var localPool bool
		if utils.IsTTY() && len(lbTracks) > 1 {
			var err error
			pool, err = pb.StartPool()
			if err != nil {
				ui.Error.Printf("❌ Failed to start progress bar pool: %v\n", err)
			} else {
				localPool = true
			}

			for _, lbTrack := range lbTracks {
				trackName := lbTrack.Name + " - " + lbTrack.Artist
				selectedItems, itemTypes, err := ui.HandleSearch(context.Background(), a, trackName, "track", debug, auto)
				if err != nil {
					ui.Error.Printf("❌ Search failed for track %s: %v\n", trackName, err)
					continue
				}

				if len(selectedItems) == 0 {
					ui.Warning.Printf("⚠️ No results found for track: %s\n", trackName)
					continue
				}

				time.Sleep(500 * time.Millisecond)

				for i, selectedItem := range selectedItems {
					itemType := itemTypes[i]
					if itemType == "track" {
						track := selectedItem.(models.Track)
						ui.Info.Println("🎵 Starting track download for:", track.Title, "by", track.Artist)
						if err := d.DownloadSingleTrack(context.Background(), track, debug, conf.Format, conf.Bitrate, pool, conf, nil); err != nil {
							ui.Error.Printf("❌ Failed to download track %s: %v\n", track.Title, err)
						} else {
							ui.Success.Println("✅ Track download completed for", track.Title)
						}
					}
				}
			}

			if localPool && pool != nil {
				pool.Stop()
			}
		}
	},
}
