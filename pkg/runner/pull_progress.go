package runner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// PullProgress represents a Docker pull progress event
type PullProgress struct {
	Status         string `json:"status"`
	ID             string `json:"id"`
	Progress       string `json:"progress"`
	ProgressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"progressDetail"`
}

// PullProgressDisplay handles parsing and displaying Docker pull progress
type PullProgressDisplay struct {
	layers map[string]*layerState
}

type layerState struct {
	status   string
	current  int64
	total    int64
	complete bool
}

// NewPullProgressDisplay creates a new progress display
func NewPullProgressDisplay() *PullProgressDisplay {
	return &PullProgressDisplay{
		layers: make(map[string]*layerState),
	}
}

// ProcessPullOutput reads Docker pull output and displays clean progress
func (p *PullProgressDisplay) ProcessPullOutput(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	lastLine := ""

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var progress PullProgress
		if err := json.Unmarshal([]byte(line), &progress); err != nil {
			// Not JSON, just print it
			fmt.Println(line)
			continue
		}

		// Update layer state
		if progress.ID != "" {
			if _, exists := p.layers[progress.ID]; !exists {
				p.layers[progress.ID] = &layerState{}
			}
			layer := p.layers[progress.ID]
			layer.status = progress.Status
			if progress.ProgressDetail.Total > 0 {
				layer.current = progress.ProgressDetail.Current
				layer.total = progress.ProgressDetail.Total
			}
			if progress.Status == "Pull complete" || progress.Status == "Download complete" {
				layer.complete = true
			}
		}

		// Generate display line
		displayLine := p.formatProgress(progress)
		if displayLine != "" && displayLine != lastLine {
			// Clear previous line and print new one
			fmt.Printf("\r\033[K%s", displayLine)
			lastLine = displayLine
		}

		// Print status messages on new lines
		if progress.Status == "Digest:" || strings.HasPrefix(progress.Status, "Status:") {
			fmt.Printf("\n%s\n", progress.Status)
		}
	}

	fmt.Println() // Final newline
	return scanner.Err()
}

func (p *PullProgressDisplay) formatProgress(prog PullProgress) string {
	switch prog.Status {
	case "Pulling from library/debian", "Pulling from library/python", "Pulling from library/gcc":
		return ""
	case "Pulling fs layer":
		return fmt.Sprintf("📦 Preparing layer %s...", prog.ID[:12])
	case "Downloading":
		return p.formatDownloadProgress()
	case "Extracting":
		return p.formatExtractProgress()
	case "Pull complete":
		return fmt.Sprintf("✅ Layer %s complete", prog.ID[:12])
	case "Download complete":
		return fmt.Sprintf("📥 Layer %s downloaded", prog.ID[:12])
	default:
		if strings.HasPrefix(prog.Status, "Pulling from") {
			return ""
		}
		return ""
	}
}

func (p *PullProgressDisplay) formatDownloadProgress() string {
	var totalCurrent, totalSize int64
	downloadingCount := 0
	completeCount := 0

	for _, layer := range p.layers {
		if layer.complete {
			completeCount++
			continue
		}
		if layer.total > 0 {
			totalCurrent += layer.current
			totalSize += layer.total
			downloadingCount++
		}
	}

	if totalSize == 0 {
		return "📥 Downloading..."
	}

	percent := float64(totalCurrent) / float64(totalSize) * 100
	bar := p.progressBar(percent, 30)

	return fmt.Sprintf("📥 Downloading: %s %.0f%% (%s/%s)",
		bar, percent,
		formatBytes(totalCurrent), formatBytes(totalSize))
}

func (p *PullProgressDisplay) formatExtractProgress() string {
	extractingCount := 0
	completeCount := 0

	for _, layer := range p.layers {
		if layer.status == "Extracting" {
			extractingCount++
		}
		if layer.complete {
			completeCount++
		}
	}

	total := len(p.layers)
	if total == 0 {
		return "📦 Extracting..."
	}

	percent := float64(completeCount) / float64(total) * 100
	bar := p.progressBar(percent, 30)

	return fmt.Sprintf("📦 Extracting:  %s %.0f%% (%d/%d layers)",
		bar, percent, completeCount, total)
}

func (p *PullProgressDisplay) progressBar(percent float64, width int) string {
	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return bar
}

func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fGB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1fMB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
