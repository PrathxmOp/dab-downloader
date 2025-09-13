package main

import "github.com/fatih/color"

// Package-level color variables
var (
	colorInfo    = color.New(color.FgCyan)
	colorSuccess = color.New(color.FgGreen)
	colorWarning = color.New(color.FgYellow)
	colorError   = color.New(color.FgRed)
	colorPrompt  = color.New(color.FgBlue, color.Bold) // Added for user prompts
)
