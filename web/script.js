document.getElementById('downloadAllBtn').addEventListener('click', () => {
    const url = document.getElementById('urlInput').value;
    const statusDiv = document.getElementById('status');

    if (!url) {
        statusDiv.textContent = 'Please enter a URL.';
        return;
    }

    statusDiv.textContent = 'Starting download...';

    fetch('/api/download', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ url: url })
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            statusDiv.textContent = 'Download finished successfully!';
        } else {
            statusDiv.textContent = 'Error: ' + data.error;
        }
    })
    .catch(error => {
        console.error('Error:', error);
        statusDiv.textContent = 'An unexpected error occurred.';
    });
});

document.getElementById('searchBtn').addEventListener('click', () => {
    const url = document.getElementById('urlInput').value;
    const statusDiv = document.getElementById('status');
    const trackListContainer = document.getElementById('track-list-container');

    if (!url) {
        statusDiv.textContent = 'Please enter a URL.';
        return;
    }

    statusDiv.textContent = 'Searching for tracks...';
    trackListContainer.innerHTML = '';

    fetch('/api/search', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ url: url })
    })
    .then(response => response.json())
    .then(data => {
        if (data.error) {
            statusDiv.textContent = 'Error: ' + data.error;
            return;
        }

        if (!data.tracks || data.tracks.length === 0) {
            statusDiv.textContent = 'No tracks found.';
            return;
        }

        statusDiv.textContent = 'Select tracks to download:';
        renderTrackList(data.tracks);
    })
    .catch(error => {
        console.error('Error:', error);
        statusDiv.textContent = 'An unexpected error occurred.';
    });
});

function renderTrackList(tracks) {
    const trackListContainer = document.getElementById('track-list-container');
    trackListContainer.innerHTML = '';

    const trackList = document.createElement('ul');
    trackList.className = 'track-list';

    tracks.forEach((track, index) => {
        const listItem = document.createElement('li');
        
        const spotifyTrackInfo = document.createElement('div');
        spotifyTrackInfo.className = 'spotify-track-info';
        spotifyTrackInfo.textContent = `${track.spotify_track.name} - ${track.spotify_track.artist}`;
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
            document.getElementById('status').textContent = 'Please select at least one track.';
            return;
        }

        document.getElementById('status').textContent = 'Downloading selected tracks...';

        fetch('/api/download-tracks', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ tracks: selectedTracks })
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                document.getElementById('status').textContent = 'Selected tracks downloaded successfully!';
            } else {
                document.getElementById('status').textContent = 'Error: ' + data.error;
            }
        })
        .catch(error => {
            console.error('Error:', error);
            document.getElementById('status').textContent = 'An unexpected error occurred.';
        });
    });
}
