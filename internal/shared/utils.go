package shared

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"math/rand"
	"log"
	"net/http"
	"time"

	"github.com/mattn/go-isatty"
)

// Constants
const (
	DefaultMaxRetries = 3
	UserAgent        = "dab-downloader/2.0"
)

// HTTPError represents an HTTP error with status code
type HTTPError struct {
	StatusCode int
	Status     string
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s - %s", e.StatusCode, e.Status, e.Message)
}

// IsRetryableHTTPError checks if an HTTP error should be retried
func IsRetryableHTTPError(err error) bool {
	// Unwrap the error if it's wrapped
	for err != nil {
		if httpErr, ok := err.(*HTTPError); ok {
			switch httpErr.StatusCode {
			case http.StatusServiceUnavailable, // 503
				http.StatusTooManyRequests,     // 429
				http.StatusBadGateway,          // 502
				http.StatusGatewayTimeout:      // 504
				return true
			}
		}
		// Try to unwrap the error further
		if unwrapped, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapped.Unwrap()
		} else {
			break
		}
	}
	return false
}

// RetryWithBackoff retries the given function with exponential backoff.
func RetryWithBackoff(maxRetries int, initialDelaySec int, fn func() error) error {
	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}

		// Calculate delay with exponential backoff and some jitter
		delay := time.Duration(initialDelaySec) * time.Second * (1 << attempt)
		jitter := time.Duration(rand.Intn(100)) * time.Millisecond
		time.Sleep(delay + jitter)
	}
	return fmt.Errorf("failed after %d attempts: %w", maxRetries, err)
}

// RetryWithBackoffForHTTP retries HTTP requests with smart error handling
func RetryWithBackoffForHTTP(maxRetries int, initialDelay time.Duration, maxDelay time.Duration, fn func() error) error {
	return RetryWithBackoffForHTTPWithDebug(maxRetries, initialDelay, maxDelay, fn, false)
}

// RetryWithBackoffForHTTPWithDebug retries HTTP requests with smart error handling and optional debug logging
func RetryWithBackoffForHTTPWithDebug(maxRetries int, initialDelay time.Duration, maxDelay time.Duration, fn func() error, debug bool) error {
	var lastErr error

	if maxRetries == 0 { // If no retries, just execute once
		return fn()
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		// Check if this is a retryable HTTP error
		if !IsRetryableHTTPError(lastErr) {
			return lastErr // Don't retry non-retryable errors
		}

		if attempt == maxRetries-1 {
			break // Don't sleep on the last attempt
		}

		// Calculate delay with exponential backoff and jitter
		delay := initialDelay * time.Duration(1<<uint(attempt))
		if delay > maxDelay {
			delay = maxDelay
		}
		
		// Add jitter (±25% of delay)
		jitter := time.Duration(rand.Int63n(int64(delay/2))) - delay/4
		finalDelay := delay + jitter
		
		if finalDelay < 0 {
			finalDelay = delay
		}

		// Only log retry messages in debug mode
		if debug {
			log.Printf("HTTP request failed (attempt %d/%d): %v. Retrying in %v", 
				attempt+1, maxRetries, lastErr, finalDelay)
		}
		
		time.Sleep(finalDelay)
	}
	
	return fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// GetUserInput prompts the user for input with a default value
func GetUserInput(prompt, defaultValue string) string {
	if defaultValue != "" {
		prompt = fmt.Sprintf("%s [%s]", prompt, defaultValue)
	}
	ColorPrompt.Print(prompt + ": ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input == "" && defaultValue != "" {
			return defaultValue
		}
		return input
	}
	return defaultValue
}

// SanitizeFileName cleans a string to make it safe for use as a file name
func SanitizeFileName(name string) string {
	// Replace invalid characters with underscores
	invalidChars := []string{"<", ">", ":", `"`, `/`, `\\`, `|`, `?`, `*`, "\x00"}
	result := name
	for _, char := range invalidChars {
		result = strings.ReplaceAll(result, char, "_")
	}
	// Remove leading/trailing spaces and periods
	result = strings.Trim(result, " .")
	// Limit length to avoid filesystem issues
	if len(result) > 255 {
		result = result[:255]
	}
	// Ensure the name is not empty
	if result == "" {
		result = "unknown"
	}
	return result
}

// FileExists checks if a file exists at the given path
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// GetTrackFilename generates a filename for a track
func GetTrackFilename(trackNumber int, title string) string {
	if trackNumber == 0 {
		return fmt.Sprintf("%s.flac", SanitizeFileName(title))
	}
	return fmt.Sprintf("%02d - %s.flac", trackNumber, SanitizeFileName(title))
}

// TruncateString truncates a string to the specified length, adding ellipsis if truncated.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func IdToString(id interface{}) string {
	switch v := id.(type) {
	case string:
		return v
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	default:
		return ""
	}
}

// GetYesNoInput prompts the user for a yes/no input with a default value
func GetYesNoInput(prompt string, defaultValue string) bool {
	for {
		input := GetUserInput(prompt, defaultValue)
		switch strings.ToLower(input) {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			ColorError.Printf("❌ Invalid input. Please enter 'y' or 'n'.\n")
		}
	}
}

// ParseSelectionInput parses a string like "1-7, 10, 12-15" into a slice of unique integers.
func ParseSelectionInput(input string, max int) ([]int, error) {
	selected := make(map[int]bool)
	var result []int

	parts := strings.Split(input, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") {
			// Handle range, e.g., "1-7"
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range format: %s", part)
			}
			start, err1 := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err1 != nil {
				return nil, fmt.Errorf("invalid start of range: %s", rangeParts[0])
			}
			end, err2 := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err2 != nil {
				return nil, fmt.Errorf("invalid end of range: %s", rangeParts[1])
			}

			if start > end {
				start, end = end, start // Swap if start is greater than end
			}

			for i := start; i <= end; i++ {
				if i >= 1 && i <= max && !selected[i] {
					selected[i] = true
					result = append(result, i)
				}
			}
		} else {
			// Handle single number, e.g., "10"
			num, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid number: %s", part)
			}
			if num >= 1 && num <= max && !selected[num] {
				selected[num] = true
				result = append(result, num)
			}
		}
	}

	return result, nil
}

func IsTTY() bool {
	return isatty.IsTerminal(os.Stdout.Fd())
}

// removeSuffix removes a suffix from a track title
func RemoveSuffix(trackTitle string, suffix string) string {
	re := regexp.MustCompile(fmt.Sprintf(`(?i)( - |\s*\()((\d{4} )?)?(%s(ed)?( Version)?|Digital (Master?|%s(ed)?)|Remix)( \d{4})?(\))?`, suffix, suffix))
	return re.ReplaceAllString(trackTitle, "")
}

// VerifyFileSize checks if a file exists and matches the expected size
func VerifyFileSize(filePath string, expectedSize int64) (bool, int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return false, 0, fmt.Errorf("failed to stat file: %w", err)
	}
	
	actualSize := info.Size()
	return actualSize == expectedSize, actualSize, nil
}

// VerifyFileIntegrity performs additional checks on downloaded files
func VerifyFileIntegrity(filePath string, expectedSize int64, debug bool) error {
	if expectedSize <= 0 {
		if debug {
			fmt.Printf("DEBUG: Skipping file integrity check for %s - no expected size available\n", filePath)
		}
		return nil // Skip verification if no expected size
	}

	matches, actualSize, err := VerifyFileSize(filePath, expectedSize)
	if err != nil {
		return fmt.Errorf("file verification failed: %w", err)
	}

	if !matches {
		return fmt.Errorf("file size mismatch: expected %d bytes, got %d bytes", expectedSize, actualSize)
	}

	if debug {
		fmt.Printf("DEBUG: File integrity verified for %s - %d bytes\n", filePath, actualSize)
	}

	return nil
}

// CreateDirIfNotExists creates a directory if it doesn't exist
func CreateDirIfNotExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// CheckFFmpeg checks if ffmpeg is installed and available in the system's PATH.
func CheckFFmpeg() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

// FormatBitrateInfo formats audio quality information with colors
func FormatBitrateInfo(audioQuality AudioQuality) string {
	if audioQuality.MaximumSamplingRate == 0 && audioQuality.MaximumBitDepth == 0 {
		return ""
	}
	
	// Format sampling rate with proper alignment
	var samplingRateStr string
	if audioQuality.MaximumSamplingRate == float64(int(audioQuality.MaximumSamplingRate)) {
		// Whole number sampling rates
		intRate := int(audioQuality.MaximumSamplingRate)
		if intRate < 100 { // 24, 48, 96 need extra space for alignment
			samplingRateStr = fmt.Sprintf(" %d", intRate)
		} else {
			samplingRateStr = fmt.Sprintf("%d", intRate)
		}
	} else {
		// Decimal sampling rates (like 44.1)
		samplingRateStr = fmt.Sprintf("%.1f", audioQuality.MaximumSamplingRate)
	}
	
	bitrateInfo := fmt.Sprintf("[%s/%d]", samplingRateStr, audioQuality.MaximumBitDepth)
	
	// Color the bitrate info based on quality
	if audioQuality.IsHiRes || audioQuality.MaximumBitDepth >= 24 || audioQuality.MaximumSamplingRate > 48.0 {
		return ColorSuccess.Sprint(bitrateInfo) // Green for hi-res
	} else if audioQuality.MaximumBitDepth >= 16 {
		return ColorWarning.Sprint(bitrateInfo) // Yellow for CD quality
	} else {
		return ColorError.Sprint(bitrateInfo) // Red for lower quality
	}
}

// calculateRawBitrateLength calculates the length of bitrate info without ANSI color codes
func calculateRawBitrateLength(audioQuality AudioQuality) int {
	if audioQuality.MaximumSamplingRate == 0 && audioQuality.MaximumBitDepth == 0 {
		return 0
	}
	
	// Use the same logic as FormatBitrateInfo for consistency
	var samplingRateStr string
	if audioQuality.MaximumSamplingRate == float64(int(audioQuality.MaximumSamplingRate)) {
		// Whole number sampling rates
		intRate := int(audioQuality.MaximumSamplingRate)
		if intRate < 100 { // 24, 48, 96 need extra space for alignment
			samplingRateStr = fmt.Sprintf(" %d", intRate)
		} else {
			samplingRateStr = fmt.Sprintf("%d", intRate)
		}
	} else {
		// Decimal sampling rates (like 44.1)
		samplingRateStr = fmt.Sprintf("%.1f", audioQuality.MaximumSamplingRate)
	}
	
	return len(fmt.Sprintf("[%s/%d]", samplingRateStr, audioQuality.MaximumBitDepth))
}

// GetTerminalWidth returns the terminal width, defaulting to 80 if unable to determine
func GetTerminalWidth() int {
	// Try to get terminal width from environment or use a reasonable default
	if IsTTY() {
		// For now, use a reasonable default. In a full implementation, you might use
		// a library like golang.org/x/term to get actual terminal dimensions
		return 100
	}
	return 80
}

// FormatAlbumWithBitrate formats an album line with right-aligned bitrate info
func FormatAlbumWithBitrate(prefix, title, artist, date string, audioQuality AudioQuality) string {
	bitrateInfo := FormatBitrateInfo(audioQuality)
	if bitrateInfo == "" {
		return fmt.Sprintf("%s%s - %s (%s)", prefix, title, artist, date)
	}
	
	// Calculate the main text without bitrate
	mainText := fmt.Sprintf("%s%s - %s (%s)", prefix, title, artist, date)
	
	// Get terminal width and calculate spacing
	termWidth := GetTerminalWidth()
	
	// Calculate raw bitrate length for spacing calculation
	rawBitrateLen := calculateRawBitrateLength(audioQuality)
	
	// Calculate spacing needed
	spacesNeeded := termWidth - len(mainText) - rawBitrateLen - 1
	if spacesNeeded < 1 {
		spacesNeeded = 1 // At least one space
	}
	
	return fmt.Sprintf("%s%s%s", mainText, strings.Repeat(" ", spacesNeeded), bitrateInfo)
}

// FormatAlbumWithTrackCount formats an album line with track count and right-aligned bitrate info
func FormatAlbumWithTrackCount(prefix, title, artist, date, trackCount string, audioQuality AudioQuality) string {
	bitrateInfo := FormatBitrateInfo(audioQuality)
	
	// Calculate the main text without track count and bitrate
	mainText := fmt.Sprintf("%s%s - %s (%s)", prefix, title, artist, date)
	
	// If no bitrate info, just append track count
	if bitrateInfo == "" {
		return fmt.Sprintf("%s %s", mainText, trackCount)
	}
	
	// Get terminal width and calculate spacing
	termWidth := GetTerminalWidth()
	
	// Calculate raw lengths for spacing (without ANSI color codes)
	rawBitrateLen := calculateRawBitrateLength(audioQuality)
	
	// Estimate raw track count length (without ANSI color codes)
	// Simple estimation: "[X Track]" or "[XX Tracks]"
	rawTrackCountLen := 9 // Reasonable estimate for "[X Track]" or "[XX Tracks]"
	if strings.Contains(trackCount, "Tracks]") {
		rawTrackCountLen = 11 // "[XX Tracks]"
	}
	
	// Calculate spacing needed between main text and track count + bitrate
	totalRightSideLen := rawTrackCountLen + 1 + rawBitrateLen // +1 for space between track count and bitrate
	spacesNeeded := termWidth - len(mainText) - totalRightSideLen - 1
	if spacesNeeded < 1 {
		spacesNeeded = 1 // At least one space
	}
	
	return fmt.Sprintf("%s%s%s %s", mainText, strings.Repeat(" ", spacesNeeded), trackCount, bitrateInfo)
}

// FormatAlbumWithTrackCountProfessional formats an album line with professional styling and better alignment
func FormatAlbumWithTrackCountProfessional(prefix, title, artist, date, trackCount string, audioQuality AudioQuality) string {
	bitrateInfo := FormatBitrateInfo(audioQuality)
	
	// Create the main album info with better formatting
	// Use a more subtle separator and better typography
	albumInfo := fmt.Sprintf("%s • %s", title, artist)
	if date != "" {
		albumInfo = fmt.Sprintf("%s (%s)", albumInfo, date)
	}
	
	// Combine prefix with album info
	mainText := fmt.Sprintf("%s%s", prefix, albumInfo)
	
	// Handle very long titles by truncating if necessary
	termWidthForTrunc := GetTerminalWidth()
	if termWidthForTrunc < 100 {
		termWidthForTrunc = 100
	}
	maxMainTextLen := termWidthForTrunc - 30 // Reserve space for track count and bitrate
	if len(mainText) > maxMainTextLen {
		// Truncate and add ellipsis
		mainText = mainText[:maxMainTextLen-3] + "..."
	}
	
	// If no bitrate info, just append track count with some spacing
	if bitrateInfo == "" {
		// Add some padding for better alignment
		spacesNeeded := 80 - len(mainText) - 12 // Estimate for track count
		if spacesNeeded < 2 {
			spacesNeeded = 2
		}
		return fmt.Sprintf("%s%s%s", mainText, strings.Repeat(" ", spacesNeeded), trackCount)
	}
	
	// Get terminal width and calculate spacing for professional alignment
	termWidth := GetTerminalWidth()
	if termWidth < 100 {
		termWidth = 100 // Minimum width for professional display
	}
	
	// Calculate raw lengths for spacing (without ANSI color codes)
	rawBitrateLen := calculateRawBitrateLength(audioQuality)
	
	// Estimate raw track count length (without ANSI color codes)
	// New format: "[ 1 Track ]" or "[ X Tracks]" or "[XX Tracks]"
	rawTrackCountLen := 11 // Reasonable estimate for "[ X Tracks]" or "[XX Tracks]"
	if strings.Contains(trackCount, " 1 Track ]") {
		rawTrackCountLen = 10 // "[ 1 Track ]"
	}
	
	// Calculate spacing for better alignment
	// Reserve space for track count + space + bitrate
	rightSideLen := rawTrackCountLen + 1 + rawBitrateLen
	availableSpace := termWidth - len(mainText) - rightSideLen
	
	// Ensure minimum spacing
	if availableSpace < 2 {
		availableSpace = 2
	}
	
	// Create clean spacing for professional alignment
	spacing := strings.Repeat(" ", availableSpace)
	
	return fmt.Sprintf("%s%s%s %s", mainText, spacing, trackCount, bitrateInfo)
}

// FormatTrackWithBitrate formats a track line with right-aligned bitrate info
func FormatTrackWithBitrate(prefix, title, artist, album string, audioQuality AudioQuality) string {
	bitrateInfo := FormatBitrateInfo(audioQuality)
	if bitrateInfo == "" {
		return fmt.Sprintf("%s%s - %s (%s)", prefix, title, artist, album)
	}
	
	// Calculate the main text without bitrate
	mainText := fmt.Sprintf("%s%s - %s (%s)", prefix, title, artist, album)
	
	// Get terminal width and calculate spacing
	termWidth := GetTerminalWidth()
	
	// Calculate raw bitrate length for spacing calculation
	rawBitrateLen := calculateRawBitrateLength(audioQuality)
	
	// Calculate spacing needed
	spacesNeeded := termWidth - len(mainText) - rawBitrateLen - 1
	if spacesNeeded < 1 {
		spacesNeeded = 1 // At least one space
	}
	
	return fmt.Sprintf("%s%s%s", mainText, strings.Repeat(" ", spacesNeeded), bitrateInfo)
}