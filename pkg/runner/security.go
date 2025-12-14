package runner

import (
	"fmt"
	"os"
	"strings"

	"github.com/container-make/cm/pkg/config"
)

// SecurityWarning represents a security concern
type SecurityWarning struct {
	Level       string // "critical", "warning", "info"
	Title       string
	Description string
	Suggestion  string
}

// SecurityChecker analyzes configuration for security issues
type SecurityChecker struct {
	config   *config.DevContainerConfig
	warnings []SecurityWarning
}

// NewSecurityChecker creates a new security checker
func NewSecurityChecker(cfg *config.DevContainerConfig) *SecurityChecker {
	return &SecurityChecker{
		config:   cfg,
		warnings: []SecurityWarning{},
	}
}

// Check performs security analysis and returns warnings
func (s *SecurityChecker) Check() []SecurityWarning {
	s.warnings = []SecurityWarning{}

	s.checkDockerSocket()
	s.checkPrivilegedMode()
	s.checkCapabilities()
	s.checkSecurityOpts()
	s.checkMounts()

	return s.warnings
}

// checkDockerSocket checks if docker.sock is mounted
func (s *SecurityChecker) checkDockerSocket() {
	if s.config == nil {
		return
	}

	// Check mounts
	for _, mount := range s.config.Mounts {
		if containsDockerSocket(mount) {
			s.warnings = append(s.warnings, SecurityWarning{
				Level:       "critical",
				Title:       "Docker Socket Mounted",
				Description: "Mounting /var/run/docker.sock gives the container FULL ROOT ACCESS to the host system.",
				Suggestion:  "Consider using Docker-in-Docker (DinD) with --privileged or Rootless Docker instead.",
			})
			return
		}
	}

	// Check runArgs
	for i, arg := range s.config.RunArgs {
		if (arg == "-v" || arg == "--volume") && i+1 < len(s.config.RunArgs) {
			if containsDockerSocket(s.config.RunArgs[i+1]) {
				s.warnings = append(s.warnings, SecurityWarning{
					Level:       "critical",
					Title:       "Docker Socket in RunArgs",
					Description: "Docker socket mount detected in runArgs. This is a security risk.",
					Suggestion:  "Remove docker.sock mount or use Rootless Docker.",
				})
				return
			}
		}
	}
}

// checkPrivilegedMode checks for privileged container
func (s *SecurityChecker) checkPrivilegedMode() {
	if s.config == nil {
		return
	}

	for _, arg := range s.config.RunArgs {
		if arg == "--privileged" {
			s.warnings = append(s.warnings, SecurityWarning{
				Level:       "warning",
				Title:       "Privileged Mode Enabled",
				Description: "Container will run with full host privileges. This disables most security features.",
				Suggestion:  "Only use --privileged if absolutely necessary (e.g., for Docker-in-Docker).",
			})
			return
		}
	}
}

// checkCapabilities checks for dangerous capability additions
func (s *SecurityChecker) checkCapabilities() {
	if s.config == nil {
		return
	}

	dangerouscaps := map[string]string{
		"CAP_SYS_ADMIN":  "Allows container escape and host filesystem access",
		"CAP_NET_ADMIN":  "Allows network configuration changes",
		"CAP_SYS_PTRACE": "Required for debugging but can be used for container escape",
	}

	for i, arg := range s.config.RunArgs {
		if arg == "--cap-add" && i+1 < len(s.config.RunArgs) {
			cap := s.config.RunArgs[i+1]
			if desc, ok := dangerouscaps[cap]; ok {
				s.warnings = append(s.warnings, SecurityWarning{
					Level:       "warning",
					Title:       fmt.Sprintf("Capability Added: %s", cap),
					Description: desc,
					Suggestion:  "Only add this capability if required by your development workflow.",
				})
			}
		}
	}
}

// checkSecurityOpts checks security options
func (s *SecurityChecker) checkSecurityOpts() {
	if s.config == nil {
		return
	}

	for i, arg := range s.config.RunArgs {
		if arg == "--security-opt" && i+1 < len(s.config.RunArgs) {
			opt := s.config.RunArgs[i+1]
			if strings.Contains(opt, "seccomp=unconfined") {
				s.warnings = append(s.warnings, SecurityWarning{
					Level:       "warning",
					Title:       "Seccomp Disabled",
					Description: "Seccomp filtering is disabled. This allows all system calls.",
					Suggestion:  "Use a custom seccomp profile instead of disabling entirely.",
				})
			}
			if strings.Contains(opt, "apparmor=unconfined") {
				s.warnings = append(s.warnings, SecurityWarning{
					Level:       "warning",
					Title:       "AppArmor Disabled",
					Description: "AppArmor protection is disabled.",
					Suggestion:  "Consider using a custom AppArmor profile.",
				})
			}
		}
	}
}

// checkMounts checks for dangerous mount paths
func (s *SecurityChecker) checkMounts() {
	if s.config == nil {
		return
	}

	dangerousPaths := []string{
		"/etc/passwd",
		"/etc/shadow",
		"/etc/sudoers",
		"/root",
		"/proc",
		"/sys",
	}

	for _, mount := range s.config.Mounts {
		for _, dangerous := range dangerousPaths {
			if strings.Contains(mount, dangerous) {
				s.warnings = append(s.warnings, SecurityWarning{
					Level:       "warning",
					Title:       fmt.Sprintf("Sensitive Path Mounted: %s", dangerous),
					Description: "Mounting sensitive host paths can expose system configuration.",
					Suggestion:  "Review if this mount is necessary for your workflow.",
				})
			}
		}
	}
}

// containsDockerSocket checks if a mount string contains docker socket
func containsDockerSocket(mount string) bool {
	dockerSockets := []string{
		"/var/run/docker.sock",
		"/run/docker.sock",
		"docker.sock",
		"/var/run/podman/podman.sock",
	}

	for _, sock := range dockerSockets {
		if strings.Contains(mount, sock) {
			return true
		}
	}
	return false
}

// FormatWarnings formats warnings for display
func (s *SecurityChecker) FormatWarnings() string {
	if len(s.warnings) == 0 {
		return ""
	}

	var sb strings.Builder

	// Count by level
	critical := 0
	warning := 0
	for _, w := range s.warnings {
		if w.Level == "critical" {
			critical++
		} else if w.Level == "warning" {
			warning++
		}
	}

	sb.WriteString("\n")
	if critical > 0 {
		sb.WriteString("⛔ SECURITY WARNINGS ⛔\n")
	} else {
		sb.WriteString("⚠️  Security Notices ⚠️\n")
	}
	sb.WriteString(strings.Repeat("─", 50) + "\n")

	for _, w := range s.warnings {
		icon := "ℹ️"
		if w.Level == "critical" {
			icon = "🔴"
		} else if w.Level == "warning" {
			icon = "🟡"
		}

		sb.WriteString(fmt.Sprintf("%s %s\n", icon, w.Title))
		sb.WriteString(fmt.Sprintf("   %s\n", w.Description))
		sb.WriteString(fmt.Sprintf("   💡 %s\n\n", w.Suggestion))
	}

	return sb.String()
}

// CheckAndWarn performs security check and prints warnings
func CheckAndWarn(cfg *config.DevContainerConfig) {
	checker := NewSecurityChecker(cfg)
	warnings := checker.Check()

	if len(warnings) > 0 {
		fmt.Fprint(os.Stderr, checker.FormatWarnings())

		// For critical warnings, ask for confirmation
		for _, w := range warnings {
			if w.Level == "critical" {
				fmt.Fprint(os.Stderr, "Continue anyway? [y/N] ")
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					fmt.Println("Aborted.")
					os.Exit(1)
				}
				break
			}
		}
	}
}

// IsRootlessDocker detects if Docker is running in rootless mode
func IsRootlessDocker() bool {
	// Check DOCKER_HOST for rootless socket
	dockerHost := os.Getenv("DOCKER_HOST")
	if strings.Contains(dockerHost, "rootless") {
		return true
	}

	// Check for rootless socket paths
	home := os.Getenv("HOME")
	if home != "" {
		rootlessSocket := fmt.Sprintf("%s/.docker/run/docker.sock", home)
		if _, err := os.Stat(rootlessSocket); err == nil {
			return true
		}
	}

	// Check XDG_RUNTIME_DIR
	xdgRuntime := os.Getenv("XDG_RUNTIME_DIR")
	if xdgRuntime != "" {
		rootlessSocket := fmt.Sprintf("%s/docker.sock", xdgRuntime)
		if _, err := os.Stat(rootlessSocket); err == nil {
			return true
		}
	}

	return false
}
