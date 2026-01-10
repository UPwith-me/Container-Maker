package scan

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

type TrivyScanner struct{}

// NewTrivyScanner creates a new Trivy scanner
func NewTrivyScanner() *TrivyScanner {
	return &TrivyScanner{}
}

func (s *TrivyScanner) IsAvailable() bool {
	_, err := exec.LookPath("trivy")
	return err == nil
}

// Internal Trivy JSON structure
type trivyOutput struct {
	Results []struct {
		Target          string          `json:"Target"`
		Vulnerabilities []Vulnerability `json:"Vulnerabilities"`
	} `json:"Results"`
}

func (s *TrivyScanner) Scan(ctx context.Context, image string) (*Report, error) {
	if !s.IsAvailable() {
		return nil, fmt.Errorf("trivy not found in PATH")
	}

	// trivy image --format json --output - <image>
	// -q to suppress progress bar
	cmd := exec.CommandContext(ctx, "trivy", "image", "-q", "--format", "json", image)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("trivy failed: %s (stderr: %s)", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("trivy failed: %w", err)
	}

	var raw trivyOutput
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse trivy output: %w", err)
	}

	report := &Report{
		Image:     image,
		ScannedAt: time.Now().Format(time.RFC3339),
		Summary:   make(map[string]int),
		Vulns:     []Vulnerability{},
	}

	// Flatten results
	for _, res := range raw.Results {
		report.Vulns = append(report.Vulns, res.Vulnerabilities...)
	}

	// Calculate summary
	for _, v := range report.Vulns {
		report.Summary[v.Severity]++
	}

	return report, nil
}
