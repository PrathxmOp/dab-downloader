package listenbrainz

import (
	"net/http"
	"time"

	"dab-downloader/pkg/netutil"
)

type ListenBrainzClient struct {
	BaseURL string
	Client  *http.Client
}

func NewListenBrainzClient() *ListenBrainzClient {
	return &ListenBrainzClient{
		BaseURL: "https://api.listenbrainz.org/1",
		Client: netutil.NewRobustHTTPClient(30*time.Second, false),
	}
}

type ListenBrainzTrack struct {
	Name        string
	Artist      string
	AlbumName   string
	AlbumArtist string
}
