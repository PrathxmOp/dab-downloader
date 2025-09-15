package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	musicBrainzAPI = "https://musicbrainz.org/ws/2/"
	musicBrainzUserAgent = "dab-downloader/2.0 ( prathxm.in@gmail.com )" // Replace with your actual email or project contact
)

// MusicBrainzClient for making requests to the MusicBrainz API
type MusicBrainzClient struct {
	client *http.Client
}

// NewMusicBrainzClient creates a new MusicBrainz API client
func NewMusicBrainzClient() *MusicBrainzClient {
	return &MusicBrainzClient{
		client: &http.Client{
			Timeout: 30 * time.Second, // MusicBrainz API can be slow
		},
	}
}

// get makes a GET request to the MusicBrainz API
func (mb *MusicBrainzClient) get(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", musicBrainzAPI + path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", musicBrainzUserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := mb.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MusicBrainz API request failed with status: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return body, nil
}

// GetTrackMetadata fetches track metadata from MusicBrainz by MBID
func (mb *MusicBrainzClient) GetTrackMetadata(mbid string) (*MusicBrainzTrack, error) {
	path := fmt.Sprintf("recording/%s?inc=artists+releases+url-rels", mbid)
	body, err := mb.get(path)
	if err != nil {
		return nil, err
	}

	var track MusicBrainzTrack
	if err := json.Unmarshal(body, &track); err != nil {
		return nil, fmt.Errorf("failed to unmarshal MusicBrainz track metadata: %w", err)
	}
	return &track, nil
}

// GetReleaseMetadata fetches release (album) metadata from MusicBrainz by MBID
func (mb *MusicBrainzClient) GetReleaseMetadata(mbid string) (*MusicBrainzRelease, error) {
	path := fmt.Sprintf("release/%s?inc=artists+labels+recordings+url-rels", mbid)
	body, err := mb.get(path)
	if err != nil {
		return nil, err
	}

	var release MusicBrainzRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("failed to unmarshal MusicBrainz release metadata: %w", err)
	}
	return &release, nil
}

// MusicBrainzTrack represents a simplified MusicBrainz recording (track)
type MusicBrainzTrack struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	ArtistCredit []struct {
		Artist struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"artist"`
	} `json:"artist-credit"`
	Releases []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Date  string `json:"date"`
		Media []struct {
			Format string `json:"format"`
			Discs  []struct {
				ID string `json:"id"`
			} `json:"discs"`
			Tracks []struct {
				ID string `json:"id"`
				Number string `json:"number"`
				Title string `json:"title"`
				Length int `json:"length"`
			} `json:"tracks"`
		} `json:"media"`
	} `json:"releases"`
	Length int `json:"length"` // Duration in milliseconds
}

// MusicBrainzRelease represents a simplified MusicBrainz release (album)
type MusicBrainzRelease struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Date    string `json:"date"`
	Country string `json:"country"`
	ArtistCredit []struct {
		Artist struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"artist"`
	} `json:"artist-credit"`
	LabelInfo []struct {
		CatalogNumber string `json:"catalog-number"`
		Label struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"label"`
	} `json:"label-info"`
	Media []struct {
		Format string `json:"format"`
		Discs  []struct {
			ID string `json:"id"`
		} `json:"discs"`
		Tracks []struct {
			ID string `json:"id"`
			Number string `json:"number"`
			Title string `json:"title"`
			Length int `json:"length"`
		} `json:"tracks"`
	} `json:"media"`
	TextRepresentation struct {
		Language string `json:"language"`
		Script   string `json:"script"`
	} `json:"text-representation"`
	Packaging string `json:"packaging"`
	Barcode   string `json:"barcode"`
	// Add other fields as needed
}
