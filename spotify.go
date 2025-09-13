package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2/clientcredentials"
)

// SpotifyTrack represents a track from Spotify
type SpotifyTrack struct {
	Name   string
	Artist string
}

// Authenticate authenticates the client with the spotify api
func (s *SpotifyClient) Authenticate() error {
	ctx := context.Background()
	config := &clientcredentials.Config{
		ClientID:     s.ID,
		ClientSecret: s.Secret,
		TokenURL:     spotifyauth.TokenURL,
	}
	token, err := config.Token(ctx)
	if err != nil {
		return err
	}

	httpClient := spotifyauth.New().Client(ctx, token)
	s.client = spotify.New(httpClient)
	return nil
}

// GetPlaylistTracks gets the tracks from a spotify playlist
func (s *SpotifyClient) GetPlaylistTracks(playlistURL string) ([]SpotifyTrack, string, error) { // Updated signature
	parts := strings.Split(playlistURL, "/")
	if len(parts) < 5 {
		return nil, "", fmt.Errorf("invalid playlist URL")
	}
	playlistIDStr := strings.Split(parts[4], "?")[0]
	playlistID := spotify.ID(playlistIDStr)

	log.Printf("Fetching tracks from playlist: %s", playlistID)

	playlist, err := s.client.GetPlaylist(context.Background(), playlistID)
	if err != nil {
		return nil, "", err // Updated return
	}
	log.Printf("Spotify Playlist Name: %s", playlist.Name)

	var tracks []SpotifyTrack // Updated type
	for _, item := range playlist.Tracks.Tracks {
		trackName := item.Track.Name
		artistName := item.Track.Artists[0].Name
		tracks = append(tracks, SpotifyTrack{Name: trackName, Artist: artistName}) // Updated append
	}

	return tracks, playlist.Name, nil // Updated return to include playlist.Name
}
