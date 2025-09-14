# Enhanced DAB Music Downloader v3.0

A modular, high-quality FLAC music downloader with comprehensive metadata support for the DAB API.

## Features

- **Search**: Search for artists, albums, and tracks directly from the CLI.
- **Complete Discography Downloads**: Download entire artist discographies with smart categorization.
- **Comprehensive Metadata**: Full metadata support including genre, composer, producer, ISRC, copyright, and more.
- **Smart Album Detection**: Automatically categorizes albums, EPs, and singles.
- **Duplicate Detection**: Checks for existing downloads and skips duplicates.
- **Cover Art Support**: Downloads and embeds high-quality cover art.
- **Concurrent Downloads**: Fast parallel downloading with a detailed progress dashboard.
- **Retry Logic**: Robust error handling with automatic retries.
- **Modular Architecture**: Clean, maintainable code structure.

## Project Structure

```
dab-downloader/
├── main.go              # Entry point and command-line interface
├── search.go            # Search command logic
├── types.go             # Data structures and types
├── api.go               # API client methods
├── downloader.go        # Core download logic
├── artist_downloader.go # Artist discography handling
├── metadata.go          # FLAC metadata processing
├── utils.go             # Utility functions
├── colours.go           # Color utility functions
├── debug.go             # Debugging utilities
├── navidrome.go         # Navidrome API client methods
├── navidrome_types.go   # Navidrome data structures and types
├── retry.go             # Retry logic utility
├── spotify.go           # Spotify API client methods
├── spotify_types.go     # Spotify data structures and types
├── go.mod              # Go module dependencies
└── README.md           # This file
```

## Installation

If you don't want to build from source, you can download the latest pre-built executables from the [GitHub Releases page](https://github.com/PrathxmOp/dab-downloader/releases).

### Build from Source

1. **Install Go** (version 1.19 or later)
   ```bash
   # Download from https://golang.org/dl/
   ```

2. **Clone/Create the project**
   ```bash
   git clone https://github.com/your-username/dab-downloader.git
   cd dab-downloader
   ```

3. **Initialize Go Modules and Build**
   First, initialize and tidy up Go modules:
   ```bash
   go mod tidy
   ```
   Then, build the application:
   ```bash
   go build -o dab-downloader
   ```

## Usage

### Basic Commands

- **Search for music:**
  ```bash
  ./dab-downloader search "query"
  ```
  You can also specify the type of content to search for:
  ```bash
  ./dab-downloader search "query" --type=artist
  ./dab-downloader search "query" --type=album
  ./dab-downloader search "query" --type=track
  ```

- **Download an album:**
  ```bash
  ./dab-downloader album <album_id>
  ```



- **Download an artist's discography:**
  ```bash
  ./dab-downloader artist <artist_id>
  ```

### Advanced Features

#### Spotify Playlist Downloader

The `dab-downloader` can download entire Spotify playlists. This feature requires Spotify API credentials (Client ID and Client Secret) which you can obtain from the [Spotify Developer Dashboard](https://developer.spotify.com/dashboard/applications).

**Configuration:**
Ensure your `config.json` (or command-line flags) includes your Spotify Client ID and Client Secret:
```json
{
  "SpotifyClientID": "YOUR_SPOTIFY_CLIENT_ID",
  "SpotifyClientSecret": "YOUR_SPOTIFY_CLIENT_SECRET"
}
```

**Usage:**
To download a Spotify playlist, provide its URL:
```bash
./dab-downloader spotify <spotify_playlist_url>
```
The application will fetch all tracks from the playlist, search for them on the DAB API, and download them.

**Automatic Download:**
You can use the `--auto` flag to automatically download the first matching result for each track without interactive selection:
```bash
./dab-downloader spotify <spotify_playlist_url> --auto
```

#### Artist Discography Download

When downloading an artist's discography, you can choose between interactive and non-interactive modes.

- **Interactive Mode (default):**
  ```bash
  ./dab-downloader artist <artist_id>
  ```
  The application will prompt you to select what to download (all, albums, EPs, singles, or a custom selection).

- **Non-Interactive Mode:**
  Use the `--filter` flag to specify what to download without any prompts.
  ```bash
  ./dab-downloader artist <artist_id> --filter=albums,eps
  ```
  Available filter options: `albums`, `eps`, `singles`.

  You can also skip the confirmation prompt using the `--no-confirm` flag:
  ```bash
  ./dab-downloader artist <artist_id> --filter=all --no-confirm
  ```

#### Navidrome Integration

The `dab-downloader` can also integrate with Navidrome to manage your music library.

- **Copy a Spotify playlist to Navidrome:**
  ```bash
  ./dab-downloader navidrome <spotify_playlist_url>
  ```
  This command will search for tracks from the Spotify playlist in your Navidrome library. If a track is not found in Navidrome, it will attempt to download it via DAB and then add it to Navidrome.

- **Add songs to an existing Navidrome playlist:**
  ```bash
  ./dab-downloader add-to-playlist <playlist_id> <song_id_1> [song_id_2...]
  ```
  This allows you to add one or more songs (by their Navidrome song IDs) to a specified Navidrome playlist.

#### Download Dashboard

When downloading an album or an artist's discography, you will see a detailed download dashboard with individual progress bars for each track, including download speed and ETA.

#### Metadata Features

Each downloaded track includes:
- **Basic Tags**: Title, Artist, Album, Track Number
- **Advanced Tags**: Genre, Composer, Producer, Year, ISRC
- **Album Information**: Album Artist, Total Tracks, Disc Numbers
- **Technical Tags**: Encoder info, Source, Duration
- **Cover Art**: Embedded high-quality album artwork

## Configuration

The `dab-downloader` uses a `config.json` file located in the same directory as the executable for its settings.

### First Run Setup

On the first run, if `config.json` does not exist, the application will interactively prompt you for the following essential settings:

- **DAB API URL**: The URL of the DAB API endpoint (e.g., `https://dab.yeet.su`).
- **Download Location**: The default directory where downloaded music will be saved (e.g., `/home/user/Music`).
- **Parallel Downloads**: The number of tracks to download concurrently (default: `5`).

Your responses will be saved to `config.json` for future runs. This file is automatically added to `.gitignore` to prevent it from being committed to version control.

### Example Configuration

An `example-config.json` file is provided in the project root, which you can copy and modify to create your `config.json`. **Remember to rename `example-config.json` to `config.json` for the application to use it.**

```json
{
  "APIURL": "https://dab.yeet.su",
  "DownloadLocation": "/home/user/Music",
  "Parallelism": 5
}
```

### Command-line Flags

You can override the configuration file settings using command-line flags:
- `--api-url`: Set the DAB API URL.
- `--download-location`: Set the directory to save downloads.
- `--debug`: Enable verbose debug logging for internal operations.
- `debug`: A new command with subcommands for various debugging utilities:
  - `api-availability`: Test basic DAB API connectivity.
  - `artist-endpoints <artist_id>`: Test different artist endpoint formats for a given artist ID.
  - `comprehensive-artist-debug <artist_id>`: Perform comprehensive debugging for an artist ID (API connectivity, endpoint formats, and ID type checks).

### Directory Structure

Downloads are organized as:
```
Music/
├── Artist Name/
│   ├── artist.jpg          # Artist photo
│   ├── Album 1/
│   │   ├── cover.jpg       # Album cover
│   │   ├── 01 - Track 1.flac
│   │   └── 02 - Track 2.flac
│   ├── EP Name/
│   │   └── ...
│   └── Singles/
│       └── Single Track.flac
```

### Quality Settings

- **Audio Quality**: Highest available FLAC (quality level 27)
- **Cover Art**: Original resolution, auto-format detection
- **Metadata**: Comprehensive Vorbis comments with all available fields

## Troubleshooting

### Common Issues

1. **"Failed to get album/artist/track"**
   - Verify the ID is correct.
   - Check your internet connection.
   - Ensure the DAB API is accessible.

2. **"Failed to create directory"**
   - Check disk space.
   - Verify write permissions.
   - Ensure the path is valid.

3. **"Download failed" or timeout errors**
   - The application will automatically retry failed downloads.
   - Check your internet connection stability.
   - Some tracks may be unavailable.

4. **Progress bars not displaying correctly**
   - If you are not seeing the progress bars, try running the application with the `--debug` flag and provide the output when reporting an issue.

### Performance Tips

- **Concurrent Downloads**: The app downloads multiple tracks simultaneously by default.
- **Large Discographies**: The app handles large collections efficiently with a detailed progress dashboard.
- **Network Issues**: Built-in retry logic handles temporary network problems.

## API Information

This downloader works with the DAB (Deemix Alternative Backend) API:
- **Default Endpoint**: https://dab.yeet.su
- **Required IDs**: Album IDs, Track IDs, Artist IDs from the DAB service.
- **Quality**: Downloads highest quality FLAC files available.

## Legal Notice

This tool is for educational purposes only. Users are responsible for complying with all applicable laws and terms of service. Only download content you have the legal right to access.

## Contributing

The modular structure makes it easy to contribute:
- **api.go**: Add new API endpoints or improve error handling.
- **metadata.go**: Enhance metadata processing or add new fields.
- **downloader.go**: Improve download logic or add features.
- **utils.go**: Add utility functions.

## License

This project is provided as-is for educational purposes.