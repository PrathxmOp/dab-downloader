# 0.1.0 (2025-09-19)


### Bug Fixes

* add blank import for embed package in main.go and bump version to 0.0.29-dev ([5eec9ac](https://github.com/PrathxmOp/dab-downloader/commit/5eec9acbed04b5d75c10b1dcdb5bdc9077bf86b9))
* Add go mod tidy and download to workflow ([88d3f73](https://github.com/PrathxmOp/dab-downloader/commit/88d3f7315edf9a8acd00745a569293ce2bef1253))
* **build:** embed version at build time and fix progress bar errors ([f598c4f](https://github.com/PrathxmOp/dab-downloader/commit/f598c4f3ee12060dab6966a66ab9b702087a1542))
* Correct GitHub repository name in updater.go ([94e35e2](https://github.com/PrathxmOp/dab-downloader/commit/94e35e29508fcef249f3367a62a769c39e0f2714))
* Correctly display newlines in terminal output and update .gitignore ([206373e](https://github.com/PrathxmOp/dab-downloader/commit/206373e407e666b718b253337c49d628df7005a8))
* Create and push Git tag before creating GitHub Release ([b0316f8](https://github.com/PrathxmOp/dab-downloader/commit/b0316f85f71a0366fb4c6f13a5c6beb03b3a63d3))
* Deduplicate artist search results ([f3699f8](https://github.com/PrathxmOp/dab-downloader/commit/f3699f81dac0a08b6a7e4c28b78dbd8c0c7f23c9)), closes [#11](https://github.com/PrathxmOp/dab-downloader/issues/11)
* **downloader:** resolve progress bar race condition ([6ffa805](https://github.com/PrathxmOp/dab-downloader/commit/6ffa805c4d98d05923e0cc3d0146cbd48822b1ac))
* embed version.json into binary ([1487f54](https://github.com/PrathxmOp/dab-downloader/commit/1487f54620a8e9ae8a5a4ecbafd9592b66ba967c))
* Ensure TAG_NAME is correctly formed in workflow ([504cb14](https://github.com/PrathxmOp/dab-downloader/commit/504cb14cbbd7bae2985711584efd3d97e1f7f387))
* Handle numeric artist IDs from API ([283887b](https://github.com/PrathxmOp/dab-downloader/commit/283887b926c3bba7c290d098589d8b70fe06f036))
* handle pagination in spotify playlists and create config dir if not exists ([114edc8](https://github.com/PrathxmOp/dab-downloader/commit/114edc83531cecc6e98af314b66cbcdd551b16a1))
* **metadata:** correct musicbrainz id tagging ([5296fd4](https://github.com/PrathxmOp/dab-downloader/commit/5296fd4a5da0dda73b56e15fd8bb9790fe53504c))
* Preserve metadata when converting to other formats ([eec19e5](https://github.com/PrathxmOp/dab-downloader/commit/eec19e5e9ee6125450e331b2ad79c2fd1995e256))
* remove duplicated code in main.go and bump version to 0.0.28-dev ([78172db](https://github.com/PrathxmOp/dab-downloader/commit/78172dbaca4e72293f0cee1536d68e96bb4d2e6c))
* Rename macOS executable to dab-downloader-macos-amd64 ([260bc25](https://github.com/PrathxmOp/dab-downloader/commit/260bc2593d29fbeff32fa339977672ac7a8ff435))
* Replace deprecated release actions with softprops/action-gh-release ([590426d](https://github.com/PrathxmOp/dab-downloader/commit/590426d1d2c66b9948041a8c4a0a3d33c0932c4c))
* replace unavailable discord webhook action ([dad3ce6](https://github.com/PrathxmOp/dab-downloader/commit/dad3ce660f1af996e49677b095b28af35ec47b53))
* update link is now fixed ([b930179](https://github.com/PrathxmOp/dab-downloader/commit/b930179d837553592d18bbe06fff2e05e2332baa))
* Update setup-go action and Go version in workflow ([1d9ce37](https://github.com/PrathxmOp/dab-downloader/commit/1d9ce37604fdfde4a89b5bc8baa7b1ff6ffc8363))
* use cross-platform home directory for default download location ([74a6667](https://github.com/PrathxmOp/dab-downloader/commit/74a6667dfb3b87d0133d427123129e9ae1719dff))


### Features

* add --expand flag to navidrome command ([82bf8c4](https://github.com/PrathxmOp/dab-downloader/commit/82bf8c4bce364745b6d314260b130e92abb8f3ec))
* Add --ignore-suffix flag to ignore any suffix ([c1183d5](https://github.com/PrathxmOp/dab-downloader/commit/c1183d54857ed0935e31494ae2c5d56aa366f19f)), closes [#8](https://github.com/PrathxmOp/dab-downloader/issues/8)
* Add ARM64 build to release workflow, enabling execution on Termux and other ARM-based Linux systems. ([36ed9eb](https://github.com/PrathxmOp/dab-downloader/commit/36ed9ebbdbe384ecb22bbdedc4c56600c7460bc9))
* Add diagnostic step to GitHub Release workflow ([6ba9a35](https://github.com/PrathxmOp/dab-downloader/commit/6ba9a358ee04ce0a6b744ff39a261f7d8dbeb752))
* Add format and bitrate options and fix various bugs ([881c8ef](https://github.com/PrathxmOp/dab-downloader/commit/881c8efb115219a0b50d76cbff32f29940334c38))
* Add GitHub Actions for releases and update README ([bd0ccf6](https://github.com/PrathxmOp/dab-downloader/commit/bd0ccf62e3512919c8134bea164afa5f8672a13f))
* add manual update guide and improve update prompt ([7701fc0](https://github.com/PrathxmOp/dab-downloader/commit/7701fc01c9b1fde2862b742d1a019b75f3f93985))
* Add option to save album art ([a50c64c](https://github.com/PrathxmOp/dab-downloader/commit/a50c64c777598e0805944177923e3de7d0bbb8da)), closes [#9](https://github.com/PrathxmOp/dab-downloader/issues/9)
* automate changelog generation and Discord notifications ([2853005](https://github.com/PrathxmOp/dab-downloader/commit/2853005c66bdcb12dfa91ec8c95924157a3866b0))
* Automate release creation on push to main ([df733c5](https://github.com/PrathxmOp/dab-downloader/commit/df733c56ed76245a58ceca4268440a085527d0fb))
* **downloader:** add log message for skipping existing tracks ([8063c8b](https://github.com/PrathxmOp/dab-downloader/commit/8063c8b90568fdca128efdd641ca47ac95414d99))
* Enhance CLI help, configuration, and Navidrome integration ([c3375fd](https://github.com/PrathxmOp/dab-downloader/commit/c3375fd70ab021d9476c73a98ff29006912c7fcb))
* Enhance update notification with prompt, browser opening, and README guide ([9fb25ac](https://github.com/PrathxmOp/dab-downloader/commit/9fb25acb7ea1dc10b9ff0f00b1ef26703490a051))
* Implement explicit version command and colored update status ([393a7cd](https://github.com/PrathxmOp/dab-downloader/commit/393a7cd87d2d22cb85351117ab4f1a619c29fbbc))
* Implement format conversion ([26b9829](https://github.com/PrathxmOp/dab-downloader/commit/26b9829f71d3a08749bf7a74a690f6a33b9c6972))
* Implement playlist expansion to download full albums ([acea8ea](https://github.com/PrathxmOp/dab-downloader/commit/acea8ea40ef41c70d51efa5d206d80c0604f2670)), closes [#12](https://github.com/PrathxmOp/dab-downloader/issues/12)
* Implement rate limiting, MusicBrainz, enhanced progress, and artist search fix ([cdf07d9](https://github.com/PrathxmOp/dab-downloader/commit/cdf07d97e2470b482b018188f1829cab99d11ebe))
* Implement robust versioning and Docker integration and fix [#15](https://github.com/PrathxmOp/dab-downloader/issues/15) ([26f2413](https://github.com/PrathxmOp/dab-downloader/commit/26f2413a597f6603b6c09380626d4ab503dccebb))
* Implement versioning and update mechanism improvements. New Versioning Scheme: Uses version/version.json for manual updates. Semantic Versioning: Uses github.com/hashicorp/go-version for robust comparisons. Configurable Update Repository: Added UpdateRepo option to config.json. Docker-Aware Update Behavior: Prevents browser attempts in headless environments, allows disabling update checks. Improved Version Display: Reads from version/version.json at runtime. Workflow Updates: Removed build-time ldflags, modified Docker build to copy version.json, updated Docker image tagging, removed redundant TAG_NAME generation. Bug Fixes: Corrected fmt.Errorf usage, fixed \n literal display in UI. This significantly enhances the flexibility and reliability of the application's update process. ([2f037fe](https://github.com/PrathxmOp/dab-downloader/commit/2f037feae5102ddcd3b6165f797e9318abc26717))
* Improve user experience and update project metadata ([9bd358a](https://github.com/PrathxmOp/dab-downloader/commit/9bd358a6240b5dcc91f476a660450761ed329101))
* **navidrome:** add --auto flag to navidrome command ([e2f86a3](https://github.com/PrathxmOp/dab-downloader/commit/e2f86a30843f03d72176bd970f026cad5bed79a8))
* Overhaul README and add Docker support ([b63de2c](https://github.com/PrathxmOp/dab-downloader/commit/b63de2ceee8fbfee2444fa7fd57ca89fc0f1dce9))
* Re-implement multi-select for downloads ([b4347d5](https://github.com/PrathxmOp/dab-downloader/commit/b4347d51ec6dca6c0b15b8f610257630fd6f504d))
* Update README with auto-dl.sh as primary quick start option and bump version ([bda991e](https://github.com/PrathxmOp/dab-downloader/commit/bda991ef2d0ae35a2f59af87a1c6a77eea91e87d))



