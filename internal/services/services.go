package services

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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
	
	filteredAlbums := ds.filterAlbumsByType(artist.Albums, filter)
	if len(filteredAlbums) == 0 {
		return &shared.DownloadStats{}, nil
	}
	
	if !noConfirm && !ds.confirmDownload(filteredAlbums) {
		return &shared.DownloadStats{}, shared.ErrDownloadCancelled
	}
	
	stats := ds.downloadAlbumsWithParallelism(ctx, filteredAlbums, cfg, debug, format, bitrate)
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

func (ds *DownloadService) prefetchMetadata(ctx context.Context, tracks []shared.Track, album *shared.Album, cfg *config.Config, debug bool) {
	maxWorkers := ds.getParallelism(cfg)
	
	if debug {
		ds.logger.Debug("Pre-fetching metadata for %d tracks using %d workers", len(tracks), maxWorkers)
	}
	
	// Pre-fetch track-specific metadata in parallel
	ds.prefetchTrackMetadata(ctx, tracks, album, maxWorkers, debug)
}

func (ds *DownloadService) downloadAlbumsWithParallelism(ctx context.Context, albums []shared.Album, cfg *config.Config, debug bool, format string, bitrate string) *shared.DownloadStats {
	maxWorkers := ds.getParallelism(cfg)
	
	if debug {
		ds.logger.Debug("Using parallelism setting: %d workers for %d albums", maxWorkers, len(albums))
	}
	
	if len(albums) == 1 || maxWorkers == 1 {
		return ds.downloadAlbumsSequentially(ctx, albums, cfg, debug, format, bitrate)
	}
	
	return ds.downloadAlbumsParallel(ctx, albums, cfg, debug, format, bitrate, maxWorkers)
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

func (ds *DownloadService) downloadAlbumsSequentially(ctx context.Context, albums []shared.Album, cfg *config.Config, debug bool, format string, bitrate string) *shared.DownloadStats {
	totalStats := &shared.DownloadStats{}
	
	for _, album := range albums {
		albumStats, err := ds.DownloadAlbum(ctx, album.ID, cfg, debug, format, bitrate)
		if err != nil {
			ds.logger.Error("Failed to download album %s: %v", album.Title, err)
			totalStats.FailedCount++
			totalStats.FailedItems = append(totalStats.FailedItems, album.Title)
			continue
		}
		
		ds.mergeStats(totalStats, albumStats)
	}
	
	return totalStats
}

func (ds *DownloadService) downloadAlbumsParallel(ctx context.Context, albums []shared.Album, cfg *config.Config, debug bool, format string, bitrate string, maxWorkers int) *shared.DownloadStats {
	totalStats := &shared.DownloadStats{}
	
	albumJobs := make(chan shared.Album, len(albums))
	results := make(chan *shared.DownloadStats, len(albums))
	
	// Start worker goroutines
	for i := 0; i < maxWorkers; i++ {
		go ds.albumWorker(ctx, i, albumJobs, results, cfg, debug, format, bitrate)
	}
	
	// Send jobs
	for _, album := range albums {
		albumJobs <- album
	}
	close(albumJobs)
	
	// Collect results
	for i := 0; i < len(albums); i++ {
		albumStats := <-results
		ds.mergeStats(totalStats, albumStats)
	}
	
	return totalStats
}

func (ds *DownloadService) albumWorker(ctx context.Context, workerID int, jobs <-chan shared.Album, results chan<- *shared.DownloadStats, cfg *config.Config, debug bool, format string, bitrate string) {
	for album := range jobs {
		if debug {
			ds.logger.Debug("Worker %d: Starting download for album %s by %s", workerID, album.Title, album.Artist)
		}
		
		albumStats, err := ds.DownloadAlbum(ctx, album.ID, cfg, debug, format, bitrate)
		if err != nil {
			ds.logger.Error("Worker %d: Failed to download album %s: %v", workerID, album.Title, err)
			albumStats = &shared.DownloadStats{
				FailedCount: 1,
				FailedItems: []string{album.Title},
			}
		}
		
		results <- albumStats
	}
}

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