package team

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/UPwith-me/Container-Maker/pkg/userconfig"
)

// AuthType constants
const (
	AuthTypeNone  = "none"
	AuthTypeSSH   = "ssh"
	AuthTypeToken = "token"
)

// AuthResult represents authentication check result
type AuthResult struct {
	Type       string
	IsValid    bool
	Reason     string
	Suggestion string
}

// DetectAuthType automatically detects the best auth method for a URL
func DetectAuthType(url string) string {
	if strings.HasPrefix(url, "git@") || strings.Contains(url, "ssh://") {
		return AuthTypeSSH
	}
	return AuthTypeToken
}

// CheckAuth verifies authentication is properly configured for a repository
func CheckAuth(repo *userconfig.TeamRepository) AuthResult {
	authType := repo.AuthType
	if authType == "" {
		authType = DetectAuthType(repo.URL)
	}

	switch authType {
	case AuthTypeSSH:
		return checkSSHAuth()
	case AuthTypeToken:
		return checkTokenAuth(repo.TokenEnvVar)
	default:
		return AuthResult{
			Type:    AuthTypeNone,
			IsValid: true,
			Reason:  "No authentication required",
		}
	}
}

// checkSSHAuth verifies SSH authentication is available
func checkSSHAuth() AuthResult {
	result := AuthResult{Type: AuthTypeSSH}

	// Check if SSH agent is running
	if os.Getenv("SSH_AUTH_SOCK") != "" {
		// Try to list keys
		cmd := exec.Command("ssh-add", "-l")
		output, err := cmd.Output()
		if err == nil && len(output) > 0 && !strings.Contains(string(output), "no identities") {
			result.IsValid = true
			result.Reason = "SSH agent available with keys loaded"
			return result
		}
	}

	// Check for SSH keys in ~/.ssh
	home, _ := os.UserHomeDir()
	sshDir := filepath.Join(home, ".ssh")
	keyFiles := []string{"id_rsa", "id_ed25519", "id_ecdsa"}

	for _, keyFile := range keyFiles {
		keyPath := filepath.Join(sshDir, keyFile)
		if _, err := os.Stat(keyPath); err == nil {
			result.IsValid = true
			result.Reason = fmt.Sprintf("SSH key found: %s", keyFile)
			return result
		}
	}

	result.IsValid = false
	result.Reason = "No SSH keys found and SSH agent not running"
	result.Suggestion = "Run: ssh-keygen -t ed25519 -C 'your-email@example.com' or use 'cm team auth --token'"
	return result
}

// checkTokenAuth verifies token authentication is configured
func checkTokenAuth(tokenEnvVar string) AuthResult {
	result := AuthResult{Type: AuthTypeToken}

	// Check custom token env var first
	if tokenEnvVar != "" {
		if token := os.Getenv(tokenEnvVar); token != "" {
			result.IsValid = true
			result.Reason = fmt.Sprintf("Token found in $%s", tokenEnvVar)
			return result
		}
	}

	// Check common token env vars
	commonVars := []string{"GITHUB_TOKEN", "GH_TOKEN", "GITLAB_TOKEN", "GIT_TOKEN"}
	for _, envVar := range commonVars {
		if token := os.Getenv(envVar); token != "" {
			result.IsValid = true
			result.Reason = fmt.Sprintf("Token found in $%s", envVar)
			return result
		}
	}

	// Check git credential helper
	cmd := exec.Command("git", "config", "--global", "credential.helper")
	output, err := cmd.Output()
	if err == nil && len(strings.TrimSpace(string(output))) > 0 {
		result.IsValid = true
		result.Reason = fmt.Sprintf("Git credential helper configured: %s", strings.TrimSpace(string(output)))
		return result
	}

	result.IsValid = false
	result.Reason = "No authentication token found"
	result.Suggestion = "Set GITHUB_TOKEN environment variable or run: cm team auth --token <your-token>"
	return result
}

// TestConnection tests if a repository can be accessed
func TestConnection(repo *userconfig.TeamRepository) error {
	// Use git ls-remote to test connection without cloning
	args := []string{"ls-remote", "--heads", repo.URL}

	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0") // Prevent password prompts

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("connection failed: %s\n%s", err, string(output))
	}

	return nil
}

// InjectTokenAuth creates a URL with embedded token for HTTPS repos
func InjectTokenAuth(url, token string) string {
	if token == "" || strings.HasPrefix(url, "git@") {
		return url
	}

	// Convert https://github.com/user/repo to https://token@github.com/user/repo
	if strings.HasPrefix(url, "https://") {
		return strings.Replace(url, "https://", fmt.Sprintf("https://%s@", token), 1)
	}

	return url
}
