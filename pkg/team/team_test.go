package team

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/userconfig"
)

func TestDetectAuthType(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"SSH URL", "git@github.com:user/repo.git", AuthTypeSSH},
		{"SSH Protocol", "ssh://git@github.com/user/repo.git", AuthTypeSSH},
		{"HTTPS URL", "https://github.com/user/repo.git", AuthTypeToken},
		{"HTTP URL", "http://github.com/user/repo.git", AuthTypeToken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectAuthType(tt.url)
			if result != tt.expected {
				t.Errorf("DetectAuthType(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestInjectTokenAuth(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		token    string
		expected string
	}{
		{"Empty token", "https://github.com/user/repo.git", "", "https://github.com/user/repo.git"},
		{"With token", "https://github.com/user/repo.git", "mytoken", "https://mytoken@github.com/user/repo.git"},
		{"SSH URL ignores token", "git@github.com:user/repo.git", "mytoken", "git@github.com:user/repo.git"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InjectTokenAuth(tt.url, tt.token)
			if result != tt.expected {
				t.Errorf("InjectTokenAuth(%q, %q) = %q, want %q", tt.url, tt.token, result, tt.expected)
			}
		})
	}
}

func TestIsCacheValid(t *testing.T) {
	tests := []struct {
		name         string
		lastSyncTime int64
		cacheTTL     int
		expected     bool
	}{
		{"Never synced", 0, 24, false},
		{"Recently synced", time.Now().Unix(), 24, true},
		{"Expired cache", time.Now().Add(-48 * time.Hour).Unix(), 24, false},
		{"Custom TTL valid", time.Now().Add(-12 * time.Hour).Unix(), 24, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &userconfig.TeamRepository{
				LastSyncTime: tt.lastSyncTime,
				CacheTTL:     tt.cacheTTL,
			}
			result := IsCacheValid(repo)
			if result != tt.expected {
				t.Errorf("IsCacheValid() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetCacheDir(t *testing.T) {
	dir, err := GetCacheDir()
	if err != nil {
		t.Fatalf("GetCacheDir() error = %v", err)
	}

	if dir == "" {
		t.Error("GetCacheDir() returned empty string")
	}

	if !filepath.IsAbs(dir) {
		t.Errorf("GetCacheDir() returned relative path: %s", dir)
	}
}

func TestGetRepoCacheDir(t *testing.T) {
	dir, err := GetRepoCacheDir("test-repo")
	if err != nil {
		t.Fatalf("GetRepoCacheDir() error = %v", err)
	}

	if !filepath.IsAbs(dir) {
		t.Errorf("GetRepoCacheDir() returned relative path: %s", dir)
	}

	if filepath.Base(dir) != "test-repo" {
		t.Errorf("GetRepoCacheDir() expected 'test-repo' in path, got: %s", dir)
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"Empty", "", []string{}},
		{"Single line", "hello", []string{"hello"}},
		{"Multiple lines", "line1\nline2\nline3", []string{"line1", "line2", "line3"}},
		{"Trailing newline", "line1\nline2\n", []string{"line1", "line2"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitLines(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("splitLines(%q) length = %d, want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("splitLines(%q)[%d] = %q, want %q", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestLogAudit_Disabled(t *testing.T) {
	// When audit is disabled, LogAudit should silently succeed
	entry := AuditEntry{
		Action:   ActionTemplateUse,
		Template: "test-template",
	}

	// Should not error even if audit is disabled
	err := LogAudit(entry)
	if err != nil {
		t.Errorf("LogAudit() with disabled audit = %v, want nil", err)
	}
}

func TestGetAuditLog_NoFile(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	entries, err := GetAuditLog(10)
	if err != nil {
		t.Errorf("GetAuditLog() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("GetAuditLog() returned %d entries, want 0", len(entries))
	}
}
