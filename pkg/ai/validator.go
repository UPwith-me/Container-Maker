package ai

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ValidationError represents a validation error
type ValidationError struct {
	Type         string `json:"type"`     // "syntax", "schema", "security", "image"
	Severity     string `json:"severity"` // "error", "warning", "info"
	Field        string `json:"field,omitempty"`
	Message      string `json:"message"`
	Line         int    `json:"line,omitempty"`
	SuggestedFix string `json:"suggestedFix,omitempty"`
}

// ValidationResult holds all validation results
type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []ValidationError `json:"warnings,omitempty"`
	Info     []ValidationError `json:"info,omitempty"`
}

// Validator validates devcontainer.json configurations
type Validator struct {
	strictMode bool
}

// NewValidator creates a new validator
func NewValidator(strict bool) *Validator {
	return &Validator{strictMode: strict}
}

// Validate performs comprehensive validation on a config string
func (v *Validator) Validate(configJSON string) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// 1. JSON syntax validation
	syntaxErrors := v.validateSyntax(configJSON)
	for _, e := range syntaxErrors {
		if e.Severity == "error" {
			result.Valid = false
			result.Errors = append(result.Errors, e)
		}
	}

	// If syntax is invalid, don't continue
	if !result.Valid {
		return result
	}

	// Parse the config
	var config map[string]interface{}
	json.Unmarshal([]byte(configJSON), &config)

	// 2. Schema validation
	schemaErrors := v.validateSchema(config)
	for _, e := range schemaErrors {
		if e.Severity == "error" {
			result.Valid = false
			result.Errors = append(result.Errors, e)
		} else if e.Severity == "warning" {
			result.Warnings = append(result.Warnings, e)
		} else {
			result.Info = append(result.Info, e)
		}
	}

	// 3. Security checks
	securityIssues := v.checkSecurity(config)
	for _, e := range securityIssues {
		if e.Severity == "error" && v.strictMode {
			result.Valid = false
			result.Errors = append(result.Errors, e)
		} else if e.Severity == "warning" || e.Severity == "error" {
			result.Warnings = append(result.Warnings, e)
		} else {
			result.Info = append(result.Info, e)
		}
	}

	// 4. Best practices
	bestPractices := v.checkBestPractices(config)
	result.Info = append(result.Info, bestPractices...)

	return result
}

// validateSyntax checks JSON syntax
func (v *Validator) validateSyntax(configJSON string) []ValidationError {
	var errors []ValidationError

	// Try to parse as JSON
	var js json.RawMessage
	if err := json.Unmarshal([]byte(configJSON), &js); err != nil {
		// Try to extract line number from error
		line := 0
		if syntaxErr, ok := err.(*json.SyntaxError); ok {
			// Calculate line number from offset
			lines := strings.Split(configJSON[:syntaxErr.Offset], "\n")
			line = len(lines)
		}

		errors = append(errors, ValidationError{
			Type:     "syntax",
			Severity: "error",
			Message:  fmt.Sprintf("Invalid JSON: %v", err),
			Line:     line,
		})
	}

	return errors
}

// validateSchema checks against devcontainer.json schema
func (v *Validator) validateSchema(config map[string]interface{}) []ValidationError {
	var errors []ValidationError

	// Check required fields
	if _, hasImage := config["image"]; !hasImage {
		if _, hasBuild := config["build"]; !hasBuild {
			if _, hasDockerFile := config["dockerFile"]; !hasDockerFile {
				if _, hasDockerCompose := config["dockerComposeFile"]; !hasDockerCompose {
					errors = append(errors, ValidationError{
						Type:         "schema",
						Severity:     "error",
						Field:        "image",
						Message:      "Must specify 'image', 'build', 'dockerFile', or 'dockerComposeFile'",
						SuggestedFix: `Add "image": "mcr.microsoft.com/devcontainers/base:ubuntu"`,
					})
				}
			}
		}
	}

	// Validate image format
	if image, ok := config["image"].(string); ok {
		if !isValidImageName(image) {
			errors = append(errors, ValidationError{
				Type:     "schema",
				Severity: "warning",
				Field:    "image",
				Message:  fmt.Sprintf("Image name '%s' may not be valid", image),
			})
		}
	}

	// Validate forwardPorts
	if ports, ok := config["forwardPorts"].([]interface{}); ok {
		for i, port := range ports {
			switch p := port.(type) {
			case float64:
				if p < 1 || p > 65535 {
					errors = append(errors, ValidationError{
						Type:     "schema",
						Severity: "error",
						Field:    fmt.Sprintf("forwardPorts[%d]", i),
						Message:  fmt.Sprintf("Port %v is out of valid range (1-65535)", p),
					})
				}
			case string:
				// Validate port mapping format (e.g., "8080:8080")
				if !isValidPortMapping(p) {
					errors = append(errors, ValidationError{
						Type:     "schema",
						Severity: "error",
						Field:    fmt.Sprintf("forwardPorts[%d]", i),
						Message:  fmt.Sprintf("Invalid port mapping: %s", p),
					})
				}
			}
		}
	}

	// Validate runArgs
	if runArgs, ok := config["runArgs"].([]interface{}); ok {
		for i, arg := range runArgs {
			if argStr, ok := arg.(string); ok {
				if strings.HasPrefix(argStr, "--rm") {
					errors = append(errors, ValidationError{
						Type:     "schema",
						Severity: "warning",
						Field:    fmt.Sprintf("runArgs[%d]", i),
						Message:  "--rm flag may cause unexpected behavior with persistent containers",
					})
				}
			}
		}
	}

	// Validate features format
	if features, ok := config["features"].(map[string]interface{}); ok {
		for featureName := range features {
			if !strings.Contains(featureName, "/") && !strings.HasPrefix(featureName, ".") {
				errors = append(errors, ValidationError{
					Type:     "schema",
					Severity: "warning",
					Field:    "features",
					Message:  fmt.Sprintf("Feature '%s' should be a full OCI reference (e.g., ghcr.io/devcontainers/features/go:1)", featureName),
				})
			}
		}
	}

	return errors
}

// checkSecurity checks for security issues
func (v *Validator) checkSecurity(config map[string]interface{}) []ValidationError {
	var errors []ValidationError

	// Check for privileged mode
	if runArgs, ok := config["runArgs"].([]interface{}); ok {
		for _, arg := range runArgs {
			if argStr, ok := arg.(string); ok {
				if argStr == "--privileged" {
					errors = append(errors, ValidationError{
						Type:         "security",
						Severity:     "error",
						Field:        "runArgs",
						Message:      "SEC-001: Privileged mode is a security risk",
						SuggestedFix: "Remove --privileged and use specific capabilities with --cap-add instead",
					})
				}
			}
		}
	}

	// Check for Docker socket mount
	if mounts, ok := config["mounts"].([]interface{}); ok {
		for _, mount := range mounts {
			mountStr := ""
			switch m := mount.(type) {
			case string:
				mountStr = m
			case map[string]interface{}:
				if source, ok := m["source"].(string); ok {
					mountStr = source
				}
			}

			if strings.Contains(mountStr, "docker.sock") {
				errors = append(errors, ValidationError{
					Type:         "security",
					Severity:     "warning",
					Field:        "mounts",
					Message:      "SEC-002: Docker socket mount allows container escape",
					SuggestedFix: "Use Docker-in-Docker feature instead: ghcr.io/devcontainers/features/docker-in-docker:2",
				})
			}

			if strings.Contains(mountStr, "/etc/shadow") || strings.Contains(mountStr, "/etc/passwd") {
				errors = append(errors, ValidationError{
					Type:     "security",
					Severity: "error",
					Field:    "mounts",
					Message:  "SEC-003: Mounting sensitive system files is dangerous",
				})
			}
		}
	}

	// Check for root user
	if remoteUser, ok := config["remoteUser"].(string); ok {
		if remoteUser == "root" {
			errors = append(errors, ValidationError{
				Type:         "security",
				Severity:     "warning",
				Field:        "remoteUser",
				Message:      "SEC-004: Running as root is not recommended",
				SuggestedFix: `Set "remoteUser": "vscode" or create a non-root user`,
			})
		}
	}

	// Check containerEnv for sensitive data
	if containerEnv, ok := config["containerEnv"].(map[string]interface{}); ok {
		sensitivePatterns := []string{"PASSWORD", "SECRET", "TOKEN", "KEY", "CREDENTIAL"}
		for key, value := range containerEnv {
			for _, pattern := range sensitivePatterns {
				if strings.Contains(strings.ToUpper(key), pattern) {
					if valueStr, ok := value.(string); ok && len(valueStr) > 0 {
						errors = append(errors, ValidationError{
							Type:     "security",
							Severity: "warning",
							Field:    fmt.Sprintf("containerEnv.%s", key),
							Message:  fmt.Sprintf("SEC-005: Sensitive data in '%s' - consider using secrets management", key),
						})
					}
				}
			}
		}
	}

	return errors
}

// checkBestPractices checks for best practices
func (v *Validator) checkBestPractices(config map[string]interface{}) []ValidationError {
	var info []ValidationError

	// Check for name
	if _, hasName := config["name"]; !hasName {
		info = append(info, ValidationError{
			Type:     "best-practice",
			Severity: "info",
			Field:    "name",
			Message:  "BP-001: Consider adding a 'name' field for better identification",
		})
	}

	// Check for postCreateCommand
	if _, hasPostCreate := config["postCreateCommand"]; !hasPostCreate {
		info = append(info, ValidationError{
			Type:     "best-practice",
			Severity: "info",
			Field:    "postCreateCommand",
			Message:  "BP-002: Consider adding postCreateCommand to install dependencies",
		})
	}

	// Check for VS Code extensions in customizations
	if customizations, ok := config["customizations"].(map[string]interface{}); ok {
		if vscode, ok := customizations["vscode"].(map[string]interface{}); ok {
			if extensions, ok := vscode["extensions"].([]interface{}); ok {
				if len(extensions) == 0 {
					info = append(info, ValidationError{
						Type:     "best-practice",
						Severity: "info",
						Field:    "customizations.vscode.extensions",
						Message:  "BP-003: Consider adding VS Code extensions for better development experience",
					})
				}
			}
		}
	} else {
		info = append(info, ValidationError{
			Type:     "best-practice",
			Severity: "info",
			Field:    "customizations",
			Message:  "BP-003: Consider adding VS Code customizations (extensions, settings)",
		})
	}

	// Check for resource limits
	hasResourceLimits := false
	if runArgs, ok := config["runArgs"].([]interface{}); ok {
		for _, arg := range runArgs {
			if argStr, ok := arg.(string); ok {
				if strings.Contains(argStr, "--memory") || strings.Contains(argStr, "--cpus") {
					hasResourceLimits = true
					break
				}
			}
		}
	}
	if !hasResourceLimits {
		info = append(info, ValidationError{
			Type:     "best-practice",
			Severity: "info",
			Field:    "runArgs",
			Message:  "BP-004: Consider adding resource limits (--memory, --cpus) for predictable behavior",
		})
	}

	return info
}

// isValidImageName checks if an image name is valid
func isValidImageName(image string) bool {
	// Basic validation - not empty and has reasonable format
	if image == "" {
		return false
	}

	// Check for common patterns
	validPatterns := []string{
		`^[a-z0-9]+$`,                           // Simple name like "ubuntu"
		`^[a-z0-9-_]+/[a-z0-9-_]+$`,             // user/repo
		`^[a-z0-9.-]+/[a-z0-9-_]+/[a-z0-9-_]+$`, // registry/user/repo
		`^[a-z0-9.-]+:[0-9]+/`,                  // registry:port/...
		`^mcr\.microsoft\.com/`,                 // Microsoft container registry
		`^ghcr\.io/`,                            // GitHub container registry
		`^gcr\.io/`,                             // Google container registry
	}

	for _, pattern := range validPatterns {
		if matched, _ := regexp.MatchString(pattern, strings.Split(image, ":")[0]); matched {
			return true
		}
	}

	return true // Be permissive
}

// isValidPortMapping validates a port mapping string
func isValidPortMapping(mapping string) bool {
	// Format: "host:container" or "host:container/protocol"
	parts := strings.Split(mapping, ":")
	if len(parts) != 2 {
		return false
	}

	// Validate each part is a number (or number/protocol)
	for _, part := range parts {
		port := strings.Split(part, "/")[0]
		var portNum int
		if _, err := fmt.Sscanf(port, "%d", &portNum); err != nil {
			return false
		}
		if portNum < 1 || portNum > 65535 {
			return false
		}
	}

	return true
}

// FormatValidationResult formats the result for display
func FormatValidationResult(result *ValidationResult) string {
	var sb strings.Builder

	if result.Valid {
		sb.WriteString("✅ Configuration is valid\n")
	} else {
		sb.WriteString("❌ Configuration has errors\n")
	}
	sb.WriteString("\n")

	if len(result.Errors) > 0 {
		sb.WriteString("🔴 Errors:\n")
		for _, e := range result.Errors {
			sb.WriteString(fmt.Sprintf("   • [%s] %s\n", e.Type, e.Message))
			if e.SuggestedFix != "" {
				sb.WriteString(fmt.Sprintf("     💡 Fix: %s\n", e.SuggestedFix))
			}
		}
		sb.WriteString("\n")
	}

	if len(result.Warnings) > 0 {
		sb.WriteString("🟡 Warnings:\n")
		for _, e := range result.Warnings {
			sb.WriteString(fmt.Sprintf("   • [%s] %s\n", e.Type, e.Message))
			if e.SuggestedFix != "" {
				sb.WriteString(fmt.Sprintf("     💡 Fix: %s\n", e.SuggestedFix))
			}
		}
		sb.WriteString("\n")
	}

	if len(result.Info) > 0 {
		sb.WriteString("ℹ️  Suggestions:\n")
		for _, e := range result.Info {
			sb.WriteString(fmt.Sprintf("   • %s\n", e.Message))
		}
	}

	return sb.String()
}
