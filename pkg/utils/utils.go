package utils

import (
	"fmt"
	"runtime"
	"strings"
)

// ANSI colors
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
)

// DrawProgress draws a simple progress bar to the terminal.
func DrawProgress(step string, percent int) {
	width := 30
	filled := (percent * width) / 100
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	fmt.Printf("\r%s[%s]%s %-20s [%s] %d%%", ColorCyan, "gios", ColorReset, step, bar, percent)
	if percent >= 100 {
		fmt.Println()
	}
}

// Prompt displays a prompt and returns the user input.
func Prompt(label, def string) string {
	fmt.Printf("%s%s%s [%s]: ", ColorBold, label, ColorReset, def)
	var input string
	fmt.Scanln(&input)
	if input == "" {
		return def
	}
	return input
}
// GetTargetString returns the current platform in 'os-arch' format
func GetTargetString() string {
	os := strings.ToLower(runtime.GOOS)
	arch := strings.ToLower(runtime.GOARCH)
	if arch == "x86_64" {
		arch = "amd64"
	} else if arch == "aarch64" {
		arch = "arm64"
	}
	return fmt.Sprintf("%s-%s", os, arch)
}
