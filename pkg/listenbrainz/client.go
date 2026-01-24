package listenbrainz

import (
	"net/http"
	"time"
)

type ListenBrainzClient struct {
	BaseURL string
	Client  *http.Client
}

func NewListenBrainzClient() *ListenBrainzClient {
	return &ListenBrainzClient{
		BaseURL: "https://api.listenbrainz.org/1",
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type ListenBrainzTrack struct {
	Name        string
	Artist      string
	AlbumName   string
	AlbumArtist string
}
