## Changelog

### CI/CD
- `6b4aef5`: ci(release): auto-generate release notes
- `f598c4f`: fix(build): embed version at build time and fix progress bar errors
- `86ec26f`: chore: Update GitHub Actions workflow for version.json tagging
- `df733c5`: feat: Automate release creation on push to main

### Features
- `2f037fe`: feat: Implement versioning and update mechanism improvements
- `acea8ea`: feat: Implement playlist expansion to download full albums
- `cdf07d9`: feat: Implement rate limiting, MusicBrainz, enhanced progress, and artist search fix
- `9fb25ac`: feat: Enhance update notification with prompt, browser opening, and README guide
- `393a7cd`: feat: Implement explicit version command and colored update status
- `36ed9eb`: feat: Add ARM64 build to release workflow
- `a50c64c`: feat: Add option to save album art
- `c1183d5`: feat: Add --ignore-suffix flag to ignore any suffix
- `26b9829`: feat: Implement format conversion
- `b63de2c`: feat: Overhaul README and add Docker support
- `b4347d5`: feat: Re-implement multi-select for downloads

### Fixes
- `6ffa805`: fix(downloader): resolve progress bar race condition
- `5296fd4`: fix(metadata): correct musicbrainz id tagging
- `b930179`: fix: update link is now fixed
- `f3699f8`: fix: Deduplicate artist search results
- `206373e`: fix: Correctly display newlines in terminal output and update .gitignore
- `94e35e2`: fix: Correct GitHub repository name in updater.go
- `89b79a5`: Fix: Artist search not returning results
- `eec19e5`: fix: Preserve metadata when converting to other formats
- `74a6667`: fix: use cross-platform home directory for default download location
- `114edc8`: fix: handle pagination in spotify playlists and create config dir if not exists
- `283887b`: fix: Handle numeric artist IDs from API
