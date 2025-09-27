# 🎵 DAB Music Downloader

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.19-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-Educational-green.svg)](#license)
[![Release](https://img.shields.io/github/v/release/PrathxmOp/dab-downloader)](https://github.com/PrathxmOp/dab-downloader/releases/latest)
[![Discord Support](https://img.shields.io/badge/Support-Discord-blue.svg?logo=discord&logoColor=white)](https://discord.gg/q9RnuVza2)
![Development Status](https://img.shields.io/badge/status-unstable%20development-orange.svg)

> A powerful, modular music downloader that delivers high-quality FLAC files with comprehensive metadata support through the DAB API.

## Table of Contents
- [⚠️ IMPORTANT: Development Status](#️-important-development-status)
- [✨ Key Features](#-key-features)
- [📸 Screenshots](#-screenshots)
- [🚀 Quick Start](#-quick-start)
  - [Option 1: Using `auto-dl.sh` Script (Recommended)](#option-1-using-auto-dlsh-script-recommended)
  - [Option 2: Pre-built Binary](#option-2-pre-built-binary)
  - [Option 3: Build from Source](#option-3-build-from-source)
  - [Option 4: Docker (Containerized)](#option-4-docker-containerized)
- [🔄 CRITICAL: Staying Updated](#-critical-staying-updated)
  - [🚨 Daily Update Routine (Recommended)](#-daily-update-routine-recommended)
  - [Versioning Format](#versioning-format)
  - [Option 1: Pre-built Binary Updates](#option-1-pre-built-binary-updates)
  - [Option 2: Source Code Updates](#option-2-source-code-updates)
  - [Option 3: Docker Updates](#option-3-docker-updates)
  - [🔔 Get Update Notifications](#-get-update-notifications)
- [📋 Usage Guide](#-usage-guide)
  - [🔍 Search and Discover](#-search-and-discover)
  - [📀 Download Content](#-download-content)
  - [🎧 Spotify Integration](#-spotify-integration)
  - [🎵 Navidrome Integration](#-navidrome-integration)
- [⚙️ Configuration](#️-configuration)
  - [First-Time Setup](#first-time-setup)
  - [Configuration File](#configuration-file)
- [⚙️ Command-Line Flags](#️-command-line-flags)
  - [Global Flags (Persistent Flags)](#global-flags-persistent-flags)
  - [Command-Specific Flags](#command-specific-flags)
    - [`album` command](#album-command)
    - [`artist` command](#artist-command)
    - [`search` command](#search-command)
    - [`spotify` command](#spotify-command)
    - [`navidrome` command](#navidrome-command)
- [📁 File Organization](#-file-organization)
- [🔧 Advanced Features](#-advanced-features)
  - [Debug Tools](#debug-tools)
  - [Quality & Metadata](#quality--metadata)
- [🐛 Troubleshooting](#-troubleshooting)
- [💬 Support & Community](#-support--community)
- [🏗️ Project Architecture](#️-project-architecture)
- [🤝 Contributing](#-contributing)
  - [Development Areas Needing Help](#development-areas-needing-help)
- [⚖️ Legal Notice](#️-legal-notice)
- [📄 License](#-license)
- [🌟 Support the Project](#-support-the-project)
- [Changelog](#changelog)
- [Update Guide](#update-guide)

## ⚠️ **IMPORTANT: Development Status**

🚧 **This project is currently in active, unstable development.** 🚧

- **Frequent Breaking Changes**: Features may work one day and break the next
- **Regular Updates Required**: You'll need to update frequently to get the latest fixes
- **Expect Issues**: Something always seems to break when i fix something else
- **Pre-Stable Release**: We're working toward a stable v1.0, but we're not there yet

**📢 We strongly recommend:**
- [Discord Support Group](https://discord.gg/q9RnuVza2) for real-time updates
- ✅ Checking for updates daily if you're actively using the tool
- ✅ Being prepared to troubleshoot and report issues
- ✅ Having patience as we work through the bugs

💬 **Need Help?** Join our [Discord Support Group](https://discord.gg/q9RnuVza2) for instant community support and the latest stability updates!



## ✨ Key Features

🔍 **Smart Search** - Find artists, albums, and tracks with intelligent filtering  
📦 **Complete Discographies** - Download entire artist catalogs with automatic categorization  
🏷️ **Rich Metadata** - Full tag support including genre, composer, producer, ISRC, and copyright  
🎨 **High-Quality Artwork** - Embedded album covers in original resolution  
- **Concurrent Downloads** - Fast parallel processing with real-time progress tracking  
- **Intelligent Retry Logic** - Robust error handling for reliable downloads  
- **Spotify Integration** - Import and download entire Spotify playlists and albums  
- **Format Conversion** - Convert downloaded FLAC files to MP3, OGG, Opus with configurable bitrates (requires FFmpeg)  
- **Navidrome Support** - Seamless integration with your music server  
- **Customizable Naming** - Define your own file and folder structure with configurable naming masks

## 📸 Screenshots

![img1](./screenshots/ScreenShot1.png)
![img1](./screenshots/ScreenShot2.png)

## 🚀 Quick Start

### Option 1: Using `auto-dl.sh` Script (Recommended)

This script simplifies the process of downloading and keeping `dab-downloader` updated. It fetches the latest version and places it in your current directory.

**Direct execution with curl:**
```bash
curl -fsSL https://raw.githubusercontent.com/PrathxmOp/Support-group-junk/main/Scripts/auto-dl.sh | bash
```

**Alternative methods:**

**Using wget (if curl is not available):**
```bash
wget -qO- https://raw.githubusercontent.com/PrathxmOp/Support-group-junk/main/Scripts/auto-dl.sh | bash
```

**Download first, then execute (safer approach):**
```bash
curl -fsSL -o auto-dl.sh https://raw.githubusercontent.com/PrathxmOp/Support-group-junk/main/Scripts/auto-dl.sh
chmod +x auto-dl.sh
./auto-dl.sh
```

### Option 2: Pre-built Binary

1. Download the latest release from our [GitHub Releases](https://github.com/PrathxmOp/dab-downloader/releases/latest)
2. Extract the archive.
3. Grant execute permissions to the binary:
   ```bash
   chmod +x ./dab-downloader-linux-arm64 # Or the appropriate binary for your system
   ```
4. Run the executable:
   ```bash
   ./dab-downloader-linux-arm64 # Or the appropriate binary for your system
   ```
5. Follow the interactive setup on first launch

### Option 3: Build from Source

**Prerequisites:**
- Go 1.19 or later ([Download here](https://golang.org/dl/))

```bash
# Clone the repository
git clone https://github.com/PrathxmOp/dab-downloader.git
cd dab-downloader

# Install dependencies and build
go mod tidy
go build -o dab-downloader
```

### Option 4: Docker (Containerized)

To run dab-downloader using a pre-built Docker image from Docker Hub:

1.  **Ensure Docker is installed:** Follow the official Docker installation guide for your system.
2.  **Configure with Docker Compose:**
    *   Make sure your `docker-compose.yml` file is configured to use the `prathxm/dab-downloader:latest` image (as updated by the latest changes).
    *   Create `config` and `music` directories if they don't exist:
        ```bash
        mkdir -p config music
        ```
    *   Copy the example configuration:
        ```bash
        cp config/example-config.json config/config.json
        ```
3.  **Run any command:**
    ```bash
    docker compose run dab-downloader search "your favorite artist"
    ```
    Or, to run in detached mode:
    ```bash
    docker compose up -d
    ```

## 🔄 **CRITICAL: Staying Updated**

Due to the unstable nature of this project, **regular updates are essential**:

### 🚨 **Daily Update Routine (Recommended)**

Since we're constantly fixing bugs and pushing updates, we recommend checking for updates daily:

```bash
# Check for new releases
./dab-downloader --version
```

### Versioning Format

The application uses a versioning format of `vYYYYMMDD-gCOMMIT_HASH` (e.g., `v20250916-g9fb25ac`). This version is embedded into all binaries and Docker images during the build process, ensuring accurate version reporting and update checks.


### Option 1: Pre-built Binary Updates

1.  **Check Daily:** Visit the [GitHub Releases page](https://github.com/PrathxmOp/dab-downloader/releases/latest) or watch the repository for notifications
2.  **Download:** Get the latest binary for your operating system and architecture
3.  **Replace:** Replace your existing `dab-downloader` executable with the newly downloaded one
4.  **Permissions (Linux/macOS):** If you encounter an "Exec format error" or "Permission denied":
    ```bash
    chmod +x ./dab-downloader-linux-arm64 # Or the appropriate binary for your system
    ```

### Option 2: Source Code Updates

If you built from source, update frequently:

1.  **Pull Latest Changes:**
    ```bash
    git pull origin main
    ```
2.  **Rebuild:**
    ```bash
    go mod tidy
    go build -o dab-downloader
    ```

### Option 3: Docker Updates

For Docker users, pull the latest image from Docker Hub:

1.  **Pull Latest Image:**
    ```bash
    docker compose pull
    ```
2.  **Restart Service:**
    ```bash
    docker compose up -d
    ```

### 🔔 **Get Update Notifications**

- **Watch this repository** on GitHub for release notifications
- **Join our Discord group** for immediate update announcements
- **Enable GitHub notifications** to know when new releases are available

## 📋 Usage Guide

### 🔍 Search and Discover

```bash
# General search
./dab-downloader search "Arctic Monkeys"

# Targeted search
./dab-downloader search "AM" --type=album
./dab-downloader search "Do I Wanna Know" --type=track
./dab-downloader search "Alex Turner" --type=artist
```

### 📀 Download Content

```bash
# Download a specific album
./dab-downloader album <album_id>

# Download artist's complete discography
./dab-downloader artist <artist_id>

# Download with filters (non-interactive)
./dab-downloader artist <artist_id> --filter=albums,eps --no-confirm
```

### 🎧 Spotify Integration

**Setup:** Get your [Spotify API credentials](https://developer.spotify.com/dashboard/applications)

```bash
# Download entire Spotify playlist
./dab-downloader spotify <playlist_url>

# Download entire Spotify album
./dab-downloader spotify <album_url>

# Expand playlist to download full albums
./dab-downloader spotify <playlist_url> --expand

# Auto-download (no manual selection)
./dab-downloader spotify <playlist_url> --auto

# Auto-download expanded albums from a playlist
./dab-downloader spotify <playlist_url> --expand --auto
```

### 🎵 Navidrome Integration

```bash
# Copy Spotify playlist to Navidrome
./dab-downloader navidrome <spotify_playlist_url>

# Add songs to existing playlist
./dab-downloader add-to-playlist <playlist_id> <song_id_1> <song_id_2>
```

## ⚙️ Configuration

### First-Time Setup

The application will guide you through initial configuration:

1. **DAB API URL** (e.g., `https://dabmusic.xyz`)
2. **Download Directory** (e.g., `/home/user/Music`)
3. **Concurrent Downloads** (recommended: `5`)

### Configuration File

The application will create `config/config.json` on first run.
You can also create or modify it manually.
An example configuration is available at `config/example-config.json`.

```json
{
  "APIURL": "https://your-dab-api-url.com",
  "DownloadLocation": "/path/to/your/music/folder",
  "Parallelism": 5,
  "SpotifyClientID": "YOUR_SPOTIFY_CLIENT_ID",
  "SpotifyClientSecret": "YOUR_SPOTIFY_CLIENT_SECRET",
  "NavidromeURL": "https://your-navidrome-url.com",
  "NavidromeUsername": "your_navidrome_username",
  "NavidromePassword": "your_navidrome_password",
  "Format": "flac",
  "Bitrate": "320",
  "saveAlbumArt": false,
  "naming": {
    "album_folder_mask": "{artist}/{artist} - {album} ({year})",
    "ep_folder_mask": "{artist}/EPs/{artist} - {album} ({year})",
    "single_folder_mask": "{artist}/Singles/{artist} - {album} ({year})",
    "file_mask": "{track_number} - {artist} - {title}"
  }
}
```

## ⚙️ Command-Line Flags

You can override configuration settings and control application behavior using command-line flags. Flags can be global (persistent) or specific to certain commands.

### Global Flags (Persistent Flags)

These flags can be used with any command.

-   `--api-url <URL>`: Specifies the DAB API endpoint to use.
    -   **Example:** `--api-url https://dab.example.com`
-   `--download-location <path>`: Sets the directory where all downloaded music will be saved.
    -   **Example:** `--download-location /home/user/Music`
-   `--debug`: Enables verbose logging for debugging purposes.
    -   **Example:** `--debug`
-   `--insecure`: Skips TLS certificate verification for API connections. Use with caution.
    -   **Example:** `--insecure`
-   `--spotify-client-id <ID>`: Your Spotify application Client ID for Spotify integration.
    -   **Example:** `--spotify-client-id your_spotify_client_id`
-   `--spotify-client-secret <SECRET>`: Your Spotify application Client Secret for Spotify integration.
    -   **Example:** `--spotify-client-secret your_spotify_client_secret`
-   `--navidrome-url <URL>`: The URL of your Navidrome server for integration.
    -   **Example:** `--navidrome-url https://navidrome.example.com`
-   `--navidrome-username <username>`: Your Navidrome username.
    -   **Example:** `--navidrome-username admin`
-   `--navidrome-password <password>`: Your Navidrome password.
    -   **Example:** `--navidrome-password your_navidrome_password`
-   `--warnings <mode>`: Controls how warnings are displayed during downloads.
    -   **Modes:** `summary` (default), `immediate`, `silent`
    -   **Example:** `--warnings immediate` for real-time warnings, `--warnings silent` for clean output

### Command-Specific Flags

These flags are only available for their respective commands.

#### `album` command

-   `--format <format>`: Specifies the output format for downloaded tracks. Requires FFmpeg.
    -   **Supported formats:** `flac` (default), `mp3`, `ogg`, `opus`
    -   **Example:** `dab-downloader album <album_id> --format mp3`
-   `--bitrate <kbps>`: Sets the bitrate for lossy formats (MP3, OGG, Opus).
    -   **Supported bitrates:** `192`, `256`, `320` (default)
    -   **Example:** `dab-downloader album <album_id> --format mp3 --bitrate 256`

#### `artist` command

-   `--filter <types>`: Filters the types of items to download from an artist's discography.
    -   **Supported types:** `albums`, `eps`, `singles` (comma-separated)
    -   **Example:** `dab-downloader artist <artist_id> --filter albums,singles`
-   `--no-confirm`: Skips the confirmation prompt before starting downloads.
    -   **Example:** `dab-downloader artist <artist_id> --no-confirm`
-   `--format <format>`: Same as `album` command's `--format`.
-   `--bitrate <kbps>`: Same as `album` command's `--bitrate`.

#### `search` command

-   `--type <type>`: Specifies the type of content to search for.
    -   **Supported types:** `artist`, `album`, `track`, `all` (default)
    -   **Example:** `dab-downloader search "Arctic Monkeys" --type artist`
-   `--auto`: Automatically downloads the first search result without prompting for selection.
    -   **Example:** `dab-downloader search "Do I Wanna Know" --type track --auto`
-   `--format <format>`: Same as `album` command's `--format`.
-   `--bitrate <kbps>`: Same as `album` command's `--bitrate`.

#### `spotify` command

-   `--auto`: Automatically downloads the first matching DAB result for each Spotify track without prompting.
    -   **Example:** `dab-downloader spotify <playlist_url> --auto`
-   `--expand`: When downloading a Spotify playlist, this flag will search for and download the full albums for each unique album found in the playlist, instead of individual tracks.
    -   **Example:** `dab-downloader spotify <playlist_url> --expand`
-   `--format <format>`: Same as `album` command's `--format`.
-   `--bitrate <kbps>`: Same as `album` command's `--bitrate`.

#### `navidrome` command

-   `--ignore-suffix <suffix>`: Specifies a suffix to ignore when searching for tracks on Navidrome. Useful for cleaning up track titles.
    -   **Example:** `dab-downloader navidrome <spotify_url> --ignore-suffix "(Remastered)"`
-   `--expand`: When copying a Spotify playlist to Navidrome, this flag will search for and download the full albums for each unique album found in the playlist to your download location, and then attempt to add those tracks to the Navidrome playlist.
    -   **Example:** `dab-downloader navidrome <spotify_url> --expand`
-   `--auto`: Automatically selects the first matching DAB result when searching for tracks to add to Navidrome, without prompting.
    -   **Example:** `dab-downloader navidrome <spotify_url> --auto`

#### `add-to-playlist` command

-   This command takes a playlist ID and one or more song IDs as arguments.
    -   **Example:** `dab-downloader add-to-playlist <playlist_id> <song_id_1> <song_id_2>`


## 📁 File Organization

Your music library will be organized like this:

```
Music/
├── Arctic Monkeys/
│   ├── artist.jpg
│   ├── AM (2013)/
│   │   ├── cover.jpg
│   │   ├── 01 - Do I Wanna Know.flac
│   │   └── 02 - R U Mine.flac
│   ├── Humbug (2009)/
│   │   └── ...
│   └── Singles/
│       └── I Bet You Look Good on the Dancefloor.flac
```

**Note:** You can customize this structure using the `naming` masks in your `config/config.json` file.

## 🔧 Advanced Features

### Debug Tools

```bash
# Test API connectivity
./dab-downloader debug api-availability

# Test artist endpoints
./dab-downloader debug artist-endpoints <artist_id>

# Comprehensive debugging
./dab-downloader debug comprehensive-artist-debug <artist_id>
```

### Quality & Metadata

- **Audio Format:** FLAC (highest quality available), or converted to MP3/OGG/Opus
- **Metadata Tags:** Title, Artist, Album, Genre, Year, ISRC, Producer, Composer
- **Cover Art:** Original resolution, auto-format detection
- **File Naming:** Consistent, organized structure

## 🐛 Troubleshooting

<details>
<summary><strong>Common Issues & Solutions</strong></summary>

**"Something that worked yesterday is broken today"**
- ✅ **First step:** Check for and install the latest update
- ✅ Check the Discord group for known issues
- ✅ Report the issue with your version number

**"Failed to get album/artist/track"**
- ✅ Update to the latest version first
- ✅ Verify the ID is correct
- ✅ Check internet connection
- ✅ Confirm DAB API accessibility

**"Failed to create directory"**
- ✅ Check available disk space
- ✅ Verify write permissions
- ✅ Ensure valid file path

**"Download failed" or timeouts**
- ✅ App auto-retries failed downloads
- ✅ Check connection stability
- ✅ Some tracks may be unavailable
- ✅ Update to latest version if issues persist

**Progress bars not showing**
- ✅ Run with `--debug` flag
- ✅ Check terminal compatibility
- ✅ Report output when filing issues

**"It worked fine last week but now nothing works"**
- ✅ This is expected during development - update immediately
- ✅ Join Discord group for real-time fixes
- ✅ Help me by reporting what broke

</details>

## 💬 Support & Community

Due to the unstable nature of this project and it being a solo-developed tool, community support is essential:

📱 **[Discord Support Group](https://discord.gg/q9RnuVza2)** - **HIGHLY RECOMMENDED**
- Get real-time help and updates
- Learn about breaking changes immediately  
- Connect with other users experiencing similar issues
- Get notified when critical fixes are released
- Help the solo developer by reporting issues and testing fixes

🐛 **[GitHub Issues](https://github.com/PrathxmOp/dab-downloader/issues)** - Report bugs and request features
- Please include your version number and operating system
- Describe what worked before vs. what's broken now
- Check recent issues - your problem might already be reported
- Be patient - I'm one person handling all development and support

## 🏗️ Project Architecture

```
dab-downloader/
├── main.go              # CLI entry point
├── search.go            # Search functionality
├── api.go               # DAB API client
├── downloader.go        # Download engine
├── artist_downloader.go # Artist catalog handling
├── metadata.go          # FLAC metadata processing
├── spotify.go           # Spotify integration
├── navidrome.go         # Navidrome integration
├── utils.go             # Utility functions
└── docker-compose.yml   # Container setup
```

🤝 Contributing

I especially welcome contributions during this unstable development phase:

1.  **🐛 Report bugs** - Even small issues help me stabilize faster
2.  **💡 Test features** - Help me catch breaking changes early  
3.  **🔧 Submit PRs** - Fixes for stability issues are prioritized
4.  **📖 Improve docs** - Help other users navigate the instability

### Development Areas Needing Help

- **Stability Testing** - Help me identify what breaks between versions
- **API Client** (`api.go`) - Enhance error handling and resilience
- **Metadata** (`metadata.go`) - Fix edge cases and improve reliability
- **Downloads** (`downloader.go`) - Improve robustness and error recovery
- **Cross-platform Testing** - Help me ensure updates work across different systems

## 💖 Contributors

A huge thank you to all the amazing people who have contributed to this project and helped make it better! Your contributions are greatly appreciated.

- **[NimbleAINinja](https://github.com/NimbleAINinja)**: For their outstanding work on the warning collector, MusicBrainz optimizations, and retroactive metadata updates.

If you've contributed to the project and your name is missing, please feel free to add it!



## ⚖️ Legal Notice

This software is provided for **educational purposes only**. Users are responsible for:

- ✅ Complying with all applicable laws
- ✅ Respecting the terms of service of any third-party services used with this tool
- ✅ Only downloading content they legally own or have permission to access

The developers and contributors of this project do not endorse piracy or any form of copyright infringement.

## 📄 License

This project is open-source and available under the [MIT License](LICENSE). See the [LICENSE](LICENSE) file for more details.

## 🌟 Support the Project

If you find this project useful and want to support its development, here are a few ways you can help:

- ⭐ Star this repository
- 🐛 Report issues and bugs
- 💡 Suggest new features
- 🤝 Contribute to the codebase
- 💬 Join our [Discord community](https://discord.gg/q9RnuVza2) and help other users

Your support and feedback are invaluable to the project's growth and improvement! 🙏

---

<div align="center">
  <strong>Made with ❤️ for music lovers</strong><br>
  <sub>Download responsibly • Respect artists • Support music</sub><br><br>
  <strong>⚠️ Remember: Update frequently during development! ⚠️</strong>
</div>

---

## Changelog

### Features
- feat: Add --insecure flag to skip TLS verification
- feat(config): Add configurable naming masks
- feat: Enhance MusicBrainz metadata fetching
- feat(downloader): add log message for skipping existing tracks
- feat(navidrome): add --auto flag to navidrome command
- feat: add --expand flag to navidrome command
- feat: Update README with auto-dl.sh as primary quick start option and bump version
- feat: add manual update guide and improve update prompt
- feat: Implement versioning and update mechanism improvements. New Versioning Scheme: Uses version/version.json for manual updates. Semantic Versioning: Uses github.com/hashicorp/go-version for robust comparisons. Configurable Update Repository: Added UpdateRepo option to config.json. Docker-Aware Update Behavior: Prevents browser attempts in headless environments, allows disabling update checks. Improved Version Display: Reads from version/version.json at runtime. Workflow Updates: Removed build-time ldflags, modified Docker build to copy version.json, updated Docker image tagging, removed redundant TAG_NAME generation. Bug Fixes: Corrected fmt.Errorf usage, fixed \n literal display in UI. This significantly enhances the flexibility and reliability of the application's update process.
- feat: Implement robust versioning and Docker integration and fix #15
- feat: Implement playlist expansion to download full albums
- feat: Implement rate limiting, MusicBrainz, enhanced progress, and artist search fix
- feat: Enhance update notification with prompt, browser opening, and README guide
- feat: Implement explicit version command and colored update status
- feat: Add ARM64 build to release workflow, enabling execution on Termux and other ARM-based Linux systems.
- feat: Add diagnostic step to GitHub Release workflow
- feat: Add option to save album art
- feat: Add --ignore-suffix flag to ignore any suffix
- feat: Add format and bitrate options and fix various bugs
- feat: Implement format conversion
- feat: Overhaul README and add Docker support
- feat: Improve user experience and update project metadata
- feat: Automate release creation on push to main
- feat: Add GitHub Actions for releases and update README
- feat: Enhance CLI help, configuration, and Navidrome integration
- feat: Re-implement multi-select for downloads

### Fixes
- fix: Correct build errors
- fix: add blank import for embed package in main.go and bump version to 0.0.29-dev
- fix: remove duplicated code in main.go and bump version to 0.0.28-dev
- fix: embed version.json into binary
- fix(downloader): resolve progress bar race condition
- fix(metadata): correct musicbrainz id tagging
- fix(build): embed version at build time and fix progress bar errors
- fix: update link is now fixed
- fix: Deduplicate artist search results
- fix: Correctly display newlines in terminal output and update .gitignore
- fix: Correct GitHub repository name in updater.go
- Fix: Artist search not returning results
- fix: Preserve metadata when converting to other formats
- fix: use cross-platform home directory for default download location
- fix: handle pagination in spotify playlists and create config dir if not exists
- fix: Replace deprecated release actions with softprops/action-gh-release
- fix: Rename macOS executable to dab-downloader-macos-amd64
- fix: Ensure TAG_NAME is correctly formed in workflow
- fix: Create and push Git tag before creating GitHub Release
- fix: Update setup-go action and Go version in workflow
- fix: Add go mod tidy and download to workflow
- fix: Handle numeric artist IDs from API

### Chore
- chore: Bump version to 0.9.0-dev
- chore(version): bump version to 0.8.0-dev
- chore(version): bump version to 0.0.32-dev
- chore: bump version to 0.0.27-dev
- chore: bump version to 0.0.26-dev
- chore: Update GitHub Actions workflow for version.json tagging. Version Source: Reads from version/version.json. Release Tagging: Uses version.json for GitHub Releases and Git tags. Docker Image Tagging: Docker images are also tagged with the version from version.json.

### Docs
- docs: update README with changelog
- docs: Update README.md for versioning and Docker integration
- docs: Update README with --expand flag usage
- docs: Update README with development status, update guide, and support information
- docs: Add instructions for granting execute permissions to binaries in README
- Docs: Update README release links
- docs: Enhance README with go mod tidy and Spotify downloader details

### Refactor
- refactor(navidrome): improve album and track searching to prevent re-downloads

### Other
- Added Discord Group Link
- Shooot my config pushed
- Enhancement in README
- Added option to copy playlist from spotify
- initial change

---

## Update Guide

The tool has a built-in update checker. If a new version is available, it will prompt you to update and attempt to open the update guide in your browser.

If the tool fails to open the browser, you can manually update by following these steps:

1.  **Go to the [GitHub Releases page](https://github.com/PrathxmOp/dab-downloader/releases/latest).**
2.  **Download the latest release** for your operating system and architecture.
3.  **Extract the archive.**
4.  **Replace your existing `dab-downloader` executable** with the newly downloaded one.
5.  **(Linux/macOS only) Grant execute permissions** to the new binary:

    ```bash
    chmod +x ./dab-downloader-linux-amd64
    ```

    (Replace `./dab-downloader-linux-amd64` with the actual name of the binary you downloaded).
