package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"dab-downloader/internal/api"
	"dab-downloader/internal/config"
	"dab-downloader/internal/downloader"
	"dab-downloader/internal/ui"
	"dab-downloader/internal/updater"
	"dab-downloader/internal/utils"
	"dab-downloader/internal/version"
	"dab-downloader/pkg/netutil"
)

var toolVersion string

const authorName = "Prathxm"

//go:embed version.json
var versionJSON []byte

var (
	apiURL              string
	downloadLocation    string
	debug               bool
	filter              string
	noConfirm           bool
	searchType          string
	spotifyPlaylist     string
	spotifyClientID     string
	spotifyClientSecret string
	auto                bool
	expandPlaylist      bool
	expandNavidrome     bool
	navidromeURL        string
	navidromeUsername   string
	navidromePassword   string
	format              string = "flac"
	bitrate             string = "320"
	ignoreSuffix        string
	insecure            bool
	warningBehavior     string = "summary"
)

var rootCmd = &cobra.Command{
	Use:   "dab-downloader",
	Short: "A high-quality FLAC music downloader for the DAB API.",
}

func initConfigAPIAndDownloader() (*config.Config, *api.DabAPI, *downloader.Downloader) {
	color.NoColor = !utils.IsTTY()
	homeDir, err := os.UserHomeDir()
	if err != nil {
		ui.Warning.Println("⚠️ Could not determine home directory, will use current directory for downloads.")
		homeDir = "."
	}

	conf := &config.Config{
		APIURL:           "https://dabmusic.xyz",
		DownloadLocation: filepath.Join(homeDir, "Music"),
		Parallelism:      5,
		UpdateRepo:       "PrathxmOp/dab-downloader",
		VerifyDownloads:  true,
		MaxRetryAttempts: config.DefaultMaxRetries,
		WarningBehavior:  "summary",
	}

	configFile := config.GetConfigFilePath()

	if !utils.FileExists(configFile) {
		ui.Info.Println("✨ Welcome to DAB Downloader! Let's set up your configuration.")

		conf.APIURL = utils.GetUserInput(fmt.Sprintf("Enter DAB API URL (e.g., %s)", conf.APIURL), conf.APIURL)
		conf.DownloadLocation = utils.GetUserInput(fmt.Sprintf("Enter download location (e.g., %s)", conf.DownloadLocation), conf.DownloadLocation)

		defaultParallelism := strconv.Itoa(conf.Parallelism)
		parallelismStr := utils.GetUserInput(fmt.Sprintf("Enter number of parallel downloads (default: %s)", defaultParallelism), defaultParallelism)
		if p, err := strconv.Atoi(parallelismStr); err == nil && p > 0 {
			conf.Parallelism = p
		}

		conf.SpotifyClientID = utils.GetUserInput("Enter your Spotify Client ID", "")
		conf.SpotifyClientSecret = utils.GetUserInput("Enter your Spotify Client Secret", "")
		conf.NavidromeURL = utils.GetUserInput("Enter your Navidrome URL", "")
		conf.NavidromeUsername = utils.GetUserInput("Enter your Navidrome Username", "")
		conf.NavidromePassword = utils.GetUserInput("Enter your Navidrome Password", "")
		conf.Format = utils.GetUserInput("Enter default output format (e.g., flac, mp3, ogg, opus)", "flac")
		conf.Bitrate = utils.GetUserInput("Enter default bitrate for lossy formats (e.g., 320)", "320")
		conf.UpdateRepo = utils.GetUserInput("Enter GitHub repository for updates", "PrathxmOp/dab-downloader")

		if err := config.SaveConfig(configFile, conf); err != nil {
			ui.Error.Printf("❌ Failed to save initial config: %v\n", err)
		} else {
			ui.Success.Println("✅ Configuration saved to", configFile)
		}
	} else {
		if err := config.LoadConfig(configFile, conf); err != nil {
			ui.Error.Printf("❌ Failed to load config from %s: %v\n", configFile, err)
		} else {
			ui.Info.Println("✅ Loaded configuration from", configFile)
			if conf.Format == "" {
				conf.Format = "flac"
			}
			if conf.Bitrate == "" {
				conf.Bitrate = "320"
			}
		}
	}

	if apiURL != "" {
		conf.APIURL = apiURL
	}
	if downloadLocation != "" {
		conf.DownloadLocation = downloadLocation
	}
	if spotifyClientID != "" {
		conf.SpotifyClientID = spotifyClientID
	}
	if spotifyClientSecret != "" {
		conf.SpotifyClientSecret = spotifyClientSecret
	}
	if navidromeURL != "" {
		conf.NavidromeURL = navidromeURL
	}
	if navidromeUsername != "" {
		conf.NavidromeUsername = navidromeUsername
	}
	if navidromePassword != "" {
		conf.NavidromePassword = navidromePassword
	}
	if format != "flac" {
		conf.Format = format
	}
	if bitrate != "320" {
		conf.Bitrate = bitrate
	}
	if warningBehavior != "summary" {
		conf.WarningBehavior = warningBehavior
	}

	httpClient := netutil.NewRobustHTTPClient(10*time.Minute, insecure)

	a := api.NewDabAPI(conf.APIURL, conf.DownloadLocation, httpClient)
	d := downloader.NewDownloader(a, conf.DownloadLocation)
	return conf, a, d
}

func init() {
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "", "DAB API URL")
	rootCmd.PersistentFlags().StringVar(&downloadLocation, "download-location", "", "Directory to save downloads")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&insecure, "insecure", false, "Skip TLS certificate verification")
	rootCmd.PersistentFlags().StringVar(&warningBehavior, "warnings", "summary", "Warning behavior")

	albumCmd.Flags().StringVar(&format, "format", "flac", "Format to convert to")
	albumCmd.Flags().StringVar(&bitrate, "bitrate", "320", "Bitrate for lossy formats")

	artistCmd.Flags().StringVar(&filter, "filter", "all", "Filter by item type")
	artistCmd.Flags().BoolVar(&noConfirm, "no-confirm", false, "Skip confirmation prompt")
	artistCmd.Flags().StringVar(&format, "format", "flac", "Format to convert to")
	artistCmd.Flags().StringVar(&bitrate, "bitrate", "320", "Bitrate for lossy formats")

	searchCmd.Flags().StringVar(&searchType, "type", "all", "Type of content to search for")
	searchCmd.Flags().BoolVar(&auto, "auto", false, "Automatically download first result")
	searchCmd.Flags().StringVar(&format, "format", "flac", "Format to convert to")
	searchCmd.Flags().StringVar(&bitrate, "bitrate", "320", "Bitrate for lossy formats")

	playlistCmd.Flags().StringVar(&format, "format", "flac", "Format to convert to")
	playlistCmd.Flags().StringVar(&bitrate, "bitrate", "320", "Bitrate for lossy formats")

	spotifyCmd.Flags().StringVar(&spotifyPlaylist, "spotify", "", "Spotify playlist URL")
	spotifyCmd.Flags().BoolVar(&auto, "auto", false, "Automatically download first result")
	spotifyCmd.Flags().BoolVar(&expandPlaylist, "expand", false, "Expand playlist to full albums")
	spotifyCmd.Flags().StringVar(&format, "format", "flac", "Format to convert to")
	spotifyCmd.Flags().StringVar(&bitrate, "bitrate", "320", "Bitrate for lossy formats")

	rootCmd.PersistentFlags().StringVar(&spotifyClientID, "spotify-client-id", "", "Spotify Client ID")
	rootCmd.PersistentFlags().StringVar(&spotifyClientSecret, "spotify-client-secret", "", "Spotify Client Secret")

	rootCmd.PersistentFlags().StringVar(&navidromeURL, "navidrome-url", "", "Navidrome URL")
	rootCmd.PersistentFlags().StringVar(&navidromeUsername, "navidrome-username", "", "Navidrome Username")
	rootCmd.PersistentFlags().StringVar(&navidromePassword, "navidrome-password", "", "Navidrome Password")
	navidromeCmd.Flags().StringVar(&ignoreSuffix, "ignore-suffix", "", "Ignore suffix in search")
	navidromeCmd.Flags().BoolVar(&expandNavidrome, "expand", false, "Expand to full albums")
	navidromeCmd.Flags().BoolVar(&auto, "auto", false, "Automatically download first result")

	listenbrainzCmd.Flags().BoolVar(&auto, "auto", false, "Automatically download first result")
	listenbrainzCmd.Flags().StringVar(&format, "format", "flac", "Format to convert to")
	listenbrainzCmd.Flags().StringVar(&bitrate, "bitrate", "320", "Bitrate for lossy formats")

	rootCmd.AddCommand(artistCmd)
	rootCmd.AddCommand(albumCmd)
	rootCmd.AddCommand(playlistCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(spotifyCmd)
	rootCmd.AddCommand(navidromeCmd)
	rootCmd.AddCommand(listenbrainzCmd)
	rootCmd.AddCommand(batchCmd)
	rootCmd.AddCommand(addToPlaylistCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(debugCmd)

	debugCmd.AddCommand(testApiAvailabilityCmd)
	debugCmd.AddCommand(testArtistEndpointsCmd)
	debugCmd.AddCommand(testPlaylistEndpointsCmd)
	debugCmd.AddCommand(comprehensiveArtistDebugCmd)

	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	var vInfo version.VersionInfo
	if err := json.Unmarshal(versionJSON, &vInfo); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading embedded version.json: %v\n", err)
		os.Exit(1)
	}

toolVersion = vInfo.Version
	rootCmd.Version = toolVersion

	conf, _, _ := initConfigAPIAndDownloader()

	if _, err := os.Stat("/.dockerenv"); err == nil {
		conf.IsDockerContainer = true
	}

	updater.CheckForUpdates(conf, toolVersion)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
