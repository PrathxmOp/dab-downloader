package spotify

import (
	"github.com/zmb3/spotify/v2"
)

// SpotifyClient holds the spotify client and other required fields
type SpotifyClient struct {
	client *spotify.Client
	ID     string
	Secret string
}

// NewSpotifyClient creates a new spotify client
func NewSpotifyClient(id, secret string) *SpotifyClient {
	return &SpotifyClient{
		ID:     id,
		Secret: secret,
	}
}

// SpotifyTrack represents a track from Spotify
type SpotifyTrack struct {
	Name        string
	Artist      string
	AlbumName   string
	AlbumArtist string
}