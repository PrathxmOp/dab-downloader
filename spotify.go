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
	for {
		for _, item := range playlist.Tracks.Tracks {
			trackName := item.Track.Name
			artistName := item.Track.Artists[0].Name
			tracks = append(tracks, SpotifyTrack{Name: trackName, Artist: artistName}) // Updated append
		}

		err = s.client.NextPage(context.Background(), &playlist.Tracks)
		if err == spotify.ErrNoMorePages {
			break
		}
		if err != nil {
			return nil, "", err
		}
	}

	return tracks, playlist.Name, nil // Updated return to include playlist.Name
}

// GetAlbumTracks gets the tracks from a spotify album
func (s *SpotifyClient) GetAlbumTracks(albumURL string) ([]SpotifyTrack, string, error) {
	parts := strings.Split(albumURL, "/")
	if len(parts) < 5 || parts[3] != "album" {
		return nil, "", fmt.Errorf("invalid album URL")
	}
	albumIDStr := strings.Split(parts[4], "?")[0]
	albumID := spotify.ID(albumIDStr)

	log.Printf("Fetching tracks from album: %s", albumID)

	album, err := s.client.GetAlbum(context.Background(), albumID)
	if err != nil {
		return nil, "", err
	}
	log.Printf("Spotify Album Name: %s", album.Name)

	var tracks []SpotifyTrack
	for _, track := range album.Tracks.Tracks {
		trackName := track.Name
		artistName := track.Artists[0].Name
		tracks = append(tracks, SpotifyTrack{Name: trackName, Artist: artistName})
	}

	return tracks, album.Name, nil
}
