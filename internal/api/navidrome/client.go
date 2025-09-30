package navidrome

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	subsonic "github.com/delucks/go-subsonic"
)

// Authenticate authenticates the client with the navidrome api
func (n *NavidromeClient) Authenticate() error {
	// Ping the server to get the salt
	pingURL := fmt.Sprintf("%s/rest/ping.view?v=1.16.1&c=dab-downloader&f=json", n.URL)
	resp, err := http.Get(pingURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var pingResponse struct {
		SubsonicResponse struct {
			Status string `json:"status"`
			Salt   string `json:"salt"`
		} `json:"subsonic-response"`
	}

	if err := json.Unmarshal(body, &pingResponse); err != nil {
		return err
	}

	if pingResponse.SubsonicResponse.Status != "ok" {
		// Try with auth
		pingURL = fmt.Sprintf("%s/rest/ping.view?u=%s&p=%s&v=1.16.1&c=dab-downloader&f=json", n.URL, n.Username, n.Password)
		resp, err = http.Get(pingURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		if err := json.Unmarshal(body, &pingResponse); err != nil {
			return err
		}

		if pingResponse.SubsonicResponse.Status != "ok" {
			return fmt.Errorf("ping failed: %s", pingResponse.SubsonicResponse.Status)
		}
	}

	n.Salt = pingResponse.SubsonicResponse.Salt
	n.Token = getSaltedPassword(n.Password, n.Salt)

	n.Client = subsonic.Client{
		Client:       http.DefaultClient,
		BaseUrl:      n.URL,
		User:         n.Username,
		ClientName:   "dab-downloader",
		PasswordAuth: true,
	}
	return n.Client.Authenticate(n.Password)
}

// SearchTrack searches for a track on the navidrome server
func (n *NavidromeClient) SearchTrack(trackName, artistName, albumName string) (*subsonic.Child, error) {
	log.Printf("Searching Navidrome for Track: %s, Artist: %s, Album: %s", trackName, artistName, albumName)

	// First, search for the album
	album, err := n.SearchAlbum(albumName, artistName)
	if err != nil {
		log.Printf("Error searching for album '%s' by '%s': %v", albumName, artistName, err)
		// Continue to track-based search as a fallback
	}

	if album != nil {
		// Album found, now get the tracks of the album
		albumData, err := n.Client.GetAlbum(album.ID)
		if err != nil {
			log.Printf("Error getting album details for '%s': %v", albumName, err)
		} else {
			for _, song := range albumData.Song {
				if strings.EqualFold(song.Title, trackName) {
					log.Printf("Found exact track match in album: %s by %s (ID: %s)", song.Title, song.Artist, song.ID)
					return song, nil
				}
			}
		}
	}

	// Fallback to original search logic if album search is not conclusive
	log.Printf("Fallback: Searching for track directly.")
	// Try searching with track name and artist name combined first
	combinedQuery := fmt.Sprintf("%s %s", trackName, artistName)
	log.Printf("Trying combined query first: '%s'", combinedQuery)
	searchResult, err := n.Client.Search2(combinedQuery, map[string]string{"songCount": "5"})
	if err != nil {
		log.Printf("Error during combined search for '%s': %v", combinedQuery, err)
		// Don't return yet, fall back to track name only
	}

	if searchResult != nil && len(searchResult.Song) > 0 {
		log.Printf("Search result for combined query '%s': %v songs found", combinedQuery, len(searchResult.Song))
		// Check for exact match, as search can be fuzzy
		for _, song := range searchResult.Song {
			if strings.EqualFold(song.Title, trackName) && strings.EqualFold(song.Artist, artistName) {
				log.Printf("  Found exact match: %s by %s (ID: %s)", song.Title, song.Artist, song.ID)
				return song, nil
			}
		}
		// If no exact match, return the first result as a best guess
		log.Printf("  No exact match found, returning first result as best guess: %s by %s (ID: %s)", searchResult.Song[0].Title, searchResult.Song[0].Artist, searchResult.Song[0].ID)
		return searchResult.Song[0], nil
	}

	// Fallback to searching with just the track name
	log.Printf("No results from combined query, falling back to track name only: '%s'", trackName)
	searchResult, err = n.Client.Search2(trackName, map[string]string{"songCount": "10"})
	if err != nil {
		log.Printf("Error during track name search for '%s': %v", trackName, err)
		return nil, err
	}

	if searchResult != nil && len(searchResult.Song) > 0 {
		log.Printf("Search result for '%s': %v songs found", trackName, len(searchResult.Song))
		for _, song := range searchResult.Song {
			log.Printf("  Found song: %s by %s (ID: %s)", song.Title, song.Artist, song.ID)
			// Check if the artist name matches (case-insensitive)
			if strings.EqualFold(song.Artist, artistName) {
				log.Printf("  Exact artist match found for '%s'", artistName)
				return song, nil
			}
		}
	}

	log.Printf("Track '%s' by '%s' not found after all attempts.", trackName, artistName)
	return nil, nil
}

// SearchAlbum searches for an album on the navidrome server
func (n *NavidromeClient) SearchAlbum(albumName string, artistName string) (*subsonic.Child, error) {
	log.Printf("Searching Navidrome for Album: %s, Artist: %s", albumName, artistName)

	// Search for the album by name
	searchResult, err := n.Client.Search2(albumName, map[string]string{"albumCount": "5"})
	if err != nil {
		return nil, fmt.Errorf("error searching for album '%s': %w", albumName, err)
	}

	if searchResult != nil && len(searchResult.Album) > 0 {
		log.Printf("Found %d albums for query '%s'", len(searchResult.Album), albumName)
		for _, album := range searchResult.Album {
			log.Printf("  Checking album: %s by %s (ID: %s)", album.Title, album.Artist, album.ID)
			if strings.EqualFold(album.Title, albumName) && strings.EqualFold(album.Artist, artistName) {
				log.Printf("  Found exact album match: %s by %s (ID: %s)", album.Title, album.Artist, album.ID)
				return album, nil
			}
		}
		log.Printf("No exact album match found for '%s' by '%s'.", albumName, artistName)
	} else {
		log.Printf("No albums found for query '%s'", albumName)
	}

	return nil, nil // Album not found
}

// CreatePlaylist creates a new playlist on the navidrome server
func (n *NavidromeClient) CreatePlaylist(name string) error {
	// Use url.Values to properly encode the playlist name
	data := url.Values{}
	data.Set("name", name)

	// Construct the URL for creating the playlist
	createURL := fmt.Sprintf("%s/rest/createPlaylist.view?%s&u=%s&t=%s&s=%s&v=1.16.1&c=dab-downloader&f=json",
		n.URL, data.Encode(), n.Username, n.Token, n.Salt)

	// Create a new HTTP GET request
	req, err := http.NewRequest("GET", createURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Execute the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create playlist: status code %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdatePlaylist updates a playlist
func (n *NavidromeClient) UpdatePlaylist(playlistID string, name *string, comment *string, public *bool) error {
	updateMap := make(map[string]string)
	if name != nil {
		updateMap["name"] = *name
	}
	if comment != nil {
		updateMap["comment"] = *comment
	}
	if public != nil {
		updateMap["public"] = strconv.FormatBool(*public)
	}
	err := n.Client.UpdatePlaylist(playlistID, updateMap)
	if err != nil {
		return err
	}
	return nil
}

// AddTracksToPlaylist adds multiple tracks to a playlist in a single call
func (n *NavidromeClient) AddTracksToPlaylist(playlistID string, trackIDs []string) error {
	params := url.Values{}
	params.Add("playlistId", playlistID)
	params.Add("u", n.Username)
	params.Add("t", n.Token)
	params.Add("s", n.Salt)
	params.Add("v", "1.16.1")
	params.Add("c", "dab-downloader")
	params.Add("f", "json")

	for _, songID := range trackIDs {
		params.Add("songIdToAdd", songID)
	}

	updateURL := fmt.Sprintf("%s/rest/updatePlaylist.view?%s", n.URL, params.Encode())

	log.Printf("Calling update playlist URL: %s", updateURL)

	// Create a new HTTP GET request
	req, err := http.NewRequest("GET", updateURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Execute the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update playlist: status code %d, body: %s", resp.StatusCode, string(body))
	}

	// Unmarshal the response to check for Subsonic errors
	var subsonicResponse struct {
		SubsonicResponse struct {
			Status string `json:"status"`
			Error  struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		} `json:"subsonic-response"`
	}

	if err := json.Unmarshal(body, &subsonicResponse); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if subsonicResponse.SubsonicResponse.Status == "failed" {
		return fmt.Errorf("failed to update playlist: %s (code %d)", subsonicResponse.SubsonicResponse.Error.Message, subsonicResponse.SubsonicResponse.Error.Code)
	}

	return nil
}

// GetPlaylistTracks returns the tracks in a playlist
func (n *NavidromeClient) GetPlaylistTracks(playlistID string) ([]*subsonic.Child, error) {
	playlist, err := n.Client.GetPlaylist(playlistID)
	if err != nil {
		return nil, err
	}
	return playlist.Entry, nil
}

// SearchPlaylist searches for a playlist by name and returns its ID
func (n *NavidromeClient) SearchPlaylist(playlistName string) (string, error) {
	playlists, err := n.Client.GetPlaylists(map[string]string{})
	if err != nil {
		return "", err
	}

	for _, playlist := range playlists {
		if playlist.Name == playlistName {
			return playlist.ID, nil
		}
	}

	return "", fmt.Errorf("playlist '%s' not found", playlistName)
}

// getSaltedPassword returns the salted password for navidrome
func getSaltedPassword(password string, salt string) string {
	hasher := md5.New()
	hasher.Write([]byte(password + salt))
	return hex.EncodeToString(hasher.Sum(nil))
}