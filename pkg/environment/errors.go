package environment

import (
	"fmt"
)

// Error types for environment management
var (
	ErrEnvironmentNotFound   = &EnvironmentError{Code: "ENV_NOT_FOUND", Message: "environment not found"}
	ErrEnvironmentExists     = &EnvironmentError{Code: "ENV_EXISTS", Message: "environment already exists"}
	ErrEnvironmentRunning    = &EnvironmentError{Code: "ENV_RUNNING", Message: "environment is running"}
	ErrEnvironmentStopped    = &EnvironmentError{Code: "ENV_STOPPED", Message: "environment is stopped"}
	ErrContainerNotFound     = &EnvironmentError{Code: "CONTAINER_NOT_FOUND", Message: "container not found"}
	ErrNetworkNotFound       = &EnvironmentError{Code: "NETWORK_NOT_FOUND", Message: "network not found"}
	ErrNetworkInUse          = &EnvironmentError{Code: "NETWORK_IN_USE", Message: "network is in use"}
	ErrInvalidName           = &EnvironmentError{Code: "INVALID_NAME", Message: "invalid environment name"}
	ErrInvalidConfig         = &EnvironmentError{Code: "INVALID_CONFIG", Message: "invalid configuration"}
	ErrDockerNotAvailable    = &EnvironmentError{Code: "DOCKER_UNAVAILABLE", Message: "Docker is not available"}
	ErrGPUNotAvailable       = &EnvironmentError{Code: "GPU_UNAVAILABLE", Message: "requested GPU is not available"}
	ErrInsufficientResources = &EnvironmentError{Code: "INSUFFICIENT_RESOURCES", Message: "insufficient resources"}
	ErrLinkExists            = &EnvironmentError{Code: "LINK_EXISTS", Message: "environments are already linked"}
	ErrLinkNotFound          = &EnvironmentError{Code: "LINK_NOT_FOUND", Message: "environments are not linked"}
	ErrSelfLink              = &EnvironmentError{Code: "SELF_LINK", Message: "cannot link environment to itself"}
	ErrStateCorrupted        = &EnvironmentError{Code: "STATE_CORRUPTED", Message: "environment state is corrupted"}
	ErrOperationTimeout      = &EnvironmentError{Code: "OPERATION_TIMEOUT", Message: "operation timed out"}
)

// EnvironmentError represents an environment-specific error
type EnvironmentError struct {
	Code       string // Machine-readable error code
	Message    string // Human-readable message
	Cause      error  // Underlying error
	EnvID      string // Related environment ID
	EnvName    string // Related environment name
	Suggestion string // Suggested fix
}

func (e *EnvironmentError) Error() string {
	if e.EnvName != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.EnvName, e.Message)
	}
	if e.EnvID != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.EnvID, e.Message)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *EnvironmentError) Unwrap() error {
	return e.Cause
}

// WithCause adds the underlying cause to the error
func (e *EnvironmentError) WithCause(cause error) *EnvironmentError {
	return &EnvironmentError{
		Code:       e.Code,
		Message:    e.Message,
		Cause:      cause,
		EnvID:      e.EnvID,
		EnvName:    e.EnvName,
		Suggestion: e.Suggestion,
	}
}

// WithEnv adds environment context to the error
func (e *EnvironmentError) WithEnv(id, name string) *EnvironmentError {
	return &EnvironmentError{
		Code:       e.Code,
		Message:    e.Message,
		Cause:      e.Cause,
		EnvID:      id,
		EnvName:    name,
		Suggestion: e.Suggestion,
	}
}

// WithSuggestion adds a fix suggestion to the error
func (e *EnvironmentError) WithSuggestion(suggestion string) *EnvironmentError {
	return &EnvironmentError{
		Code:       e.Code,
		Message:    e.Message,
		Cause:      e.Cause,
		EnvID:      e.EnvID,
		EnvName:    e.EnvName,
		Suggestion: suggestion,
	}
}

// Is implements errors.Is for error comparison
func (e *EnvironmentError) Is(target error) bool {
	if t, ok := target.(*EnvironmentError); ok {
		return e.Code == t.Code
	}
	return false
}

// NewError creates a new environment error with custom message
func NewError(code, message string) *EnvironmentError {
	return &EnvironmentError{
		Code:    code,
		Message: message,
	}
}

// WrapError wraps an error with environment context
func WrapError(err error, code, message string) *EnvironmentError {
	return &EnvironmentError{
		Code:    code,
		Message: message,
		Cause:   err,
	}
}

// FormatUserError formats an error for user display with actionable suggestions
func FormatUserError(err error) string {
	if envErr, ok := err.(*EnvironmentError); ok {
		result := fmt.Sprintf("Error: %s\n", envErr.Message)

		if envErr.EnvName != "" {
			result += fmt.Sprintf("Environment: %s\n", envErr.EnvName)
		}

		if envErr.Cause != nil {
			result += fmt.Sprintf("Details: %v\n", envErr.Cause)
		}

		if envErr.Suggestion != "" {
			result += fmt.Sprintf("\nSuggestion: %s\n", envErr.Suggestion)
		} else {
			// Default suggestions based on error code
			switch envErr.Code {
			case "ENV_NOT_FOUND":
				result += "\nSuggestion: Run 'cm env list' to see available environments\n"
			case "ENV_EXISTS":
				result += "\nSuggestion: Use 'cm env delete <name>' to remove the existing environment, or use --force\n"
			case "DOCKER_UNAVAILABLE":
				result += "\nSuggestion: Run 'cm doctor' to diagnose Docker issues\n"
			case "GPU_UNAVAILABLE":
				result += "\nSuggestion: Run 'cm gpu list' to see available GPUs\n"
			case "INSUFFICIENT_RESOURCES":
				result += "\nSuggestion: Stop other environments with 'cm env stop' or reduce resource requests\n"
			}
		}

		return result
	}

	return fmt.Sprintf("Error: %v\n", err)
}
