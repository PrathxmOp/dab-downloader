package services

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"dab-downloader/internal/api/dab"
	"dab-downloader/internal/api/spotify"
	"dab-downloader/internal/api/navidrome"
	"dab-downloader/internal/config"
	"dab-downloader/internal/core/downloader"
	"dab-downloader/internal/core/search"
	"dab-downloader/internal/core/updater"
	"dab-downloader/internal/interfaces"
	"dab-downloader/internal/shared"
)

// ============================================================================
// 1. Constants and Types
// ============================================================================

const (
	MaxParallelWorkers = 10
	DefaultParallelism = 1
)

// ServiceContainer holds all application services
type ServiceContainer struct {
	Config           interfaces.ConfigService
	APIClient        interfaces.APIClient
	DownloadService  interfaces.DownloadService
	SearchService    interfaces.SearchService
	SpotifyService   interfaces.SpotifyService
	NavidromeService interfaces.NavidromeService
	UpdaterService   interfaces.UpdaterService
	FileSystem       interfaces.FileSystemService
	Logger           interfaces.LoggerService
	WarningCollector interfaces.WarningCollectorService
	Metadata         interfaces.MetadataService
	Conversion       interfaces.ConversionService
}

// ============================================================================
// 2. Service Container Constructor
// ============================================================================

// NewServiceContainer creates a new service container with all services initialized
func NewServiceContainer(cfg *config.Config, httpClient *http.Client) *ServiceContainer {
	// Create core services first
	logger := NewConsoleLogger()
	warningCollector := shared.NewWarningCollector(true)
	fileSystem := NewFileSystemService(cfg)
	
	// Create API clients
	apiClient := dab.NewDabAPI(cfg.APIURL, cfg.DownloadLocation, httpClient)
	spotifyClient := spotify.NewSpotifyClient(cfg.SpotifyClientID, cfg.SpotifyClientSecret)
	navidromeClient := navidrome.NewNavidromeClient(cfg.NavidromeURL, cfg.NavidromeUsername, cfg.NavidromePassword)
	
	// Create business logic services
	configService := NewConfigService()
	downloadService := NewDownloadService(apiClient, fileSystem, logger, warningCollector)
	searchService := NewSearchService(apiClient)
	updaterService := NewUpdaterService(httpClient)
	metadataService := NewMetadataService(warningCollector)
	conversionService := NewConversionService()
	
	return &ServiceContainer{
		Config:           configService,
		APIClient:        apiClient,
		DownloadService:  downloadService,
		SearchService:    searchService,
		SpotifyService:   NewSpotifyServiceWrapper(spotifyClient),
		NavidromeService: NewNavidromeServiceWrapper(navidromeClient),
		UpdaterService:   updaterService,
		FileSystem:       fileSystem,
		Logger:           logger,
		WarningCollector: warningCollector,
		Metadata:         metadataService,
		Conversion:       conversionService,
	}
}

// ============================================================================
// 3. Config Service Implementation
// ============================================================================

type ConfigService struct{}

func NewConfigService() *ConfigService {
	return &ConfigService{}
}

func (cs *ConfigService) LoadConfig(configFile string) (*config.Config, error) {
	cfg := &config.Config{}
	return cfg, config.LoadConfig(configFile, cfg)
}

func (cs *ConfigService) SaveConfig(configFile string, cfg *config.Config) error {
	return config.SaveConfig(configFile, cfg)
}

func (cs *ConfigService) ValidateConfig(cfg *config.Config) error {
	if cfg.APIURL == "" {
		return fmt.Errorf("API URL is required")
	}
	if cfg.DownloadLocation == "" {
		return fmt.Errorf("download location is required")
	}
	return nil
}

func (cs *ConfigService) GetDefaultConfig() *config.Config {
	cfg := &config.Config{
		APIURL:           "https://api.dab.com",
		DownloadLocation: "./downloads",
		Parallelism:      3,
		Format:           "flac",
		Bitrate:          "320",
		SaveAlbumArt:     true,
		VerifyDownloads:  true,
		MaxRetryAttempts: 3,
		WarningBehavior:  "display",
	}
	cfg.ApplyDefaultNamingMasks()
	return cfg
}

func (cs *ConfigService) EnsureConfigExists(configFile string) error {
	if !shared.FileExists(configFile) {
		defaultConfig := cs.GetDefaultConfig()
		return cs.SaveConfig(configFile, defaultConfig)
	}
	return nil
}

// ============================================================================
// 4. Download Service Implementation
// ============================================================================

type DownloadService struct {
	apiClient        *dab.DabAPI
	fileSystem       *FileSystemService
	logger           interfaces.LoggerService
	warningCollector *shared.WarningCollector
	downloader       *downloader.TrackDownloader
}

func NewDownloadService(apiClient interfaces.APIClient, fileSystem interfaces.FileSystemService, logger interfaces.LoggerService, warningCollector interfaces.WarningCollectorService) *DownloadService {
	dabAPI := apiClient.(*dab.DabAPI)
	fileSystemService := fileSystem.(*FileSystemService)
	warningCollectorService := warningCollector.(*shared.WarningCollector)
	
	// Create a track downloader instance
	trackDownloader := downloader.NewTrackDownloader(dabAPI, fileSystemService.config)
	
	return &DownloadService{
		apiClient:        dabAPI,
		fileSystem:       fileSystemService,
		logger:           logger,
		warningCollector: warningCollectorService,
		downloader:       trackDownloader,
	}
}

// ============================================================================
// 4.1 Public Download Methods
// ============================================================================

func (ds *DownloadService) DownloadAlbum(ctx context.Context, albumID string, cfg *config.Config, debug bool, format string, bitrate string) (*shared.DownloadStats, error) {
	album, err := ds.apiClient.GetAlbum(ctx, albumID)
	if err != nil {
		return nil, fmt.Errorf("failed to get album: %w", err)
	}
	
	return ds.DownloadTracks(ctx, album.Tracks, album, cfg, debug, format, bitrate)
}

func (ds *DownloadService) DownloadArtist(ctx context.Context, artistID string, cfg *config.Config, debug bool, format string, bitrate string, filter string, noConfirm bool) (*shared.DownloadStats, error) {
	ds.apiClient.SetDebugMode(debug)
	artist, err := ds.apiClient.GetArtist(ctx, artistID, cfg, debug)
	if err != nil {
		return nil, fmt.Errorf("failed to get artist: %w", err)
	}
	
	// Check if user explicitly provided a filter
	var filteredAlbums []shared.Album
	var usedCustomSelection bool
	
	// If a specific filter was provided, use it directly
	if filter != "" && filter != "all" {
		filteredAlbums = ds.filterAlbumsByType(artist.Albums, filter)
	} else {
		// Present menu options to user
		selectedFilter, cancelled := ds.presentDownloadMenu(artist.Albums)
		if cancelled {
			return &shared.DownloadStats{}, shared.ErrDownloadCancelled
		}
		
		if selectedFilter == "custom" {
			// Use existing custom selection logic
			filteredAlbums = ds.selectCustomAlbums(artist.Albums)
			usedCustomSelection = true
			if len(filteredAlbums) == 0 {
				return &shared.DownloadStats{}, shared.ErrNoItemsSelected
			}
		} else {
			// Simulate custom selection for menu options 1-4
			filteredAlbums = ds.simulateCustomSelection(artist.Albums, selectedFilter)
			usedCustomSelection = true
			if len(filteredAlbums) == 0 {
				return &shared.DownloadStats{}, shared.ErrNoItemsSelected
			}
		}
		
		// For all menu-based selections, use individual feedback mode
		if debug {
			ds.logger.Debug("DEBUG: Using menu-based selection, downloading %d albums with individual feedback", len(filteredAlbums))
		}
		stats := ds.downloadAlbumsUnified(ctx, filteredAlbums, cfg, debug, format, bitrate, true)
		if debug && stats != nil {
			ds.logger.Debug("DEBUG: Download completed - Success: %d, Failed: %d, Skipped: %d", stats.SuccessCount, stats.FailedCount, stats.SkippedCount)
		}
		return stats, nil
	}
	
	if len(filteredAlbums) == 0 {
		return &shared.DownloadStats{}, nil
	}
	
	// Skip confirmation if we already did custom selection or if noConfirm is set
	if !noConfirm && !usedCustomSelection {
		if !ds.confirmDownload(filteredAlbums) {
			return &shared.DownloadStats{}, shared.ErrDownloadCancelled
		}
	}
	
	stats := ds.downloadAlbumsUnified(ctx, filteredAlbums, cfg, debug, format, bitrate, false)
	return stats, nil
}

func (ds *DownloadService) DownloadTrack(ctx context.Context, trackID string, cfg *config.Config, debug bool, format string, bitrate string) (*shared.DownloadStats, error) {
	track, err := ds.apiClient.GetTrack(ctx, trackID)
	if err != nil {
		return nil, fmt.Errorf("failed to get track: %w", err)
	}
	
	album := ds.createMinimalAlbum(track)
	return ds.DownloadTracks(ctx, []shared.Track{*track}, album, cfg, debug, format, bitrate)
}

func (ds *DownloadService) DownloadTrackDirect(ctx context.Context, track shared.Track, cfg *config.Config, debug bool, format string, bitrate string) (*shared.DownloadStats, error) {
	album := ds.createMinimalAlbumFromTrack(track)
	return ds.DownloadTracks(ctx, []shared.Track{track}, album, cfg, debug, format, bitrate)
}

func (ds *DownloadService) DownloadTracks(ctx context.Context, tracks []shared.Track, album *shared.Album, cfg *config.Config, debug bool, format string, bitrate string) (*shared.DownloadStats, error) {
	ds.downloader.SetDebugMode(debug)
	ds.logger.SetDebugMode(debug)
	ds.apiClient.SetDebugMode(debug)
	
	// Pre-populate MusicBrainz metadata
	if album != nil && len(tracks) > 0 {
		downloader.FindReleaseIDFromISRC(tracks, album.Artist, album.Title)
	}
	
	// Pre-fetch metadata if using parallelism
	if ds.shouldPrefetchMetadata(cfg) {
		ds.prefetchMetadata(ctx, tracks, album, cfg, debug)
	}
	
	// Download cover art
	coverData := ds.downloadCoverArt(ctx, album)
	
	// Save cover art as cover.jpg if configured to do so
	if err := ds.saveCoverArtToFile(coverData, album, cfg); err != nil {
		ds.logger.Warning("Failed to save cover art file: %v", err)
	}
	
	// Download tracks with appropriate parallelism
	return ds.downloadTracksWithParallelism(ctx, tracks, album, coverData, cfg, debug, format, bitrate)
}

// ============================================================================
// 4.2 Info Retrieval Methods
// ============================================================================

func (ds *DownloadService) GetArtistInfo(ctx context.Context, artistID string, cfg *config.Config, debug bool) (*shared.Artist, error) {
	ds.apiClient.SetDebugMode(debug)
	artist, err := ds.apiClient.GetArtist(ctx, artistID, cfg, debug)
	if err != nil {
		return nil, fmt.Errorf("failed to get artist: %w", err)
	}
	return artist, nil
}

func (ds *DownloadService) GetAlbumInfo(ctx context.Context, albumID string, cfg *config.Config, debug bool) (*shared.Album, error) {
	ds.apiClient.SetDebugMode(debug)
	album, err := ds.apiClient.GetAlbum(ctx, albumID)
	if err != nil {
		return nil, fmt.Errorf("failed to get album: %w", err)
	}
	return album, nil
}

// ============================================================================
// 4.3 Private Helper Methods
// ============================================================================

func (ds *DownloadService) shouldPrefetchMetadata(cfg *config.Config) bool {
	return cfg != nil && cfg.Parallelism > 1
}

func (ds *DownloadService) getParallelism(cfg *config.Config) int {
	if cfg == nil || cfg.Parallelism <= 0 {
		return DefaultParallelism
	}
	
	parallelism := cfg.Parallelism
	if parallelism > MaxParallelWorkers {
		parallelism = MaxParallelWorkers
	}
	
	return parallelism
}

func (ds *DownloadService) presentDownloadMenu(albums []shared.Album) (string, bool) {
	// Count albums by type
	albumCount := 0
	epCount := 0
	singleCount := 0
	
	for _, album := range albums {
		switch strings.ToLower(album.Type) {
		case "album":
			albumCount++
		case "ep":
			epCount++
		case "single":
			singleCount++
		}
	}
	
	// Display album summary
	ds.logger.Info("Found %d albums:", len(albums))
	
	// Group and sort albums by type for display
	groupedAlbums := ds.groupAndSortAlbums(albums)
	
	// Display albums grouped by type with improved formatting
	counter := 1
	for _, albumType := range []string{"ALBUM", "EP", "SINGLE"} {
		albumsOfType := groupedAlbums[albumType]
		if len(albumsOfType) > 0 {
			// Display section header with better styling
			fmt.Printf("\n")
			shared.ColorInfo.Printf("┌─ %ss ", albumType)
			shared.ColorWarning.Printf("(%d)", len(albumsOfType))
			fmt.Printf("\n")
			
			for _, album := range albumsOfType {
				// Get track count
				trackCount := ds.getAlbumTrackCount(album)
				
				// Format track count with blue color and proper alignment
				var trackCountStr string
				if trackCount == 1 {
					trackCountStr = shared.ColorPrompt.Sprintf("[ 1 Track ]")
				} else if trackCount > 0 {
					if trackCount < 10 {
						trackCountStr = shared.ColorPrompt.Sprintf("[ %d Tracks]", trackCount)
					} else {
						trackCountStr = shared.ColorPrompt.Sprintf("[%d Tracks]", trackCount)
					}
				} else {
					trackCountStr = shared.ColorPrompt.Sprintf("[ ? Tracks]")
				}
				
				// Create a more professional display format
				prefix := fmt.Sprintf("│ %2d. ", counter)
				formattedLine := shared.FormatAlbumWithTrackCountProfessional(prefix, album.Title, album.Artist, album.ReleaseDate, trackCountStr, album.AudioQuality)
				fmt.Println(formattedLine)
				counter++
			}
		}
	}
	
	// Add a bottom border
	if counter > 1 {
		termWidth := shared.GetTerminalWidth()
		if termWidth < 80 {
			termWidth = 80
		}
		borderLine := "└" + strings.Repeat("─", termWidth-2) + "\n"
		fmt.Printf(borderLine)
	}
	
	fmt.Println("\nWhat would you like to download?")
	fmt.Printf("1) Everything (albums + EPs + singles)\n")
	fmt.Printf("2) Only albums (%d)\n", albumCount)
	fmt.Printf("3) Only EPs (%d)\n", epCount)
	fmt.Printf("4) Only singles (%d)\n", singleCount)
	fmt.Printf("5) Custom selection\n")
	
	for {
		choice := shared.GetUserInput("Choose option (1-5, or q to quit)", "1")
		
		switch strings.ToLower(choice) {
		case "q", "quit":
			return "", true
		case "1", "":
			return "all", false
		case "2":
			if albumCount == 0 {
				shared.ColorError.Printf("❌ No albums found for this artist.\n")
				continue
			}
			return "albums", false
		case "3":
			if epCount == 0 {
				shared.ColorError.Printf("❌ No EPs found for this artist.\n")
				continue
			}
			return "eps", false
		case "4":
			if singleCount == 0 {
				shared.ColorError.Printf("❌ No singles found for this artist.\n")
				continue
			}
			return "singles", false
		case "5":
			return "custom", false
		default:
			shared.ColorError.Printf("❌ Invalid option. Please choose 1-5 or q to quit.\n")
		}
	}
}

func (ds *DownloadService) selectCustomAlbums(albums []shared.Album) []shared.Album {
	if len(albums) == 0 {
		return albums
	}
	
	// Create the same display order as shown in presentDownloadMenu
	groupedAlbums := ds.groupAndSortAlbums(albums)
	var displayOrderAlbums []shared.Album
	
	// Build display order (same as in presentDownloadMenu)
	for _, albumType := range []string{"ALBUM", "EP", "SINGLE"} {
		albumsOfType := groupedAlbums[albumType]
		for _, album := range albumsOfType {
			displayOrderAlbums = append(displayOrderAlbums, album)
		}
	}
	
	// Don't show the list again - it was already shown in presentDownloadMenu
	for {
		input := shared.GetUserInput("Enter numbers to download (e.g., '1,3,5-7' or 'q' to quit)", "")
		if input == "" {
			shared.ColorError.Printf("❌ Please enter a selection or 'q' to quit.\n")
			continue
		}
		
		if strings.ToLower(input) == "q" || strings.ToLower(input) == "quit" {
			return []shared.Album{}
		}
		
		selectedIndices, err := shared.ParseSelectionInput(input, len(displayOrderAlbums))
		if err != nil {
			shared.ColorError.Printf("❌ Invalid selection: %v\n", err)
			continue
		}
		
		if len(selectedIndices) == 0 {
			shared.ColorError.Printf("❌ No valid albums selected.\n")
			continue
		}
		
		// Convert indices to albums using display order (indices are 1-based)
		var selectedAlbums []shared.Album
		for _, index := range selectedIndices {
			if index >= 1 && index <= len(displayOrderAlbums) {
				selectedAlbums = append(selectedAlbums, displayOrderAlbums[index-1])
			}
		}
		
		return selectedAlbums
	}
}

func (ds *DownloadService) confirmDownload(albums []shared.Album) bool {
	ds.logger.Info("Found %d albums to download:", len(albums))
	for i, album := range albums {
		prefix := fmt.Sprintf("%d. [%s] ", i+1, strings.ToUpper(album.Type))
		formattedLine := shared.FormatAlbumWithBitrate(prefix, album.Title, album.Artist, album.ReleaseDate, album.AudioQuality)
		fmt.Println(formattedLine)
	}
	
	confirmation := shared.GetUserInput("Continue with download? (y/n)", "y")
	return strings.ToLower(confirmation) == "y" || strings.ToLower(confirmation) == "yes"
}

// groupAndSortAlbums groups albums by type and sorts them alphabetically within each group
func (ds *DownloadService) groupAndSortAlbums(albums []shared.Album) map[string][]shared.Album {
	grouped := make(map[string][]shared.Album)
	
	for _, album := range albums {
		albumType := strings.ToUpper(album.Type)
		// Normalize type names
		switch albumType {
		case "ALBUM", "LP", "":
			albumType = "ALBUM"
		case "EP":
			albumType = "EP"
		case "SINGLE":
			albumType = "SINGLE"
		default:
			// Default unknown types to ALBUM
			albumType = "ALBUM"
		}
		grouped[albumType] = append(grouped[albumType], album)
	}
	
	// Sort each group alphabetically by title
	for albumType := range grouped {
		albums := grouped[albumType]
		sort.Slice(albums, func(i, j int) bool {
			return strings.ToLower(albums[i].Title) < strings.ToLower(albums[j].Title)
		})
		grouped[albumType] = albums
	}
	
	return grouped
}

// getAlbumTrackCount attempts to get the track count for an album
func (ds *DownloadService) getAlbumTrackCount(album shared.Album) int {
	// First, try TotalTracks field
	if album.TotalTracks > 0 {
		return album.TotalTracks
	}
	
	// Second, try counting the Tracks slice if it's populated
	if len(album.Tracks) > 0 {
		return len(album.Tracks)
	}
	
	// If all else fails, return 0 (will display as "? Tracks")
	return 0
}

// displayDownloadSummary shows a summary of the download results
func (ds *DownloadService) displayDownloadSummary(stats *shared.DownloadStats, albums []shared.Album, cfg *config.Config) {
	if stats == nil || (stats.SuccessCount == 0 && stats.FailedCount == 0 && stats.SkippedCount == 0) {
		return
	}
	
	// Get artist name from the first album
	artistName := "Unknown Artist"
	if len(albums) > 0 {
		artistName = albums[0].Artist
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
	shared.ColorSuccess.Printf("📁 Artist discography downloaded to: %s\n", cfg.DownloadLocation)
	shared.ColorSuccess.Printf("🎉 Discography download completed for %s\n", artistName)
}

// simulateCustomSelection automatically selects albums based on filter type
func (ds *DownloadService) simulateCustomSelection(albums []shared.Album, filter string) []shared.Album {
	if len(albums) == 0 {
		return albums
	}
	
	// Create the same grouped and sorted display order as shown in the menu
	groupedAlbums := ds.groupAndSortAlbums(albums)
	var displayOrderAlbums []shared.Album
	
	// Build display order (same as in presentDownloadMenu)
	for _, albumType := range []string{"ALBUM", "EP", "SINGLE"} {
		albumsOfType := groupedAlbums[albumType]
		for _, album := range albumsOfType {
			displayOrderAlbums = append(displayOrderAlbums, album)
		}
	}
	
	// Filter based on the selected option
	var selectedAlbums []shared.Album
	for _, album := range displayOrderAlbums {
		albumType := strings.ToLower(album.Type)
		
		switch filter {
		case "all":
			selectedAlbums = append(selectedAlbums, album)
		case "albums":
			if albumType == "album" {
				selectedAlbums = append(selectedAlbums, album)
			}
		case "eps":
			if albumType == "ep" {
				selectedAlbums = append(selectedAlbums, album)
			}
		case "singles":
			if albumType == "single" {
				selectedAlbums = append(selectedAlbums, album)
			}
		}
	}
	
	return selectedAlbums
}

// downloadAlbumsUnified is the single method for downloading multiple albums
// individualFeedback: true = show individual album start/complete messages (like search command)
//                    false = use bulk download approach (like discography command)
func (ds *DownloadService) downloadAlbumsUnified(ctx context.Context, albums []shared.Album, cfg *config.Config, debug bool, format string, bitrate string, individualFeedback bool) *shared.DownloadStats {
	if individualFeedback {
		// Individual feedback mode - show start/complete message for each album
		totalStats := &shared.DownloadStats{}
		
		for _, album := range albums {
			ds.logger.Info("🎵 Starting album download for: %s by %s", album.Title, album.Artist)
			albumStats, err := ds.DownloadAlbum(ctx, album.ID, cfg, debug, format, bitrate)
			if err != nil {
				ds.logger.Error("❌ Failed to download album %s: %v", album.Title, err)
				totalStats.FailedCount++
				totalStats.FailedItems = append(totalStats.FailedItems, album.Title)
				continue
			}
			
			// Merge stats
			if albumStats != nil {
				totalStats.SuccessCount += albumStats.SuccessCount
				totalStats.FailedCount += albumStats.FailedCount
				totalStats.SkippedCount += albumStats.SkippedCount
				totalStats.FailedItems = append(totalStats.FailedItems, albumStats.FailedItems...)
			}
			
			ds.logger.Success("✅ Album download completed for %s", album.Title)
		}
		
		return totalStats
	} else {
		// Bulk mode - use parallelism without individual messages
		maxWorkers := ds.getParallelism(cfg)
		
		if debug {
			ds.logger.Debug("Using parallelism setting: %d workers for %d albums", maxWorkers, len(albums))
		}
		
		if len(albums) == 1 || maxWorkers == 1 {
			// Sequential download for single album or single worker
			totalStats := &shared.DownloadStats{}
			
			for _, album := range albums {
				albumStats, err := ds.DownloadAlbum(ctx, album.ID, cfg, debug, format, bitrate)
				if err != nil {
					ds.logger.Error("Failed to download album %s: %v", album.Title, err)
					totalStats.FailedCount++
					totalStats.FailedItems = append(totalStats.FailedItems, album.Title)
					continue
				}
				
				if albumStats != nil {
					totalStats.SuccessCount += albumStats.SuccessCount
					totalStats.FailedCount += albumStats.FailedCount
					totalStats.SkippedCount += albumStats.SkippedCount
					totalStats.FailedItems = append(totalStats.FailedItems, albumStats.FailedItems...)
				}
			}
			
			return totalStats
		} else {
			// Parallel download
			totalStats := &shared.DownloadStats{}
			albumChan := make(chan shared.Album, len(albums))
			statsChan := make(chan *shared.DownloadStats, len(albums))
			
			// Start workers
			for i := 0; i < maxWorkers; i++ {
				go func() {
					for album := range albumChan {
						stats, err := ds.DownloadAlbum(ctx, album.ID, cfg, debug, format, bitrate)
						if err != nil {
							stats = &shared.DownloadStats{
								FailedCount: 1,
								FailedItems: []string{album.Title},
							}
						}
						statsChan <- stats
					}
				}()
			}
			
			// Send albums to workers
			go func() {
				defer close(albumChan)
				for _, album := range albums {
					albumChan <- album
				}
			}()
			
			// Collect results
			for i := 0; i < len(albums); i++ {
				stats := <-statsChan
				if stats != nil {
					totalStats.SuccessCount += stats.SuccessCount
					totalStats.FailedCount += stats.FailedCount
					totalStats.SkippedCount += stats.SkippedCount
					totalStats.FailedItems = append(totalStats.FailedItems, stats.FailedItems...)
				}
			}
			
			return totalStats
		}
	}
}



func (ds *DownloadService) createMinimalAlbum(track *shared.Track) *shared.Album {
	return &shared.Album{
		ID:     track.AlbumID,
		Title:  track.Album,
		Artist: track.AlbumArtist,
		Tracks: []shared.Track{*track},
	}
}

func (ds *DownloadService) createMinimalAlbumFromTrack(track shared.Track) *shared.Album {
	albumTitle := track.AlbumTitle
	if albumTitle == "" {
		albumTitle = track.Album
	}
	
	albumArtist := track.AlbumArtist
	if albumArtist == "" {
		albumArtist = track.Artist
	}
	
	return &shared.Album{
		ID:     track.AlbumID,
		Title:  albumTitle,
		Artist: albumArtist,
		Tracks: []shared.Track{track},
	}
}

func (ds *DownloadService) downloadCoverArt(ctx context.Context, album *shared.Album) []byte {
	if album == nil || album.Cover == "" {
		return nil
	}
	
	coverData, err := ds.apiClient.DownloadCover(ctx, album.Cover)
	if err != nil {
		ds.logger.Warning("Failed to download cover art: %v", err)
		return nil
	}
	
	return coverData
}

// saveCoverArtToFile saves cover art data as cover.jpg in the album directory if SaveAlbumArt is enabled
func (ds *DownloadService) saveCoverArtToFile(coverData []byte, album *shared.Album, cfg *config.Config) error {
	if coverData == nil || len(coverData) == 0 || album == nil {
		return nil // Nothing to save
	}
	
	// Check if saving album art is enabled in config
	if cfg == nil || !cfg.SaveAlbumArt {
		return nil // Album art saving is disabled
	}

	// Create a dummy track to use the naming system to get the album directory
	dummyTrack := shared.Track{
		Album:       album.Title,
		AlbumArtist: album.Artist,
		Artist:      album.Artist,
		Title:       "dummy",
		TrackNumber: 1,
	}

	// Get the album directory path by getting a track path and removing the filename
	trackPath := ds.fileSystem.GetDownloadPathWithTrack(dummyTrack, album, "flac", cfg)
	albumDir := filepath.Dir(trackPath)

	// Ensure the album directory exists
	if err := ds.fileSystem.EnsureDirectoryExists(albumDir); err != nil {
		return fmt.Errorf("failed to create album directory: %w", err)
	}

	// Define the cover art file path
	coverPath := filepath.Join(albumDir, "cover.jpg")

	// Check if cover.jpg already exists
	if ds.fileSystem.FileExists(coverPath) {
		return nil // Cover art already exists, skip
	}

	// Write the cover art data to cover.jpg
	if err := os.WriteFile(coverPath, coverData, 0644); err != nil {
		return fmt.Errorf("failed to write cover art file: %w", err)
	}

	return nil
}

func (ds *DownloadService) prefetchMetadata(ctx context.Context, tracks []shared.Track, album *shared.Album, cfg *config.Config, debug bool) {
	maxWorkers := ds.getParallelism(cfg)
	
	if debug {
		ds.logger.Debug("Pre-fetching metadata for %d tracks using %d workers", len(tracks), maxWorkers)
	}
	
	// Pre-fetch track-specific metadata in parallel
	ds.prefetchTrackMetadata(ctx, tracks, album, maxWorkers, debug)
}



func (ds *DownloadService) downloadTracksWithParallelism(ctx context.Context, tracks []shared.Track, album *shared.Album, coverData []byte, cfg *config.Config, debug bool, format string, bitrate string) (*shared.DownloadStats, error) {
	stats := &shared.DownloadStats{}
	trackWorkers := ds.getParallelism(cfg)
	
	if debug && len(tracks) > 1 {
		ds.logger.Debug("Using parallelism setting: %d workers for %d tracks in album '%s'", trackWorkers, len(tracks), album.Title)
	}
	
	if len(tracks) == 1 || trackWorkers == 1 {
		ds.downloadTracksSequentially(ctx, tracks, album, coverData, cfg, debug, format, bitrate, stats)
	} else {
		ds.downloadTracksParallel(ctx, tracks, album, coverData, cfg, debug, format, bitrate, trackWorkers, stats)
	}
	
	return stats, nil
}

func (ds *DownloadService) filterAlbumsByType(albums []shared.Album, filter string) []shared.Album {
	if filter == "all" || filter == "" {
		return albums
	}
	
	filters := strings.Split(strings.ToLower(filter), ",")
	var filtered []shared.Album
	
	for _, album := range albums {
		albumType := strings.ToLower(album.Type)
		for _, f := range filters {
			f = strings.TrimSpace(f)
			if ds.matchesFilter(f, albumType) {
				filtered = append(filtered, album)
				break
			}
		}
	}
	
	return filtered
}

func (ds *DownloadService) matchesFilter(filter, albumType string) bool {
	return filter == albumType ||
		(filter == "albums" && albumType == "album") ||
		(filter == "eps" && albumType == "ep") ||
		(filter == "singles" && albumType == "single")
}

// ============================================================================
// 4.4 Parallel Processing Methods
// ============================================================================





func (ds *DownloadService) downloadTracksSequentially(ctx context.Context, tracks []shared.Track, album *shared.Album, coverData []byte, cfg *config.Config, debug bool, format string, bitrate string, stats *shared.DownloadStats) {
	for _, track := range tracks {
		outputPath := ds.fileSystem.GetDownloadPathWithTrack(track, album, format, cfg)
		
		if ds.fileSystem.FileExists(outputPath) {
			ds.logger.Info("Skipping %s - already exists", track.Title)
			stats.SkippedCount++
			continue
		}
		
		_, err := downloader.DownloadTrack(ctx, ds.apiClient, track, album, outputPath, coverData, nil, debug, format, bitrate, cfg, ds.warningCollector)
		if err != nil {
			ds.logger.Error("Failed to download %s: %v", track.Title, err)
			stats.FailedCount++
			stats.FailedItems = append(stats.FailedItems, track.Title)
			continue
		}
		
		stats.SuccessCount++
	}
}

func (ds *DownloadService) downloadTracksParallel(ctx context.Context, tracks []shared.Track, album *shared.Album, coverData []byte, cfg *config.Config, debug bool, format string, bitrate string, maxWorkers int, stats *shared.DownloadStats) {
	trackJobs := make(chan shared.Track, len(tracks))
	results := make(chan trackDownloadResult, len(tracks))
	
	// Start worker goroutines
	for i := 0; i < maxWorkers; i++ {
		go ds.trackWorker(ctx, i, trackJobs, results, album, coverData, cfg, debug, format, bitrate)
	}
	
	// Send jobs
	for _, track := range tracks {
		trackJobs <- track
	}
	close(trackJobs)
	
	// Collect results
	for i := 0; i < len(tracks); i++ {
		result := <-results
		ds.updateStatsFromResult(stats, result)
	}
}

func (ds *DownloadService) trackWorker(ctx context.Context, workerID int, jobs <-chan shared.Track, results chan<- trackDownloadResult, album *shared.Album, coverData []byte, cfg *config.Config, debug bool, format string, bitrate string) {
	for track := range jobs {
		result := trackDownloadResult{track: track}
		
		outputPath := ds.fileSystem.GetDownloadPathWithTrack(track, album, format, cfg)
		
		if ds.fileSystem.FileExists(outputPath) {
			if debug {
				ds.logger.Debug("Worker %d: Skipping %s - already exists", workerID, track.Title)
			}
			result.skipped = true
			results <- result
			continue
		}
		
		_, err := downloader.DownloadTrack(ctx, ds.apiClient, track, album, outputPath, coverData, nil, debug, format, bitrate, cfg, ds.warningCollector)
		if err != nil {
			if debug {
				ds.logger.Error("Worker %d: Failed to download %s: %v", workerID, track.Title, err)
			}
			result.err = err
		} else {
			result.success = true
			if debug {
				ds.logger.Debug("Worker %d: Successfully downloaded %s", workerID, track.Title)
			}
		}
		
		results <- result
	}
}

func (ds *DownloadService) prefetchTrackMetadata(ctx context.Context, tracks []shared.Track, album *shared.Album, maxWorkers int, debug bool) {
	trackJobs := make(chan shared.Track, len(tracks))
	results := make(chan bool, len(tracks))
	
	// Start worker goroutines for metadata fetching
	for i := 0; i < maxWorkers; i++ {
		go ds.metadataWorker(ctx, i, trackJobs, results, tracks, album, debug)
	}
	
	// Send jobs
	for _, track := range tracks {
		trackJobs <- track
	}
	close(trackJobs)
	
	// Wait for completion
	for i := 0; i < len(tracks); i++ {
		<-results
	}
	
	if debug {
		ds.logger.Debug("Metadata pre-fetching completed for %d tracks", len(tracks))
	}
}

func (ds *DownloadService) metadataWorker(ctx context.Context, workerID int, jobs <-chan shared.Track, results chan<- bool, tracks []shared.Track, album *shared.Album, debug bool) {
	for track := range jobs {
		if debug {
			ds.logger.Debug("Metadata Worker %d: Pre-fetching metadata for %s", workerID, track.Title)
		}
		
		if track.ISRC != "" {
			expectedTrackCount := len(tracks)
			if album != nil && album.TotalTracks > 0 {
				expectedTrackCount = album.TotalTracks
			}
			
			_, err := downloader.GetISRCMetadataWithTrackCount(track.ISRC, expectedTrackCount)
			if err != nil && debug {
				ds.logger.Debug("Metadata Worker %d: ISRC lookup failed for %s: %v", workerID, track.Title, err)
			}
		}
		
		results <- true
	}
}

// ============================================================================
// 4.5 Utility Methods
// ============================================================================

type trackDownloadResult struct {
	track   shared.Track
	success bool
	skipped bool
	err     error
}

func (ds *DownloadService) mergeStats(total, addition *shared.DownloadStats) {
	total.SuccessCount += addition.SuccessCount
	total.SkippedCount += addition.SkippedCount
	total.FailedCount += addition.FailedCount
	total.FailedItems = append(total.FailedItems, addition.FailedItems...)
}

func (ds *DownloadService) updateStatsFromResult(stats *shared.DownloadStats, result trackDownloadResult) {
	if result.skipped {
		stats.SkippedCount++
	} else if result.success {
		stats.SuccessCount++
	} else {
		stats.FailedCount++
		stats.FailedItems = append(stats.FailedItems, result.track.Title)
	}
}

// ============================================================================
// 5. Search Service Implementation
// ============================================================================

type SearchService struct {
	apiClient interfaces.APIClient
}

func NewSearchService(apiClient interfaces.APIClient) *SearchService {
	return &SearchService{
		apiClient: apiClient,
	}
}

func (ss *SearchService) HandleSearch(ctx context.Context, query string, searchType string, debug bool, auto bool, cfg *config.Config) ([]interface{}, []string, error) {
	return search.HandleSearch(ctx, ss.apiClient.(*dab.DabAPI), query, searchType, debug, auto, cfg)
}

func (ss *SearchService) Search(ctx context.Context, query string, searchType string, limit int, debug bool) (*shared.SearchResults, error) {
	return ss.apiClient.Search(ctx, query, searchType, limit, debug)
}

// ============================================================================
// 6. Spotify Service Wrapper
// ============================================================================

type SpotifyServiceWrapper struct {
	client *spotify.SpotifyClient
}

func NewSpotifyServiceWrapper(client *spotify.SpotifyClient) *SpotifyServiceWrapper {
	return &SpotifyServiceWrapper{client: client}
}

func (ssw *SpotifyServiceWrapper) Authenticate() error {
	return ssw.client.Authenticate()
}

func (ssw *SpotifyServiceWrapper) GetPlaylistTracks(playlistURL string) ([]shared.SpotifyTrack, string, error) {
	tracks, name, err := ssw.client.GetPlaylistTracks(playlistURL)
	if err != nil {
		return nil, "", err
	}
	
	return ssw.convertSpotifyTracks(tracks), name, nil
}

func (ssw *SpotifyServiceWrapper) GetAlbumTracks(albumURL string) ([]shared.SpotifyTrack, string, error) {
	tracks, name, err := ssw.client.GetAlbumTracks(albumURL)
	if err != nil {
		return nil, "", err
	}
	
	return ssw.convertSpotifyTracks(tracks), name, nil
}

func (ssw *SpotifyServiceWrapper) convertSpotifyTracks(tracks []spotify.SpotifyTrack) []shared.SpotifyTrack {
	sharedTracks := make([]shared.SpotifyTrack, len(tracks))
	for i, track := range tracks {
		sharedTracks[i] = shared.SpotifyTrack{
			Name:        track.Name,
			Artist:      track.Artist,
			AlbumName:   track.AlbumName,
			AlbumArtist: track.AlbumArtist,
		}
	}
	return sharedTracks
}

// ============================================================================
// 7. Navidrome Service Wrapper
// ============================================================================

type NavidromeServiceWrapper struct {
	client *navidrome.NavidromeClient
}

func NewNavidromeServiceWrapper(client *navidrome.NavidromeClient) *NavidromeServiceWrapper {
	return &NavidromeServiceWrapper{client: client}
}

func (nsw *NavidromeServiceWrapper) Authenticate() error {
	return fmt.Errorf("Navidrome authentication not implemented")
}

func (nsw *NavidromeServiceWrapper) CreatePlaylist(name string, tracks []shared.Track) error {
	return fmt.Errorf("Navidrome playlist creation not implemented")
}

func (nsw *NavidromeServiceWrapper) GetPlaylists() ([]shared.NavidromePlaylist, error) {
	return nil, fmt.Errorf("Navidrome playlist retrieval not implemented")
}

func (nsw *NavidromeServiceWrapper) AddTracksToPlaylist(playlistID string, tracks []shared.Track) error {
	return fmt.Errorf("Navidrome add tracks not implemented")
}

// ============================================================================
// 8. Updater Service Implementation
// ============================================================================

type UpdaterService struct {
	httpClient *http.Client
}

func NewUpdaterService(httpClient *http.Client) *UpdaterService {
	return &UpdaterService{httpClient: httpClient}
}

func (us *UpdaterService) CheckForUpdates(ctx context.Context, currentVersion string, updateRepo string) (*shared.UpdateInfo, error) {
	cfg := &config.Config{
		UpdateRepo:          updateRepo,
		DisableUpdateCheck: false,
	}
	
	updater.CheckForUpdates(cfg, currentVersion)
	
	return &shared.UpdateInfo{
		Version:     "unknown",
		DownloadURL: "",
		ReleaseDate: "",
		Notes:       "",
	}, fmt.Errorf("update check not fully implemented in service layer")
}

func (us *UpdaterService) DownloadUpdate(ctx context.Context, updateInfo *shared.UpdateInfo, outputPath string) error {
	return fmt.Errorf("update download not implemented")
}

func (us *UpdaterService) ApplyUpdate(updatePath string, currentBinaryPath string) error {
	return fmt.Errorf("update application not implemented")
}

// ============================================================================
// 9. File System Service Implementation
// ============================================================================

type FileSystemService struct {
	config *config.Config
}

func NewFileSystemService(cfg *config.Config) *FileSystemService {
	return &FileSystemService{config: cfg}
}

func (fss *FileSystemService) EnsureDirectoryExists(path string) error {
	return config.CreateDirIfNotExists(path)
}

func (fss *FileSystemService) FileExists(path string) bool {
	return shared.FileExists(path)
}

func (fss *FileSystemService) GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (fss *FileSystemService) ValidateDownloadLocation(path string) error {
	if err := fss.EnsureDirectoryExists(path); err != nil {
		return fmt.Errorf("cannot create download directory: %w", err)
	}
	
	testFile := filepath.Join(path, ".write_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("download directory is not writable: %w", err)
	}
	
	os.Remove(testFile)
	return nil
}

func (fss *FileSystemService) SanitizeFileName(filename string) string {
	return shared.SanitizeFileName(filename)
}

func (fss *FileSystemService) GetDownloadPath(artist, album, track string, format string, cfg *config.Config) string {
	ext := ".flac"
	if format != "flac" {
		ext = "." + format
	}
	
	if cfg.NamingMasks.FileMask != "" {
		trackFileName := fss.SanitizeFileName(track) + ext
		albumDir := filepath.Join(cfg.DownloadLocation, fss.SanitizeFileName(artist), fss.SanitizeFileName(album))
		return filepath.Join(albumDir, trackFileName)
	}
	
	// Default structure
	artist = fss.SanitizeFileName(artist)
	album = fss.SanitizeFileName(album)
	track = fss.SanitizeFileName(track)
	
	artistDir := filepath.Join(cfg.DownloadLocation, artist)
	albumDir := filepath.Join(artistDir, album)
	trackFileName := track + ext
	
	return filepath.Join(albumDir, trackFileName)
}

func (fss *FileSystemService) GetDownloadPathWithTrack(track shared.Track, album *shared.Album, format string, cfg *config.Config) string {
	ext := ".flac"
	if format != "flac" {
		ext = "." + format
	}
	
	cfg.ApplyDefaultNamingMasks()
	
	fileName := fss.ProcessNamingMaskForFile(cfg.NamingMasks.FileMask, track, album) + ext
	
	var folderMask string
	if album != nil {
		switch strings.ToLower(album.Type) {
		case "ep":
			folderMask = cfg.NamingMasks.EpFolderMask
		case "single":
			folderMask = cfg.NamingMasks.SingleFolderMask
		default:
			folderMask = cfg.NamingMasks.AlbumFolderMask
		}
	}
	
	if folderMask == "" {
		folderMask = cfg.NamingMasks.AlbumFolderMask
	}
	
	folderPath := fss.ProcessNamingMaskForFolder(folderMask, track, album)
	fullFolderPath := filepath.Join(cfg.DownloadLocation, folderPath)
	
	return filepath.Join(fullFolderPath, fileName)
}

func (fss *FileSystemService) ProcessNamingMask(mask string, track shared.Track, album *shared.Album) string {
	if mask == "" {
		return ""
	}
	
	result := mask
	
	// Replace track-specific placeholders
	result = strings.ReplaceAll(result, "{title}", track.Title)
	result = strings.ReplaceAll(result, "{artist}", track.Artist)
	result = strings.ReplaceAll(result, "{track_number}", fmt.Sprintf("%02d", track.TrackNumber))
	
	// Replace album-specific placeholders
	if album != nil {
		result = strings.ReplaceAll(result, "{album}", album.Title)
		result = strings.ReplaceAll(result, "{year}", album.Year)
		result = strings.ReplaceAll(result, "{album_artist}", album.Artist)
	}
	
	return result
}

func (fss *FileSystemService) ProcessNamingMaskForFile(mask string, track shared.Track, album *shared.Album) string {
	result := fss.ProcessNamingMask(mask, track, album)
	return fss.SanitizeFileName(result)
}

func (fss *FileSystemService) ProcessNamingMaskForFolder(mask string, track shared.Track, album *shared.Album) string {
	result := fss.ProcessNamingMask(mask, track, album)
	
	parts := strings.Split(result, "/")
	for i, part := range parts {
		parts[i] = fss.SanitizeFileName(part)
	}
	
	return strings.Join(parts, "/")
}

// ============================================================================
// 10. Supporting Services
// ============================================================================

type MetadataService struct {
	warningCollector *shared.WarningCollector
}

func NewMetadataService(warningCollector interfaces.WarningCollectorService) *MetadataService {
	return &MetadataService{
		warningCollector: warningCollector.(*shared.WarningCollector),
	}
}

func (ms *MetadataService) AddMetadata(filePath string, track shared.Track, album *shared.Album, coverData []byte, totalTracks int) error {
	return fmt.Errorf("metadata service not implemented")
}

func (ms *MetadataService) ExtractMetadata(filePath string) (*shared.Track, error) {
	return nil, fmt.Errorf("metadata extraction not implemented")
}

func (ms *MetadataService) ValidateMetadata(filePath string, expectedTrack shared.Track) error {
	return fmt.Errorf("metadata validation not implemented")
}

type ConversionService struct{}

func NewConversionService() *ConversionService {
	return &ConversionService{}
}

func (cs *ConversionService) ConvertTrack(inputPath string, format string, bitrate string) (string, error) {
	return "", fmt.Errorf("conversion service not implemented")
}

func (cs *ConversionService) GetSupportedFormats() []string {
	return []string{"flac", "mp3", "aac"}
}

func (cs *ConversionService) ValidateFormat(format string) error {
	return fmt.Errorf("format validation not implemented")
}

func (cs *ConversionService) ValidateBitrate(format string, bitrate string) error {
	return fmt.Errorf("bitrate validation not implemented")
}

type ConsoleLogger struct {
	debugEnabled bool
}

func NewConsoleLogger() *ConsoleLogger {
	return &ConsoleLogger{
		debugEnabled: false,
	}
}

func (cl *ConsoleLogger) Info(format string, args ...interface{}) {
	shared.ColorInfo.Printf("[INFO] "+format+"\n", args...)
}

func (cl *ConsoleLogger) Warning(format string, args ...interface{}) {
	shared.ColorWarning.Printf("[WARN] "+format+"\n", args...)
}

func (cl *ConsoleLogger) Error(format string, args ...interface{}) {
	shared.ColorError.Printf("[ERROR] "+format+"\n", args...)
}

func (cl *ConsoleLogger) Debug(format string, args ...interface{}) {
	if cl.debugEnabled {
		shared.ColorDebug.Printf("[DEBUG] "+format+"\n", args...)
	}
}

func (cl *ConsoleLogger) Success(format string, args ...interface{}) {
	shared.ColorSuccess.Printf("[SUCCESS] "+format+"\n", args...)
}

func (cl *ConsoleLogger) SetDebugMode(enabled bool) {
	cl.debugEnabled = enabled
}