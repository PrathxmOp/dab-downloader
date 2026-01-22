package api

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"dab-downloader/internal/models"
)

// TestArtistEndpoints tests different possible artist endpoint formats
func (api *DabAPI) TestArtistEndpoints(ctx context.Context, artistID string) {
	fmt.Printf("🔍 Testing different artist endpoint formats for ID: %s\n", artistID)

	// Test different endpoint variations
	endpoints := []struct {
		path   string
		params []models.QueryParam
		description string
	}{
		{"discography", []models.QueryParam{{Name: "artistId", Value: artistID}}, "Correct endpoint (discography?artistId=)"},
		{"api/discography", []models.QueryParam{{Name: "artistId", Value: artistID}}, "With api prefix (api/discography?artistId=)"},
		{"discography", []models.QueryParam{{Name: "id", Value: artistID}}, "Alternative param (discography?id=)"},
		{"api/artist", []models.QueryParam{{Name: "artistId", Value: artistID}}, "Old format (api/artist?artistId=)"},
		{"api/artist", []models.QueryParam{{Name: "id", Value: artistID}}, "Alternative param (api/artist?id=)"},
		{"api/artists", []models.QueryParam{{Name: "artistId", Value: artistID}}, "Plural endpoint (api/artists?artistId=)"},
	}

	for i, endpoint := range endpoints {
		fmt.Printf("\n🧪 Test %d: %s\n", i+1, endpoint.description)

		resp, err := api.Request(ctx, endpoint.path, true, endpoint.params)
		if err != nil {
			fmt.Printf("   ❌ Failed: %v\n", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			fmt.Printf("   ❌ Failed to read body: %v\n", err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			fmt.Printf("   ✅ SUCCESS! Status: %d, Body length: %d bytes\n", resp.StatusCode, len(body))
			fmt.Printf("   Response preview: %.200s...\n", string(body))
		} else {
			fmt.Printf("   ⚠️  Status: %d, Body: %s\n", resp.StatusCode, string(body))
		}
	}
}

// TestAPIAvailability tests basic API connectivity
func (api *DabAPI) TestAPIAvailability(ctx context.Context) {
	fmt.Println("🌐 Testing basic API connectivity...")

	// Try a simple request to the base API
	resp, err := api.Request(ctx, "", true, nil)
	if err != nil {
		fmt.Printf("❌ Base API test failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("✅ Base API accessible. Status: %d, Response: %.200s\n", resp.StatusCode, string(body))
}

// DebugArtistID performs comprehensive debugging for an artist ID
func (api *DabAPI) DebugArtistID(ctx context.Context, artistID string) {
	fmt.Printf("🐛 Starting comprehensive debug for artist ID: %s\n", artistID)

	// Test basic connectivity
	api.TestAPIAvailability(ctx)

	// Test different endpoint formats
	api.TestArtistEndpoints(ctx, artistID)

	// Check if it might be an album or track ID instead
	fmt.Printf("\n🔄 Testing if ID might be for album or track instead...\n")

	// Test as album ID
	resp, err := api.Request(ctx, "api/album", true, []models.QueryParam{{Name: "albumId", Value: artistID}})
	if err == nil {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			fmt.Printf("⚠️  ID works as ALBUM ID! You might have provided an album ID instead of artist ID\n")
			fmt.Printf("   Album response preview: %.200s...\n", string(body))
		}
	}

	// Test as track ID
	resp, err = api.Request(ctx, "api/track", true, []models.QueryParam{{Name: "trackId", Value: artistID}})
	if err == nil {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			fmt.Printf("⚠️  ID works as TRACK ID! You might have provided a track ID instead of artist ID\n")
			fmt.Printf("   Track response preview: %.200s...\n", string(body))
		}
	}
}