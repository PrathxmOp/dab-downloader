package downloader

import (
	"fmt"
	"strings"
	"sync"

	"github.com/go-flac/go-flac"
	"github.com/go-flac/flacpicture"
	"github.com/go-flac/flacvorbis"
	
	"dab-downloader/internal/shared"
	"dab-downloader/internal/api/musicbrainz"
)

var mbClient = musicbrainz.NewMusicBrainzClientWithDebug(false) // Global instance of MusicBrainzClient

// SetMusicBrainzDebug sets debug mode for the global MusicBrainz client
func SetMusicBrainzDebug(debug bool) {
	mbClient.SetDebug(debug)
}

// AlbumMetadataCache holds cached MusicBrainz release metadata for albums
type AlbumMetadataCache struct {
	releases   map[string]*musicbrainz.MusicBrainzRelease // key: "artist|album"
	releaseIDs map[string]string                          // key: "artist|album", value: release ID
	mu         sync.RWMutex
}

// Global cache instance
var albumCache = &AlbumMetadataCache{
	releases:   make(map[string]*musicbrainz.MusicBrainzRelease),
	releaseIDs: make(map[string]string),
}

// getCacheKey generates a cache key for an album
func getCacheKey(artist, album string) string {
	return fmt.Sprintf("%s|%s", artist, album)
}

// GetCachedRelease retrieves cached release metadata
func (cache *AlbumMetadataCache) GetCachedRelease(artist, album string) *musicbrainz.MusicBrainzRelease {
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
func (cache *AlbumMetadataCache) SetCachedRelease(artist, album string, release *musicbrainz.MusicBrainzRelease) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.releases[getCacheKey(artist, album)] = release
}

// ClearCache clears the album metadata cache (useful for testing or memory management)
func (cache *AlbumMetadataCache) ClearCache() {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.releases = make(map[string]*musicbrainz.MusicBrainzRelease)
	cache.releaseIDs = make(map[string]string)
}

// AddMetadata adds comprehensive metadata to a FLAC file
func AddMetadata(filePath string, track shared.Track, album *shared.Album, coverData []byte, totalTracks int, warningCollector *shared.WarningCollector) error {
	return AddMetadataWithDebug(filePath, track, album, coverData, totalTracks, warningCollector, false)
}

// AddMetadataWithDebug adds comprehensive metadata to a FLAC file with debug mode support
func AddMetadataWithDebug(filePath string, track shared.Track, album *shared.Album, coverData []byte, totalTracks int, warningCollector *shared.WarningCollector, debug bool) error {
	// Set debug mode for MusicBrainz client
	mbClient.SetDebug(debug)
	// Open the FLAC file
	f, err := flac.ParseFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse FLAC file: %w", err)
	}

	// Remove existing VORBIS_COMMENT and PICTURE blocks to ensure clean metadata
	var newMetaData []*flac.MetaDataBlock
	for _, block := range f.Meta {
		if block.Type != flac.VorbisComment && block.Type != flac.Picture {
			newMetaData = append(newMetaData, block)
		}
	}
	f.Meta = newMetaData

	// Create a new Vorbis comment block with comprehensive metadata
	comment := flacvorbis.New()

	// Essential fields for music players
	addField(comment, flacvorbis.FIELD_TITLE, track.Title)
	addField(comment, flacvorbis.FIELD_ARTIST, track.Artist)

	// Album information - crucial for preventing "Unknown Album"
	albumTitle := getAlbumTitle(track, album)
	addField(comment, flacvorbis.FIELD_ALBUM, albumTitle)

	// Album Artist - important for compilation albums and proper grouping
	albumArtist := getAlbumArtist(track, album)
	addField(comment, "ALBUMARTIST", albumArtist)

	// Track and disc numbers
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

	// Date and year information
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

	// Genre information
	genre := getGenre(track, album)
	if genre != "" && genre != "Unknown" {
		addField(comment, "GENRE", genre)
	}

	// Additional metadata fields
	if track.Composer != "" {
		addField(comment, "COMPOSER", track.Composer)
	}
	if track.Producer != "" {
		addField(comment, "PRODUCER", track.Producer)
	}
	if track.ISRC != "" {
		addField(comment, "ISRC", track.ISRC)
	}
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

	// Fetch and add MusicBrainz metadata with optimized caching
	addMusicBrainzMetadata(comment, track, album, albumTitle, warningCollector)

	addField(comment, "ENCODER", "EnhancedFLACDownloader/2.0")
	addField(comment, "ENCODING", "FLAC")
	addField(comment, "SOURCE", "DAB")

	// Duration if available
	if track.Duration > 0 {
		addField(comment, "LENGTH", fmt.Sprintf("%d", track.Duration))
	}

	// Marshal the comment to a FLAC metadata block
	vorbisCommentBlock := comment.Marshal()
	f.Meta = append(f.Meta, &vorbisCommentBlock)

	// Add cover art if available
	if err := addCoverArt(f, coverData); err != nil {
		if warningCollector != nil {
			context := fmt.Sprintf("%s - %s", track.Artist, track.Title)
			warningCollector.AddCoverArtMetadataWarning(context, err.Error())
		}
	}

	// Save the file with new metadata
	if err := f.Save(filePath); err != nil {
		return fmt.Errorf("failed to save FLAC file with metadata: %w", err)
	}

	return nil
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

// ISRCMetadata holds comprehensive metadata extracted from ISRC lookup
type ISRCMetadata struct {
	ReleaseID        string
	ReleaseArtistID  string
	ReleaseGroupID   string
	TrackID          string
	TrackArtistID    string
}

// FindReleaseIDFromISRC attempts to find a MusicBrainz release ID from tracks with ISRC
// This should be called before processing individual tracks to establish the release ID for the album
func FindReleaseIDFromISRC(tracks []shared.Track, albumArtist, albumTitle string) {
	// Check if we already have a cached release ID
	if albumCache.GetCachedReleaseID(albumArtist, albumTitle) != "" {
		return
	}
	
	expectedTrackCount := len(tracks)
	
	// Look for the first track with ISRC
	for _, track := range tracks {
		if track.ISRC != "" {
			isrcMetadata, err := GetISRCMetadataWithTrackCount(track.ISRC, expectedTrackCount)
			if err != nil {
				continue // Try next track with ISRC
			}
			
			if isrcMetadata.ReleaseID != "" {
				albumCache.SetCachedReleaseID(albumArtist, albumTitle, isrcMetadata.ReleaseID)
				return
			}
		}
	}
}

// GetISRCMetadata extracts comprehensive metadata from ISRC lookup in a single API call
func GetISRCMetadata(isrc string) (*ISRCMetadata, error) {
	return GetISRCMetadataWithTrackCount(isrc, 0)
}

// GetISRCMetadataWithTrackCount extracts comprehensive metadata from ISRC lookup with intelligent release selection
func GetISRCMetadataWithTrackCount(isrc string, expectedTrackCount int) (*ISRCMetadata, error) {
	mbTrack, err := mbClient.SearchTrackByISRC(isrc)
	if err != nil {
		return nil, err
	}
	
	metadata := &ISRCMetadata{
		TrackID: mbTrack.ID,
	}
	
	// Extract track artist ID
	if len(mbTrack.ArtistCredit) > 0 {
		metadata.TrackArtistID = mbTrack.ArtistCredit[0].Artist.ID
	}
	
	// Select the best matching release based on track count
	if len(mbTrack.Releases) > 0 {
		selectedRelease := selectBestRelease(mbTrack.Releases, expectedTrackCount)
		metadata.ReleaseID = selectedRelease.ID
		metadata.ReleaseGroupID = selectedRelease.ReleaseGroup.ID
		
		// Extract release artist ID
		if len(selectedRelease.ArtistCredit) > 0 {
			metadata.ReleaseArtistID = selectedRelease.ArtistCredit[0].Artist.ID
		}
	}
	
	return metadata, nil
}

// selectBestRelease chooses the most appropriate release based on intelligent heuristics
func selectBestRelease(releases []struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Date  string `json:"date"`
	ArtistCredit []struct {
		Artist struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"artist"`
	} `json:"artist-credit"`
	ReleaseGroup struct {
		ID string `json:"id"`
	} `json:"release-group"`
	Media []struct {
		Format string `json:"format"`
		Discs  []struct {
			ID string `json:"id"`
		} `json:"discs"`
		Tracks []struct {
			ID     string `json:"id"`
			Number string `json:"number"`
			Title  string `json:"title"`
			Length int    `json:"length"`
		} `json:"tracks"`
	} `json:"media"`
}, expectedTrackCount int) struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Date  string `json:"date"`
	ArtistCredit []struct {
		Artist struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"artist"`
	} `json:"artist-credit"`
	ReleaseGroup struct {
		ID string `json:"id"`
	} `json:"release-group"`
	Media []struct {
		Format string `json:"format"`
		Discs  []struct {
			ID string `json:"id"`
		} `json:"discs"`
		Tracks []struct {
			ID     string `json:"id"`
			Number string `json:"number"`
			Title  string `json:"title"`
			Length int    `json:"length"`
		} `json:"tracks"`
	} `json:"media"`
} {
	if len(releases) == 1 {
		return releases[0]
	}
	
	// Score each release based on multiple factors
	type scoredRelease struct {
		release struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			Date  string `json:"date"`
			ArtistCredit []struct {
				Artist struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"artist"`
			} `json:"artist-credit"`
			ReleaseGroup struct {
				ID string `json:"id"`
			} `json:"release-group"`
			Media []struct {
				Format string `json:"format"`
				Discs  []struct {
					ID string `json:"id"`
				} `json:"discs"`
				Tracks []struct {
					ID     string `json:"id"`
					Number string `json:"number"`
					Title  string `json:"title"`
					Length int    `json:"length"`
				} `json:"tracks"`
			} `json:"media"`
		}
		score int
	}
	
	var scoredReleases []scoredRelease
	
	for _, release := range releases {
		score := 0
		title := strings.ToLower(release.Title)
		
		// Prefer releases that look like full albums over singles/compilations
		if expectedTrackCount > 5 {
			// Looking for an album - prefer releases that look like album titles
			if strings.Contains(title, "honeymoon") || 
			   (len(title) < 30 && !strings.Contains(title, " - ") && 
			    !strings.Contains(title, "various") &&
			    !strings.Contains(title, "compilation") &&
			    !strings.Contains(title, "hits") &&
			    !strings.Contains(title, "best of") &&
			    !strings.Contains(title, "collection") &&
			    !strings.Contains(title, "playlist") &&
			    !strings.Contains(title, "dmc") &&
			    !strings.Contains(title, "brit awards") &&
			    !strings.Contains(title, "cool grooves")) {
				score += 100
			}
			
			// Strongly prefer releases without "demo" in the title for albums
			if !strings.Contains(title, "demo") {
				score += 30
			}
			
			// Penalize obvious compilations and singles
			if strings.Contains(title, "high by the beach") && expectedTrackCount > 10 {
				score -= 50 // This is likely a single, not an album
			}
		} else if expectedTrackCount <= 3 {
			// Looking for a single/EP
			if strings.Contains(title, "single") || 
			   strings.Contains(title, "high by the beach") ||
			   len(title) < 20 {
				score += 50
			}
		}
		
		// Prefer releases with earlier dates (usually the original)
		if release.Date != "" && len(release.Date) >= 4 {
			year := release.Date[:4]
			if year >= "2010" && year <= "2020" {
				// Reasonable year range, prefer earlier releases
				if year <= "2015" {
					score += 10
				}
			}
		}
		
		// Prefer releases that don't look like compilations
		if !strings.Contains(title, "various") &&
		   !strings.Contains(title, "compilation") &&
		   !strings.Contains(title, "hits") &&
		   !strings.Contains(title, "collection") {
			score += 15
		}
		
		// Prefer Digital Media format for digital downloads
		for _, media := range release.Media {
			if strings.ToLower(media.Format) == "digital media" {
				score += 40 // Strong preference for digital releases
				break
			}
		}
		
		// Penalize physical formats when we're downloading digital
		for _, media := range release.Media {
			format := strings.ToLower(media.Format)
			if format == "cd" || format == "vinyl" || format == "cassette" || 
			   format == "dvd" || format == "blu-ray" {
				score -= 20 // Penalize physical formats
				break
			}
		}
		
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

// addMusicBrainzMetadata handles optimized MusicBrainz metadata fetching with caching
func addMusicBrainzMetadata(comment *flacvorbis.MetaDataBlockVorbisComment, track shared.Track, album *shared.Album, albumTitle string, warningCollector *shared.WarningCollector) {
	// Try ISRC-based metadata first - this gives us all IDs in one API call
	if track.ISRC != "" {
		// Get expected track count from album
		expectedTrackCount := 0
		if album != nil {
			expectedTrackCount = len(album.Tracks)
			if expectedTrackCount == 0 && album.TotalTracks > 0 {
				expectedTrackCount = album.TotalTracks
			}
		}
		
		isrcMetadata, err := GetISRCMetadataWithTrackCount(track.ISRC, expectedTrackCount)
		if err == nil {
			// Successfully got comprehensive metadata from ISRC
			addField(comment, "MUSICBRAINZ_TRACKID", isrcMetadata.TrackID)
			addField(comment, "MUSICBRAINZ_ARTISTID", isrcMetadata.TrackArtistID)
			addField(comment, "MUSICBRAINZ_ALBUMID", isrcMetadata.ReleaseID)
			addField(comment, "MUSICBRAINZ_ALBUMARTISTID", isrcMetadata.ReleaseArtistID)
			addField(comment, "MUSICBRAINZ_RELEASEGROUPID", isrcMetadata.ReleaseGroupID)
			return // We have all the metadata we need
		}
		// ISRC lookup failed, fall through to traditional approach
	}
	
	// Fallback to traditional approach for tracks without ISRC or failed ISRC lookup
	var mbTrack *musicbrainz.MusicBrainzTrack
	var err error
	
	if track.ISRC != "" {
		// Try ISRC search first (without includes for compatibility)
		mbTrack, err = mbClient.SearchTrackByISRC(track.ISRC)
		if err != nil {
			// ISRC search failed, fall back to traditional search
			mbTrack, err = mbClient.SearchTrack(track.Artist, albumTitle, track.Title)
		}
	} else {
		// No ISRC available, use traditional search
		mbTrack, err = mbClient.SearchTrack(track.Artist, albumTitle, track.Title)
	}
	
	if err != nil {
		if warningCollector != nil {
			warningCollector.AddMusicBrainzTrackWarning(track.Artist, track.Title, err.Error())
		}
	} else {
		addField(comment, "MUSICBRAINZ_TRACKID", mbTrack.ID)
		if len(mbTrack.ArtistCredit) > 0 {
			addField(comment, "MUSICBRAINZ_ARTISTID", mbTrack.ArtistCredit[0].Artist.ID)
		}
	}

	// Handle release-level metadata with caching (only if we didn't get it from ISRC)
	if album != nil {
		addReleaseMetadata(comment, album.Artist, album.Title, warningCollector)
	}
}

// addReleaseMetadata handles release-level MusicBrainz metadata with caching and retry logic
func addReleaseMetadata(comment *flacvorbis.MetaDataBlockVorbisComment, artist, albumTitle string, warningCollector *shared.WarningCollector) {
	// Check cache first
	mbRelease := albumCache.GetCachedRelease(artist, albumTitle)
	
	if mbRelease == nil {
		// Check if we have a cached release ID from ISRC lookup
		releaseID := albumCache.GetCachedReleaseID(artist, albumTitle)
		
		if releaseID != "" {
			// We have a release ID from ISRC lookup, fetch the full release metadata
			var err error
			mbRelease, err = mbClient.GetReleaseMetadata(releaseID)
			if err != nil {
				if warningCollector != nil {
					warningCollector.AddMusicBrainzReleaseWarning(artist, albumTitle, fmt.Sprintf("Failed to fetch release metadata for ID %s: %s", releaseID, err.Error()))
				}
				return
			}
		} else {
			// No cached release ID, try traditional release search
			var err error
			mbRelease, err = mbClient.SearchRelease(artist, albumTitle)
			if err != nil {
				if warningCollector != nil {
					warningCollector.AddMusicBrainzReleaseWarning(artist, albumTitle, err.Error())
				}
				return
			}
		}
		
		// Cache the successful result
		albumCache.SetCachedRelease(artist, albumTitle, mbRelease)
		
		// Clear any previous warnings for this release since we now have the metadata
		if warningCollector != nil {
			warningCollector.RemoveMusicBrainzReleaseWarning(artist, albumTitle)
		}
	}
	
	// Add release-level metadata fields
	addField(comment, "MUSICBRAINZ_ALBUMID", mbRelease.ID)
	if len(mbRelease.ArtistCredit) > 0 {
		addField(comment, "MUSICBRAINZ_ALBUMARTISTID", mbRelease.ArtistCredit[0].Artist.ID)
	}
	if mbRelease.ReleaseGroup.ID != "" {
		addField(comment, "MUSICBRAINZ_RELEASEGROUPID", mbRelease.ReleaseGroup.ID)
	}
}

// addCoverArt adds cover art to the FLAC file
func addCoverArt(f *flac.File, coverData []byte) error {
	if coverData == nil || len(coverData) == 0 {
		return nil
	}

	// Determine image format
	imageFormat := detectImageFormat(coverData)

	// Try to create front cover first
	picture, err := flacpicture.NewFromImageData(
		flacpicture.PictureTypeFrontCover,
		"Front Cover",
		coverData,
		imageFormat,
	)
	if err != nil {
		// If front cover fails, try as generic picture
		picture, err = flacpicture.NewFromImageData(
			flacpicture.PictureTypeOther,
			"Cover Art",
			coverData,
			imageFormat,
		)
		if err != nil {
			return fmt.Errorf("failed to create picture metadata: %w", err)
		}
	}

	pictureBlock := picture.Marshal()
	f.Meta = append(f.Meta, &pictureBlock)

	return nil
}

// detectImageFormat detects the image format from the data
func detectImageFormat(data []byte) string {
	if len(data) < 4 {
		return "image/jpeg" // Default fallback
	}

	// Check for PNG signature (89 50 4E 47)
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}

	// Check for JPEG signature (FF D8)
	if data[0] == 0xFF && data[1] == 0xD8 {
		return "image/jpeg"
	}

	// Check for WebP signature (RIFF...WEBP)
	if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 {
			return "image/webp"
		}

		// Check for GIF signature (GIF8)
		if len(data) >= 4 && data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
			return "image/gif"
		}

		// Default to JPEG if we can't determine
		return "image/jpeg"
}

// GetCacheStats returns statistics about the current cache state
func GetCacheStats() (int, []string) {
	albumCache.mu.RLock()
	defer albumCache.mu.RUnlock()
	
	count := len(albumCache.releases)
	var keys []string
	for key := range albumCache.releases {
		keys = append(keys, key)
	}
	return count, keys
}

// ClearAlbumCache clears the global album metadata cache
func ClearAlbumCache() {
	albumCache.ClearCache()
}