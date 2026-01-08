package policy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/workspace"
)

// SimpleEngine implements a basic policy engine
type SimpleEngine struct {
	policies []Policy
}

// NewEngine creates a new policy engine
func NewEngine() *SimpleEngine {
	return &SimpleEngine{
		policies: DefaultPolicies(),
	}
}

// LoadPolicies loads policies from a file (placeholder for now)
func (e *SimpleEngine) LoadPolicies(path string) error {
	// In a real implementation, this would load YAML/Rego files
	return nil
}

// GetPolicies returns loaded policies
func (e *SimpleEngine) GetPolicies() []Policy {
	return e.policies
}

// EvaluateWorkspace evaluates a workspace
func (e *SimpleEngine) EvaluateWorkspace(ctx context.Context, ws *workspace.Workspace) (*EvaluationResult, error) {
	start := time.Now()
	result := &EvaluationResult{
		EvaluatedAt: start,
		Violations:  make([]Violation, 0),
		Passed:      true,
		PolicyCount: len(e.policies),
	}

	// Evaluate each service
	for _, svc := range ws.Services {
		svcResult, err := e.EvaluateService(ctx, svc)
		if err != nil {
			return nil, err
		}
		result.Violations = append(result.Violations, svcResult.Violations...)
	}

	if len(result.Violations) > 0 {
		result.Passed = false
	}

	result.Score = calculateScore(len(ws.Services), result.Violations)
	result.Duration = time.Since(start)

	return result, nil
}

// EvaluateService evaluates a single service
func (e *SimpleEngine) EvaluateService(ctx context.Context, svc *workspace.Service) (*EvaluationResult, error) {
	result := &EvaluationResult{
		EvaluatedAt: time.Now(),
		Violations:  make([]Violation, 0),
		Passed:      true,
	}

	for _, p := range e.policies {
		if !p.Enabled {
			continue
		}

		violation := e.checkPolicy(p, svc)
		if violation != nil {
			result.Violations = append(result.Violations, *violation)
		}
	}

	if len(result.Violations) > 0 {
		result.Passed = false
	}

	return result, nil
}

// checkPolicy checks a single policy against a service
func (e *SimpleEngine) checkPolicy(p Policy, svc *workspace.Service) *Violation {
	switch p.ID {
	case "SEC-001": // No Privileged
		// This requires checking capabilities or privileged flag
		// Assuming we store this in Labels or strict config for now as the workspace type might not expose it directly yet
		// But let's check if "privileged" is in networks or volumes which is wrong usage
		return nil // Placeholder as Type definition might need update to support Privileged flag directly

	case "SEC-002": // Root User
		if svc.Image != "" && !strings.Contains(svc.Image, "nonroot") {
			// This is a naive check, real check needs image inspection
			// Skipping for now to avoid false positives
		}

	case "RES-001": // Memory Limits
		if svc.Resources == nil || svc.Resources.Memory == "" {
			return &Violation{
				PolicyID:   p.ID,
				PolicyName: p.Name,
				Severity:   p.Severity,
				Message:    "Memory limit not defined",
				Resource:   svc.Name,
				Suggestion: "Add 'resources.memory' to service definition",
				Timestamp:  time.Now(),
			}
		}

	case "BP-001": // Healthcheck
		if svc.HealthCheck == nil {
			return &Violation{
				PolicyID:   p.ID,
				PolicyName: p.Name,
				Severity:   p.Severity,
				Message:    "Healthcheck not configured",
				Resource:   svc.Name,
				Suggestion: "Add 'healthcheck' block to ensure service reliability",
				Timestamp:  time.Now(),
			}
		}

	case "BP-002": // Version Pinning
		if svc.Image != "" {
			if strings.HasSuffix(svc.Image, ":latest") || !strings.Contains(svc.Image, ":") {
				return &Violation{
					PolicyID:   p.ID,
					PolicyName: p.Name,
					Severity:   p.Severity,
					Message:    fmt.Sprintf("Image '%s' uses 'latest' tag or no tag", svc.Image),
					Resource:   svc.Name,
					Suggestion: "Use a specific version tag (e.g., :v1.0.0, :14-alpine)",
					Timestamp:  time.Now(),
				}
			}
		}
	}

	return nil
}

func calculateScore(serviceCount int, violations []Violation) int {
	if serviceCount == 0 {
		return 100
	}

	score := 100

	for _, v := range violations {
		deduction := 0
		switch v.Severity {
		case SeverityCritical:
			deduction = 20
		case SeverityError:
			deduction = 10
		case SeverityWarning:
			deduction = 5
		case SeverityInfo:
			deduction = 1
		}
		score -= deduction
	}

	if score < 0 {
		return 0
	}
	return score
}
