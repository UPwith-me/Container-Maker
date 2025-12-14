package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/mount"
)

// CacheConfig defines cache volumes for a language/tool
type CacheConfig struct {
	Language      string
	VolumeName    string
	ContainerPath string
	EnvVar        string // Optional environment variable to set
}

// LanguageCaches defines cache configurations for supported languages
var LanguageCaches = map[string][]CacheConfig{
	"go": {
		{Language: "go", VolumeName: "cm-go-pkg", ContainerPath: "/go/pkg", EnvVar: ""},
		{Language: "go", VolumeName: "cm-go-build", ContainerPath: "/root/.cache/go-build", EnvVar: "GOCACHE=/root/.cache/go-build"},
	},
	"rust": {
		{Language: "rust", VolumeName: "cm-cargo-registry", ContainerPath: "/root/.cargo/registry", EnvVar: ""},
		{Language: "rust", VolumeName: "cm-cargo-git", ContainerPath: "/root/.cargo/git", EnvVar: ""},
		{Language: "rust", VolumeName: "cm-cargo-target", ContainerPath: "/target", EnvVar: "CARGO_TARGET_DIR=/target"},
	},
	"node": {
		{Language: "node", VolumeName: "cm-npm-cache", ContainerPath: "/root/.npm", EnvVar: ""},
		{Language: "node", VolumeName: "cm-yarn-cache", ContainerPath: "/root/.yarn", EnvVar: ""},
		{Language: "node", VolumeName: "cm-pnpm-cache", ContainerPath: "/root/.pnpm-store", EnvVar: ""},
	},
	"python": {
		{Language: "python", VolumeName: "cm-pip-cache", ContainerPath: "/root/.cache/pip", EnvVar: "PIP_CACHE_DIR=/root/.cache/pip"},
		{Language: "python", VolumeName: "cm-pipenv-cache", ContainerPath: "/root/.cache/pipenv", EnvVar: ""},
	},
	"cpp": {
		{Language: "cpp", VolumeName: "cm-ccache", ContainerPath: "/root/.ccache", EnvVar: "CCACHE_DIR=/root/.ccache"},
	},
	"java": {
		{Language: "java", VolumeName: "cm-maven-cache", ContainerPath: "/root/.m2", EnvVar: ""},
		{Language: "java", VolumeName: "cm-gradle-cache", ContainerPath: "/root/.gradle", EnvVar: ""},
	},
	"dotnet": {
		{Language: "dotnet", VolumeName: "cm-nuget-cache", ContainerPath: "/root/.nuget", EnvVar: ""},
	},
}

// CacheManager handles build cache volume management
type CacheManager struct {
	backend       string
	projectDir    string
	detectedLangs []string
}

// NewCacheManager creates a new cache manager
func NewCacheManager(backend, projectDir string) *CacheManager {
	return &CacheManager{
		backend:    backend,
		projectDir: projectDir,
	}
}

// DetectLanguages detects programming languages in the project
func (c *CacheManager) DetectLanguages() []string {
	if len(c.detectedLangs) > 0 {
		return c.detectedLangs
	}

	langs := []string{}

	// Check for language-specific files
	checks := map[string][]string{
		"go":     {"go.mod", "go.sum"},
		"rust":   {"Cargo.toml", "Cargo.lock"},
		"node":   {"package.json", "package-lock.json", "yarn.lock", "pnpm-lock.yaml"},
		"python": {"requirements.txt", "pyproject.toml", "Pipfile", "setup.py"},
		"cpp":    {"CMakeLists.txt", "Makefile", "*.cpp", "*.c", "*.h"},
		"java":   {"pom.xml", "build.gradle", "build.gradle.kts"},
		"dotnet": {"*.csproj", "*.sln", "*.fsproj"},
	}

	for lang, files := range checks {
		for _, pattern := range files {
			if strings.Contains(pattern, "*") {
				// Glob pattern
				matches, _ := filepath.Glob(filepath.Join(c.projectDir, pattern))
				if len(matches) > 0 {
					langs = append(langs, lang)
					break
				}
			} else {
				// Exact file
				if _, err := os.Stat(filepath.Join(c.projectDir, pattern)); err == nil {
					langs = append(langs, lang)
					break
				}
			}
		}
	}

	c.detectedLangs = langs
	return langs
}

// GetCacheMounts returns Docker mount configurations for detected languages
func (c *CacheManager) GetCacheMounts() []mount.Mount {
	langs := c.DetectLanguages()
	mounts := []mount.Mount{}

	for _, lang := range langs {
		if caches, ok := LanguageCaches[lang]; ok {
			for _, cache := range caches {
				mounts = append(mounts, mount.Mount{
					Type:   mount.TypeVolume,
					Source: cache.VolumeName,
					Target: cache.ContainerPath,
				})
			}
		}
	}

	return mounts
}

// GetCacheEnvVars returns environment variables for cache configuration
func (c *CacheManager) GetCacheEnvVars() []string {
	langs := c.DetectLanguages()
	envVars := []string{}

	for _, lang := range langs {
		if caches, ok := LanguageCaches[lang]; ok {
			for _, cache := range caches {
				if cache.EnvVar != "" {
					envVars = append(envVars, cache.EnvVar)
				}
			}
		}
	}

	return envVars
}

// EnsureCacheVolumes creates cache volumes if they don't exist
func (c *CacheManager) EnsureCacheVolumes(ctx context.Context) error {
	langs := c.DetectLanguages()

	for _, lang := range langs {
		if caches, ok := LanguageCaches[lang]; ok {
			for _, cache := range caches {
				// Create volume if it doesn't exist
				cmd := exec.CommandContext(ctx, c.backend, "volume", "create", cache.VolumeName)
				_ = cmd.Run() // Ignore errors - volume may already exist
			}
		}
	}

	return nil
}

// GetCacheStats returns statistics about cache volumes
func (c *CacheManager) GetCacheStats(ctx context.Context) string {
	var sb strings.Builder
	sb.WriteString("📦 Build Cache Status:\n")

	langs := c.DetectLanguages()
	if len(langs) == 0 {
		sb.WriteString("   No languages detected\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("   Detected: %s\n", strings.Join(langs, ", ")))

	for _, lang := range langs {
		if caches, ok := LanguageCaches[lang]; ok {
			for _, cache := range caches {
				// Check if volume exists
				cmd := exec.CommandContext(ctx, c.backend, "volume", "inspect", cache.VolumeName)
				if err := cmd.Run(); err == nil {
					sb.WriteString(fmt.Sprintf("   ✅ %s -> %s\n", cache.VolumeName, cache.ContainerPath))
				} else {
					sb.WriteString(fmt.Sprintf("   ⭕ %s (will be created)\n", cache.VolumeName))
				}
			}
		}
	}

	return sb.String()
}

// CleanCaches removes all cache volumes
func (c *CacheManager) CleanCaches(ctx context.Context) error {
	for _, caches := range LanguageCaches {
		for _, cache := range caches {
			cmd := exec.CommandContext(ctx, c.backend, "volume", "rm", "-f", cache.VolumeName)
			_ = cmd.Run() // Ignore errors
		}
	}
	return nil
}
