package downloader

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CheckFFmpeg checks if ffmpeg is installed and available in the system's PATH.
func CheckFFmpeg() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

// ConvertTrack converts a track to the specified format using ffmpeg.
func ConvertTrack(inputFile, format, bitrate string) (string, error) {
	outputFile := strings.TrimSuffix(inputFile, filepath.Ext(inputFile)) + "." + format

	var cmd *exec.Cmd
	switch format {
	case "mp3":
		cmd = exec.Command("ffmpeg", "-i", inputFile, "-b:a", bitrate+"k", "-vn", "-map_metadata", "0", outputFile)
	case "ogg":
		// For ogg, -q:a (quality) is often preferred over bitrate.
		// A mapping from bitrate to quality could be implemented if needed.
		// For now, using a high quality setting.
		cmd = exec.Command("ffmpeg", "-i", inputFile, "-c:a", "libvorbis", "-q:a", "8", "-vn", "-map_metadata", "0", outputFile)
	case "opus":
		cmd = exec.Command("ffmpeg", "-i", inputFile, "-c:a", "libopus", "-b:a", bitrate+"k", "-vn", "-map_metadata", "0", outputFile)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to convert track: %w\nffmpeg output: %s", err, string(output))
	}

	// Verify that the output file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		return "", fmt.Errorf("converted file not found after conversion")
	}

	return outputFile, nil
}