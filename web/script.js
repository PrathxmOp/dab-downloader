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

    tracks.forEach(track => {
        const listItem = document.createElement('li');
        const checkbox = document.createElement('input');
        checkbox.type = 'checkbox';
        checkbox.value = JSON.stringify(track);
        checkbox.id = track.id;

        const label = document.createElement('label');
        label.htmlFor = track.id;
        label.textContent = `${track.title} - ${track.artist}`;

        listItem.appendChild(checkbox);
        listItem.appendChild(label);
        trackList.appendChild(listItem);
    });

    trackListContainer.appendChild(trackList);

    const downloadSelectedBtn = document.createElement('button');
    downloadSelectedBtn.textContent = 'Download Selected';
    downloadSelectedBtn.id = 'downloadSelectedBtn';
    trackListContainer.appendChild(downloadSelectedBtn);

    downloadSelectedBtn.addEventListener('click', () => {
        const selectedTracks = [];
        const checkboxes = document.querySelectorAll('.track-list input[type="checkbox"]:checked');
        checkboxes.forEach(checkbox => {
            selectedTracks.push(JSON.parse(checkbox.value));
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