package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/mattn/go-isatty"
)

// GetUserInput prompts the user for input with a default value
func GetUserInput(prompt, defaultValue string) string {
	if defaultValue != "" {
		prompt = fmt.Sprintf("%s [%s]", prompt, defaultValue)
	}
	colorPrompt.Print(prompt + ": ")
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

// CreateDirIfNotExists creates a directory if it does not exist
func CreateDirIfNotExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// GetTrackFilename generates a filename for a track
func GetTrackFilename(trackNumber int, title string) string {
	if trackNumber == 0 {
		return fmt.Sprintf("%s.flac", SanitizeFileName(title))
	}
	return fmt.Sprintf("%02d - %s.flac", trackNumber, SanitizeFileName(title))
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(filePath string, config *Config) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %%w", err)
	}
	if err := json.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %%w", err)
	}
	return nil
}

// SaveConfig saves configuration to a JSON file
func SaveConfig(filePath string, config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %%w", err)
	}
	dir := filepath.Dir(filePath)
	if err := CreateDirIfNotExists(dir); err != nil {
		return fmt.Errorf("failed to create config directory: %%w", err)
	}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %%w", err)
	}
	return nil
}


// TruncateString truncates a string to the specified length, adding ellipsis if truncated.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func idToString(id interface{}) string {
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
			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid start of range: %s", rangeParts[0])
			}
			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
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

	// Sort the results for consistent order
	// sort.Ints(result) // Not strictly necessary for functionality, but good for consistency

	return result, nil
}

func isTTY() bool {
	return isatty.IsTerminal(os.Stdout.Fd())
}

// removeSuffix removes a suffix from a track title
func removeSuffix(trackTitle string, suffix string) string {
	re := regexp.MustCompile(fmt.Sprintf(`(?i)( - |\s*\()((\d{4} )?)?(%s(ed)?( Version)?|Digital (Master?|%s(ed)?)|Remix)( \d{4})?(\))?$`, suffix, suffix))
	return re.ReplaceAllString(trackTitle, "")
}