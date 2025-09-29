package navidrome

import (
	subsonic "github.com/delucks/go-subsonic"
)

// NavidromeClient holds the navidrome client and other required fields
type NavidromeClient struct {
	URL      string
	Username string
	Password string
	Client   subsonic.Client
	Salt     string
	Token    string
}

// NewNavidromeClient creates a new navidrome client
func NewNavidromeClient(url, username, password string) *NavidromeClient {
	return &NavidromeClient{
		URL:      url,
		Username: username,
		Password: password,
	}
}