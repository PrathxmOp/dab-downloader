package downloader

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/go-flac/go-flac"
	"github.com/go-flac/flacpicture"
	"github.com/go-flac/flacvorbis"
	
	"dab-downloader/internal/shared"
	"dab-downloader/internal/api/musicbrainz"
)

// ============================================================================
// 1. Constants and Types
// ============================================================================

const (
	DefaultEncoder = "EnhancedFLACDownloader/2.0"
	DefaultEncoding = "FLAC"
	DefaultSource = "DAB"
)

// ISRCMetadata holds comprehensive metadata extracted from ISRC lookup
type ISRCMetadata struct {
	ReleaseID        string
	ReleaseArtistID  string
	ReleaseGroupID   string
	TrackID          string
	TrackArtistID    string
}

// These types are now imported from the musicbrainz package

// ============================================================================
// 2. Constructor and Configuration
// ============================================================================

// MetadataProcessor handles FLAC metadata operations
type MetadataProcessor struct {
	mbClient *musicbrainz.Client
	cache    *AlbumMetadataCache
}

// NewMetadataProcessor creates a new metadata processor with default settings
func NewMetadataProcessor() *MetadataProcessor {
	return &MetadataProcessor{
		mbClient: musicbrainz.NewClient(),
		cache:    NewAlbumMetadataCache(),
	}
}

// SetDebugMode enables or disables debug mode for MusicBrainz client
func (mp *MetadataProcessor) SetDebugMode(debug bool) {
	mp.mbClient.SetDebug(debug)
}

// ============================================================================
// 3. Cache Management
// ============================================================================

// AlbumMetadataCache holds cached MusicBrainz release metadata for albums
type AlbumMetadataCache struct {
	releases   map[string]*musicbrainz.Release
	releaseIDs map[string]string
	mu         sync.RWMutex
}

// NewAlbumMetadataCache creates a new album metadata cache
func NewAlbumMetadataCache() *AlbumMetadataCache {
	return &AlbumMetadataCache{
		releases:   make(map[string]*musicbrainz.Release),
		releaseIDs: make(map[string]string),
	}
}

// GetCachedRelease retrieves cached release metadata
func (cache *AlbumMetadataCache) GetCachedRelease(artist, album string) *musicbrainz.Release {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return cache.releases[getCacheKey(artist, album)]
}

// GetCachedReleaseID retrieves cached release ID
func (cache *AlbumMetadataCache) GetCachedReleaseID(artist, album string) string {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return cache.releaseIDs[getCacheKey(artist, album)]
}

// SetCachedReleaseID stores release ID in cache
func (cache *AlbumMetadataCache) SetCachedReleaseID(artist, album, releaseID string) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.releaseIDs[getCacheKey(artist, album)] = releaseID
}

// SetCachedRelease stores release metadata in cache
func (cache *AlbumMetadataCache) SetCachedRelease(artist, album string, release *musicbrainz.Release) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.releases[getCacheKey(artist, album)] = release
}

// ClearCache clears the album metadata cache
func (cache *AlbumMetadataCache) ClearCache() {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.releases = make(map[string]*musicbrainz.Release)
	cache.releaseIDs = make(map[string]string)
}

// GetStats returns cache statistics
func (cache *AlbumMetadataCache) GetStats() (int, []string) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	
	count := len(cache.releases)
	keys := make([]string, 0, count)
	for key := range cache.releases {
		keys = append(keys, key)
	}
	return count, keys
}

// ============================================================================
// 4. Public API Methods
// ============================================================================

// AddMetadata adds comprehensive metadata to a FLAC file
func (mp *MetadataProcessor) AddMetadata(filePath string, track shared.Track, album *shared.Album, coverData []byte, totalTracks int, warningCollector *shared.WarningCollector) error {
	return mp.AddMetadataWithDebug(filePath, track, album, coverData, totalTracks, warningCollector, false)
}

// AddMetadataWithDebug adds comprehensive metadata to a FLAC file with debug mode support
func (mp *MetadataProcessor) AddMetadataWithDebug(filePath string, track shared.Track, album *shared.Album, coverData []byte, totalTracks int, warningCollector *shared.WarningCollector, debug bool) error {
	mp.SetDebugMode(debug)
	
	f, err := mp.openAndCleanFLACFile(filePath)
	if err != nil {
		return err
	}

	comment := mp.buildVorbisComment(track, album, totalTracks, warningCollector)
	
	vorbisCommentBlock := comment.Marshal()
	f.Meta = append(f.Meta, &vorbisCommentBlock)

	if err := mp.addCoverArt(f, coverData, warningCollector, track); err != nil {
		// Cover art errors are non-fatal, already logged to warningCollector
	}

	return mp.saveFLACFile(f, filePath)
}

// FindReleaseIDFromISRC attempts to find a MusicBrainz release ID from tracks with ISRC
func (mp *MetadataProcessor) FindReleaseIDFromISRC(tracks []shared.Track, albumArtist, albumTitle string) {
	if mp.cache.GetCachedReleaseID(albumArtist, albumTitle) != "" {
		return
	}
	
	expectedTrackCount := len(tracks)
	
	for _, track := range tracks {
		if track.ISRC == "" {
			continue
		}
		
		isrcMetadata, err := mp.GetISRCMetadataWithTrackCount(track.ISRC, expectedTrackCount)
		if err != nil {
			continue
		}
		
		if isrcMetadata.ReleaseID != "" {
			mp.cache.SetCachedReleaseID(albumArtist, albumTitle, isrcMetadata.ReleaseID)
			return
		}
	}
}

// GetISRCMetadata extracts comprehensive metadata from ISRC lookup
func (mp *MetadataProcessor) GetISRCMetadata(isrc string) (*ISRCMetadata, error) {
	return mp.GetISRCMetadataWithTrackCount(isrc, 0)
}

// GetISRCMetadataWithTrackCount extracts comprehensive metadata from ISRC lookup with intelligent release selection
func (mp *MetadataProcessor) GetISRCMetadataWithTrackCount(isrc string, expectedTrackCount int) (*ISRCMetadata, error) {
	ctx := context.Background()
	mbTrack, err := mp.mbClient.SearchTrackByISRC(ctx, isrc)
	if err != nil {
		return nil, err
	}
	
	metadata := &ISRCMetadata{
		TrackID: mbTrack.ID,
	}
	
	if len(mbTrack.ArtistCredit) > 0 {
		metadata.TrackArtistID = mbTrack.ArtistCredit[0].Artist.ID
	}
	
	if len(mbTrack.Releases) > 0 {
		selectedRelease := mp.selectBestTrackRelease(mbTrack.Releases, expectedTrackCount)
		metadata.ReleaseID = selectedRelease.ID
		metadata.ReleaseGroupID = selectedRelease.ReleaseGroup.ID
		
		if len(selectedRelease.ArtistCredit) > 0 {
			metadata.ReleaseArtistID = selectedRelease.ArtistCredit[0].Artist.ID
		}
	}
	
	return metadata, nil
}

// ClearCache clears the metadata cache
func (mp *MetadataProcessor) ClearCache() {
	mp.cache.ClearCache()
}

// GetCacheStats returns cache statistics
func (mp *MetadataProcessor) GetCacheStats() (int, []string) {
	return mp.cache.GetStats()
}

// ============================================================================
// 5. Private Core Methods
// ============================================================================

// openAndCleanFLACFile opens a FLAC file and removes existing metadata blocks
func (mp *MetadataProcessor) openAndCleanFLACFile(filePath string) (*flac.File, error) {
	f, err := flac.ParseFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse FLAC file: %w", err)
	}

	// Remove existing VORBIS_COMMENT and PICTURE blocks
	var newMetaData []*flac.MetaDataBlock
	for _, block := range f.Meta {
		if block.Type != flac.VorbisComment && block.Type != flac.Picture {
			newMetaData = append(newMetaData, block)
		}
	}
	f.Meta = newMetaData

	return f, nil
}

// buildVorbisComment creates a comprehensive Vorbis comment block
func (mp *MetadataProcessor) buildVorbisComment(track shared.Track, album *shared.Album, totalTracks int, warningCollector *shared.WarningCollector) *flacvorbis.MetaDataBlockVorbisComment {
	comment := flacvorbis.New()

	// Essential metadata
	mp.addEssentialMetadata(comment, track, album)
	
	// Track and disc information
	mp.addTrackDiscMetadata(comment, track, album, totalTracks)
	
	// Date and temporal metadata
	mp.addDateMetadata(comment, track, album)
	
	// Additional metadata
	mp.addExtendedMetadata(comment, track, album)
	
	// MusicBrainz metadata
	mp.addMusicBrainzMetadata(comment, track, album, warningCollector)
	
	// Technical metadata
	mp.addTechnicalMetadata(comment, track)

	return comment
}

// addEssentialMetadata adds core track information
func (mp *MetadataProcessor) addEssentialMetadata(comment *flacvorbis.MetaDataBlockVorbisComment, track shared.Track, album *shared.Album) {
	addField(comment, flacvorbis.FIELD_TITLE, track.Title)
	addField(comment, flacvorbis.FIELD_ARTIST, track.Artist)
	addField(comment, flacvorbis.FIELD_ALBUM, getAlbumTitle(track, album))
	addField(comment, "ALBUMARTIST", getAlbumArtist(track, album))
	
	genre := getGenre(track, album)
	if genre != "" && genre != "Unknown" {
		addField(comment, "GENRE", genre)
	}
}

// addTrackDiscMetadata adds track and disc numbering information
func (mp *MetadataProcessor) addTrackDiscMetadata(comment *flacvorbis.MetaDataBlockVorbisComment, track shared.Track, album *shared.Album, totalTracks int) {
	trackNumber := track.TrackNumber
	if trackNumber == 0 {
		trackNumber = 1
	}
	addField(comment, flacvorbis.FIELD_TRACKNUMBER, fmt.Sprintf("%d", trackNumber))

	if totalTracks > 0 {
		addField(comment, "TOTALTRACKS", fmt.Sprintf("%d", totalTracks))
	} else if album != nil && album.TotalTracks > 0 {
		addField(comment, "TOTALTRACKS", fmt.Sprintf("%d", album.TotalTracks))
	}

	discNumber := track.DiscNumber
	if discNumber == 0 {
		discNumber = 1
	}
	addField(comment, "DISCNUMBER", fmt.Sprintf("%d", discNumber))

	totalDiscs := 1
	if album != nil && album.TotalDiscs > 0 {
		totalDiscs = album.TotalDiscs
	}
	addField(comment, "TOTALDISCS", fmt.Sprintf("%d", totalDiscs))
}

// addDateMetadata adds date and year information
func (mp *MetadataProcessor) addDateMetadata(comment *flacvorbis.MetaDataBlockVorbisComment, track shared.Track, album *shared.Album) {
	releaseDate := getReleaseDate(track, album)
	if releaseDate != "" {
		addField(comment, flacvorbis.FIELD_DATE, releaseDate)
		if len(releaseDate) >= 4 {
			year := releaseDate[:4]
			addField(comment, "YEAR", year)
			addField(comment, "ORIGINALDATE", releaseDate)
		}
	} else if track.Year != "" {
		addField(comment, "YEAR", track.Year)
		addField(comment, flacvorbis.FIELD_DATE, track.Year)
	}
}

// addExtendedMetadata adds additional metadata fields
func (mp *MetadataProcessor) addExtendedMetadata(comment *flacvorbis.MetaDataBlockVorbisComment, track shared.Track, album *shared.Album) {
	addField(comment, "COMPOSER", track.Composer)
	addField(comment, "PRODUCER", track.Producer)
	addField(comment, "ISRC", track.ISRC)
	
	// Copyright information
	if track.Copyright != "" {
		addField(comment, "COPYRIGHT", track.Copyright)
	} else if album != nil && album.Copyright != "" {
		addField(comment, "COPYRIGHT", album.Copyright)
	}

	// Label information
	if album != nil && album.Label != nil {
		if label, ok := album.Label.(string); ok {
			addField(comment, "LABEL", label)
		}
	}

	// Catalog numbers
	if album != nil && album.UPC != "" {
		addField(comment, "CATALOGNUMBER", album.UPC)
		addField(comment, "UPC", album.UPC)
	}
}

// addTechnicalMetadata adds technical and encoding information
func (mp *MetadataProcessor) addTechnicalMetadata(comment *flacvorbis.MetaDataBlockVorbisComment, track shared.Track) {
	addField(comment, "ENCODER", DefaultEncoder)
	addField(comment, "ENCODING", DefaultEncoding)
	addField(comment, "SOURCE", DefaultSource)

	if track.Duration > 0 {
		addField(comment, "LENGTH", fmt.Sprintf("%d", track.Duration))
	}
}

// addMusicBrainzMetadata handles MusicBrainz metadata fetching with caching
func (mp *MetadataProcessor) addMusicBrainzMetadata(comment *flacvorbis.MetaDataBlockVorbisComment, track shared.Track, album *shared.Album, warningCollector *shared.WarningCollector) {
	albumTitle := getAlbumTitle(track, album)
	
	// Try ISRC-based metadata first
	if track.ISRC != "" {
		expectedTrackCount := mp.getExpectedTrackCount(album)
		
		if isrcMetadata, err := mp.GetISRCMetadataWithTrackCount(track.ISRC, expectedTrackCount); err == nil {
			mp.addISRCMetadataFields(comment, isrcMetadata)
			return
		}
	}
	
	// Fallback to traditional approach
	mp.addTrackMetadata(comment, track, albumTitle, warningCollector)
	
	// Handle release-level metadata
	if album != nil {
		mp.addReleaseMetadata(comment, album.Artist, album.Title, warningCollector)
	}
}

// addISRCMetadataFields adds MusicBrainz fields from ISRC metadata
func (mp *MetadataProcessor) addISRCMetadataFields(comment *flacvorbis.MetaDataBlockVorbisComment, metadata *ISRCMetadata) {
	addField(comment, "MUSICBRAINZ_TRACKID", metadata.TrackID)
	addField(comment, "MUSICBRAINZ_ARTISTID", metadata.TrackArtistID)
	addField(comment, "MUSICBRAINZ_ALBUMID", metadata.ReleaseID)
	addField(comment, "MUSICBRAINZ_ALBUMARTISTID", metadata.ReleaseArtistID)
	addField(comment, "MUSICBRAINZ_RELEASEGROUPID", metadata.ReleaseGroupID)
}

// addTrackMetadata adds track-level MusicBrainz metadata
func (mp *MetadataProcessor) addTrackMetadata(comment *flacvorbis.MetaDataBlockVorbisComment, track shared.Track, albumTitle string, warningCollector *shared.WarningCollector) {
	var mbTrack *musicbrainz.Track
	var err error
	ctx := context.Background()
	
	if track.ISRC != "" {
		mbTrack, err = mp.mbClient.SearchTrackByISRC(ctx, track.ISRC)
		if err != nil {
			mbTrack, err = mp.mbClient.SearchTrack(ctx, track.Artist, albumTitle, track.Title)
		}
	} else {
		mbTrack, err = mp.mbClient.SearchTrack(ctx, track.Artist, albumTitle, track.Title)
	}
	
	if err != nil {
		if warningCollector != nil {
			warningCollector.AddMusicBrainzTrackWarning(track.Artist, track.Title, err.Error())
		}
		return
	}
	
	addField(comment, "MUSICBRAINZ_TRACKID", mbTrack.ID)
	if len(mbTrack.ArtistCredit) > 0 {
		addField(comment, "MUSICBRAINZ_ARTISTID", mbTrack.ArtistCredit[0].Artist.ID)
	}
}

// addReleaseMetadata handles release-level MusicBrainz metadata with caching
func (mp *MetadataProcessor) addReleaseMetadata(comment *flacvorbis.MetaDataBlockVorbisComment, artist, albumTitle string, warningCollector *shared.WarningCollector) {
	mbRelease := mp.cache.GetCachedRelease(artist, albumTitle)
	ctx := context.Background()
	
	if mbRelease == nil {
		releaseID := mp.cache.GetCachedReleaseID(artist, albumTitle)
		
		var err error
		if releaseID != "" {
			mbRelease, err = mp.mbClient.GetReleaseMetadata(ctx, releaseID)
		} else {
			mbRelease, err = mp.mbClient.SearchRelease(ctx, artist, albumTitle)
		}
		
		if err != nil {
			if warningCollector != nil {
				warningCollector.AddMusicBrainzReleaseWarning(artist, albumTitle, err.Error())
			}
			return
		}
		
		mp.cache.SetCachedRelease(artist, albumTitle, mbRelease)
		
		if warningCollector != nil {
			warningCollector.RemoveMusicBrainzReleaseWarning(artist, albumTitle)
		}
	}
	
	addField(comment, "MUSICBRAINZ_ALBUMID", mbRelease.ID)
	if len(mbRelease.ArtistCredit) > 0 {
		addField(comment, "MUSICBRAINZ_ALBUMARTISTID", mbRelease.ArtistCredit[0].Artist.ID)
	}
	if mbRelease.ReleaseGroup.ID != "" {
		addField(comment, "MUSICBRAINZ_RELEASEGROUPID", mbRelease.ReleaseGroup.ID)
	}
}

// addCoverArt adds cover art to the FLAC file
func (mp *MetadataProcessor) addCoverArt(f *flac.File, coverData []byte, warningCollector *shared.WarningCollector, track shared.Track) error {
	if len(coverData) == 0 {
		return nil
	}

	imageFormat := detectImageFormat(coverData)
	
	picture, err := flacpicture.NewFromImageData(
		flacpicture.PictureTypeFrontCover,
		"Front Cover",
		coverData,
		imageFormat,
	)
	if err != nil {
		picture, err = flacpicture.NewFromImageData(
			flacpicture.PictureTypeOther,
			"Cover Art",
			coverData,
			imageFormat,
		)
		if err != nil {
			if warningCollector != nil {
				context := fmt.Sprintf("%s - %s", track.Artist, track.Title)
				warningCollector.AddCoverArtMetadataWarning(context, err.Error())
			}
			return fmt.Errorf("failed to create picture metadata: %w", err)
		}
	}

	pictureBlock := picture.Marshal()
	f.Meta = append(f.Meta, &pictureBlock)
	return nil
}

// saveFLACFile saves the FLAC file with new metadata
func (mp *MetadataProcessor) saveFLACFile(f *flac.File, filePath string) error {
	if err := f.Save(filePath); err != nil {
		return fmt.Errorf("failed to save FLAC file with metadata: %w", err)
	}
	return nil
}

// selectBestTrackRelease chooses the most appropriate track release based on intelligent heuristics
func (mp *MetadataProcessor) selectBestTrackRelease(releases []musicbrainz.TrackRelease, expectedTrackCount int) musicbrainz.TrackRelease {
	if len(releases) == 1 {
		return releases[0]
	}
	
	type scoredRelease struct {
		release musicbrainz.TrackRelease
		score   int
	}
	
	var scoredReleases []scoredRelease
	
	for _, release := range releases {
		score := mp.scoreTrackRelease(release, expectedTrackCount)
		scoredReleases = append(scoredReleases, scoredRelease{
			release: release,
			score:   score,
		})
	}
	
	// Find the release with the highest score
	bestRelease := scoredReleases[0]
	for _, sr := range scoredReleases[1:] {
		if sr.score > bestRelease.score {
			bestRelease = sr
		}
	}
	
	return bestRelease.release
}

// selectBestRelease chooses the most appropriate release based on intelligent heuristics
func (mp *MetadataProcessor) selectBestRelease(releases []musicbrainz.Release, expectedTrackCount int) musicbrainz.Release {
	if len(releases) == 1 {
		return releases[0]
	}
	
	type scoredRelease struct {
		release musicbrainz.Release
		score   int
	}
	
	var scoredReleases []scoredRelease
	
	for _, release := range releases {
		score := mp.scoreRelease(release, expectedTrackCount)
		scoredReleases = append(scoredReleases, scoredRelease{
			release: release,
			score:   score,
		})
	}
	
	// Find the release with the highest score
	bestRelease := scoredReleases[0]
	for _, sr := range scoredReleases[1:] {
		if sr.score > bestRelease.score {
			bestRelease = sr
		}
	}
	
	return bestRelease.release
}

// scoreTrackRelease calculates a score for a track release based on various factors
func (mp *MetadataProcessor) scoreTrackRelease(release musicbrainz.TrackRelease, expectedTrackCount int) int {
	score := 0
	title := strings.ToLower(release.Title)
	
	// Prefer releases that look like full albums over singles/compilations
	if expectedTrackCount > 5 {
		if mp.looksLikeAlbum(title) {
			score += 100
		}
		
		if !strings.Contains(title, "demo") {
			score += 30
		}
		
		if mp.looksLikeSingle(title) && expectedTrackCount > 10 {
			score -= 50
		}
	} else if expectedTrackCount <= 3 {
		if mp.looksLikeSingle(title) {
			score += 50
		}
	}
	
	// Prefer releases with reasonable dates
	if release.Date != "" && len(release.Date) >= 4 {
		year := release.Date[:4]
		if year >= "2010" && year <= "2020" && year <= "2015" {
			score += 10
		}
	}
	
	// Prefer non-compilation releases
	if !mp.looksLikeCompilation(title) {
		score += 15
	}
	
	// Format preferences
	score += mp.scoreByFormat(release.Media)
	
	return score
}

// scoreRelease calculates a score for a release based on various factors
func (mp *MetadataProcessor) scoreRelease(release musicbrainz.Release, expectedTrackCount int) int {
	score := 0
	title := strings.ToLower(release.Title)
	
	// Prefer releases that look like full albums over singles/compilations
	if expectedTrackCount > 5 {
		if mp.looksLikeAlbum(title) {
			score += 100
		}
		
		if !strings.Contains(title, "demo") {
			score += 30
		}
		
		if mp.looksLikeSingle(title) && expectedTrackCount > 10 {
			score -= 50
		}
	} else if expectedTrackCount <= 3 {
		if mp.looksLikeSingle(title) {
			score += 50
		}
	}
	
	// Prefer releases with reasonable dates
	if release.Date != "" && len(release.Date) >= 4 {
		year := release.Date[:4]
		if year >= "2010" && year <= "2020" && year <= "2015" {
			score += 10
		}
	}
	
	// Prefer non-compilation releases
	if !mp.looksLikeCompilation(title) {
		score += 15
	}
	
	// Format preferences
	score += mp.scoreByFormat(release.Media)
	
	return score
}

// ============================================================================
// 6. Helper/Utility Functions
// ============================================================================

// getExpectedTrackCount determines expected track count from album
func (mp *MetadataProcessor) getExpectedTrackCount(album *shared.Album) int {
	if album == nil {
		return 0
	}
	
	if len(album.Tracks) > 0 {
		return len(album.Tracks)
	}
	
	return album.TotalTracks
}

// looksLikeAlbum determines if a title looks like an album
func (mp *MetadataProcessor) looksLikeAlbum(title string) bool {
	return len(title) < 30 && 
		   !strings.Contains(title, " - ") && 
		   !mp.looksLikeCompilation(title)
}

// looksLikeSingle determines if a title looks like a single
func (mp *MetadataProcessor) looksLikeSingle(title string) bool {
	return strings.Contains(title, "single") || len(title) < 20
}

// looksLikeCompilation determines if a title looks like a compilation
func (mp *MetadataProcessor) looksLikeCompilation(title string) bool {
	compilationKeywords := []string{
		"various", "compilation", "hits", "best of", 
		"collection", "playlist", "dmc", "brit awards", "cool grooves",
	}
	
	for _, keyword := range compilationKeywords {
		if strings.Contains(title, keyword) {
			return true
		}
	}
	return false
}

// scoreByFormat scores releases based on their media format
func (mp *MetadataProcessor) scoreByFormat(media []musicbrainz.Media) int {
	for _, m := range media {
		format := strings.ToLower(m.Format)
		switch format {
		case "digital media":
			return 40
		case "cd", "vinyl", "cassette", "dvd", "blu-ray":
			return -20
		}
	}
	return 0
}

// getCacheKey generates a cache key for an album
func getCacheKey(artist, album string) string {
	return fmt.Sprintf("%s|%s", artist, album)
}

// addField adds a field to vorbis comment only if value is not empty
func addField(comment *flacvorbis.MetaDataBlockVorbisComment, field, value string) {
	if value != "" {
		comment.Add(field, value)
	}
}

// getAlbumTitle determines the best album title to use
func getAlbumTitle(track shared.Track, album *shared.Album) string {
	if album != nil && album.Title != "" {
		return album.Title
	}
	if track.Album != "" {
		return track.Album
	}
	return "Unknown Album"
}

// getAlbumArtist determines the best album artist to use
func getAlbumArtist(track shared.Track, album *shared.Album) string {
	if album != nil && album.Artist != "" {
		return album.Artist
	}
	if track.AlbumArtist != "" {
		return track.AlbumArtist
	}
	return track.Artist
}

// getReleaseDate determines the best release date to use
func getReleaseDate(track shared.Track, album *shared.Album) string {
	if track.ReleaseDate != "" {
		return track.ReleaseDate
	}
	if album != nil && album.ReleaseDate != "" {
		return album.ReleaseDate
	}
	return ""
}

// getGenre determines the best genre to use
func getGenre(track shared.Track, album *shared.Album) string {
	if track.Genre != "" && track.Genre != "Unknown" {
		return track.Genre
	}
	if album != nil && album.Genre != "" && album.Genre != "Unknown" {
		return album.Genre
	}
	return ""
}

// detectImageFormat detects the image format from the data
func detectImageFormat(data []byte) string {
	if len(data) < 4 {
		return "image/jpeg"
	}

	// PNG signature (89 50 4E 47)
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}

	// JPEG signature (FF D8)
	if data[0] == 0xFF && data[1] == 0xD8 {
		return "image/jpeg"
	}

	// WebP signature (RIFF...WEBP)
	if len(data) >= 12 && 
		data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 {
		return "image/webp"
	}

	// GIF signature (GIF8)
	if len(data) >= 4 && 
		data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
		return "image/gif"
	}

	return "image/jpeg"
}

// ============================================================================
// 7. Global Compatibility Layer (for backward compatibility)
// ============================================================================

var (
	// Global instance for backward compatibility
	globalProcessor = NewMetadataProcessor()
)

// SetMusicBrainzDebug sets debug mode for the global MusicBrainz client
func SetMusicBrainzDebug(debug bool) {
	globalProcessor.SetDebugMode(debug)
}

// AddMetadata adds comprehensive metadata to a FLAC file (global function for compatibility)
func AddMetadata(filePath string, track shared.Track, album *shared.Album, coverData []byte, totalTracks int, warningCollector *shared.WarningCollector) error {
	return globalProcessor.AddMetadata(filePath, track, album, coverData, totalTracks, warningCollector)
}

// AddMetadataWithDebug adds comprehensive metadata to a FLAC file with debug mode support (global function for compatibility)
func AddMetadataWithDebug(filePath string, track shared.Track, album *shared.Album, coverData []byte, totalTracks int, warningCollector *shared.WarningCollector, debug bool) error {
	return globalProcessor.AddMetadataWithDebug(filePath, track, album, coverData, totalTracks, warningCollector, debug)
}

// FindReleaseIDFromISRC attempts to find a MusicBrainz release ID from tracks with ISRC (global function for compatibility)
func FindReleaseIDFromISRC(tracks []shared.Track, albumArtist, albumTitle string) {
	globalProcessor.FindReleaseIDFromISRC(tracks, albumArtist, albumTitle)
}

// GetISRCMetadata extracts comprehensive metadata from ISRC lookup (global function for compatibility)
func GetISRCMetadata(isrc string) (*ISRCMetadata, error) {
	return globalProcessor.GetISRCMetadata(isrc)
}

// GetISRCMetadataWithTrackCount extracts comprehensive metadata from ISRC lookup with intelligent release selection (global function for compatibility)
func GetISRCMetadataWithTrackCount(isrc string, expectedTrackCount int) (*ISRCMetadata, error) {
	return globalProcessor.GetISRCMetadataWithTrackCount(isrc, expectedTrackCount)
}

// GetCacheStats returns statistics about the current cache state (global function for compatibility)
func GetCacheStats() (int, []string) {
	return globalProcessor.GetCacheStats()
}

// ClearAlbumCache clears the global album metadata cache (global function for compatibility)
func ClearAlbumCache() {
	globalProcessor.ClearCache()
}