// Package policy provides Policy as Code capabilities for Container-Maker.
// It allows defining and enforcing security, compliance, and best practice rules.
package policy

import (
	"context"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/workspace"
)

// SeverityLevel defines the severity of a policy violation
type SeverityLevel string

const (
	SeverityInfo     SeverityLevel = "info"
	SeverityWarning  SeverityLevel = "warning"
	SeverityError    SeverityLevel = "error"
	SeverityCritical SeverityLevel = "critical"
)

// PolicyType defines the type of policy
type PolicyType string

const (
	PolicyTypeSecurity     PolicyType = "security"
	PolicyTypeBestPractice PolicyType = "best_practice"
	PolicyTypeResource     PolicyType = "resource"
	PolicyTypeLicense      PolicyType = "license"
	PolicyTypeCustom       PolicyType = "custom"
)

// Policy represents a single policy rule
type Policy struct {
	ID          string        `json:"id" yaml:"id"`
	Name        string        `json:"name" yaml:"name"`
	Description string        `json:"description" yaml:"description"`
	Type        PolicyType    `json:"type" yaml:"type"`
	Severity    SeverityLevel `json:"severity" yaml:"severity"`
	Enabled     bool          `json:"enabled" yaml:"enabled"`

	// Rule logic (rego or internal)
	Rule       string                 `json:"rule,omitempty" yaml:"rule,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// Violation represents a policy violation
type Violation struct {
	PolicyID   string        `json:"policy_id"`
	PolicyName string        `json:"policy_name"`
	Severity   SeverityLevel `json:"severity"`
	Message    string        `json:"message"`
	Resource   string        `json:"resource"`           // e.g., service name
	Location   string        `json:"location,omitempty"` // file:line or field path
	Suggestion string        `json:"suggestion,omitempty"`
	Timestamp  time.Time     `json:"timestamp"`
}

// EvaluationResult represents the result of a policy evaluation
type EvaluationResult struct {
	Passed      bool          `json:"passed"`
	Score       int           `json:"score"` // 0-100 security score
	Violations  []Violation   `json:"violations"`
	EvaluatedAt time.Time     `json:"evaluated_at"`
	Duration    time.Duration `json:"duration"`
	PolicyCount int           `json:"policy_count"`
}

// Engine defines the policy enforcement engine interface
type Engine interface {
	// LoadPolicies loads policies from a directory or file
	LoadPolicies(path string) error

	// EvaluateWorkspace evaluates a workspace against loaded policies
	EvaluateWorkspace(ctx context.Context, ws *workspace.Workspace) (*EvaluationResult, error)

	// EvaluateService evaluates a single service
	EvaluateService(ctx context.Context, svc *workspace.Service) (*EvaluationResult, error)

	// GetPolicies returns loaded policies
	GetPolicies() []Policy
}

// DefaultPolicies returns a set of built-in default policies
func DefaultPolicies() []Policy {
	return []Policy{
		{
			ID:          "SEC-001",
			Name:        "No Privileged Containers",
			Description: "Containers should not run in privileged mode",
			Type:        PolicyTypeSecurity,
			Severity:    SeverityCritical,
			Enabled:     true,
		},
		{
			ID:          "SEC-002",
			Name:        "Root User Check",
			Description: "Services should mostly not run as root",
			Type:        PolicyTypeSecurity,
			Severity:    SeverityWarning,
			Enabled:     true,
		},
		{
			ID:          "RES-001",
			Name:        "Memory Limits",
			Description: "Services must have memory limits defined",
			Type:        PolicyTypeResource,
			Severity:    SeverityWarning,
			Enabled:     true,
		},
		{
			ID:          "BP-001",
			Name:        "Healthcheck Defined",
			Description: "Services should have healthchecks",
			Type:        PolicyTypeBestPractice,
			Severity:    SeverityInfo,
			Enabled:     true,
		},
		{
			ID:          "BP-002",
			Name:        "Version Pinning",
			Description: "Images should use specific tags, not 'latest'",
			Type:        PolicyTypeBestPractice,
			Severity:    SeverityWarning,
			Enabled:     true,
		},
	}
}
