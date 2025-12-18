package updater

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	version "github.com/hashicorp/go-version"
	
	"dab-downloader/internal/shared"
	"dab-downloader/internal/config"
)

// CheckForUpdates checks for a newer version on GitHub
func CheckForUpdates(config *config.Config, currentVersion string) {
	if config.DisableUpdateCheck {
		shared.ColorInfo.Println("Skipping update check as DisableUpdateCheck is enabled in config.")
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
		shared.ColorError.Printf("Error checking for updates: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		shared.ColorError.Printf("Error checking for updates: GitHub API returned status %d\n", resp.StatusCode)
		return
	}

	var remoteVersionInfo shared.VersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&remoteVersionInfo); err != nil {
		shared.ColorError.Printf("Error decoding remote version.json: %v\n", err)
		return
	}

	latestVersion := remoteVersionInfo.Version

	if isNewerVersion(latestVersion, currentVersion) {
		shared.ColorError.Printf("🚨 You are using an outdated version (%s) of dab-downloader! A new version (%s) is available.\n", currentVersion, latestVersion)
		shared.ColorPrompt.Print("Would you like to update now? (Y/n): ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.ToLower(strings.TrimSpace(input))

		if input == "y" || input == "" {
			shared.ColorInfo.Println("Attempting to open the Update Guide in your browser...")
			updateURL := "https://github.com/PrathxmOp/dab-downloader/#option-1-using-auto-dl.sh-script-recommended"
			if err := openBrowser(updateURL, config); err != nil {
				shared.ColorWarning.Printf("Failed to open browser automatically: %v\n", err)
				shared.ColorInfo.Println("Please refer to the 'Update Guide' section in the README for detailed instructions:")
				shared.ColorInfo.Println("https://github.com/PrathxmOp/dab-downloader/#update-guide")
			}
		} else {
			shared.ColorInfo.Println("You can update later by referring to the 'Update Guide' in the README.")
		}
	} else {
		shared.ColorSuccess.Println("✅ You are running the latest version of dab-downloader.")
	}
}

func openBrowser(url string, config *config.Config) error {
	if config.IsDockerContainer {
		shared.ColorInfo.Printf("Running in Docker, please open the update guide manually: %s\n", url)
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
		shared.ColorWarning.Printf("⚠️ Error parsing latest version '%s': %v\n", latest, err)
		return false // Cannot determine if newer, assume not
	}

	vCurrent, err := version.NewVersion(current)
	if err != nil {
		shared.ColorWarning.Printf("⚠️ Error parsing current version '%s': %v\n", current, err)
		return false // Cannot determine if newer, assume not
	}

	return vLatest.GreaterThan(vCurrent)
}