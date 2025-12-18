# DAB Downloader - Go Project Structure

This document outlines the reorganized project structure following Go community standards.

## Directory Structure

```
dab-downloader/
├── cmd/                           # Command-line applications
│   └── dab-downloader/           # Main CLI application
│       ├── commands/             # CLI command implementations
│       │   ├── root.go          # Root command and setup
│       │   ├── album.go         # Album download command
│       │   ├── artist.go        # Artist download command
│       │   ├── search.go        # Search command
│       │   ├── spotify.go       # Spotify integration command
│       │   ├── navidrome.go     # Navidrome integration command
│       │   ├── server.go        # Web server command
│       │   ├── debug.go         # Debug utilities command
│       │   └── version.go       # Version command
│       ├── main.go              # Application entry point
│       ├── app.go               # Application orchestration
│       └── interfaces.go        # Application-level interfaces
├── internal/                     # Private application code
│   ├── api/                     # External API clients
│   │   ├── dab/                 # DAB API client
│   │   │   └── client.go
│   │   ├── spotify/             # Spotify API client
│   │   │   ├── client.go
│   │   │   └── types.go
│   │   ├── navidrome/           # Navidrome API client
│   │   │   ├── client.go
│   │   │   └── types.go
│   │   └── musicbrainz/         # MusicBrainz API client
│   │       └── client.go
│   ├── core/                    # Core business logic
│   │   ├── downloader/          # Download functionality
│   │   │   ├── downloader.go    # Main download logic
│   │   │   ├── artist.go        # Artist-specific downloads
│   │   │   ├── retry.go         # Retry mechanisms
│   │   │   ├── ffmpeg.go        # Audio conversion
│   │   │   └── metadata.go      # Metadata handling
│   │   ├── search/              # Search functionality
│   │   │   └── search.go
│   │   └── updater/             # Application updates
│   │       └── updater.go
│   ├── config/                  # Configuration management
│   │   └── config.go
│   ├── shared/                  # Shared utilities and types
│   │   ├── types.go             # Common data types
│   │   ├── utils.go             # Utility functions
│   │   ├── colors.go            # Color/styling utilities
│   │   ├── debug.go             # Debug utilities
│   │   └── warnings.go          # Warning collection
│   ├── interfaces/              # Application interfaces
│   │   └── interfaces.go
│   └── services/                # Service layer
│       └── services.go
├── config/                      # Configuration files
├── docs/                        # Documentation
├── version/                     # Version information
├── go.mod                       # Go module definition
├── go.sum                       # Go module checksums
└── README.md                    # Project documentation
```

## Package Organization

### `cmd/dab-downloader/`
Contains the main application entry point and command-line interface. This follows the Go standard of placing executable commands in the `cmd/` directory.

### `internal/`
Contains all private application code that should not be imported by other projects. This enforces encapsulation and API boundaries.

#### `internal/api/`
External API client implementations, organized by service:
- **dab/**: DAB music API client
- **spotify/**: Spotify Web API client  
- **navidrome/**: Navidrome server API client
- **musicbrainz/**: MusicBrainz metadata API client

#### `internal/core/`
Core business logic, organized by domain:
- **downloader/**: All download-related functionality
- **search/**: Search and discovery functionality
- **updater/**: Application self-update functionality

#### `internal/config/`
Configuration management, including loading, saving, and validation.

#### `internal/shared/`
Shared utilities, types, and helper functions used across the application.

#### `internal/interfaces/`
Application-wide interface definitions for dependency injection and testing.

#### `internal/services/`
Service layer that orchestrates business logic and manages dependencies.

## Benefits of This Structure

1. **Clear Separation of Concerns**: Each package has a single, well-defined responsibility
2. **Encapsulation**: Internal packages prevent external dependencies on implementation details
3. **Testability**: Clean interfaces make unit testing easier
4. **Maintainability**: Related code is grouped together, making it easier to find and modify
5. **Go Standards Compliance**: Follows established Go community conventions
6. **Scalability**: Structure supports growth without major refactoring

