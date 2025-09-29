# DAB Music Downloader - Development Guide

## Development Environment Setup

### Prerequisites

#### Required Software
- **Go 1.21+**: Primary development language
- **Git**: Version control
- **FFmpeg**: Audio format conversion (optional for development)
- **Docker**: Container development and testing (optional)

#### Development Tools (Recommended)
- **VS Code** with Go extension
- **GoLand** or other Go IDE
- **Postman** or similar for API testing
- **Terminal** with good Unicode support

### Project Setup

#### Clone and Initialize
```bash
# Clone the repository
git clone https://github.com/PrathxmOp/dab-downloader.git
cd dab-downloader

# Initialize Go modules
go mod tidy

# Build the project
go build -o dab-downloader ./cmd/dab-downloader

# Run tests (when available)
go test ./...
```

#### Configuration for Development
```bash
# Copy example configuration
cp config/example-config.json config/config.json

# Edit configuration with your settings
# - Set APIURL to development DAB API endpoint
# - Configure test credentials for integrations
# - Set appropriate download location
```

## Project Structure and Organization

### Core Modules

#### CLI Layer (`cmd/dab-downloader/`)
- **Responsibility**: Command-line interface and user interaction
- **Key Components**: Cobra commands, flag parsing, user prompts, application lifecycle
- **Development Focus**: User experience, command validation, help text
- **Files**: `main.go`, `app.go`, `interfaces.go`, `commands/*.go`

#### API Clients (`internal/api/`)
- **Responsibility**: External API communication
- **Key Components**: HTTP clients, authentication, response parsing, rate limiting
- **Development Focus**: Error handling, retry logic, API compatibility
- **Packages**: `dab/`, `spotify/`, `navidrome/`, `musicbrainz/`

#### Core Business Logic (`internal/core/`)
- **Responsibility**: Core application functionality
- **Key Components**: Download engine, search functionality, update management
- **Development Focus**: Performance, reliability, business logic correctness
- **Packages**: `downloader/`, `search/`, `updater/`

#### Configuration Management (`internal/config/`)
- **Responsibility**: Application configuration handling
- **Key Components**: Config loading, validation, defaults
- **Development Focus**: Validation, cross-platform compatibility

### Integration Modules

#### Spotify Integration (`spotify.go`, `spotify_types.go`)
- **Responsibility**: Spotify API communication
- **Key Components**: OAuth2 authentication, playlist/album retrieval
- **Development Focus**: API compatibility, pagination handling

#### Navidrome Integration (`navidrome.go`, `navidrome_types.go`)
- **Responsibility**: Navidrome/Subsonic API communication
- **Key Components**: Authentication, search, playlist management
- **Development Focus**: Search accuracy, batch operations

#### MusicBrainz Integration (`musicbrainz.go`)
- **Responsibility**: Enhanced metadata retrieval
- **Key Components**: Search queries, metadata mapping
- **Development Focus**: Rate limiting compliance, data accuracy

### Utility Modules

#### Format Conversion (`ffmpeg.go`)
- **Responsibility**: Audio format conversion using FFmpeg
- **Key Components**: Process execution, format validation, metadata preservation
- **Development Focus**: Error handling, format support, quality settings

#### Configuration Management (`utils.go`)
- **Responsibility**: Configuration loading, file operations, user input
- **Key Components**: JSON parsing, file system operations, input validation
- **Development Focus**: Validation, default handling, cross-platform compatibility

## Development Patterns and Conventions

### Code Organization

#### Package Structure
```go
// Follow Go standard project layout
// cmd/dab-downloader/ - CLI application
package main

// internal/ packages - Private application code
package api      // internal/api/dab/
package config   // internal/config/
package shared   // internal/shared/

// Import organization:
// 1. Standard library imports
// 2. Third-party imports  
// 3. Internal package imports
import (
    "context"
    "fmt"
    
    "github.com/spf13/cobra"
    
    "dab-downloader/internal/config"
    "dab-downloader/internal/services"
)
```

#### Error Handling Pattern
```go
// Consistent error wrapping
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}

// Context-aware operations
func (api *DabAPI) Operation(ctx context.Context, params) error {
    // Check context cancellation
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // Perform operation
}
```

#### Logging and Debug Output
```go
// Use color-coded output for user feedback
colorInfo.Println("Information message")
colorSuccess.Println("Success message")
colorWarning.Println("Warning message")
colorError.Println("Error message")

// Debug output with conditional display
if debug {
    fmt.Printf("DEBUG: %s\n", debugInfo)
}
```

### Concurrency Patterns

#### Semaphore-Based Concurrency
```go
// Use semaphores for resource control
sem := semaphore.NewWeighted(int64(parallelism))

// Acquire before operation
if err := sem.Acquire(ctx, 1); err != nil {
    return err
}
defer sem.Release(1)
```

#### Progress Tracking
```go
// Use progress bar pools for concurrent operations
pool, err := pb.StartPool()
if err != nil {
    // Handle gracefully
}
defer pool.Stop()

// Create individual progress bars
bar := pb.New(0)
bar.SetTemplateString(`{{ string . "prefix" }} {{ bar . }} {{ percent . }}`)
pool.Add(bar)
```

### Configuration Management

#### Hierarchical Configuration
```go
// Priority order: CLI flags > config file > defaults
if flagValue != "" {
    config.Value = flagValue
} else if config.Value == "" {
    config.Value = defaultValue
}
```

#### Validation Patterns
```go
// Validate configuration on load
func (c *Config) Validate() error {
    if c.APIURL == "" {
        return fmt.Errorf("API URL is required")
    }
    if c.Parallelism < 1 || c.Parallelism > 20 {
        return fmt.Errorf("parallelism must be between 1 and 20")
    }
    return nil
}
```

## Testing Strategy

### Unit Testing Approach

#### Test File Organization
```go
// Create test files alongside source files
// main_test.go, api_test.go, downloader_test.go, etc.

package main

import (
    "testing"
    "context"
    "net/http"
    "net/http/httptest"
)
```

#### Mock External Dependencies
```go
// Mock HTTP server for API testing
func TestAPIClient(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Mock response
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"test": "response"}`))
    }))
    defer server.Close()
    
    // Test with mock server
    client := NewDabAPI(server.URL, "/tmp", http.DefaultClient)
    // ... test operations
}
```

#### Test Data Management
```go
// Use table-driven tests for multiple scenarios
func TestSearchParsing(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected SearchResults
        wantErr  bool
    }{
        {
            name:     "valid artist search",
            input:    `{"artists": [{"id": "123", "name": "Test Artist"}]}`,
            expected: SearchResults{Artists: []Artist{{ID: "123", Name: "Test Artist"}}},
            wantErr:  false,
        },
        // ... more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Integration Testing

#### API Integration Tests
```go
// Test against real APIs with proper credentials
func TestSpotifyIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    clientID := os.Getenv("SPOTIFY_CLIENT_ID")
    clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
    if clientID == "" || clientSecret == "" {
        t.Skip("Spotify credentials not provided")
    }
    
    // Test integration
}
```

#### File System Tests
```go
// Test file operations with temporary directories
func TestDownloadFile(t *testing.T) {
    tmpDir := t.TempDir() // Automatically cleaned up
    
    // Test file download and organization
    api := NewDabAPI("http://test", tmpDir, http.DefaultClient)
    // ... test file operations
    
    // Verify file structure
    if !FileExists(filepath.Join(tmpDir, "expected", "file.flac")) {
        t.Error("Expected file not created")
    }
}
```

### Performance Testing

#### Benchmark Tests
```go
func BenchmarkMetadataProcessing(b *testing.B) {
    // Setup test data
    track := Track{Title: "Test", Artist: "Test Artist"}
    album := &Album{Title: "Test Album"}
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // Benchmark metadata processing
        AddMetadata("test.flac", track, album, nil, 10)
    }
}
```

#### Concurrency Testing
```go
func TestConcurrentDownloads(t *testing.T) {
    // Test concurrent download behavior
    var wg sync.WaitGroup
    errors := make(chan error, 10)
    
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            // Simulate concurrent download
            if err := simulateDownload(id); err != nil {
                errors <- err
            }
        }(i)
    }
    
    wg.Wait()
    close(errors)
    
    // Check for errors
    for err := range errors {
        t.Errorf("Concurrent download error: %v", err)
    }
}
```

## Debugging and Troubleshooting

### Debug Mode Implementation

#### Verbose Logging
```go
// Add debug output throughout the codebase
if debug {
    colorInfo.Printf("DEBUG: API request to %s with params %+v\n", endpoint, params)
}

// Log response details
if debug {
    fmt.Printf("DEBUG: Response status: %d, body length: %d\n", resp.StatusCode, len(body))
    fmt.Printf("DEBUG: Response body: %s\n", string(body))
}
```

#### API Debugging Tools
```go
// Implement comprehensive API testing
func (api *DabAPI) DebugEndpoint(ctx context.Context, endpoint string, params []QueryParam) {
    colorInfo.Printf("Testing endpoint: %s\n", endpoint)
    
    resp, err := api.Request(ctx, endpoint, true, params)
    if err != nil {
        colorError.Printf("Request failed: %v\n", err)
        return
    }
    defer resp.Body.Close()
    
    body, _ := io.ReadAll(resp.Body)
    colorSuccess.Printf("Status: %d, Body: %s\n", resp.StatusCode, string(body))
}
```

### Common Development Issues

#### API Rate Limiting
```go
// Implement proper rate limiting
type RateLimiter struct {
    ticker *time.Ticker
    mu     sync.Mutex
}

func (rl *RateLimiter) Wait() {
    rl.mu.Lock()
    <-rl.ticker.C
    rl.mu.Unlock()
}
```

#### Memory Management
```go
// Avoid memory leaks in long-running operations
func (api *DabAPI) DownloadWithCleanup(ctx context.Context, url string) error {
    resp, err := api.client.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close() // Always close response bodies
    
    // Process response
    return nil
}
```

#### Context Cancellation
```go
// Respect context cancellation in long operations
func (api *DabAPI) LongOperation(ctx context.Context) error {
    for i := 0; i < 1000; i++ {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        
        // Perform work
        time.Sleep(100 * time.Millisecond)
    }
    return nil
}
```

## Contributing Guidelines

### Code Style

#### Formatting
```bash
# Use gofmt for consistent formatting
go fmt ./...

# Use goimports for import organization
goimports -w .

# Run linting
golangci-lint run
```

#### Naming Conventions
- **Functions**: PascalCase for exported, camelCase for internal
- **Variables**: camelCase, descriptive names
- **Constants**: UPPER_CASE for package-level constants
- **Types**: PascalCase, clear and descriptive

#### Documentation
```go
// Package-level documentation
// Package main implements a high-quality music downloader for the DAB API.

// Function documentation with examples
// DownloadAlbum downloads all tracks from the specified album.
// It returns download statistics and any error encountered.
//
// Example:
//   stats, err := api.DownloadAlbum(ctx, "album123", config, false, nil)
//   if err != nil {
//       log.Fatal(err)
//   }
func (api *DabAPI) DownloadAlbum(ctx context.Context, albumID string, config *Config, debug bool, pool *pb.Pool) (*DownloadStats, error) {
    // Implementation
}
```

### Pull Request Process

#### Branch Naming
- `feature/description`: New features
- `bugfix/description`: Bug fixes
- `refactor/description`: Code refactoring
- `docs/description`: Documentation updates

#### Commit Messages
```
type(scope): brief description

Longer description if needed, explaining what and why.

Fixes #123
```

**Types**: feat, fix, docs, style, refactor, test, chore

#### Code Review Checklist
- [ ] Code follows project style guidelines
- [ ] All tests pass
- [ ] New functionality includes tests
- [ ] Documentation is updated
- [ ] No breaking changes (or properly documented)
- [ ] Performance impact considered
- [ ] Security implications reviewed

### Release Process

#### Version Management
```json
// Update version/version.json
{
  "version": "1.0.0"
}
```

#### Build Process
```bash
# Cross-platform builds
GOOS=linux GOARCH=amd64 go build -o dab-downloader-linux-amd64 ./cmd/dab-downloader
GOOS=windows GOARCH=amd64 go build -o dab-downloader-windows-amd64.exe ./cmd/dab-downloader
GOOS=darwin GOARCH=amd64 go build -o dab-downloader-macos-amd64 ./cmd/dab-downloader
```

#### Docker Build
```bash
# Build Docker image
docker build -t dab-downloader:latest .

# Test Docker image
docker run --rm dab-downloader:latest version
```

## Performance Optimization

### Profiling

#### CPU Profiling
```go
import _ "net/http/pprof"

// Add profiling endpoint in debug mode
if debug {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
}
```

#### Memory Profiling
```bash
# Generate memory profile
go tool pprof http://localhost:6060/debug/pprof/heap

# Analyze allocations
go tool pprof http://localhost:6060/debug/pprof/allocs
```

### Optimization Strategies

#### Reduce Allocations
```go
// Reuse buffers
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 32*1024)
    },
}

func processData(data []byte) {
    buffer := bufferPool.Get().([]byte)
    defer bufferPool.Put(buffer)
    
    // Use buffer for processing
}
```

#### Optimize I/O Operations
```go
// Use buffered I/O for better performance
func writeMetadata(file *os.File, data []byte) error {
    writer := bufio.NewWriter(file)
    defer writer.Flush()
    
    _, err := writer.Write(data)
    return err
}
```

## Security Considerations

### Input Validation
```go
// Validate all user inputs
func validateAlbumID(id string) error {
    if id == "" {
        return fmt.Errorf("album ID cannot be empty")
    }
    if len(id) > 100 {
        return fmt.Errorf("album ID too long")
    }
    // Additional validation
    return nil
}
```

### Secure File Operations
```go
// Prevent directory traversal
func sanitizePath(path string) (string, error) {
    clean := filepath.Clean(path)
    if strings.Contains(clean, "..") {
        return "", fmt.Errorf("invalid path: %s", path)
    }
    return clean, nil
}
```

### Credential Management
```go
// Never log sensitive information
func logRequest(url string, headers map[string]string) {
    safeHeaders := make(map[string]string)
    for k, v := range headers {
        if strings.ToLower(k) == "authorization" {
            safeHeaders[k] = "[REDACTED]"
        } else {
            safeHeaders[k] = v
        }
    }
    log.Printf("Request: %s, Headers: %+v", url, safeHeaders)
}
```

