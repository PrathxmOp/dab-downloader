package shared

import (
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"os"
)

// Package-level color variables
var (
	ColorInfo    = color.New(color.FgCyan)
	ColorSuccess = color.New(color.FgGreen)
	ColorWarning = color.New(color.FgYellow)
	ColorError   = color.New(color.FgRed)
	ColorPrompt  = color.New(color.FgBlue, color.Bold) // Added for user prompts
)

// InitializeColors initializes color output based on TTY detection
func InitializeColors() {
	color.NoColor = !isatty.IsTerminal(os.Stdout.Fd())
}