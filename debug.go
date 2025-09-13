package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// TestArtistEndpoints tests different possible artist endpoint formats
func (api *DabAPI) TestArtistEndpoints(ctx context.Context, artistID string) {
	colorInfo.Printf("🔍 Testing different artist endpoint formats for ID: %s\n", artistID)

	// Test different endpoint variations
	endpoints := []struct {
		path   string
		params []QueryParam
		description string
	}{
		{"discography", []QueryParam{{Name: "artistId", Value: artistID}}, "Correct endpoint (discography?artistId=)"},
		{"api/discography", []QueryParam{{Name: "artistId", Value: artistID}}, "With api prefix (api/discography?artistId=)"},
		{"discography", []QueryParam{{Name: "id", Value: artistID}}, "Alternative param (discography?id=)"},
		{"api/artist", []QueryParam{{Name: "artistId", Value: artistID}}, "Old format (api/artist?artistId=)"},
		{"api/artist", []QueryParam{{Name: "id", Value: artistID}}, "Alternative param (api/artist?id=)"},
		{"api/artists", []QueryParam{{Name: "artistId", Value: artistID}}, "Plural endpoint (api/artists?artistId=)"},
	}

	for i, endpoint := range endpoints {
		fmt.Printf("\n🧪 Test %d: %s\n", i+1, endpoint.description)

		resp, err := api.Request(ctx, endpoint.path, true, endpoint.params)
		if err != nil {
			colorError.Printf("   ❌ Failed: %v\n", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			colorError.Printf("   ❌ Failed to read body: %v\n", err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			colorSuccess.Printf("   ✅ SUCCESS! Status: %d, Body length: %d bytes\n", resp.StatusCode, len(body))
			colorSuccess.Printf("   Response preview: %.200s...\n", string(body))
		} else {
			colorWarning.Printf("   ⚠️  Status: %d, Body: %s\n", resp.StatusCode, string(body))
		}
	}
}

// TestAPIAvailability tests basic API connectivity
func (api *DabAPI) TestAPIAvailability(ctx context.Context) {
	colorInfo.Println("🌐 Testing basic API connectivity...")

	// Try a simple request to the base API
	resp, err := api.Request(ctx, "", true, nil)
	if err != nil {
		colorError.Printf("❌ Base API test failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	colorSuccess.Printf("✅ Base API accessible. Status: %d, Response: %.200s\n", resp.StatusCode, string(body))
}

// DebugArtistID performs comprehensive debugging for an artist ID
func (api *DabAPI) DebugArtistID(ctx context.Context, artistID string) {
	colorInfo.Printf("🐛 Starting comprehensive debug for artist ID: %s\n", artistID)

	// Test basic connectivity
	api.TestAPIAvailability(ctx)

	// Test different endpoint formats
	api.TestArtistEndpoints(ctx, artistID)

	// Check if it might be an album or track ID instead
	colorInfo.Printf("\n🔄 Testing if ID might be for album or track instead...\n")

	// Test as album ID
	resp, err := api.Request(ctx, "api/album", true, []QueryParam{{Name: "albumId", Value: artistID}})
	if err == nil {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			colorWarning.Printf("⚠️  ID works as ALBUM ID! You might have provided an album ID instead of artist ID\n")
			colorWarning.Printf("   Album response preview: %.200s...\n", string(body))
		}
	}

	// Test as track ID
	resp, err = api.Request(ctx, "api/track", true, []QueryParam{{Name: "trackId", Value: artistID}})
	if err == nil {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			colorWarning.Printf("⚠️  ID works as TRACK ID! You might have provided a track ID instead of artist ID\n")
			colorWarning.Printf("   Track response preview: %.200s...\n", string(body))
		}
	}
}
