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
	"time"

	"github.com/hashicorp/go-version"
	
	"dab-downloader/internal/config"
	"dab-downloader/internal/ui"
	internalversion "dab-downloader/internal/version"
	"dab-downloader/pkg/netutil"
)

// CheckForUpdates checks for a newer version on GitHub
func CheckForUpdates(conf *config.Config, currentVersion string) {
	if conf.DisableUpdateCheck {
		ui.Info.Println("Skipping update check as DisableUpdateCheck is enabled in config.")
		return
	}

	// Fetch remote version.json
	repoURL := "PrathxmOp/dab-downloader" // Default value
	if conf.UpdateRepo != "" {
		repoURL = conf.UpdateRepo
	}
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/main/cmd/dab-downloader/version.json", repoURL)
	
	client := netutil.NewRobustHTTPClient(30*time.Second, false)
	resp, err := client.Get(rawURL)
	if err != nil {
		ui.Error.Printf("Error checking for updates: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ui.Error.Printf("Error checking for updates: GitHub API returned status %d\n", resp.StatusCode)
		return
	}

	var remoteVersionInfo internalversion.VersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&remoteVersionInfo); err != nil {
		ui.Error.Printf("Error decoding remote version.json: %v\n", err)
		return
	}

	latestVersion := remoteVersionInfo.Version


	if isNewerVersion(latestVersion, currentVersion) {
		ui.Error.Printf("🚨 You are using an outdated version (%s) of dab-downloader! A new version (%s) is available.\n", currentVersion, latestVersion)
		ui.Prompt.Print("Would you like to update now? (Y/n): ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.ToLower(strings.TrimSpace(input))

		if input == "y" || input == "" {
			ui.Info.Println("Attempting to open the Installation Guide in your browser...")
			updateURL := "https://github.com/PrathxmOp/dab-downloader#-installation"
			if err := openBrowser(updateURL, conf); err != nil {
				ui.Warning.Printf("Failed to open browser automatically: %v\n", err)
				ui.Info.Println("Please refer to the 'Installation' section in the README for detailed instructions:")
				ui.Info.Println("https://github.com/PrathxmOp/dab-downloader#-installation")
			}
		} else {
			ui.Info.Println("You can update later by referring to the 'Update Guide' in the README.")
		}
	} else {
		ui.Success.Println("✅ You are running the latest version of dab-downloader.")
	}
}

func openBrowser(url string, conf *config.Config) error {
	if conf.IsDockerContainer {
		ui.Info.Printf("Running in Docker, please open the update guide manually: %s\n", url)
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
		ui.Warning.Printf("⚠️ Error parsing latest version '%s': %v\n", latest, err)
		return false // Cannot determine if newer, assume not
	}

	vCurrent, err := version.NewVersion(current)
	if err != nil {
		ui.Warning.Printf("⚠️ Error parsing current version '%s': %v\n", current, err)
		return false // Cannot determine if newer, assume not
	}

	return vLatest.GreaterThan(vCurrent)
}