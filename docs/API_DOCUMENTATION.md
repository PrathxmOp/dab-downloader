# DAB Music Downloader - API Documentation

## Overview

This document describes the internal API structure and external integrations used by the DAB Music Downloader. The application interfaces with multiple external APIs and provides internal abstractions for music discovery and downloading.

## DAB API Integration

### Base API Client (`api.go`)

The core API client provides a unified interface to the DAB music service.

#### Client Configuration

```go
type DabAPI struct {
    endpoint       string
    outputLocation string
    client         *http.Client
    rateLimiter    *time.Ticker
}
```

#### Rate Limiting

The client implements rate limiting to respect API terms:
- 500ms interval between requests
- Mutex-protected rate limiter
- Automatic retry on rate limit errors (429)

#### Core Methods

##### `Request(ctx context.Context, path string, isPathOnly bool, params []QueryParam) (*http.Response, error)`

Generic HTTP request method with:
- Context-based cancellation
- Query parameter support
- Rate limiting enforcement
- Retry logic with exponential backoff

##### `GetAlbum(ctx context.Context, albumID string) (*Album, error)`

Retrieves complete album information including:
- Album metadata (title, artist, release date)
- Track listing with metadata
- Cover art URLs
- Genre and label information

**API Endpoint**: `api/album?albumId={id}`

##### `GetArtist(ctx context.Context, artistID string, config *Config, debug bool) (*Artist, error)`

Fetches artist discography with:
- Artist information and biography
- Complete album listing
- Automatic album type detection
- Parallel album detail fetching

**API Endpoint**: `api/discography?artistId={id}`

##### `GetTrack(ctx context.Context, trackID string) (*Track, error)`

Retrieves individual track information:
- Track metadata
- Album association
- Duration and technical details

**API Endpoint**: `api/track?trackId={id}`

##### `GetStreamURL(ctx context.Context, trackID string) (string, error)`

Generates streaming URLs for download:
- Highest quality FLAC (quality=27)
- Temporary URL generation
- Retry logic for reliability

**API Endpoint**: `api/stream?trackId={id}&quality=27`

##### `Search(ctx context.Context, query string, searchType string, limit int, debug bool) (*SearchResults, error)`

Multi-type search functionality:
- Concurrent search across content types
- Configurable result limits
- Type-specific result parsing

**API Endpoint**: `api/search?q={query}&type={type}&limit={limit}`

**Supported Search Types**:
- `artist`: Search for artists
- `album`: Search for albums  
- `track`: Search for tracks
- `all`: Search all types concurrently

## Spotify API Integration

### Authentication (`spotify.go`)

Uses OAuth2 Client Credentials flow for API access:

```go
config := &clientcredentials.Config{
    ClientID:     s.ID,
    ClientSecret: s.Secret,
    TokenURL:     spotifyauth.TokenURL,
}
```

### Playlist Operations

#### `GetPlaylistTracks(playlistURL string) ([]SpotifyTrack, string, error)`

Extracts tracks from Spotify playlists:
- Handles pagination automatically
- Extracts track and album metadata
- Returns playlist name for organization

**Features**:
- Automatic pagination handling
- Album information extraction
- Artist credit processing

#### `GetAlbumTracks(albumURL string) ([]SpotifyTrack, string, error)`

Retrieves tracks from Spotify albums:
- Complete track listing
- Album metadata extraction
- Artist information

### Data Structures

```go
type SpotifyTrack struct {
    Name        string
    Artist      string
    AlbumName   string
    AlbumArtist string
}
```

## Navidrome/Subsonic API Integration

### Authentication (`navidrome.go`)

Implements Subsonic API authentication:
- Salt-based password hashing
- Token generation for requests
- Ping-based connectivity testing

```go
func getSaltedPassword(password string, salt string) string {
    hasher := md5.New()
    hasher.Write([]byte(password + salt))
    return hex.EncodeToString(hasher.Sum(nil))
}
```

### Core Operations

#### `SearchTrack(trackName, artistName, albumName string) (*subsonic.Child, error)`

Multi-strategy track searching:
1. Album-based search with track matching
2. Combined artist+track search
3. Fallback to track name only
4. Fuzzy matching with exact match preference

#### `SearchAlbum(albumName string, artistName string) (*subsonic.Child, error)`

Album discovery with exact matching:
- Case-insensitive comparison
- Artist name validation
- Multiple result handling

#### `CreatePlaylist(name string) error`

Playlist creation via Subsonic API:
- URL-encoded playlist names
- Error handling and validation
- Response status checking

#### `AddTracksToPlaylist(playlistID string, trackIDs []string) error`

Batch track addition to playlists:
- Multiple track support in single request
- Subsonic API error handling
- Response validation

### API Endpoints

- **Ping**: `/rest/ping.view`
- **Search**: `/rest/search2.view`
- **Get Album**: `/rest/getAlbum.view`
- **Create Playlist**: `/rest/createPlaylist.view`
- **Update Playlist**: `/rest/updatePlaylist.view`
- **Get Playlist**: `/rest/getPlaylist.view`

## MusicBrainz API Integration

### Client Implementation (`musicbrainz.go`)

Provides enhanced metadata through MusicBrainz database:

```go
const (
    musicBrainzAPI       = "https://musicbrainz.org/ws/2/"
    musicBrainzUserAgent = "dab-downloader/2.0 ( prathxm.in@gmail.com )"
)
```

### Search Operations

#### `SearchTrackByISRC(isrc string) (*MusicBrainzTrack, error)`

Searches for track metadata using ISRC (International Standard Recording Code):
- More accurate than text-based searches
- Returns complete track information including associated releases
- Used for enhanced album-level metadata consistency

**Query Format**: `isrc:"ISRC_CODE"`

#### `SearchTrack(artist, album, title string) (*MusicBrainzTrack, error)`

Searches for track metadata:
- Structured query with artist, album, and title
- Returns MusicBrainz IDs for enhanced tagging
- Artist credit information

**Query Format**: `artist:"Artist" AND release:"Album" AND recording:"Title"`

#### `SearchRelease(artist, album string) (*MusicBrainzRelease, error)`

Searches for release (album) metadata:
- Album-level MusicBrainz IDs
- Release group information
- Label and catalog number data

**Query Format**: `artist:"Artist" AND release:"Album"`

### Enhanced Release Lookup Strategy

The system now implements an intelligent release lookup strategy:

1. **ISRC Priority**: When processing albums, the system first attempts to find tracks with ISRC codes
2. **Release ID Extraction**: Uses ISRC-based track lookup to extract the MusicBrainz release ID
3. **Album-wide Application**: Applies the discovered release ID to all tracks in the album
4. **Fallback Mechanism**: Falls back to traditional artist/album name searches when ISRC is unavailable

This approach ensures consistent and accurate metadata across all tracks in an album.

### Data Structures

```go
type MusicBrainzTrack struct {
    ID           string
    Title        string
    ArtistCredit []struct {
        Artist struct {
            ID   string
            Name string
        }
    }
    Length int // Duration in milliseconds
}

type MusicBrainzRelease struct {
    ID           string
    Title        string
    Status       string
    Date         string
    ArtistCredit []struct {
        Artist struct {
            ID   string
            Name string
        }
    }
    ReleaseGroup ReleaseGroup
}
```

## Error Handling Patterns

### Retry Logic (`retry.go`)

Implements exponential backoff for transient failures:

```go
func RetryWithBackoff(maxRetries int, initialDelaySec int, fn func() error) error {
    for attempt := 0; attempt < maxRetries; attempt++ {
        err = fn()
        if err == nil {
            return nil
        }
        
        delay := time.Duration(initialDelaySec) * time.Second * (1 << attempt)
        jitter := time.Duration(rand.Intn(100)) * time.Millisecond
        time.Sleep(delay + jitter)
    }
    return fmt.Errorf("failed after %d attempts: %w", maxRetries, err)
}
```

### API-Specific Error Handling

#### DAB API Errors
- Rate limiting (429) with automatic retry
- Invalid ID format detection
- Network connectivity issues
- Response parsing errors

#### Spotify API Errors
- Authentication failures
- Invalid URL formats
- Pagination errors
- Rate limiting

#### Navidrome API Errors
- Authentication failures
- Server connectivity issues
- Invalid playlist operations
- Search result parsing

#### MusicBrainz API Errors
- Rate limiting compliance
- Search result parsing
- Network timeouts
- Invalid query formats

## Request/Response Examples

### DAB API Search Request

```http
GET /api/search?q=Arctic%20Monkeys&type=artist&limit=10
User-Agent: DAB-Downloader/2.0
```

**Response**:
```json
{
  "artists": [
    {
      "id": "12345",
      "name": "Arctic Monkeys",
      "picture": "https://example.com/artist.jpg"
    }
  ]
}
```

### DAB API Album Request

```http
GET /api/album?albumId=67890
User-Agent: DAB-Downloader/2.0
```

**Response**:
```json
{
  "album": {
    "id": "67890",
    "title": "AM",
    "artist": "Arctic Monkeys",
    "cover": "https://example.com/cover.jpg",
    "releaseDate": "2013-09-09",
    "tracks": [
      {
        "id": "111",
        "title": "Do I Wanna Know?",
        "artist": "Arctic Monkeys",
        "trackNumber": 1,
        "duration": 272
      }
    ]
  }
}
```

### Spotify Playlist Request

```http
GET /v1/playlists/{playlist_id}
Authorization: Bearer {access_token}
```

### MusicBrainz Search Request

```http
GET /ws/2/recording?query=artist:"Arctic Monkeys" AND release:"AM" AND recording:"Do I Wanna Know?"&limit=1
User-Agent: dab-downloader/2.0 ( prathxm.in@gmail.com )
Accept: application/json
```

## Rate Limiting and Best Practices

### DAB API
- 500ms between requests
- Automatic retry on 429 responses
- Respect server response times

### Spotify API
- Built-in rate limiting via SDK
- OAuth2 token management
- Pagination handling

### MusicBrainz API
- 1 request per second (enforced by client)
- Proper User-Agent identification
- Graceful degradation on failures

### Navidrome API
- No specific rate limiting
- Connection pooling for efficiency
- Batch operations where possible

## Security Considerations

### Credential Management
- Environment variable support
- Configuration file encryption (future)
- No hardcoded credentials

### Network Security
- TLS verification (with bypass option)
- Timeout configuration
- Connection limits

### Data Privacy
- No user data collection
- Local-only operation
- Secure credential storage

## Performance Optimization

### Caching Strategy
- In-memory response caching
- Metadata caching for repeated operations
- Stream URL caching

### Connection Management
- HTTP client reuse
- Connection pooling
- Keep-alive connections

### Concurrent Processing
- Semaphore-based concurrency control
- Parallel API requests where appropriate
- Progress tracking for user feedback