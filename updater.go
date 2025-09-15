package main

import (
	"encoding/json"
	"fmt"
	"net/http"
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
		fmt.Printf("Error checking for updates: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error checking for updates: GitHub API returned status %d\n", resp.StatusCode)
		return
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		fmt.Printf("Error decoding GitHub API response: %v\n", err)
		return
	}

	latestVersion := release.TagName

	if isNewerVersion(latestVersion, currentVersion) {
			colorWarning.Printf("A new version (%s) of dab-downloader is available! You are running %s.\n", latestVersion, currentVersion)
			colorWarning.Println("Please update to get the latest features and bug fixes.")
		} else {
			colorSuccess.Println("You are running the latest version of dab-downloader.")
		}
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
