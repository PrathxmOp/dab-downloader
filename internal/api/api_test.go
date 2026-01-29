package api

import (
	"context"
	"dab-downloader/internal/config"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestLogin(t *testing.T) {
	// Create a temporary directory for the config/token
	tempDir, err := os.MkdirTemp("", "api_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change working directory to tempDir so "config" folder is created there
	// Actually, NewDabAPI and Login write to "config" relative to CWD.
	// Best to mock the file system or run the test in a separate process, 
	// but simpler is to change CWD for this test.
	originalWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalWd)

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/login" {
			t.Errorf("Expected path /api/auth/login, got %s", r.URL.Path)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check credentials
		var creds struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		json.NewDecoder(r.Body).Decode(&creds)
		if creds.Email != "test@example.com" || creds.Password != "password" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Set session cookie
		http.SetCookie(w, &http.Cookie{
			Name:  "session",
			Value: "mock_session_token",
		})
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Initialize API client
	api := NewDabAPI(server.URL, tempDir, server.Client())

	// Test successful login
	err = api.Login("test@example.com", "password")
	if err != nil {
		t.Errorf("Login failed: %v", err)
	}

	// Verify token file was created
	tokenPath := filepath.Join(config.GetConfigDir(), ".token")
	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		t.Errorf("Token file was not created at %s", tokenPath)
	}

	// Read token and verify
	token, _ := os.ReadFile(tokenPath)
	if string(token) != "mock_session_token" {
		t.Errorf("Token mismatch: got %s, want mock_session_token", string(token))
	}

	// Test failed login
	err = api.Login("wrong@example.com", "wrong")
	if err == nil {
		t.Errorf("Expected login failure, got nil")
	}
}

func TestGetPlaylist(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/libraries/test-playlist-id" {
			t.Errorf("Expected path /api/libraries/test-playlist-id, got %s", r.URL.Path)
			return
		}

		resp := map[string]interface{}{
			"playlist": map[string]interface{}{
				"id":    "test-playlist-id",
				"title": "Test Playlist",
				"tracks": []map[string]interface{}{
					{
						"id":     "track-1",
						"title":  "Track 1",
						"artist": "Artist 1",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	api := NewDabAPI(server.URL, ".", server.Client())
	playlist, err := api.GetPlaylist(context.Background(), "test-playlist-id")
	if err != nil {
		t.Fatalf("GetPlaylist failed: %v", err)
	}

	if playlist.Title != "Test Playlist" {
		t.Errorf("Expected title 'Test Playlist', got %s", playlist.Title)
	}
	if len(playlist.Tracks) != 1 {
		t.Errorf("Expected 1 track, got %d", len(playlist.Tracks))
	}
}
