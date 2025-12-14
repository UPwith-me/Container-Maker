package watch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/config"
	"github.com/UPwith-me/Container-Maker/pkg/runner"
	"github.com/fsnotify/fsnotify"
)

// Options configures the watcher behavior
type Options struct {
	Extensions []string      // File extensions to watch (empty = all)
	IgnoreDirs []string      // Directories to ignore
	Delay      time.Duration // Debounce delay
	Clear      bool          // Clear screen before each run
	InitialRun bool          // Run command on startup
	ProjectDir string        // Project directory
	Config     *config.DevContainerConfig
}

// DefaultOptions returns default watch options
func DefaultOptions() Options {
	return Options{
		Extensions: []string{}, // All extensions
		IgnoreDirs: []string{".git", "vendor", "node_modules", ".devcontainer", "__pycache__", ".cm"},
		Delay:      300 * time.Millisecond,
		Clear:      false,
		InitialRun: true,
	}
}

// Watcher monitors files and runs commands on changes
type Watcher struct {
	opts    Options
	command []string
	watcher *fsnotify.Watcher
	runner  *runner.PersistentRunner
	mu      sync.Mutex
	lastRun time.Time
	pending bool
}

// New creates a new file watcher
func New(opts Options, command []string) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	pr, err := runner.NewPersistentRunner(opts.Config, opts.ProjectDir)
	if err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to create runner: %w", err)
	}

	return &Watcher{
		opts:    opts,
		command: command,
		watcher: watcher,
		runner:  pr,
	}, nil
}

// Start begins watching for file changes
func (w *Watcher) Start(ctx context.Context) error {
	// Add directories recursively
	if err := w.addWatchPaths(w.opts.ProjectDir); err != nil {
		return err
	}

	// Print startup info
	w.printStartup()

	// Initial run if enabled
	if w.opts.InitialRun {
		fmt.Println("✓ Initial run:")
		w.runCommand(ctx)
	}

	fmt.Println()
	fmt.Println("⏳ Waiting for changes... (Ctrl+C to stop)")
	fmt.Println()

	// Event loop
	debounce := time.NewTimer(w.opts.Delay)
	debounce.Stop()
	var changedFiles []string

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}

			// Skip if shouldn't watch
			if !w.shouldWatch(event.Name) {
				continue
			}

			// Only care about write/create/remove events
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) != 0 {
				changedFiles = append(changedFiles, filepath.Base(event.Name))
				debounce.Reset(w.opts.Delay)
			}

		case <-debounce.C:
			if len(changedFiles) > 0 {
				// Print changed files
				if len(changedFiles) <= 3 {
					fmt.Printf("📝 Changed: %s\n", strings.Join(changedFiles, ", "))
				} else {
					fmt.Printf("📝 Changed: %s... (%d files)\n",
						strings.Join(changedFiles[:2], ", "), len(changedFiles))
				}
				changedFiles = nil

				// Clear screen if enabled
				if w.opts.Clear {
					fmt.Print("\033[H\033[2J")
				}

				// Run command
				fmt.Printf("🔄 Re-running: %s\n\n", strings.Join(w.command, " "))
				w.runCommand(ctx)
				fmt.Println()
				fmt.Println("⏳ Waiting for changes...")
				fmt.Println()
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Printf("⚠️  Watch error: %v\n", err)
		}
	}
}

// addWatchPaths adds all directories to watch recursively
func (w *Watcher) addWatchPaths(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if !info.IsDir() {
			return nil
		}

		// Skip ignored directories
		name := info.Name()
		for _, ignore := range w.opts.IgnoreDirs {
			if name == ignore {
				return filepath.SkipDir
			}
		}

		// Add directory to watcher
		if err := w.watcher.Add(path); err != nil {
			// Don't fail on individual directory errors
			return nil
		}

		return nil
	})
}

// shouldWatch checks if a file should be watched based on options
func (w *Watcher) shouldWatch(path string) bool {
	// Check ignored directories
	for _, ignore := range w.opts.IgnoreDirs {
		if strings.Contains(path, string(os.PathSeparator)+ignore+string(os.PathSeparator)) ||
			strings.HasSuffix(path, string(os.PathSeparator)+ignore) {
			return false
		}
	}

	// Check extensions if specified
	if len(w.opts.Extensions) > 0 {
		ext := strings.TrimPrefix(filepath.Ext(path), ".")
		for _, e := range w.opts.Extensions {
			if strings.EqualFold(ext, e) {
				return true
			}
		}
		return false
	}

	return true
}

// runCommand executes the command in the container
func (w *Watcher) runCommand(ctx context.Context) {
	if err := w.runner.Exec(ctx, w.command); err != nil {
		fmt.Printf("\n❌ Command failed: %v\n", err)
	}
}

// printStartup prints startup information
func (w *Watcher) printStartup() {
	fmt.Println("📡 Watching for changes...")
	fmt.Printf("   Directory: %s\n", w.opts.ProjectDir)

	if len(w.opts.Extensions) > 0 {
		fmt.Printf("   Extensions: %s\n", strings.Join(w.opts.Extensions, ", "))
	} else {
		fmt.Println("   Extensions: * (all files)")
	}

	fmt.Printf("   Command: %s\n", strings.Join(w.command, " "))
	fmt.Println()
}

// Close cleans up the watcher
func (w *Watcher) Close() error {
	return w.watcher.Close()
}
