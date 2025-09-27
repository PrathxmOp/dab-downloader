package main

import (
	"fmt"
	"sync"

	"github.com/go-flac/go-flac"
	"github.com/go-flac/flacpicture"
	"github.com/go-flac/flacvorbis"
	
)

var mbClient = NewMusicBrainzClient() // Global instance of MusicBrainzClient

// SetMusicBrainzDebug sets debug mode for the global MusicBrainz client
func SetMusicBrainzDebug(debug bool) {
	mbClient.SetDebug(debug)
}

// AlbumMetadataCache holds cached MusicBrainz release metadata for albums
type AlbumMetadataCache struct {
	releases map[string]*MusicBrainzRelease // key: "artist|album"
	mu       sync.RWMutex
}

// Global cache instance
var albumCache = &AlbumMetadataCache{
	releases: make(map[string]*MusicBrainzRelease),
}

// getCacheKey generates a cache key for an album
func getCacheKey(artist, album string) string {
	return fmt.Sprintf("%s|%s", artist, album)
}

// GetCachedRelease retrieves cached release metadata
func (cache *AlbumMetadataCache) GetCachedRelease(artist, album string) *MusicBrainzRelease {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return cache.releases[getCacheKey(artist, album)]
}

// SetCachedRelease stores release metadata in cache
func (cache *AlbumMetadataCache) SetCachedRelease(artist, album string, release *MusicBrainzRelease) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.releases[getCacheKey(artist, album)] = release
}

// ClearCache clears the album metadata cache (useful for testing or memory management)
func (cache *AlbumMetadataCache) ClearCache() {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.releases = make(map[string]*MusicBrainzRelease)
}

// AddMetadata adds comprehensive metadata to a FLAC file
func AddMetadata(filePath string, track Track, album *Album, coverData []byte, totalTracks int, warningCollector *WarningCollector) error {
	return AddMetadataWithDebug(filePath, track, album, coverData, totalTracks, warningCollector, false)
}

// AddMetadataWithDebug adds comprehensive metadata to a FLAC file with debug mode support
func AddMetadataWithDebug(filePath string, track Track, album *Album, coverData []byte, totalTracks int, warningCollector *WarningCollector, debug bool) error {
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

	// Technical and source information
	// addField(comment, "MUSICBRAINZ_TRACKID", idToString(track.ID)) // This is wrong
	// if album != nil && album.ID != "" {
	// 	addField(comment, "MUSICBRAINZ_ALBUMID", album.ID) // This is wrong
	// }

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
func getAlbumTitle(track Track, album *Album) string {
	if album != nil && album.Title != "" {
		return album.Title
	}
	if track.Album != "" {
		return track.Album
	}
	return "Unknown Album"
}

// getAlbumArtist determines the best album artist to use
func getAlbumArtist(track Track, album *Album) string {
	if album != nil && album.Artist != "" {
		return album.Artist
	}
	if track.AlbumArtist != "" {
		return track.AlbumArtist
	}
	return track.Artist
}

// getReleaseDate determines the best release date to use
func getReleaseDate(track Track, album *Album) string {
	if track.ReleaseDate != "" {
		return track.ReleaseDate
	}
	if album != nil && album.ReleaseDate != "" {
		return album.ReleaseDate
	}
	return ""
}

// getGenre determines the best genre to use
func getGenre(track Track, album *Album) string {
	if track.Genre != "" && track.Genre != "Unknown" {
		return track.Genre
	}
	if album != nil && album.Genre != "" && album.Genre != "Unknown" {
		return album.Genre
	}
	return ""
}

// addMusicBrainzMetadata handles optimized MusicBrainz metadata fetching with caching
func addMusicBrainzMetadata(comment *flacvorbis.MetaDataBlockVorbisComment, track Track, album *Album, albumTitle string, warningCollector *WarningCollector) {
	// Fetch track-specific metadata
	mbTrack, err := mbClient.SearchTrack(track.Artist, albumTitle, track.Title)
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

	// Handle release-level metadata with caching
	if album != nil {
		addReleaseMetadata(comment, album.Artist, album.Title, warningCollector)
	}
}

// addReleaseMetadata handles release-level MusicBrainz metadata with caching and retry logic
func addReleaseMetadata(comment *flacvorbis.MetaDataBlockVorbisComment, artist, albumTitle string, warningCollector *WarningCollector) {
	// Check cache first
	mbRelease := albumCache.GetCachedRelease(artist, albumTitle)
	
	if mbRelease == nil {
		// Not in cache, fetch from MusicBrainz
		var err error
		mbRelease, err = mbClient.SearchRelease(artist, albumTitle)
		if err != nil {
			if warningCollector != nil {
				warningCollector.AddMusicBrainzReleaseWarning(artist, albumTitle, err.Error())
			}
			return
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