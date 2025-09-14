document.addEventListener('DOMContentLoaded', () => {
    const tabs = document.querySelectorAll('.tab-button');
    const tabContents = document.querySelectorAll('.tab-content');
    const settingsIcon = document.getElementById('settings-icon');
    const settingsModal = document.getElementById('settings-modal');
    const closeButton = document.querySelector('.close-button');
    const settingsForm = document.getElementById('settings-form');

    loadSettings();

    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            tabs.forEach(t => t.classList.remove('active'));
            tab.classList.add('active');

            tabContents.forEach(content => content.classList.remove('active'));
            document.getElementById(tab.dataset.tab).classList.add('active');
        });
    });

    settingsIcon.addEventListener('click', () => {
        settingsModal.style.display = 'block';
    });

    closeButton.addEventListener('click', () => {
        settingsModal.style.display = 'none';
    });

    window.addEventListener('click', (event) => {
        if (event.target == settingsModal) {
            settingsModal.style.display = 'none';
        }
    });

    settingsForm.addEventListener('submit', (event) => {
        event.preventDefault();
        saveSettings();
        settingsModal.style.display = 'none';
    });

    document.getElementById('searchBtn').addEventListener('click', () => {
        const query = document.getElementById('urlInput').value;
        if (!query) {
            showStatus('Please enter a URL or search query.');
            return;
        }
        showStatus('Searching...');
        search(query);
    });

    document.getElementById('downloadAllBtn').addEventListener('click', () => {
        const query = document.getElementById('urlInput').value;
        if (!query) {
            showStatus('Please enter a URL or search query.');
            return;
        }
        showStatus('Downloading all...');
        downloadAll(query);
    });

    document.getElementById('downloadArtistBtn').addEventListener('click', () => {
        const artistId = document.getElementById('artistIdInput').value;
        if (!artistId) {
            showStatus('Please enter an artist ID.');
            return;
        }
        showStatus(`Downloading discography for artist ${artistId}...`);
        downloadArtist(artistId);
    });

    document.getElementById('downloadAlbumBtn').addEventListener('click', () => {
        const albumId = document.getElementById('albumIdInput').value;
        if (!albumId) {
            showStatus('Please enter an album ID.');
            return;
        }
        showStatus(`Downloading album ${albumId}...`);
        downloadAlbum(albumId);
    });

    document.getElementById('downloadSpotifyBtn').addEventListener('click', () => {
        const spotifyUrl = document.getElementById('spotifyUrlInput').value;
        if (!spotifyUrl) {
            showStatus('Please enter a Spotify URL.');
            return;
        }
        showStatus(`Downloading from Spotify URL ${spotifyUrl}...`);
        downloadSpotify(spotifyUrl);
    });

    document.getElementById('copyNavidromeBtn').addEventListener('click', () => {
        const navidromeUrl = document.getElementById('navidromeUrlInput').value;
        if (!navidromeUrl) {
            showStatus('Please enter a Spotify URL.');
            return;
        }
        showStatus(`Copying to Navidrome from ${navidromeUrl}...`);
        copyToNavidrome(navidromeUrl);
    });

    function showStatus(message) {
        document.getElementById('status').textContent = message;
    }

    function getSettings() {
        return {
            APIURL: document.getElementById('api-url').value,
            DownloadLocation: document.getElementById('download-location').value,
            SpotifyClientID: document.getElementById('spotify-client-id').value,
            SpotifyClientSecret: document.getElementById('spotify-client-secret').value,
            NavidromeURL: document.getElementById('navidrome-url').value,
            NavidromeUsername: document.getElementById('navidrome-username').value,
            NavidromePassword: document.getElementById('navidrome-password').value,
            Format: document.getElementById('format').value,
            Bitrate: document.getElementById('bitrate').value,
            Debug: document.getElementById('debug').checked
        };
    }

    function saveSettings() {
        const settings = getSettings();
        fetch('/api/settings', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(settings)
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                showStatus('Settings saved.');
            } else {
                showStatus('Error: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showStatus('An unexpected error occurred.');
        });
    }

    function loadSettings() {
        fetch('/api/settings')
        .then(response => response.json())
        .then(data => {
            document.getElementById('api-url').value = data.APIURL || '';
            document.getElementById('download-location').value = data.DownloadLocation || '';
            document.getElementById('spotify-client-id').value = data.SpotifyClientID || '';
            document.getElementById('spotify-client-secret').value = data.SpotifyClientSecret || '';
            document.getElementById('navidrome-url').value = data.NavidromeURL || '';
            document.getElementById('navidrome-username').value = data.NavidromeUsername || '';
            document.getElementById('navidrome-password').value = data.NavidromePassword || '';
            document.getElementById('format').value = data.Format || 'flac';
            document.getElementById('bitrate').value = data.Bitrate || '320';
            document.getElementById('debug').checked = data.Debug || false;
        });
    }

    function search(query) {
        const settings = getSettings();
        fetch('/api/search', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ query: query, config: settings })
        })
        .then(response => response.json())
        .then(data => {
            if (data.error) {
                showStatus('Error: ' + data.error);
                return;
            }
            renderTrackList(data.tracks);
        })
        .catch(error => {
            console.error('Error:', error);
            showStatus('An unexpected error occurred.');
        });
    }

    function downloadAll(query) {
        const settings = getSettings();
        fetch('/api/download-spotify', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ url: query, config: settings })
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                showStatus('Download finished successfully!');
            } else {
                showStatus('Error: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showStatus('An unexpected error occurred.');
        });
    }

    function downloadArtist(artistId) {
        const settings = getSettings();
        fetch('/api/download-artist', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ artistId: artistId, config: settings })
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                showStatus('Artist discography downloaded successfully!');
            } else {
                showStatus('Error: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showStatus('An unexpected error occurred.');
        });
    }

    function downloadAlbum(albumId) {
        const settings = getSettings();
        fetch('/api/download-album', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ albumId: albumId, config: settings })
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                showStatus('Album downloaded successfully!');
            } else {
                showStatus('Error: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showStatus('An unexpected error occurred.');
        });
    }

    function downloadSpotify(spotifyUrl) {
        const settings = getSettings();
        fetch('/api/download-spotify', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ url: spotifyUrl, config: settings })
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                showStatus('Downloaded from Spotify successfully!');
            } else {
                showStatus('Error: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showStatus('An unexpected error occurred.');
        });
    }

    function copyToNavidrome(navidromeUrl) {
        const settings = getSettings();
        fetch('/api/copy-navidrome', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ url: navidromeUrl, config: settings })
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                showStatus('Copied to Navidrome successfully!');
            } else {
                showStatus('Error: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showStatus('An unexpected error occurred.');
        });
    }

    function renderTrackList(tracks) {
        const trackListContainer = document.getElementById('track-list-container');
        trackListContainer.innerHTML = '';

        if (!tracks || tracks.length === 0) {
            showStatus('No tracks found.');
            return;
        }

        const trackList = document.createElement('ul');
        trackList.className = 'track-list';

        tracks.forEach((track, index) => {
            const listItem = document.createElement('li');
            
            const spotifyTrackInfo = document.createElement('div');
            spotifyTrackInfo.className = 'spotify-track-info';
            if (track.spotify_track) {
                spotifyTrackInfo.textContent = `${track.spotify_track.name} - ${track.spotify_track.artist}`;
            } else {
                spotifyTrackInfo.textContent = 'Search Results';
            }
            listItem.appendChild(spotifyTrackInfo);

            const dabResultsList = document.createElement('ul');
            dabResultsList.className = 'dab-results-list';

            if (track.dab_results && track.dab_results.length > 0) {
                track.dab_results.forEach(dabResult => {
                    const dabResultItem = document.createElement('li');
                    const radio = document.createElement('input');
                    radio.type = 'radio';
                    radio.name = `spotify-track-${index}`;
                    radio.value = JSON.stringify(dabResult);

                    const label = document.createElement('label');
                    label.textContent = `${dabResult.title} - ${dabResult.artist}`;

                    dabResultItem.appendChild(radio);
                    dabResultItem.appendChild(label);
                    dabResultsList.appendChild(dabResultItem);
                });
            } else {
                const noResultItem = document.createElement('li');
                noResultItem.textContent = 'No DAB results found.';
                dabResultsList.appendChild(noResultItem);
            }

            listItem.appendChild(dabResultsList);
            trackList.appendChild(listItem);
        });

        trackListContainer.appendChild(trackList);

        const downloadSelectedBtn = document.createElement('button');
        downloadSelectedBtn.textContent = 'Download Selected';
        downloadSelectedBtn.id = 'downloadSelectedBtn';
        trackListContainer.appendChild(downloadSelectedBtn);

        downloadSelectedBtn.addEventListener('click', () => {
            const selectedTracks = [];
            const radioButtons = document.querySelectorAll('.dab-results-list input[type="radio"]:checked');
            radioButtons.forEach(radio => {
                selectedTracks.push(JSON.parse(radio.value));
            });

            if (selectedTracks.length === 0) {
                showStatus('Please select at least one track.');
                return;
            }

            showStatus('Downloading selected tracks...');
            downloadSelected(selectedTracks);
        });
    }

    function downloadSelected(tracks) {
        const settings = getSettings();
        fetch('/api/download-tracks', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ tracks: tracks, config: settings })
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                showStatus('Selected tracks downloaded successfully!');
            } else {
                showStatus('Error: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showStatus('An unexpected error occurred.');
        });
    }
});
