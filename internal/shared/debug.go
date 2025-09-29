package shared

import (
	"fmt"
	"os"
)

// DebugPrint prints debug messages when debug mode is enabled
func DebugPrint(debug bool, format string, args ...interface{}) {
	if debug {
		fmt.Printf("DEBUG: "+format+"\n", args...)
	}
}

// DebugPrintln prints debug messages with newline when debug mode is enabled
func DebugPrintln(debug bool, message string) {
	if debug {
		fmt.Printf("DEBUG: %s\n", message)
	}
}

// IsDebugMode checks if debug mode is enabled via environment variable
func IsDebugMode() bool {
	return os.Getenv("DEBUG") == "1" || os.Getenv("DEBUG") == "true"
}