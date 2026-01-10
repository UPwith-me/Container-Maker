package scan

import (
	"context"
)

// Severity levels
const (
	SeverityCritical = "CRITICAL"
	SeverityHigh     = "HIGH"
	SeverityMedium   = "MEDIUM"
	SeverityLow      = "LOW"
	SeverityUnknown  = "UNKNOWN"
)

// Vulnerability represents a single security issue
type Vulnerability struct {
	VulnerabilityID  string `json:"VulnerabilityID"`
	PkgName          string `json:"PkgName"`
	InstalledVersion string `json:"InstalledVersion"`
	FixedVersion     string `json:"FixedVersion"`
	Severity         string `json:"Severity"`
	Title            string `json:"Title"`
	Description      string `json:"Description"`
}

// Report represents a scan result
type Report struct {
	Image     string          `json:"image"`
	Vulns     []Vulnerability `json:"vulnerabilities"`
	Summary   map[string]int  `json:"summary"` // Severity -> Count
	ScannedAt string          `json:"scanned_at"`
}

// Scanner defines the interface for security scanners
type Scanner interface {
	// Scan scans an image and returns a report
	Scan(ctx context.Context, image string) (*Report, error)

	// IsAvailable checks if the scanner backend is available (e.g., binary installed)
	IsAvailable() bool
}
