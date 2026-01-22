# 🎵 DAB Music Downloader

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](#license)
[![Release](https://img.shields.io/github/v/release/PrathxmOp/dab-downloader)](https://github.com/PrathxmOp/dab-downloader/releases/latest)
[![Discord Support](https://img.shields.io/badge/Support-Discord-blue.svg?logo=discord&logoColor=white)](https://discord.gg/gjf3xjMtRE)
![Development Status](https://img.shields.io/badge/status-active%20development-orange.svg)

> **A high-performance, modular music downloader designed for audiophiles.**  
> Effortlessly fetch high-quality FLAC audio with comprehensive metadata, embedded artwork, and smart organization from the DAB API.

---

## ✨ Features at a Glance

### 🚀 Core Experience
*   **High-Fidelity Audio**: Downloads pristine FLAC files directly from source.
*   **Smart Metadata**: Automatically tags files with Title, Artist, Album, Genre, Year, ISRC, and more.
*   **Format Conversion**: Built-in support for converting to MP3, OGG, or Opus on the fly (requires FFmpeg).
*   **Interactive TUI**: Navigate search results and select multiple items with a modern, terminal-based user interface.

### 🔗 Integrations
*   **Spotify**: Import and download entire playlists or albums directly.
*   **Navidrome**: Seamlessly sync downloaded content to your Navidrome media server.
*   **MusicBrainz**: Enhances metadata accuracy with advanced acoustic fingerprinting and database lookups.

### 🛠️ Power User Tools
*   **Authentication System**: Secure login command to manage your API session.
*   **Smart Resume**: Detects existing files to prevent duplicate downloads and wasted bandwidth.
*   **Concurrent Downloading**: Accelerate large batches with parallel processing.
*   **Flexible Organization**: Customize file naming and directory structures to match your library.

---

## 📸 Screenshots

![Interactive Search TUI](./screenshots/ScreenShot1.png)
*Interactive search with multi-selection support.*

![Download Progress](./screenshots/ScreenShot2.png)
*Parallel downloads with detailed progress tracking.*

---

## 🚀 Installation

### Option 1: Pre-built Binaries
1.  Visit the [**Releases Page**](https://github.com/PrathxmOp/dab-downloader/releases/latest).
2.  Download the archive matching your OS (Windows, macOS, Linux).
3.  Extract the binary and (on Linux/macOS) make it executable:
    ```bash
    chmod +x dab-downloader
    ```

### Option 2: Docker
Run entirely within a container to keep your system clean.

```bash
# Pull the latest image
docker pull prathxm/dab-downloader:latest

# Run a search command
docker run -it -v $(pwd)/music:/music -v $(pwd)/config:/config prathxm/dab-downloader search "The Weeknd"
```

### Option 3: Build from Source
If you prefer to build it yourself, you'll need Go 1.21+ installed.

```bash
git clone https://github.com/PrathxmOp/dab-downloader.git
cd dab-downloader
go build -o dab-downloader ./cmd/dab-downloader
```

---

## 📖 Usage Guide

### 🔐 Authentication (New!)
Before starting, log in to your DAB account to access all features.

```bash
./dab-downloader login "your-email@example.com" "your-password"
```
*Session tokens are securely stored locally.*

### 🔍 Interactive Search
Search for artists, albums, or tracks. Use **Arrow Keys** to navigate, **Space** to select multiple items, and **Enter** to download.

```bash
./dab-downloader search "Daft Punk"
```

### 🎧 Spotify & Navidrome
Import playlists directly from Spotify or sync to Navidrome.

```bash
# Download a Spotify Playlist
./dab-downloader spotify "https://open.spotify.com/playlist/..."

# Sync Spotify Playlist to Navidrome
./dab-downloader navidrome "https://open.spotify.com/playlist/..."
```

### ⚙️ Common Commands

| Command | Description |
| :--- | :--- |
| `artist [id]` | Download an artist's entire discography. |
| `album [id]` | Download a specific album by ID. |
| `status` | Check your current login status. |
| `logout` | Clear your session and log out. |
| `version` | Display version information. |

---

## ⚙️ Configuration

On first run, `dab-downloader` will generate a `config/config.json` file. You can customize download paths, concurrency limits, and output formats.

**Example `config.json`:**
```json
{
  "APIURL": "https://dabmusic.xyz",
  "DownloadLocation": "/home/user/Music",
  "Parallelism": 5,
  "Format": "flac", 
  "Bitrate": "320",
  "WarningBehavior": "summary"
}
```

### CLI Flags
Override config settings on the fly:
*   `--format mp3`: Convert output to MP3.
*   `--bitrate 320`: Set bitrate for lossy formats.
*   `--output "./Downloads"`: Temporarily change download folder.

---

## 🤝 Contributing

We welcome contributions! Whether it's reporting bugs, suggesting features, or submitting Pull Requests.

1.  **Report Issues**: Found a bug? Open an issue on GitHub.
2.  **Join the Community**: Discuss features and get help on our [**Discord Server**](https://discord.gg/gjf3xjMtRE).

---

## ⚖️ Legal Disclaimer

**For Educational Purposes Only.**
This tool is designed to demonstrate API interaction and metadata processing. Users are strictly responsible for:
*   Complying with all local copyright laws.
*   Respecting terms of service of third-party platforms.
*   Downloading only content they have the legal right to access.

---

<div align="center">
  <sub>Made with ❤️ by PrathxmOp and contributors</sub>
</div>