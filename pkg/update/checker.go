package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/userconfig"
)

const (
	updateCheckInterval = 24 * time.Hour
	githubAPIURL        = "https://api.github.com/repos/UPwith-me/Container-Maker/releases/latest"
)

type Release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

// CheckForUpdates performs a non-blocking check for new versions.
// It respects the leaky bucket rate limiting (once per 24h).
func CheckForUpdates(currentVersion string) {
	// 1. Load config
	cfg, err := userconfig.Load()
	if err != nil {
		return // Fail silently
	}

	// 2. Check rate limit
	lastCheck := time.Unix(cfg.LastUpdateCheck, 0)
	if time.Since(lastCheck) < updateCheckInterval {
		return
	}

	// 3. Update timestamp immediately to prevent spamming if logic fails later
	if err := userconfig.UpdateLastCheck(time.Now().Unix()); err != nil {
		// Log error but continue
	}

	// 4. Run check in background (goroutine) to not block CLI
	go func() {
		rel, err := fetchLatestRelease()
		if err != nil {
			return
		}

		// 5. Compare versions
		// Naive comparison: if tag != v + currentVersion
		// In production, use semver library
		remoteVer := strings.TrimPrefix(rel.TagName, "v")
		currentVer := strings.TrimPrefix(currentVersion, "v")

		if remoteVer != currentVer && remoteVer != "" {
			// Notify user (print to Stderr to avoid messing up pipes)
			fmt.Fprintf(os.Stderr, "\n📦 New version available: %s -> %s\n", currentVersion, rel.TagName)
			fmt.Fprintf(os.Stderr, "   Run 'cm update' or visit: %s\n", rel.HTMLURL)
		}
	}()
}

func fetchLatestRelease() (*Release, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(githubAPIURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}
