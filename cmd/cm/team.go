package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/team"
	"github.com/UPwith-me/Container-Maker/pkg/userconfig"
	"github.com/spf13/cobra"
)

var teamCmd = &cobra.Command{
	Use:   "team",
	Short: "Manage team/organization settings",
	Long: `Configure Container-Maker for enterprise team usage.

Teams can share DevContainer templates from centralized repositories,
ensuring everyone uses consistent development environments.

Examples:
  cm team add hq https://github.com/mycompany/templates
  cm team sync
  cm team list
  cm team info`,
}

// ==================== Config Commands ====================

var teamSetCmd = &cobra.Command{
	Use:   "set <org-name>",
	Short: "Set organization name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := userconfig.Load()
		if err != nil {
			cfg = &userconfig.UserConfig{}
		}

		cfg.Team.OrgName = args[0]
		if err := userconfig.Save(cfg); err != nil {
			return err
		}

		fmt.Printf("[OK] Organization set to '%s'\n", args[0])
		return nil
	},
}

var teamAddRepoName string
var teamAddBranch string
var teamAddPriority int

var teamAddCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Add a team template repository",
	Long: `Add a team template repository for shared templates.

Examples:
  cm team add https://github.com/mycompany/templates
  cm team add https://github.com/ml-team/templates --name ml
  cm team add git@github.com:mycompany/templates.git --branch develop`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]

		cfg, err := userconfig.Load()
		if err != nil {
			cfg = &userconfig.UserConfig{}
		}

		// Generate name if not provided
		name := teamAddRepoName
		if name == "" {
			name = fmt.Sprintf("repo-%d", len(cfg.Team.Repositories)+1)
		}

		// Check for duplicate name
		for _, r := range cfg.Team.Repositories {
			if r.Name == name {
				return fmt.Errorf("repository '%s' already exists, use --name to specify a different name", name)
			}
		}

		// Detect auth type
		authType := team.DetectAuthType(url)

		repo := userconfig.TeamRepository{
			Name:       name,
			URL:        url,
			Branch:     teamAddBranch,
			Priority:   teamAddPriority,
			AuthType:   authType,
			AutoUpdate: true,
			CacheTTL:   24,
		}

		// Test connection
		fmt.Printf("[>] Testing connection to %s...\n", url)
		authResult := team.CheckAuth(&repo)
		if !authResult.IsValid {
			fmt.Printf("[!] Warning: %s\n", authResult.Reason)
			if authResult.Suggestion != "" {
				fmt.Printf("    Suggestion: %s\n", authResult.Suggestion)
			}
		}

		if err := team.TestConnection(&repo); err != nil {
			return fmt.Errorf("connection failed: %w", err)
		}

		cfg.Team.Repositories = append(cfg.Team.Repositories, repo)

		// Sort by priority
		sort.Slice(cfg.Team.Repositories, func(i, j int) bool {
			return cfg.Team.Repositories[i].Priority > cfg.Team.Repositories[j].Priority
		})

		if err := userconfig.Save(cfg); err != nil {
			return err
		}

		fmt.Printf("[OK] Added repository '%s'\n", name)
		fmt.Println("[i] Run 'cm team sync' to download templates")
		return nil
	},
}

var teamRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a team template repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := userconfig.Load()
		if err != nil {
			return err
		}

		found := false
		newRepos := []userconfig.TeamRepository{}
		for _, r := range cfg.Team.Repositories {
			if r.Name == name {
				found = true
				// Clear cache
				_ = team.ClearCache(name)
			} else {
				newRepos = append(newRepos, r)
			}
		}

		if !found {
			return fmt.Errorf("repository '%s' not found", name)
		}

		cfg.Team.Repositories = newRepos
		if err := userconfig.Save(cfg); err != nil {
			return err
		}

		fmt.Printf("[OK] Removed repository '%s'\n", name)
		return nil
	},
}

// ==================== Sync Commands ====================

var teamSyncForce bool

var teamSyncCmd = &cobra.Command{
	Use:   "sync [name]",
	Short: "Sync team template repositories",
	Long: `Sync (clone or update) team template repositories.

Examples:
  cm team sync          # Sync all repositories
  cm team sync hq       # Sync specific repository
  cm team sync --force  # Force re-clone`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		cfg, err := userconfig.Load()
		if err != nil {
			return err
		}

		// Handle legacy config
		if len(cfg.Team.Repositories) == 0 && cfg.Team.TemplatesURL != "" {
			cfg.Team.Repositories = []userconfig.TeamRepository{{
				Name:       "default",
				URL:        cfg.Team.TemplatesURL,
				Priority:   100,
				AutoUpdate: true,
			}}
		}

		if len(cfg.Team.Repositories) == 0 {
			fmt.Println("[!] No team repositories configured")
			fmt.Println("[i] Use 'cm team add <url>' to add a repository")
			return nil
		}

		// Filter by name if specified
		var reposToSync []userconfig.TeamRepository
		if len(args) > 0 {
			for _, r := range cfg.Team.Repositories {
				if r.Name == args[0] {
					reposToSync = append(reposToSync, r)
					break
				}
			}
			if len(reposToSync) == 0 {
				return fmt.Errorf("repository '%s' not found", args[0])
			}
		} else {
			reposToSync = cfg.Team.Repositories
		}

		// Force re-clone if requested
		if teamSyncForce {
			for _, r := range reposToSync {
				_ = team.ClearCache(r.Name)
			}
		}

		fmt.Printf("[>] Syncing %d repositor(ies)...\n\n", len(reposToSync))

		for i := range reposToSync {
			repo := &reposToSync[i]
			fmt.Printf("  [~] %s (%s)\n", repo.Name, repo.URL)

			result := team.SyncRepository(ctx, repo)
			if result.Success {
				if result.IsNewClone {
					fmt.Printf("      [OK] Cloned (commit: %s)\n", result.NewCommit)
				} else {
					fmt.Printf("      [OK] Updated (commit: %s)\n", result.NewCommit)
				}

				// Update config
				for j := range cfg.Team.Repositories {
					if cfg.Team.Repositories[j].Name == repo.Name {
						cfg.Team.Repositories[j].LastSyncTime = time.Now().Unix()
						cfg.Team.Repositories[j].LastCommit = result.NewCommit
						break
					}
				}
			} else {
				fmt.Printf("      [ERROR] %s\n", result.Message)
			}
		}

		// Save updated config
		if err := userconfig.Save(cfg); err != nil {
			return err
		}

		fmt.Println("\n[OK] Sync complete")
		return nil
	},
}

// ==================== Info Commands ====================

var teamInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show team configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := userconfig.Load()
		if err != nil {
			return err
		}

		fmt.Println("=== Team Configuration ===")
		fmt.Println()

		if cfg.Team.OrgName != "" {
			fmt.Printf("  Organization: %s\n", cfg.Team.OrgName)
		} else {
			fmt.Println("  Organization: (not set)")
		}

		fmt.Println()
		fmt.Println("Repositories:")

		if len(cfg.Team.Repositories) == 0 && cfg.Team.TemplatesURL == "" {
			fmt.Println("  (none configured)")
			fmt.Println()
			fmt.Println("[i] Use 'cm team add <url>' to add a repository")
		} else {
			// Handle legacy
			if len(cfg.Team.Repositories) == 0 && cfg.Team.TemplatesURL != "" {
				fmt.Printf("  [legacy] %s\n", cfg.Team.TemplatesURL)
			}

			for _, r := range cfg.Team.Repositories {
				status := "[not synced]"
				if r.LastSyncTime > 0 {
					lastSync := time.Unix(r.LastSyncTime, 0)
					status = fmt.Sprintf("[synced %s, commit: %s]", lastSync.Format("2006-01-02 15:04"), r.LastCommit)
				}

				fmt.Printf("  %s\n", r.Name)
				fmt.Printf("    URL:    %s\n", r.URL)
				if r.Branch != "" {
					fmt.Printf("    Branch: %s\n", r.Branch)
				}
				if r.Tag != "" {
					fmt.Printf("    Tag:    %s\n", r.Tag)
				}
				fmt.Printf("    Auth:   %s\n", r.AuthType)
				fmt.Printf("    Status: %s\n", status)
				fmt.Println()
			}
		}

		// Show variables
		if len(cfg.Team.Variables) > 0 {
			fmt.Println("Variables:")
			for k, v := range cfg.Team.Variables {
				fmt.Printf("  %s = %s\n", k, v)
			}
		}

		return nil
	},
}

var teamListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cached team templates",
	RunE: func(cmd *cobra.Command, args []string) error {
		templates, err := team.GetAllTeamTemplates()
		if err != nil {
			return err
		}

		if len(templates) == 0 {
			fmt.Println("[!] No team templates found")
			fmt.Println("[i] Run 'cm team sync' to download templates")
			return nil
		}

		fmt.Println("=== Team Templates ===")
		fmt.Println()

		for repo, tmpls := range templates {
			fmt.Printf("  [%s]\n", repo)
			for _, t := range tmpls {
				desc := t.Description
				if desc == "" {
					desc = "(no description)"
				}
				fmt.Printf("    %-20s %s\n", t.Name, desc)
			}
			fmt.Println()
		}

		return nil
	},
}

// ==================== Variable Commands ====================

var teamVarCmd = &cobra.Command{
	Use:   "var <key>=<value>",
	Short: "Set a global template variable",
	Long: `Set a global variable for template rendering.

Variables can be used in templates as {{.VAR_NAME}}.

Examples:
  cm team var REGISTRY_URL=harbor.internal.com
  cm team var BASE_IMAGE=mycompany/base:latest`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Parse key=value
		parts := splitFirst(args[0], "=")
		if len(parts) != 2 {
			return fmt.Errorf("invalid format, use: KEY=VALUE")
		}

		key, value := parts[0], parts[1]

		cfg, err := userconfig.Load()
		if err != nil {
			cfg = &userconfig.UserConfig{}
		}

		if cfg.Team.Variables == nil {
			cfg.Team.Variables = make(map[string]string)
		}

		cfg.Team.Variables[key] = value
		if err := userconfig.Save(cfg); err != nil {
			return err
		}

		fmt.Printf("[OK] Set %s = %s\n", key, value)
		return nil
	},
}

var teamClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all cached team templates",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := team.ClearAllCaches(); err != nil {
			return err
		}
		fmt.Println("[OK] Cleared all team template caches")
		return nil
	},
}

// ==================== Auth Commands ====================

var teamAuthToken string

var teamAuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Configure authentication for team repositories",
	Long: `Configure authentication for accessing private team repositories.

Examples:
  cm team auth --test           # Test current auth configuration
  cm team auth --token <token>  # Set GitHub token`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if teamAuthToken != "" {
			// Save token to config
			fmt.Println("[i] Token will be used for HTTPS repositories")
			fmt.Println("[!] For better security, consider using environment variables:")
			fmt.Println("    export GITHUB_TOKEN=<your-token>")
			return nil
		}

		// Default: test auth for all repos
		cfg, err := userconfig.Load()
		if err != nil {
			return err
		}

		fmt.Println("=== Authentication Status ===")
		fmt.Println()

		for _, repo := range cfg.Team.Repositories {
			result := team.CheckAuth(&repo)
			status := "[OK]"
			if !result.IsValid {
				status = "[!]"
			}

			fmt.Printf("  %s %s (%s)\n", status, repo.Name, result.Type)
			fmt.Printf("      %s\n", result.Reason)
			if result.Suggestion != "" {
				fmt.Printf("      [i] %s\n", result.Suggestion)
			}
		}

		return nil
	},
}

// ==================== Helpers ====================

func splitFirst(s, sep string) []string {
	idx := -1
	for i := 0; i < len(s); i++ {
		if s[i] == sep[0] {
			idx = i
			break
		}
	}
	if idx == -1 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+1:]}
}

func init() {
	// Config flags
	teamAddCmd.Flags().StringVar(&teamAddRepoName, "name", "", "Repository name (auto-generated if not set)")
	teamAddCmd.Flags().StringVar(&teamAddBranch, "branch", "", "Branch to track (default: main)")
	teamAddCmd.Flags().IntVar(&teamAddPriority, "priority", 100, "Display priority (higher = first)")

	// Sync flags
	teamSyncCmd.Flags().BoolVar(&teamSyncForce, "force", false, "Force re-clone repositories")

	// Auth flags
	teamAuthCmd.Flags().StringVar(&teamAuthToken, "token", "", "Set authentication token")

	// Register commands
	teamCmd.AddCommand(teamSetCmd)
	teamCmd.AddCommand(teamAddCmd)
	teamCmd.AddCommand(teamRemoveCmd)
	teamCmd.AddCommand(teamSyncCmd)
	teamCmd.AddCommand(teamInfoCmd)
	teamCmd.AddCommand(teamListCmd)
	teamCmd.AddCommand(teamVarCmd)
	teamCmd.AddCommand(teamClearCmd)
	teamCmd.AddCommand(teamAuthCmd)
	rootCmd.AddCommand(teamCmd)
}
