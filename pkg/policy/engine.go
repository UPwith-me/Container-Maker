package policy

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/workspace"
	"gopkg.in/yaml.v3"
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

// LoadPolicies loads policies from a YAML file
func (e *SimpleEngine) LoadPolicies(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read policy file: %w", err)
	}

	var policyFile struct {
		Version  string   `yaml:"version"`
		Policies []Policy `yaml:"policies"`
	}

	if err := yaml.Unmarshal(data, &policyFile); err != nil {
		return fmt.Errorf("failed to parse policy file: %w", err)
	}

	// Merge with default policies, custom ones override
	policyMap := make(map[string]Policy)
	for _, p := range e.policies {
		policyMap[p.ID] = p
	}
	for _, p := range policyFile.Policies {
		p.Enabled = true // Loaded policies are enabled by default
		policyMap[p.ID] = p
	}

	e.policies = make([]Policy, 0, len(policyMap))
	for _, p := range policyMap {
		e.policies = append(e.policies, p)
	}

	fmt.Printf("📋 Loaded %d policies from %s\n", len(policyFile.Policies), path)
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
		if svc.Privileged {
			return &Violation{
				PolicyID:   p.ID,
				PolicyName: p.Name,
				Severity:   p.Severity,
				Message:    "Service runs in privileged mode",
				Resource:   svc.Name,
				Suggestion: "Avoid using privileged mode if possible. Use specific capabilities (cap_add) instead.",
				Timestamp:  time.Now(),
			}
		}

	case "SEC-002": // Root User
		if svc.User == "root" || svc.User == "0" {
			return &Violation{
				PolicyID:   p.ID,
				PolicyName: p.Name,
				Severity:   p.Severity,
				Message:    "Service runs as root user",
				Resource:   svc.Name,
				Suggestion: "Configure a non-root user in 'user' field or Dockerfile",
				Timestamp:  time.Now(),
			}
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
