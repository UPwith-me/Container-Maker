package tui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// TerminalInfo holds information about the current terminal
type TerminalInfo struct {
	Width         int
	Height        int
	IsInteractive bool
	IsTTY         bool
	Type          string // "windows", "unix", "unknown"
}

// GetTerminalInfo returns information about the current terminal
func GetTerminalInfo() TerminalInfo {
	info := TerminalInfo{
		Width:  80, // Default
		Height: 24, // Default
		Type:   "unknown",
	}

	// Check if stdout is a TTY
	if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
		info.IsTTY = true
		info.IsInteractive = true
	}

	// Detect terminal type
	if runtime.GOOS == "windows" {
		info.Type = "windows"
	} else {
		info.Type = "unix"
	}

	// Try to get terminal size
	if info.IsTTY {
		if runtime.GOOS == "windows" {
			info.Width, info.Height = getWindowsTerminalSize()
		} else {
			info.Width, info.Height = getUnixTerminalSize()
		}
	}

	return info
}

// getWindowsTerminalSize gets terminal size on Windows
func getWindowsTerminalSize() (int, int) {
	cmd := exec.Command("powershell", "-Command", "$Host.UI.RawUI.WindowSize.Width; $Host.UI.RawUI.WindowSize.Height")
	output, err := cmd.Output()
	if err != nil {
		return 80, 24
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) >= 2 {
		var w, h int
		_, _ = fmt.Sscanf(strings.TrimSpace(lines[0]), "%d", &w)
		_, _ = fmt.Sscanf(strings.TrimSpace(lines[1]), "%d", &h)
		if w > 0 && h > 0 {
			return w, h
		}
	}
	return 80, 24
}

// getUnixTerminalSize gets terminal size on Unix-like systems
func getUnixTerminalSize() (int, int) {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	output, err := cmd.Output()
	if err != nil {
		return 80, 24
	}

	var h, w int
	_, _ = fmt.Sscanf(string(output), "%d %d", &h, &w)
	if w > 0 && h > 0 {
		return w, h
	}
	return 80, 24
}

// ClearScreen clears the terminal screen
func ClearScreen() {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		_ = cmd.Run()
	} else {
		fmt.Print("\033[H\033[2J")
	}
}

// MoveCursor moves the cursor to a specific position
func MoveCursor(row, col int) {
	fmt.Printf("\033[%d;%dH", row, col)
}

// HideCursor hides the cursor
func HideCursor() {
	fmt.Print("\033[?25l")
}

// ShowCursor shows the cursor
func ShowCursor() {
	fmt.Print("\033[?25h")
}

// PrintCentered prints text centered in the terminal
func PrintCentered(text string) {
	info := GetTerminalInfo()
	padding := (info.Width - len(text)) / 2
	if padding > 0 {
		fmt.Printf("%s%s\n", strings.Repeat(" ", padding), text)
	} else {
		fmt.Println(text)
	}
}
