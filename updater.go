package main

import (
	"bufio"
	"encoding/json"
	"fmt" // Added this import
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	version "github.com/hashicorp/go-version" // Added this import
)



// CheckForUpdates checks for a newer version on GitHub
func CheckForUpdates(config *Config, currentVersion string) {
	if config.DisableUpdateCheck {
		colorInfo.Println("Skipping update check as DisableUpdateCheck is enabled in config.")
		return
	}

	// Fetch remote version.json
	repoURL := "PrathxmOp/dab-downloader" // Default value
	if config.UpdateRepo != "" {
		repoURL = config.UpdateRepo
	}
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/main/version/version.json", repoURL)
	resp, err := http.Get(rawURL)
	if err != nil {
		colorError.Printf("Error checking for updates: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		colorError.Printf("Error checking for updates: GitHub API returned status %d\n", resp.StatusCode)
		return
	}

	var remoteVersionInfo VersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&remoteVersionInfo); err != nil {
		colorError.Printf("Error decoding remote version.json: %v\n", err)
		return
	}

	latestVersion := remoteVersionInfo.Version


	if isNewerVersion(latestVersion, currentVersion) {
		colorError.Printf("🚨 You are using an outdated version (%s) of dab-downloader! A new version (%s) is available.\n", currentVersion, latestVersion)
		colorPrompt.Print("Would you like to update now? (Y/n): ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.ToLower(strings.TrimSpace(input))

		if input == "y" || input == "" {
			colorInfo.Println("Attempting to open the Update Guide in your browser...")
			updateURL := "https://github.com/PrathxmOp/dab-downloader/#option-1-using-auto-dl.sh-script-recommended"
			if err := openBrowser(updateURL, config); err != nil {
				colorWarning.Printf("Failed to open browser automatically: %v\n", err)
				colorInfo.Println("Please refer to the 'Update Guide' section in the README for detailed instructions:")
				colorInfo.Println("https://github.com/PrathxmOp/dab-downloader/#update-guide")
			}
		} else {
			colorInfo.Println("You can update later by referring to the 'Update Guide' in the README.")
		}
	} else {
		colorSuccess.Println("✅ You are running the latest version of dab-downloader.")
	}
}

func openBrowser(url string, config *Config) error {
	if config.IsDockerContainer {
		colorInfo.Printf("Running in Docker, please open the update guide manually: %s\n", url)
		return nil
	}

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

// isNewerVersion compares two versions using semantic versioning
func isNewerVersion(latest, current string) bool {
	vLatest, err := version.NewVersion(latest)
	if err != nil {
		colorWarning.Printf("⚠️ Error parsing latest version '%s': %v\n", latest, err)
		return false // Cannot determine if newer, assume not
	}

	vCurrent, err := version.NewVersion(current)
	if err != nil {
		colorWarning.Printf("⚠️ Error parsing current version '%s': %v\n", current, err)
		return false // Cannot determine if newer, assume not
	}

	return vLatest.GreaterThan(vCurrent)
}

// extractDateFromVersion is no longer needed with semantic versioning
// func extractDateFromVersion(version string) string {
// 	if strings.HasPrefix(version, "v") && strings.Contains(version, "-") {
// 		parts := strings.Split(version[1:], "-") // Remove 'v' and split by '-'
// 		if len(parts) > 0 && len(parts[0]) == 8 { // Check if it looks like YYYYMMDD
// 			return parts[0]
// 		}
// 	}
// 	return ""
// }
