package ui

import "github.com/fatih/color"

// Package-level color variables
var (
	Info    = color.New(color.FgCyan)
	Success = color.New(color.FgGreen)
	Warning = color.New(color.FgYellow)
	Error   = color.New(color.FgRed)
	Prompt  = color.New(color.FgBlue, color.Bold) // Added for user prompts
)