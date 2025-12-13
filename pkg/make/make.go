package make

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Target represents a Makefile target
type Target struct {
	Name        string
	Description string
	IsPhony     bool
}

// MakefileInfo contains parsed Makefile information
type MakefileInfo struct {
	Path    string
	Targets []Target
}

// FindMakefile looks for a Makefile in the given directory
func FindMakefile(dir string) (string, error) {
	// Check common Makefile names in order of preference
	names := []string{"Makefile", "makefile", "GNUmakefile"}

	for _, name := range names {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", os.ErrNotExist
}

// HasMakefile checks if a Makefile exists in the directory
func HasMakefile(dir string) bool {
	_, err := FindMakefile(dir)
	return err == nil
}

// ParseMakefile parses a Makefile and extracts targets
func ParseMakefile(path string) (*MakefileInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info := &MakefileInfo{
		Path:    path,
		Targets: []Target{},
	}

	// Regex patterns
	targetPattern := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_\-]*)\s*:`)
	phonyPattern := regexp.MustCompile(`^\.PHONY\s*:\s*(.+)`)
	commentPattern := regexp.MustCompile(`^##\s*(.+)`)

	phonyTargets := make(map[string]bool)
	var lastComment string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Check for .PHONY declaration
		if matches := phonyPattern.FindStringSubmatch(line); len(matches) > 1 {
			for _, t := range strings.Fields(matches[1]) {
				phonyTargets[t] = true
			}
			continue
		}

		// Check for description comment (## Description)
		if matches := commentPattern.FindStringSubmatch(line); len(matches) > 1 {
			lastComment = matches[1]
			continue
		}

		// Check for target
		if matches := targetPattern.FindStringSubmatch(line); len(matches) > 1 {
			targetName := matches[1]

			// Skip internal targets (starting with .)
			if strings.HasPrefix(targetName, ".") {
				lastComment = ""
				continue
			}

			target := Target{
				Name:        targetName,
				Description: lastComment,
				IsPhony:     phonyTargets[targetName],
			}
			info.Targets = append(info.Targets, target)
			lastComment = ""
		} else if !strings.HasPrefix(line, "#") && strings.TrimSpace(line) != "" {
			// Reset comment if we hit a non-target, non-comment line
			lastComment = ""
		}
	}

	return info, scanner.Err()
}

// ListTargets returns a formatted list of targets
func ListTargets(info *MakefileInfo) string {
	if len(info.Targets) == 0 {
		return "No targets found in Makefile."
	}

	var sb strings.Builder
	sb.WriteString("📋 Available Makefile targets:\n\n")

	// Find max target name length for alignment
	maxLen := 0
	for _, t := range info.Targets {
		if len(t.Name) > maxLen {
			maxLen = len(t.Name)
		}
	}

	for _, t := range info.Targets {
		if t.Description != "" {
			sb.WriteString("  ")
			sb.WriteString(t.Name)
			sb.WriteString(strings.Repeat(" ", maxLen-len(t.Name)+2))
			sb.WriteString(t.Description)
			sb.WriteString("\n")
		} else {
			sb.WriteString("  ")
			sb.WriteString(t.Name)
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\nTip: Run 'cm make <target>' to execute")
	return sb.String()
}
