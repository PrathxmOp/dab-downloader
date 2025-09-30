package shared

import (
	"fmt"
	"sort"
	"strings"
)

// WarningType represents different types of warnings
type WarningType int

const (
	MusicBrainzTrackWarning WarningType = iota
	MusicBrainzReleaseWarning
	CoverArtDownloadWarning
	CoverArtMetadataWarning
	AlbumFetchWarning
	TrackSkippedWarning
)

// Warning represents a single warning with context
type Warning struct {
	Type     WarningType
	Message  string
	Context  string // Track/Album context
	Details  string // Additional details like error message
}

// WarningCollector collects warnings during download operations
type WarningCollector struct {
	warnings []Warning
	enabled  bool
}

// NewWarningCollector creates a new warning collector
func NewWarningCollector(enabled bool) *WarningCollector {
	return &WarningCollector{
		warnings: make([]Warning, 0),
		enabled:  enabled,
	}
}

// AddWarning adds a warning to the collector
func (wc *WarningCollector) AddWarning(warningType WarningType, context, message, details string) {
	if !wc.enabled {
		return
	}
	
	warning := Warning{
		Type:    warningType,
		Message: message,
		Context: context,
		Details: details,
	}
	wc.warnings = append(wc.warnings, warning)
}

// AddMusicBrainzTrackWarning adds a MusicBrainz track lookup warning
func (wc *WarningCollector) AddMusicBrainzTrackWarning(artist, title, details string) {
	context := fmt.Sprintf("%s - %s", artist, title)
	wc.AddWarning(MusicBrainzTrackWarning, context, "Failed to find MusicBrainz track", details)
}

// AddMusicBrainzReleaseWarning adds a MusicBrainz release lookup warning
func (wc *WarningCollector) AddMusicBrainzReleaseWarning(artist, album, details string) {
	context := fmt.Sprintf("%s - %s", artist, album)
	wc.AddWarning(MusicBrainzReleaseWarning, context, "Failed to find MusicBrainz release", details)
}

// AddCoverArtDownloadWarning adds a cover art download warning
func (wc *WarningCollector) AddCoverArtDownloadWarning(album, details string) {
	wc.AddWarning(CoverArtDownloadWarning, album, "Could not download cover art", details)
}

// AddCoverArtMetadataWarning adds a cover art metadata warning
func (wc *WarningCollector) AddCoverArtMetadataWarning(context, details string) {
	wc.AddWarning(CoverArtMetadataWarning, context, "Failed to add cover art to metadata", details)
}

// AddAlbumFetchWarning adds an album fetch warning
func (wc *WarningCollector) AddAlbumFetchWarning(trackTitle, trackID, details string) {
	context := fmt.Sprintf("%s (ID: %s)", trackTitle, trackID)
	wc.AddWarning(AlbumFetchWarning, context, "Could not fetch album info", details)
}

// AddTrackSkippedWarning adds a track skipped warning
func (wc *WarningCollector) AddTrackSkippedWarning(trackPath string) {
	wc.AddWarning(TrackSkippedWarning, trackPath, "Track already exists", "")
}

// RemoveWarningsByTypeAndContext removes warnings of a specific type and context
func (wc *WarningCollector) RemoveWarningsByTypeAndContext(warningType WarningType, context string) {
	if !wc.enabled {
		return
	}
	
	var filteredWarnings []Warning
	for _, warning := range wc.warnings {
		// Keep warnings that don't match the type and context
		if warning.Type != warningType || warning.Context != context {
			filteredWarnings = append(filteredWarnings, warning)
		}
	}
	wc.warnings = filteredWarnings
}

// RemoveMusicBrainzReleaseWarning removes a specific MusicBrainz release warning
func (wc *WarningCollector) RemoveMusicBrainzReleaseWarning(artist, album string) {
	context := fmt.Sprintf("%s - %s", artist, album)
	wc.RemoveWarningsByTypeAndContext(MusicBrainzReleaseWarning, context)
}

// HasWarnings returns true if there are any warnings
func (wc *WarningCollector) HasWarnings() bool {
	return len(wc.warnings) > 0
}

// GetWarningCount returns the total number of warnings
func (wc *WarningCollector) GetWarningCount() int {
	return len(wc.warnings)
}

// GetWarningsByType returns warnings grouped by type
func (wc *WarningCollector) GetWarningsByType() map[WarningType][]Warning {
	grouped := make(map[WarningType][]Warning)
	for _, warning := range wc.warnings {
		grouped[warning.Type] = append(grouped[warning.Type], warning)
	}
	return grouped
}

// PrintSummary prints a formatted summary of all warnings
func (wc *WarningCollector) PrintSummary() {
	if !wc.HasWarnings() {
		return
	}

	ColorWarning.Printf("\n⚠️  Warning Summary (%d warnings):\n", len(wc.warnings))
	ColorWarning.Println(strings.Repeat("─", 50))

	grouped := wc.GetWarningsByType()
	
	// Sort warning types for consistent output
	var types []WarningType
	for warningType := range grouped {
		types = append(types, warningType)
	}
	sort.Slice(types, func(i, j int) bool { return types[i] < types[j] })

	for _, warningType := range types {
		warnings := grouped[warningType]
		wc.printWarningTypeSection(warningType, warnings)
	}
}

// printWarningTypeSection prints warnings for a specific type
func (wc *WarningCollector) printWarningTypeSection(warningType WarningType, warnings []Warning) {
	if len(warnings) == 0 {
		return
	}

	// Print section header
	sectionTitle := wc.getWarningTypeTitle(warningType)
	ColorWarning.Printf("\n%s (%d):\n", sectionTitle, len(warnings))

	// Group similar warnings to avoid repetition
	contextCounts := make(map[string]int)
	for _, warning := range warnings {
		contextCounts[warning.Context]++
	}

	// Sort contexts for consistent output
	var contexts []string
	for context := range contextCounts {
		contexts = append(contexts, context)
	}
	sort.Strings(contexts)

	// Print warnings, showing count if multiple
	for _, context := range contexts {
		count := contextCounts[context]
		if count > 1 {
			ColorWarning.Printf("  • %s (×%d)\n", context, count)
		} else {
			ColorWarning.Printf("  • %s\n", context)
		}
	}
}

// getWarningTypeTitle returns a human-readable title for a warning type
func (wc *WarningCollector) getWarningTypeTitle(warningType WarningType) string {
	switch warningType {
	case MusicBrainzTrackWarning:
		return "MusicBrainz Track Lookup Failures"
	case MusicBrainzReleaseWarning:
		return "MusicBrainz Release Lookup Failures"
	case CoverArtDownloadWarning:
		return "Cover Art Download Failures"
	case CoverArtMetadataWarning:
		return "Cover Art Metadata Failures"
	case AlbumFetchWarning:
		return "Album Information Fetch Failures"
	case TrackSkippedWarning:
		return "Tracks Skipped (Already Exist)"
	default:
		return "Other Warnings"
	}
}