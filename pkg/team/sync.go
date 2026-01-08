package team

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/userconfig"
)

// SyncResult represents the result of a sync operation
type SyncResult struct {
	RepoName   string
	Success    bool
	Message    string
	NewCommit  string
	UpdatedAt  time.Time
	IsNewClone bool
}

// GetCacheDir returns the team templates cache directory
func GetCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cm", "team-cache"), nil
}

// GetRepoCacheDir returns the cache directory for a specific repository
func GetRepoCacheDir(repoName string) (string, error) {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, repoName), nil
}

// SyncRepository syncs a single team repository
func SyncRepository(ctx context.Context, repo *userconfig.TeamRepository) SyncResult {
	result := SyncResult{
		RepoName:  repo.Name,
		UpdatedAt: time.Now(),
	}

	// Get cache directory
	repoDir, err := GetRepoCacheDir(repo.Name)
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("Failed to get cache dir: %v", err)
		return result
	}

	// Check if repo already exists
	gitDir := filepath.Join(repoDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		// Clone new repository
		result.IsNewClone = true
		if err := cloneRepo(ctx, repo, repoDir); err != nil {
			result.Success = false
			result.Message = fmt.Sprintf("Clone failed: %v", err)
			return result
		}
		result.Message = "Repository cloned successfully"
	} else {
		// Pull updates
		if err := pullRepo(ctx, repo, repoDir); err != nil {
			result.Success = false
			result.Message = fmt.Sprintf("Pull failed: %v", err)
			return result
		}
		result.Message = "Repository updated"
	}

	// Get current commit
	result.NewCommit = getCurrentCommit(repoDir)
	result.Success = true
	return result
}

// cloneRepo clones a repository to the specified directory
func cloneRepo(ctx context.Context, repo *userconfig.TeamRepository, destDir string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(destDir), 0755); err != nil {
		return err
	}

	// Build clone command
	args := []string{"clone", "--depth", "1"}

	// Add branch/tag if specified
	if repo.Tag != "" {
		args = append(args, "--branch", repo.Tag)
	} else if repo.Branch != "" {
		args = append(args, "--branch", repo.Branch)
	}

	// Get URL (potentially with token)
	url := getAuthenticatedURL(repo)
	args = append(args, url, destDir)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// pullRepo pulls updates for an existing repository
func pullRepo(ctx context.Context, repo *userconfig.TeamRepository, repoDir string) error {
	// First, fetch
	fetchCmd := exec.CommandContext(ctx, "git", "fetch", "--all", "--prune")
	fetchCmd.Dir = repoDir
	fetchCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if err := fetchCmd.Run(); err != nil {
		return fmt.Errorf("fetch failed: %w", err)
	}

	// Determine target ref
	targetRef := "origin/main"
	if repo.Tag != "" {
		targetRef = repo.Tag
	} else if repo.Branch != "" {
		targetRef = "origin/" + repo.Branch
	}

	// Reset to target (handles diverged history)
	resetCmd := exec.CommandContext(ctx, "git", "reset", "--hard", targetRef)
	resetCmd.Dir = repoDir
	return resetCmd.Run()
}

// getCurrentCommit returns the current commit hash
func getCurrentCommit(repoDir string) string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// getAuthenticatedURL returns URL with auth if needed
func getAuthenticatedURL(repo *userconfig.TeamRepository) string {
	if repo.AuthType != AuthTypeToken {
		return repo.URL
	}

	// Try to get token from env var
	var token string
	if repo.TokenEnvVar != "" {
		token = os.Getenv(repo.TokenEnvVar)
	}
	if token == "" {
		// Try common env vars
		for _, envVar := range []string{"GITHUB_TOKEN", "GH_TOKEN", "GITLAB_TOKEN"} {
			if t := os.Getenv(envVar); t != "" {
				token = t
				break
			}
		}
	}

	return InjectTokenAuth(repo.URL, token)
}

// SyncAllRepositories syncs all configured team repositories
func SyncAllRepositories(ctx context.Context) ([]SyncResult, error) {
	cfg, err := userconfig.Load()
	if err != nil {
		return nil, err
	}

	// Handle legacy single-repo config
	if len(cfg.Team.Repositories) == 0 && cfg.Team.TemplatesURL != "" {
		cfg.Team.Repositories = []userconfig.TeamRepository{{
			Name:       "default",
			URL:        cfg.Team.TemplatesURL,
			Priority:   100,
			AutoUpdate: true,
		}}
	}

	var results []SyncResult
	for i := range cfg.Team.Repositories {
		repo := &cfg.Team.Repositories[i]
		result := SyncRepository(ctx, repo)
		results = append(results, result)

		// Update last sync time in config
		if result.Success {
			repo.LastSyncTime = time.Now().Unix()
			repo.LastCommit = result.NewCommit
		}
	}

	// Save updated config
	if err := userconfig.Save(cfg); err != nil {
		return results, fmt.Errorf("failed to save config: %w", err)
	}

	return results, nil
}

// IsCacheValid checks if cached repo is still valid based on TTL
func IsCacheValid(repo *userconfig.TeamRepository) bool {
	if repo.LastSyncTime == 0 {
		return false
	}

	ttlHours := repo.CacheTTL
	if ttlHours == 0 {
		ttlHours = 24 // Default 24 hours
	}

	lastSync := time.Unix(repo.LastSyncTime, 0)
	return time.Since(lastSync) < time.Duration(ttlHours)*time.Hour
}

// ClearCache removes cached repository
func ClearCache(repoName string) error {
	repoDir, err := GetRepoCacheDir(repoName)
	if err != nil {
		return err
	}
	return os.RemoveAll(repoDir)
}

// ClearAllCaches removes all cached repositories
func ClearAllCaches() error {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return err
	}
	return os.RemoveAll(cacheDir)
}
