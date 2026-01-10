// Package sync provides file synchronization for remote development
package sync

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// SyncConfig holds configuration for file synchronization
type SyncConfig struct {
	LocalPath       string        // Local directory to sync
	RemoteHost      string        // SSH host (user@host)
	RemotePath      string        // Remote directory path
	ExcludePatterns []string      // Patterns to exclude (e.g., .git, node_modules)
	SyncInterval    time.Duration // Interval for periodic sync (0 = watch mode only)
}

// Syncer handles bidirectional file synchronization
type Syncer struct {
	config    SyncConfig
	watcher   *fsnotify.Watcher
	stopChan  chan struct{}
	mu        sync.Mutex
	running   bool
	lastSync  time.Time
	syncQueue chan string // Queue of files to sync
}

// DefaultExcludes returns common patterns to exclude from sync
func DefaultExcludes() []string {
	return []string{
		".git",
		".idea",
		".vscode",
		"node_modules",
		"__pycache__",
		".pytest_cache",
		"*.pyc",
		".DS_Store",
		"Thumbs.db",
		"*.swp",
		"*.swo",
		"vendor",
		"target",
		"bin",
		"obj",
		".cm-state.json",
	}
}

// New creates a new Syncer with the given configuration
func New(cfg SyncConfig) (*Syncer, error) {
	if cfg.LocalPath == "" || cfg.RemoteHost == "" || cfg.RemotePath == "" {
		return nil, fmt.Errorf("LocalPath, RemoteHost, and RemotePath are required")
	}

	// Use defaults if not specified
	if len(cfg.ExcludePatterns) == 0 {
		cfg.ExcludePatterns = DefaultExcludes()
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	return &Syncer{
		config:    cfg,
		watcher:   watcher,
		stopChan:  make(chan struct{}),
		syncQueue: make(chan string, 100),
	}, nil
}

// Start begins file synchronization (blocking)
func (s *Syncer) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("syncer already running")
	}
	s.running = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	// Initial full sync
	fmt.Println("🔄 Performing initial sync...")
	if err := s.SyncToRemote(); err != nil {
		return fmt.Errorf("initial sync failed: %w", err)
	}
	fmt.Println("✅ Initial sync complete")

	// Add directory to watcher
	if err := s.addWatchRecursive(s.config.LocalPath); err != nil {
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	fmt.Printf("👀 Watching %s for changes...\n", s.config.LocalPath)
	fmt.Println("   Press Ctrl+C to stop")

	// Start debounced sync worker
	go s.syncWorker(ctx)

	// Watch for file changes
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-s.stopChan:
			return nil
		case event, ok := <-s.watcher.Events:
			if !ok {
				return nil
			}
			// Filter and queue events
			if s.shouldSync(event.Name) {
				s.queueSync(event.Name)
			}
		case err, ok := <-s.watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Printf("⚠️  Watch error: %v\n", err)
		}
	}
}

// Stop stops the file synchronization
func (s *Syncer) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		close(s.stopChan)
		s.watcher.Close()
	}
}

// SyncToRemote performs a one-way sync from local to remote
func (s *Syncer) SyncToRemote() error {
	return s.rsync(s.config.LocalPath, fmt.Sprintf("%s:%s", s.config.RemoteHost, s.config.RemotePath))
}

// SyncFromRemote performs a one-way sync from remote to local
func (s *Syncer) SyncFromRemote() error {
	return s.rsync(fmt.Sprintf("%s:%s", s.config.RemoteHost, s.config.RemotePath), s.config.LocalPath)
}

// rsync executes rsync with appropriate flags
func (s *Syncer) rsync(src, dst string) error {
	args := []string{
		"-avz",       // Archive, verbose, compress
		"--delete",   // Delete extraneous files from dest
		"--progress", // Show progress
		"-e", "ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null",
	}

	// Add exclude patterns
	for _, pattern := range s.config.ExcludePatterns {
		args = append(args, "--exclude", pattern)
	}

	// Ensure trailing slash for directory sync
	if !strings.HasSuffix(src, "/") && !strings.Contains(src, ":") {
		src = src + "/"
	}

	args = append(args, src, dst)

	// Check if rsync is available
	rsyncPath := "rsync"
	useScp := false

	if runtime.GOOS == "windows" {
		// On Windows, try to find rsync in common locations
		if _, err := exec.LookPath("rsync"); err != nil {
			// Try Git Bash's rsync
			gitBashRsync := filepath.Join(os.Getenv("ProgramFiles"), "Git", "usr", "bin", "rsync.exe")
			if _, err := os.Stat(gitBashRsync); err == nil {
				rsyncPath = gitBashRsync
			} else {
				// Fallback to scp if available
				if _, err := exec.LookPath("scp"); err == nil {
					useScp = true
					fmt.Println("⚠️  rsync not found, using scp fallback (slower, excludes ignored)")
				} else {
					return fmt.Errorf("rsync not found. Install Git for Windows or WSL")
				}
			}
		}
	} else {
		if _, err := exec.LookPath("rsync"); err != nil {
			if _, err := exec.LookPath("scp"); err == nil {
				useScp = true
				fmt.Println("⚠️  rsync not found, using scp fallback (slower, excludes ignored)")
			} else {
				return fmt.Errorf("rsync not found")
			}
		}
	}

	var cmd *exec.Cmd
	if useScp {
		// SCP doesn't support exclusions or advanced progress, but works for basic sync
		scpArgs := []string{
			"-r",
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
			src, dst,
		}
		cmd = exec.Command("scp", scpArgs...)
	} else {
		cmd = exec.Command(rsyncPath, args...)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// addWatchRecursive adds a directory and all subdirectories to the watcher
func (s *Syncer) addWatchRecursive(path string) error {
	return filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip excluded directories
		if info.IsDir() {
			for _, exclude := range s.config.ExcludePatterns {
				if matched, _ := filepath.Match(exclude, info.Name()); matched {
					return filepath.SkipDir
				}
			}
			return s.watcher.Add(walkPath)
		}
		return nil
	})
}

// shouldSync checks if a file should be synced based on exclude patterns
func (s *Syncer) shouldSync(path string) bool {
	name := filepath.Base(path)
	for _, pattern := range s.config.ExcludePatterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return false
		}
		// Check if any parent directory matches
		if strings.Contains(path, string(filepath.Separator)+pattern+string(filepath.Separator)) {
			return false
		}
	}
	return true
}

// queueSync adds a file to the sync queue with debouncing
func (s *Syncer) queueSync(path string) {
	select {
	case s.syncQueue <- path:
	default:
		// Queue full, will be caught by next sync
	}
}

// syncWorker processes the sync queue with debouncing
func (s *Syncer) syncWorker(ctx context.Context) {
	var pendingSync bool
	debounceTimer := time.NewTimer(0)
	<-debounceTimer.C // Drain initial timer

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-s.syncQueue:
			// Reset debounce timer
			pendingSync = true
			debounceTimer.Reset(300 * time.Millisecond)
		case <-debounceTimer.C:
			if pendingSync {
				fmt.Printf("\n🔄 Syncing changes...\n")
				if err := s.SyncToRemote(); err != nil {
					fmt.Printf("❌ Sync failed: %v\n", err)
				} else {
					s.lastSync = time.Now()
					fmt.Printf("✅ Synced at %s\n", s.lastSync.Format("15:04:05"))
				}
				pendingSync = false
			}
		}
	}
}

// Status returns the current sync status
func (s *Syncer) Status() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return "Stopped"
	}

	if s.lastSync.IsZero() {
		return "Running (no sync yet)"
	}

	return fmt.Sprintf("Running (last sync: %s)", s.lastSync.Format("15:04:05"))
}
