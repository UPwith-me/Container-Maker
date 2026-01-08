package team

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/userconfig"
)

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	Timestamp  time.Time `json:"ts"`
	Action     string    `json:"action"`
	Template   string    `json:"template,omitempty"`
	Repository string    `json:"repo,omitempty"`
	User       string    `json:"user,omitempty"`
	Project    string    `json:"project,omitempty"`
	Result     string    `json:"result,omitempty"`
	DurationMs int64     `json:"duration_ms,omitempty"`
	Details    string    `json:"details,omitempty"`
}

// Actions
const (
	ActionTemplateUse  = "template_use"
	ActionTeamSync     = "team_sync"
	ActionTeamAdd      = "team_add"
	ActionTeamRemove   = "team_remove"
	ActionVersionPin   = "version_pin"
	ActionVersionUnpin = "version_unpin"
)

// getAuditLogPath returns the path to the audit log file
func getAuditLogPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cm", "audit.jsonl"), nil
}

// LogAudit writes an audit entry to the log file
func LogAudit(entry AuditEntry) error {
	// Check if audit is enabled
	cfg, err := userconfig.Load()
	if err != nil || !cfg.Team.AuditLog {
		return nil // Silently skip if not enabled
	}

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Get username
	if entry.User == "" {
		entry.User = os.Getenv("USER")
		if entry.User == "" {
			entry.User = os.Getenv("USERNAME")
		}
	}

	logPath, err := getAuditLogPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return err
	}

	// Append to log file
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	_, err = f.WriteString(string(data) + "\n")
	return err
}

// LogTemplateUse logs template usage
func LogTemplateUse(template, repo, project string) {
	_ = LogAudit(AuditEntry{
		Action:     ActionTemplateUse,
		Template:   template,
		Repository: repo,
		Project:    project,
		Result:     "success",
	})
}

// LogSync logs sync operations
func LogSync(repo, result string, durationMs int64) {
	_ = LogAudit(AuditEntry{
		Action:     ActionTeamSync,
		Repository: repo,
		Result:     result,
		DurationMs: durationMs,
	})
}

// PinVersion pins a repository to a specific tag/branch
func PinVersion(repoName, version string) error {
	cfg, err := userconfig.Load()
	if err != nil {
		return err
	}

	found := false
	for i := range cfg.Team.Repositories {
		if cfg.Team.Repositories[i].Name == repoName {
			cfg.Team.Repositories[i].Tag = version
			cfg.Team.Repositories[i].AutoUpdate = false // Disable auto-update when pinned
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("repository '%s' not found", repoName)
	}

	if err := userconfig.Save(cfg); err != nil {
		return err
	}

	_ = LogAudit(AuditEntry{
		Action:     ActionVersionPin,
		Repository: repoName,
		Details:    version,
		Result:     "success",
	})

	return nil
}

// UnpinVersion removes version lock from a repository
func UnpinVersion(repoName string) error {
	cfg, err := userconfig.Load()
	if err != nil {
		return err
	}

	found := false
	for i := range cfg.Team.Repositories {
		if cfg.Team.Repositories[i].Name == repoName {
			cfg.Team.Repositories[i].Tag = ""
			cfg.Team.Repositories[i].AutoUpdate = true
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("repository '%s' not found", repoName)
	}

	if err := userconfig.Save(cfg); err != nil {
		return err
	}

	_ = LogAudit(AuditEntry{
		Action:     ActionVersionUnpin,
		Repository: repoName,
		Result:     "success",
	})

	return nil
}

// GetAuditLog returns recent audit entries
func GetAuditLog(limit int) ([]AuditEntry, error) {
	logPath, err := getAuditLogPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []AuditEntry{}, nil
		}
		return nil, err
	}

	lines := splitLines(string(data))

	// Take last N entries
	start := 0
	if len(lines) > limit {
		start = len(lines) - limit
	}

	var entries []AuditEntry
	for i := start; i < len(lines); i++ {
		if lines[i] == "" {
			continue
		}
		var entry AuditEntry
		if err := json.Unmarshal([]byte(lines[i]), &entry); err == nil {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

func splitLines(s string) []string {
	var lines []string
	current := ""
	for _, c := range s {
		if c == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
