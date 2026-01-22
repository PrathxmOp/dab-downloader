package utils

import (
	"strings"
	"testing"
)

func TestSanitizeFileName(t *testing.T) {
	// Create a string of 300 null bytes
	inputLong := string(make([]byte, 300))
	// Expect 255 underscores because SanitizeFileName replaces nulls with underscores and truncates to 255
	expectedLong := strings.Repeat("_", 255)

	tests := []struct {
		input    string
		expected string
	}{
		{"ValidName", "ValidName"},
		{"Invalid:Name", "Invalid_Name"},
		{"Name/With/Slashes", "Name_With_Slashes"},
		{"<Invalid>", "_Invalid_"},
		{"  Trimmed  ", "Trimmed"},
		{"", "unknown"},
		{inputLong, expectedLong},
	}

	for _, test := range tests {
		result := SanitizeFileName(test.input)
		if result != test.expected {
			// Don't print the huge strings on failure to keep output clean
			if len(test.expected) > 50 {
				t.Errorf("SanitizeFileName(long input) = length %d; want length %d", len(result), len(test.expected))
			} else {
				t.Errorf("SanitizeFileName(%q) = %q; want %q", test.input, result, test.expected)
			}
		}
	}
}

func TestGetTrackFilename(t *testing.T) {
	tests := []struct {
		number   int
		title    string
		expected string
	}{
		{1, "Song Title", "01 - Song Title.flac"},
		{10, "Song Title", "10 - Song Title.flac"},
		{0, "Song Title", "Song Title.flac"},
		{5, "Invalid:Name", "05 - Invalid_Name.flac"},
	}

	for _, test := range tests {
		result := GetTrackFilename(test.number, test.title)
		if result != test.expected {
			t.Errorf("GetTrackFilename(%d, %q) = %q; want %q", test.number, test.title, result, test.expected)
		}
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"Short", 10, "Short"},
		{"ExactLength", 11, "ExactLength"},
		{"VeryLongString", 5, "Ve..."},
		{"VeryLongString", 10, "VeryLon..."},
	}

	for _, test := range tests {
		result := TruncateString(test.input, test.maxLen)
		if result != test.expected {
			t.Errorf("TruncateString(%q, %d) = %q; want %q", test.input, test.maxLen, result, test.expected)
		}
	}
}

func TestIDToString(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"123", "123"},
		{123, "123"},
		{123.0, "123"},
		{123.45, "123.45"},
		{nil, ""},
	}

	for _, test := range tests {
		result := IDToString(test.input)
		if result != test.expected {
			t.Errorf("IDToString(%v) = %q; want %q", test.input, result, test.expected)
		}
	}
}