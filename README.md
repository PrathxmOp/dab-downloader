# 🎵 DAB Music Downloader

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.19-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-Educational-green.svg)](#license)
[![Release](https://img.shields.io/github/v/release/PrathxmOp/dab-downloader)](https://github.com/PrathxmOp/dab-downloader/releases/latest)
[![Signal Support](https://img.shields.io/badge/Support-Signal%20Group-blue.svg)](https://signal.group/#CjQKIARVUX48EP6g9DSPb2n1v6fAkxGQvdJJSWc4KLa4KFVyEhDCRiJon09heXcckPnkX6k2)
![Development Status](https://img.shields.io/badge/status-unstable%20development-orange.svg)

> A powerful, modular music downloader that delivers high-quality FLAC files with comprehensive metadata support through the DAB API.

## ⚠️ **IMPORTANT: Development Status**

🚧 **This project is currently in active, unstable development.** 🚧

- **Frequent Breaking Changes**: Features may work one day and break the next
- **Regular Updates Required**: You'll need to update frequently to get the latest fixes
- **Expect Issues**: Something always seems to break when i fix something else
- **Pre-Stable Release**: We're working toward a stable v1.0, but we're not there yet

**📢 We strongly recommend:**
- ✅ Joining our [Signal Support Group](https://signal.group/#CjQKIARVUX48EP6g9DSPb2n1v6fAkxGQvdJJSWc4KLa4KFVyEhDCRiJon09heXcckPnkX6k2) for real-time updates
- ✅ Checking for updates daily if you're actively using the tool
- ✅ Being prepared to troubleshoot and report issues
- ✅ Having patience as we work through the bugs

💬 **Need Help?** Join our [Signal Support Group](https://signal.group/#CjQKIARVUX48EP6g9DSPb2n1v6fAkxGQvdJJSWc4KLa4KFVyEhDCRiJon09heXcckPnkX6k2) for instant community support and the latest stability updates!

**👤 Solo Developer Project:** This tool is developed and maintained by a single developer. While I work hard to push frequent updates and fixes (often multiple commits per day), expect some instability as I can't test every scenario across all systems.

## ✨ Key Features

🔍 **Smart Search** - Find artists, albums, and tracks with intelligent filtering  
📦 **Complete Discographies** - Download entire artist catalogs with automatic categorization  
🏷️ **Rich Metadata** - Full tag support including genre, composer, producer, ISRC, and copyright  
🎨 **High-Quality Artwork** - Embedded album covers in original resolution  
⚡ **Concurrent Downloads** - Fast parallel processing with real-time progress tracking  
🔄 **Intelligent Retry Logic** - Robust error handling for reliable downloads  
🎧 **Spotify Integration** - Import and download entire Spotify playlists and albums  
🎵 **Format Conversion** - Convert downloaded FLAC files to MP3, OGG, Opus with configurable bitrates (requires FFmpeg)  
📊 **Navidrome Support** - Seamless integration with your music server  

## 📸 Screenshots

![img1](./screenshots/ScreenShot1.png)
![img1](./screenshots/ScreenShot2.png)

## 🚀 Quick Start

### Option 1: Pre-built Binary (Recommended)

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

### Option 2: Build from Source

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

### Option 3: Docker (Containerized)

```bash
# Build and run with Docker Compose
docker compose build
mkdir config music
cp config/example-config.json config/config.json

# Run any command
docker compose run dab-downloader search "your favorite artist"
```

## 🔄 **CRITICAL: Staying Updated**

Due to the unstable nature of this project, **regular updates are essential**:

### 🚨 **Daily Update Routine (Recommended)**

Since we're constantly fixing bugs and pushing updates, we recommend checking for updates daily:

```bash
# Check for new releases
curl -s https://api.github.com/repos/PrathxmOp/dab-downloader/releases/latest | grep "tag_name"
```

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

For Docker users:

1.  **Pull Latest Image:**
    ```bash
    docker compose pull
    ```
2.  **Rebuild/Restart:**
    ```bash
    docker compose build
    docker compose up -d
    ```

### 🔔 **Get Update Notifications**

- **Watch this repository** on GitHub for release notifications
- **Join our Signal group** for immediate update announcements
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

# Auto-download (no manual selection)
./dab-downloader spotify <playlist_url> --auto
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

1. **DAB API URL** (e.g., `https://dab.yeet.su`)
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
  "Bitrate": "320"
}
```

### Command-Line Options

Override configuration with flags:

```bash
--api-url               # Set DAB API endpoint
--download-location     # Set download directory
--debug                 # Enable verbose logging
--auto                  # Auto-download first results
--no-confirm            # Skip confirmation prompts
--format                # Specify output format (mp3, ogg, opus)
--bitrate               # Specify bitrate for lossy formats (e.g., 192, 256, 320)
--filter                # Filter by item type for artist downloads (albums, eps, singles)
--type                  # Type of content to search for (artist, album, track, all)
--spotify-client-id     # Your Spotify Client ID
--spotify-client-secret # Your Spotify Client Secret
--navidrome-url         # Your Navidrome URL
--navidrome-username    # Your Navidrome Username
--navidrome-password    # Your Navidrome Password
```

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
- ✅ Check the Signal group for known issues
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
- ✅ Join Signal group for real-time fixes
- ✅ Help me by reporting what broke

</details>

## 💬 Support & Community

Due to the unstable nature of this project and it being a solo-developed tool, community support is essential:

📱 **[Signal Support Group](https://signal.group/#CjQKIARVUX48EP6g9DSPb2n1v6fAkxGQvdJJSWc4KLa4KFVyEhDCRiJon09heXcckPnkX6k2)** - **HIGHLY RECOMMENDED**
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

## 🤝 Contributing

I especially welcome contributions during this unstable development phase:

1. **🐛 Report bugs** - Even small issues help me stabilize faster
2. **💡 Test features** - Help me catch breaking changes early  
3. **🔧 Submit PRs** - Fixes for stability issues are prioritized
4. **📖 Improve docs** - Help other users navigate the instability

### Development Areas Needing Help

- **Stability Testing** - Help me identify what breaks between versions
- **API Client** (`api.go`) - Enhance error handling and resilience
- **Metadata** (`metadata.go`) - Fix edge cases and improve reliability
- **Downloads** (`downloader.go`) - Improve robustness and error recovery
- **Cross-platform Testing** - Help me ensure updates work across different systems

## ⚖️ Legal Notice

This software is provided for **educational purposes only**. Users are responsible for:

- ✅ Complying with all applicable laws
- ✅ Respecting terms of service
- ✅ Only downloading content you legally own or have permission to access

**Note:** The unstable nature of this software means it should not be relied upon for any critical or commercial purposes.

## 📄 License

This project is provided under an educational license. See the [LICENSE](LICENSE) file for details.

## 🌟 Support the Project

If you're willing to help us through the unstable development phase:

- ⭐ Star this repository
- 🐛 Report issues and bugs (even small ones!)
- 💡 Test new features and report what breaks
- 🤝 Contribute stability fixes
- 💬 Join our [Signal community](https://signal.group/#CjQKIARVUX48EP6g9DSPb2n1v6fAkxGQvdJJSWc4KLa4KFVyEhDCRiJon09heXcckPnkX6k2) and help other users
- 🔄 Help spread awareness about the need for frequent updates

**Your patience and feedback during this development phase is invaluable to a solo developer! 🙏**

---

<div align="center">
  <strong>Made with ❤️ for music lovers</strong><br>
  <sub>Download responsibly • Respect artists • Support music</sub><br><br>
  <strong>⚠️ Remember: Update frequently during development! ⚠️</strong>
</div>
