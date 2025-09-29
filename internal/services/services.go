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

// NewServiceContainer creates a new service container with all services initialized
func NewServiceContainer(cfg *config.Config, httpClient *http.Client) *ServiceContainer {
	// Create logger first as other services may need it
	logger := NewConsoleLogger()
	
	// Create warning collector
	warningCollector := shared.NewWarningCollector(true)
	
	// Create file system service
	fileSystem := NewFileSystemService(cfg)
	
	// Create API client
	apiClient := dab.NewDabAPI(cfg.APIURL, cfg.DownloadLocation, httpClient)
	
	// Create config service
	configService := NewConfigService()
	
	// Create download service
	downloadService := NewDownloadService(apiClient, fileSystem, logger, warningCollector)
	
	// Create search service
	searchService := NewSearchService(apiClient)
	
	// Create Spotify service
	spotifyService := spotify.NewSpotifyClient(cfg.SpotifyClientID, cfg.SpotifyClientSecret)
	
	// Create Navidrome service
	navidromeService := navidrome.NewNavidromeClient(cfg.NavidromeURL, cfg.NavidromeUsername, cfg.NavidromePassword)
	
	// Create updater service
	updaterService := NewUpdaterService(httpClient)
	
	// Create metadata service
	metadataService := NewMetadataService(warningCollector)
	
	// Create conversion service
	conversionService := NewConversionService()
	
	return &ServiceContainer{
		Config:           configService,
		APIClient:        apiClient,
		DownloadService:  downloadService,
		SearchService:    searchService,
		SpotifyService:   NewSpotifyServiceWrapper(spotifyService),
		NavidromeService: NewNavidromeServiceWrapper(navidromeService),
		UpdaterService:   updaterService,
		FileSystem:       fileSystem,
		Logger:           logger,
		WarningCollector: warningCollector,
		Metadata:         metadataService,
		Conversion:       conversionService,
	}
}

// ConfigService implementation
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
	// Add validation logic here
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
	// Apply default naming masks
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

// DownloadService implementation
type DownloadService struct {
	apiClient        interfaces.APIClient
	fileSystem       interfaces.FileSystemService
	logger           interfaces.LoggerService
	warningCollector interfaces.WarningCollectorService
}

func NewDownloadService(apiClient interfaces.APIClient, fileSystem interfaces.FileSystemService, logger interfaces.LoggerService, warningCollector interfaces.WarningCollectorService) *DownloadService {
	return &DownloadService{
		apiClient:        apiClient,
		fileSystem:       fileSystem,
		logger:           logger,
		warningCollector: warningCollector,
	}
}

func (ds *DownloadService) DownloadAlbum(ctx context.Context, albumID string, cfg *config.Config, debug bool, format string, bitrate string) (*shared.DownloadStats, error) {
	// Get album information
	album, err := ds.apiClient.GetAlbum(ctx, albumID)
	if err != nil {
		return nil, fmt.Errorf("failed to get album: %w", err)
	}
	
	return ds.DownloadTracks(ctx, album.Tracks, album, cfg, debug, format, bitrate)
}

func (ds *DownloadService) DownloadArtist(ctx context.Context, artistID string, cfg *config.Config, debug bool, format string, bitrate string, filter string, noConfirm bool) (*shared.DownloadStats, error) {
	// Get artist information
	artist, err := ds.apiClient.GetArtist(ctx, artistID, cfg, debug)
	if err != nil {
		return nil, fmt.Errorf("failed to get artist: %w", err)
	}
	
	// Filter albums based on type
	filteredAlbums := filterAlbumsByType(artist.Albums, filter)
	
	if len(filteredAlbums) == 0 {
		return &shared.DownloadStats{}, nil
	}
	
	// Show confirmation if not skipped
	if !noConfirm {
		ds.logger.Info("Found %d albums to download:", len(filteredAlbums))
		for i, album := range filteredAlbums {
			prefix := fmt.Sprintf("%d. [%s] ", i+1, strings.ToUpper(album.Type))
			formattedLine := shared.FormatAlbumWithBitrate(prefix, album.Title, album.Artist, album.ReleaseDate, album.AudioQuality)
			fmt.Println(formattedLine)
		}
		
		confirmation := shared.GetUserInput("Continue with download? (y/n)", "y")
		if strings.ToLower(confirmation) != "y" && strings.ToLower(confirmation) != "yes" {
			return &shared.DownloadStats{}, shared.ErrDownloadCancelled
		}
	}
	
	// Download all albums
	totalStats := &shared.DownloadStats{}
	for _, album := range filteredAlbums {
		albumStats, err := ds.DownloadAlbum(ctx, album.ID, cfg, debug, format, bitrate)
		if err != nil {
			ds.logger.Error("Failed to download album %s: %v", album.Title, err)
			totalStats.FailedCount++
			totalStats.FailedItems = append(totalStats.FailedItems, album.Title)
			continue
		}
		
		totalStats.SuccessCount += albumStats.SuccessCount
		totalStats.SkippedCount += albumStats.SkippedCount
		totalStats.FailedCount += albumStats.FailedCount
		totalStats.FailedItems = append(totalStats.FailedItems, albumStats.FailedItems...)
	}
	
	return totalStats, nil
}

func (ds *DownloadService) DownloadTrack(ctx context.Context, trackID string, cfg *config.Config, debug bool, format string, bitrate string) (*shared.DownloadStats, error) {
	fmt.Printf("DEBUG - DownloadTrack called with trackID: '%s'\n", trackID)
	// Get track information
	track, err := ds.apiClient.GetTrack(ctx, trackID)
	if err != nil {
		return nil, fmt.Errorf("failed to get track: %w", err)
	}
	
	// Create a minimal album for the track
	album := &shared.Album{
		ID:     track.AlbumID,
		Title:  track.Album,
		Artist: track.AlbumArtist,
		Tracks: []shared.Track{*track},
	}
	
	return ds.DownloadTracks(ctx, []shared.Track{*track}, album, cfg, debug, format, bitrate)
}

// DownloadTrackDirect downloads a track using the track data directly (bypassing GetTrack API call)
func (ds *DownloadService) DownloadTrackDirect(ctx context.Context, track shared.Track, cfg *config.Config, debug bool, format string, bitrate string) (*shared.DownloadStats, error) {
	fmt.Printf("DEBUG - DownloadTrackDirect called with track: '%s' by '%s'\n", track.Title, track.Artist)
	
	// Determine the album title - prefer AlbumTitle from search results, fallback to Album
	albumTitle := track.AlbumTitle
	if albumTitle == "" {
		albumTitle = track.Album
	}
	
	// Determine the album artist - use the track artist if AlbumArtist is empty
	albumArtist := track.AlbumArtist
	if albumArtist == "" {
		albumArtist = track.Artist
	}
	
	// Create a minimal album for the track using available data
	album := &shared.Album{
		ID:     track.AlbumID,
		Title:  albumTitle,
		Artist: albumArtist,
		Tracks: []shared.Track{track},
	}
	
	if debug {
		fmt.Printf("DEBUG - Created album: ID='%s', Title='%s', Artist='%s'\n", album.ID, album.Title, album.Artist)
	}
	
	return ds.DownloadTracks(ctx, []shared.Track{track}, album, cfg, debug, format, bitrate)
}

func (ds *DownloadService) DownloadTracks(ctx context.Context, tracks []shared.Track, album *shared.Album, cfg *config.Config, debug bool, format string, bitrate string) (*shared.DownloadStats, error) {
	stats := &shared.DownloadStats{}
	
	// Download cover art
	var coverData []byte
	if album.Cover != "" {
		var err error
		coverData, err = ds.apiClient.DownloadCover(ctx, album.Cover)
		if err != nil {
			ds.logger.Warning("Failed to download cover art: %v", err)
		}
	}
	
	// Download each track
	for _, track := range tracks {
		// Use the new method that supports naming masks
		outputPath := ds.fileSystem.(*FileSystemService).GetDownloadPathWithTrack(track, album, format, cfg)
		
		// Check if file already exists
		if ds.fileSystem.FileExists(outputPath) {
			ds.logger.Info("Skipping %s - already exists", track.Title)
			stats.SkippedCount++
			continue
		}
		
		// Download the track
		_, err := downloader.DownloadTrack(ctx, ds.apiClient.(*dab.DabAPI), track, album, outputPath, coverData, nil, debug, format, bitrate, cfg, ds.warningCollector.(*shared.WarningCollector))
		if err != nil {
			ds.logger.Error("Failed to download %s: %v", track.Title, err)
			stats.FailedCount++
			stats.FailedItems = append(stats.FailedItems, track.Title)
			continue
		}
		
		stats.SuccessCount++
		// Success message is logged by the calling command
	}
	
	return stats, nil
}

// SearchService implementation
type SearchService struct {
	apiClient interfaces.APIClient
}

func NewSearchService(apiClient interfaces.APIClient) *SearchService {
	return &SearchService{
		apiClient: apiClient,
	}
}

func (ss *SearchService) HandleSearch(ctx context.Context, query string, searchType string, debug bool, auto bool) ([]interface{}, []string, error) {
	return search.HandleSearch(ctx, ss.apiClient.(*dab.DabAPI), query, searchType, debug, auto)
}

func (ss *SearchService) Search(ctx context.Context, query string, searchType string, limit int, debug bool) (*shared.SearchResults, error) {
	return ss.apiClient.Search(ctx, query, searchType, limit, debug)
}

// SpotifyServiceWrapper wraps the Spotify client to implement the interface
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
	
	// Convert to shared types
	sharedTracks := make([]shared.SpotifyTrack, len(tracks))
	for i, track := range tracks {
		sharedTracks[i] = shared.SpotifyTrack{
			Name:        track.Name,
			Artist:      track.Artist,
			AlbumName:   track.AlbumName,
			AlbumArtist: track.AlbumArtist,
		}
	}
	
	return sharedTracks, name, nil
}

func (ssw *SpotifyServiceWrapper) GetAlbumTracks(albumURL string) ([]shared.SpotifyTrack, string, error) {
	tracks, name, err := ssw.client.GetAlbumTracks(albumURL)
	if err != nil {
		return nil, "", err
	}
	
	// Convert to shared types
	sharedTracks := make([]shared.SpotifyTrack, len(tracks))
	for i, track := range tracks {
		sharedTracks[i] = shared.SpotifyTrack{
			Name:        track.Name,
			Artist:      track.Artist,
			AlbumName:   track.AlbumName,
			AlbumArtist: track.AlbumArtist,
		}
	}
	
	return sharedTracks, name, nil
}

// NavidromeServiceWrapper wraps the Navidrome client to implement the interface
type NavidromeServiceWrapper struct {
	client *navidrome.NavidromeClient
}

func NewNavidromeServiceWrapper(client *navidrome.NavidromeClient) *NavidromeServiceWrapper {
	return &NavidromeServiceWrapper{client: client}
}

func (nsw *NavidromeServiceWrapper) Authenticate() error {
	// Implementation would call the actual Navidrome authentication
	return fmt.Errorf("Navidrome authentication not implemented")
}

func (nsw *NavidromeServiceWrapper) CreatePlaylist(name string, tracks []shared.Track) error {
	// Implementation would create a playlist in Navidrome
	return fmt.Errorf("Navidrome playlist creation not implemented")
}

func (nsw *NavidromeServiceWrapper) GetPlaylists() ([]shared.NavidromePlaylist, error) {
	// Implementation would get playlists from Navidrome
	return nil, fmt.Errorf("Navidrome playlist retrieval not implemented")
}

func (nsw *NavidromeServiceWrapper) AddTracksToPlaylist(playlistID string, tracks []shared.Track) error {
	// Implementation would add tracks to a Navidrome playlist
	return fmt.Errorf("Navidrome add tracks not implemented")
}

// UpdaterService implementation
type UpdaterService struct {
	httpClient *http.Client
}

func NewUpdaterService(httpClient *http.Client) *UpdaterService {
	return &UpdaterService{httpClient: httpClient}
}

func (us *UpdaterService) CheckForUpdates(ctx context.Context, currentVersion string, updateRepo string) (*shared.UpdateInfo, error) {
	// Create a temporary config for the updater
	cfg := &config.Config{
		UpdateRepo:          updateRepo,
		DisableUpdateCheck: false,
	}
	
	// Use the existing updater function (this is a simplified wrapper)
	// In a real implementation, we'd modify the updater to return structured data
	updater.CheckForUpdates(cfg, currentVersion)
	
	// For now, return a placeholder - this would need to be implemented properly
	return &shared.UpdateInfo{
		Version:     "unknown",
		DownloadURL: "",
		ReleaseDate: "",
		Notes:       "",
	}, fmt.Errorf("update check not fully implemented in service layer")
}

func (us *UpdaterService) DownloadUpdate(ctx context.Context, updateInfo *shared.UpdateInfo, outputPath string) error {
	// This would need to be implemented to download the update
	return fmt.Errorf("update download not implemented")
}

func (us *UpdaterService) ApplyUpdate(updatePath string, currentBinaryPath string) error {
	// This would need to be implemented to apply the update
	return fmt.Errorf("update application not implemented")
}

// Helper functions
func filterAlbumsByType(albums []shared.Album, filter string) []shared.Album {
	if filter == "all" || filter == "" {
		return albums
	}
	
	filters := strings.Split(strings.ToLower(filter), ",")
	var filtered []shared.Album
	
	for _, album := range albums {
		albumType := strings.ToLower(album.Type)
		for _, f := range filters {
			f = strings.TrimSpace(f)
			if f == albumType || (f == "albums" && albumType == "album") || (f == "eps" && albumType == "ep") || (f == "singles" && albumType == "single") {
				filtered = append(filtered, album)
				break
			}
		}
	}
	
	return filtered
}

// FileSystemService implementation
type FileSystemService struct {
	config *config.Config
}

func NewFileSystemService(cfg *config.Config) *FileSystemService {
	return &FileSystemService{config: cfg}
}

func (fss *FileSystemService) EnsureDirectoryExists(path string) error {
	return config.CreateDirIfNotExists(path)
}

func (fss *FileSystemService) GetDownloadPath(artist, album, track string, format string, cfg *config.Config) string {
	// Determine file extension
	ext := ".flac"
	if format != "flac" {
		ext = "." + format
	}
	
	// Use naming masks if configured, otherwise fall back to default structure
	if cfg.NamingMasks.FileMask != "" {
		// For now, use a simple filename structure - this would need track metadata
		trackFileName := fss.SanitizeFileName(track) + ext
		albumDir := filepath.Join(cfg.DownloadLocation, fss.SanitizeFileName(artist), fss.SanitizeFileName(album))
		return filepath.Join(albumDir, trackFileName)
	}
	
	// Default structure (legacy)
	artist = fss.SanitizeFileName(artist)
	album = fss.SanitizeFileName(album)
	track = fss.SanitizeFileName(track)
	
	artistDir := filepath.Join(cfg.DownloadLocation, artist)
	albumDir := filepath.Join(artistDir, album)
	trackFileName := track + ext
	
	return filepath.Join(albumDir, trackFileName)
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
	// Check if directory exists, create if it doesn't
	if err := fss.EnsureDirectoryExists(path); err != nil {
		return fmt.Errorf("cannot create download directory: %w", err)
	}
	
	// Test write permissions
	testFile := filepath.Join(path, ".write_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("download directory is not writable: %w", err)
	}
	
	// Clean up test file
	os.Remove(testFile)
	
	return nil
}

func (fss *FileSystemService) SanitizeFileName(filename string) string {
	return shared.SanitizeFileName(filename)
}

// ProcessNamingMask processes a naming mask template with track/album data
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

// ProcessNamingMaskForFile processes a naming mask for a filename (sanitizes the result)
func (fss *FileSystemService) ProcessNamingMaskForFile(mask string, track shared.Track, album *shared.Album) string {
	result := fss.ProcessNamingMask(mask, track, album)
	return fss.SanitizeFileName(result)
}

// ProcessNamingMaskForFolder processes a naming mask for a folder path (sanitizes each component separately)
func (fss *FileSystemService) ProcessNamingMaskForFolder(mask string, track shared.Track, album *shared.Album) string {
	result := fss.ProcessNamingMask(mask, track, album)
	
	// Split by path separator and sanitize each component
	parts := strings.Split(result, "/")
	for i, part := range parts {
		parts[i] = fss.SanitizeFileName(part)
	}
	
	return strings.Join(parts, "/")
}

// GetDownloadPathWithTrack generates the full download path using naming masks and track metadata
func (fss *FileSystemService) GetDownloadPathWithTrack(track shared.Track, album *shared.Album, format string, cfg *config.Config) string {
	// Determine file extension
	ext := ".flac"
	if format != "flac" {
		ext = "." + format
	}
	
	// Apply default naming masks if they're empty
	cfg.ApplyDefaultNamingMasks()
	
	// Process file mask (now guaranteed to have a value)
	fileName := fss.ProcessNamingMaskForFile(cfg.NamingMasks.FileMask, track, album) + ext
	
	// Determine folder mask based on album type
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
	
	// Use album folder mask as fallback
	if folderMask == "" {
		folderMask = cfg.NamingMasks.AlbumFolderMask
	}
	
	// Process folder mask (now guaranteed to have a value)
	folderPath := fss.ProcessNamingMaskForFolder(folderMask, track, album)
	return filepath.Join(cfg.DownloadLocation, folderPath, fileName)
}

// ConsoleLogger implementation
type ConsoleLogger struct {
	debugMode bool
}

func NewConsoleLogger() *ConsoleLogger {
	return &ConsoleLogger{debugMode: false}
}

func (cl *ConsoleLogger) Info(message string, args ...interface{}) {
	shared.ColorInfo.Printf(message+"\n", args...)
}

func (cl *ConsoleLogger) Warning(message string, args ...interface{}) {
	shared.ColorWarning.Printf("⚠️ "+message+"\n", args...)
}

func (cl *ConsoleLogger) Error(message string, args ...interface{}) {
	shared.ColorError.Printf("❌ "+message+"\n", args...)
}

func (cl *ConsoleLogger) Debug(message string, args ...interface{}) {
	fmt.Printf("🐛 DEBUG: "+message+"\n", args...)
}

func (cl *ConsoleLogger) Success(message string, args ...interface{}) {
	shared.ColorSuccess.Printf("✅ "+message+"\n", args...)
}

func (cl *ConsoleLogger) SetDebugMode(enabled bool) {
	cl.debugMode = enabled
}



// MetadataService implementation
type MetadataService struct {
	warningCollector interfaces.WarningCollectorService
}

func NewMetadataService(warningCollector interfaces.WarningCollectorService) *MetadataService {
	return &MetadataService{warningCollector: warningCollector}
}

func (ms *MetadataService) AddMetadata(filePath string, track shared.Track, album *shared.Album, coverData []byte, totalTracks int) error {
	return downloader.AddMetadataWithDebug(filePath, track, album, coverData, totalTracks, ms.warningCollector.(*shared.WarningCollector), false)
}

func (ms *MetadataService) ExtractMetadata(filePath string) (*shared.Track, error) {
	// This would be implemented to extract metadata from files
	return nil, fmt.Errorf("metadata extraction not implemented")
}

func (ms *MetadataService) ValidateMetadata(filePath string, expectedTrack shared.Track) error {
	// This would be implemented to validate metadata
	return fmt.Errorf("metadata validation not implemented")
}

// ConversionService implementation
type ConversionService struct{}

func NewConversionService() *ConversionService {
	return &ConversionService{}
}

func (cs *ConversionService) ConvertTrack(inputPath string, format string, bitrate string) (string, error) {
	return downloader.ConvertTrack(inputPath, format, bitrate)
}

func (cs *ConversionService) GetSupportedFormats() []string {
	return []string{"flac", "mp3", "ogg", "opus"}
}

func (cs *ConversionService) ValidateFormat(format string) error {
	supportedFormats := cs.GetSupportedFormats()
	for _, supported := range supportedFormats {
		if format == supported {
			return nil
		}
	}
	return fmt.Errorf("unsupported format: %s", format)
}

func (cs *ConversionService) ValidateBitrate(format string, bitrate string) error {
	// Basic validation - could be more sophisticated
	if format == "flac" {
		return nil // FLAC doesn't use bitrate
	}
	
	// Check if bitrate is a valid number
	validBitrates := []string{"128", "192", "256", "320"}
	for _, valid := range validBitrates {
		if bitrate == valid {
			return nil
		}
	}
	
	return fmt.Errorf("invalid bitrate for %s: %s", format, bitrate)
}