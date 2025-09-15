package main

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// GitHubRelease represents a simplified structure of a GitHub release API response
type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

// CheckForUpdates checks for a newer version on GitHub
func CheckForUpdates(currentVersion string) {
	resp, err := http.Get("https://api.github.com/repos/PrathxmOp/dab-downloader/releases/latest")
	if err != nil {
		colorError.Printf("Error checking for updates: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		colorError.Printf("Error checking for updates: GitHub API returned status %d\n", resp.StatusCode)
		return
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		colorError.Printf("Error decoding GitHub API response: %v\n", err)
		return
	}

	latestVersion := release.TagName

	if isNewerVersion(latestVersion, currentVersion) {
		colorError.Printf("🚨 You are using an outdated version (%s) of dab-downloader! A new version (%s) is available.\n", currentVersion, latestVersion)
		colorPrompt.Print("Would you like to update now? (Y/n): ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.ToLower(strings.TrimSpace(input))

		if input == "y" || input == "" {
			colorInfo.Println("Attempting to open the Update Guide in your browser...")
			updateURL := "https://github.com/PrathxmOp/dab-downloader/#-update-guide"
			if err := openBrowser(updateURL); err != nil {
				colorWarning.Printf("Failed to open browser automatically: %v\n", err)
				colorInfo.Println("Please refer to the 'Update Guide' section in the README for detailed instructions:")
				colorInfo.Println(updateURL)
			}
		} else {
			colorInfo.Println("You can update later by referring to the 'Update Guide' in the README.")
		}
	} else {
		colorSuccess.Println("✅ You are running the latest version of dab-downloader.")
	}
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}

// isNewerVersion compares two versions in vYYYYMMDD-commit_hash format
func isNewerVersion(latest, current string) bool {


	// Extract date parts
	latestDateStr := extractDateFromVersion(latest)
	currentDateStr := extractDateFromVersion(current)

	if latestDateStr == "" || currentDateStr == "" {
		// Fallback to simple string comparison if date extraction fails
		return latest > current
	}

	latestDate, err := time.Parse("20060102", latestDateStr)
	if err != nil {
		return latest > current // Fallback
	}
	currentDate, err := time.Parse("20060102", currentDateStr)
	if err != nil {
		return latest > current // Fallback
	}

	if latestDate.After(currentDate) {
		return true
	}
	if latestDate.Before(currentDate) {
		return false
	}

	// If dates are equal, compare the full version string (which includes commit hash)
	return latest > current
}

func extractDateFromVersion(version string) string {
	if strings.HasPrefix(version, "v") && strings.Contains(version, "-") {
		parts := strings.Split(version[1:], "-") // Remove 'v' and split by '-'
		if len(parts) > 0 && len(parts[0]) == 8 { // Check if it looks like YYYYMMDD
			return parts[0]
		}
	}
	return ""
}
