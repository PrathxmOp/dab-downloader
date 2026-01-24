package listenbrainz

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

type LBPlaylistResponse struct {
	Playlist struct {
		Title      string `json:"title"`
		Identifier string `json:"identifier"`
		Track      []struct {
			Title   string `json:"title"`
			Creator string `json:"creator"`
			// ListenBrainz JSPF extension
			Extension struct {
				Musicbrainz struct {
					ArtistNames []string `json:"artist_names"`
					TrackName   string   `json:"track_name"`
					ReleaseName string   `json:"release_name"`
				} `json:"https://musicbrainz.org/doc/jspf#track"`
			} `json:"extension"`
		} `json:"track"`
	} `json:"playlist"`
}

// GetPlaylistTracks gets tracks from a ListenBrainz playlist URL or ID
func (c *ListenBrainzClient) GetPlaylistTracks(playlistURL string) ([]ListenBrainzTrack, string, error) {
	// Example URL: https://listenbrainz.org/playlist/639e31d3-3760-4963-9584-601f01633516
	parts := strings.Split(playlistURL, "/")
	playlistID := parts[len(parts)-1]
	if strings.Contains(playlistID, "?") {
		playlistID = strings.Split(playlistID, "?")[0]
	}

	apiURL := fmt.Sprintf("%s/playlist/%s", c.BaseURL, playlistID)
	log.Printf("Fetching ListenBrainz playlist: %s", apiURL)

	resp, err := c.Client.Get(apiURL)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("LB API error: %d - %s", resp.StatusCode, string(body))
	}

	var lbResp LBPlaylistResponse
	if err := json.NewDecoder(resp.Body).Decode(&lbResp); err != nil {
		return nil, "", err
	}

	var tracks []ListenBrainzTrack
	for _, item := range lbResp.Playlist.Track {
		track := ListenBrainzTrack{
			Name:   item.Title,
			Artist: item.Creator,
		}

		// Use extension data if available for better accuracy
		mbExt := item.Extension.Musicbrainz
		if mbExt.TrackName != "" {
			track.Name = mbExt.TrackName
		}
		if len(mbExt.ArtistNames) > 0 {
			track.Artist = mbExt.ArtistNames[0]
			track.AlbumArtist = mbExt.ArtistNames[0]
		}
		if mbExt.ReleaseName != "" {
			track.AlbumName = mbExt.ReleaseName
		}

		tracks = append(tracks, track)
	}

	return tracks, lbResp.Playlist.Title, nil
}

// GetUserRecommendations gets the "Weekly Exploration" playlist for a user
func (c *ListenBrainzClient) GetUserRecommendations(username string) ([]ListenBrainzTrack, string, error) {
	// ListenBrainz often provides specific playlists for users like 'weekly-exploration'
	// However, the playlist ID is what we need.
	// For now, we will suggest the user to provide the playlist URL, 
	// but we can add a helper to fetch user's playlists if needed.
	return nil, "", fmt.Errorf("fetching by username not fully implemented, please provide playlist URL")
}
