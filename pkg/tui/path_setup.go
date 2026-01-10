package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// CheckAndSetupPath checks if cm is in PATH and offers to add it
func CheckAndSetupPath() {
	// Only run on Windows
	if runtime.GOOS != "windows" {
		return
	}

	// Get the directory of the current executable
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	exeDir := filepath.Dir(exePath)

	// Check if already in PATH
	userPath := os.Getenv("Path")
	if strings.Contains(strings.ToLower(userPath), strings.ToLower(exeDir)) {
		return // Already in PATH
	}

	// Check if this is the first run by looking for a marker file
	markerFile := filepath.Join(os.Getenv("USERPROFILE"), ".cm", "path_setup_done")
	if _, err := os.Stat(markerFile); err == nil {
		return // Already offered
	}

	// Ask user
	fmt.Println()
	fmt.Println("╭──────────────────────────────────────────────────────────╮")
	fmt.Println("│  📍 Add 'cm' to PATH for global access?                  │")
	fmt.Println("│                                                          │")
	fmt.Println("│  This allows you to run 'cm' from any directory          │")
	fmt.Println("│  without typing '.\\cm.exe'                               │")
	fmt.Println("╰──────────────────────────────────────────────────────────╯")
	fmt.Print("\n  Add to PATH? [Y/n] ")

	var response string
	_, _ = fmt.Scanln(&response)

	if strings.ToLower(response) == "n" {
		// Mark as done so we don't ask again
		_ = os.MkdirAll(filepath.Dir(markerFile), 0755)
		_ = os.WriteFile(markerFile, []byte("skipped"), 0644)
		fmt.Println("  Skipped. You can run 'cm install' later to add to PATH.")
		return
	}

	// Add to PATH
	if err := AddToPath(exeDir); err != nil {
		fmt.Printf("  ❌ Failed to add to PATH: %v\n", err)
		return
	}

	// Mark as done
	_ = os.MkdirAll(filepath.Dir(markerFile), 0755)
	_ = os.WriteFile(markerFile, []byte("done"), 0644)

	fmt.Println("  ✅ Added to PATH!")
	fmt.Println()
	fmt.Println("  🔄 Refreshing current session...")

	// Try to refresh PATH in current session
	RefreshPath()

	fmt.Println("  ✅ Done! 'cm' is now available globally.")
	fmt.Println()
}

// AddToPath adds a directory to the user's PATH
func AddToPath(dir string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("auto PATH setup only supported on Windows")
	}

	// Escape single quotes effectively for PowerShell
	escapedDir := strings.ReplaceAll(dir, "'", "''")

	// Use PowerShell to add to PATH
	ps := fmt.Sprintf(`
$cmPath = '%s'
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($userPath -notlike "*$cmPath*") {
    [Environment]::SetEnvironmentVariable('Path', "$userPath;$cmPath", 'User')
}
`, escapedDir)

	cmd := exec.Command("powershell", "-Command", ps)
	return cmd.Run()
}

// RefreshPath refreshes the PATH in the current session
func RefreshPath() {
	if runtime.GOOS != "windows" {
		return
	}

	// Get updated PATH
	ps := `[Environment]::GetEnvironmentVariable('Path', 'User') + ';' + [Environment]::GetEnvironmentVariable('Path', 'Machine')`
	cmd := exec.Command("powershell", "-Command", ps)
	output, err := cmd.Output()
	if err == nil {
		os.Setenv("Path", strings.TrimSpace(string(output)))
	}
}

// IsInPath checks if cm is already in PATH
func IsInPath() bool {
	exePath, err := os.Executable()
	if err != nil {
		return false
	}
	exeDir := filepath.Dir(exePath)

	userPath := os.Getenv("Path")
	return strings.Contains(strings.ToLower(userPath), strings.ToLower(exeDir))
}
