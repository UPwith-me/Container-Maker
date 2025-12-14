package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/container-make/cm/pkg/userconfig"
	"github.com/spf13/cobra"
)

var cloudAPIURL = "https://api.container-maker.dev"

var cloudCmd = &cobra.Command{
	Use:   "cloud",
	Short: "Container-Maker Cloud Control Plane",
	Long: `Manage cloud-based development environments.

Container-Maker Cloud allows you to:
  • Provision GPU instances on-demand
  • Use 14+ cloud providers (AWS, GCP, Azure, DigitalOcean, and more)
  • Share environments with your team
  • Pay-as-you-go billing

Examples:
  cm cloud login                    # Authenticate
  cm cloud instances                # List running instances
  cm cloud create --type gpu-t4     # Create GPU instance
  cm cloud connect <id>             # SSH into instance
  cm cloud delete <id>              # Terminate instance`,
}

var cloudLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Container-Maker Cloud",
	Long: `Login to Container-Maker Cloud using one of these methods:
  • Interactive browser-based OAuth (default)
  • API key (--api-key)
  • Email/password (--email, --password)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, _ := cmd.Flags().GetString("api-key")

		if apiKey != "" {
			// API key auth
			return cloudLoginWithAPIKey(apiKey)
		}

		// Interactive OAuth
		return cloudLoginInteractive()
	},
}

func cloudLoginWithAPIKey(apiKey string) error {
	// Validate API key
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", cloudAPIURL+"/api/v1/user", nil)
	req.Header.Set("X-API-Key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to cloud: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid API key")
	}

	// Save API key
	cfg, _ := userconfig.Load()
	if cfg == nil {
		cfg = &userconfig.UserConfig{}
	}
	cfg.CloudAPIKey = apiKey
	cfg.CloudAPIURL = cloudAPIURL

	if err := userconfig.Save(cfg); err != nil {
		return err
	}

	fmt.Println("✅ Logged in successfully!")
	return nil
}

func cloudLoginInteractive() error {
	fmt.Println("🔐 Opening browser for authentication...")
	fmt.Println()

	// Generate state for OAuth
	authURL := cloudAPIURL + "/auth/github?cli=true"

	// Open browser based on OS
	var browserCmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		browserCmd = exec.Command("cmd", "/c", "start", authURL)
	case "darwin":
		browserCmd = exec.Command("open", authURL)
	default:
		browserCmd = exec.Command("xdg-open", authURL)
	}

	if err := browserCmd.Start(); err != nil {
		fmt.Printf("Please open this URL in your browser:\n%s\n", authURL)
	}

	fmt.Println("Waiting for authentication...")
	fmt.Println("(Press Ctrl+C to cancel)")

	// TODO: Implement device code flow or local callback server
	fmt.Println()
	fmt.Print("Enter API key or token: ")
	var token string
	fmt.Scanln(&token)

	if token == "" {
		return fmt.Errorf("authentication cancelled")
	}

	return cloudLoginWithAPIKey(token)
}

var cloudLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out from Container-Maker Cloud",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _ := userconfig.Load()
		if cfg != nil {
			cfg.CloudAPIKey = ""
			cfg.CloudToken = ""
			userconfig.Save(cfg)
		}
		fmt.Println("✅ Logged out successfully")
		return nil
	},
}

var cloudInstancesCmd = &cobra.Command{
	Use:   "instances",
	Short: "List running cloud instances",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getCloudClient()
		if err != nil {
			return err
		}

		resp, err := client.Get(cloudAPIURL + "/api/v1/instances")
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var instances []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&instances)

		if len(instances) == 0 {
			fmt.Println("No running instances.")
			fmt.Println()
			fmt.Println("Create one with: cm cloud create --type cpu-small")
			return nil
		}

		fmt.Println("☁️  Cloud Instances")
		fmt.Println()
		fmt.Printf("  %-12s %-15s %-10s %-8s %-15s %s\n", "ID", "Name", "Type", "Status", "Provider", "IP")
		fmt.Printf("  %-12s %-15s %-10s %-8s %-15s %s\n", "────────────", "───────────────", "──────────", "────────", "───────────────", "───────────────")

		for _, inst := range instances {
			fmt.Printf("  %-12s %-15s %-10s %-8s %-15s %s\n",
				inst["id"],
				inst["name"],
				inst["instance_type"],
				inst["status"],
				inst["provider"],
				inst["public_ip"],
			)
		}

		return nil
	},
}

var cloudCreateType string
var cloudCreateProvider string
var cloudCreateRegion string
var cloudCreateName string

var cloudCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new cloud instance",
	Long: `Create a new cloud development environment.

Instance Types:
  cpu-small   2 vCPU, 4GB RAM       ~$0.02/hr
  cpu-medium  4 vCPU, 8GB RAM       ~$0.04/hr
  cpu-large   8 vCPU, 16GB RAM      ~$0.08/hr
  gpu-t4      4 vCPU, 16GB + T4     ~$0.50/hr
  gpu-a10     8 vCPU, 32GB + A10    ~$1.50/hr
  gpu-a100    8 vCPU, 80GB + A100   ~$3.00/hr

Providers:
  aws, gcp, azure, digitalocean, linode, vultr, hetzner,
  oci, alibaba, tencent, lambdalabs, runpod, vast`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getCloudClient()
		if err != nil {
			return err
		}

		name := cloudCreateName
		if name == "" {
			cwd, _ := os.Getwd()
			name = filepath.Base(cwd)
		}

		body := map[string]interface{}{
			"name":          name,
			"instance_type": cloudCreateType,
			"provider":      cloudCreateProvider,
			"region":        cloudCreateRegion,
		}

		// Check for devcontainer.json
		if _, err := os.Stat(".devcontainer/devcontainer.json"); err == nil {
			data, _ := os.ReadFile(".devcontainer/devcontainer.json")
			body["devcontainer"] = string(data)
		}

		jsonBody, _ := json.Marshal(body)

		fmt.Printf("🚀 Creating %s instance on %s...\n", cloudCreateType, cloudCreateProvider)

		resp, err := client.Post(cloudAPIURL+"/api/v1/instances", "application/json", bytes.NewReader(jsonBody))
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to create instance: %s", string(body))
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		fmt.Printf("✅ Instance created: %s\n", result["id"])
		fmt.Println()
		fmt.Printf("Connect with: cm cloud connect %s\n", result["id"])

		return nil
	},
}

var cloudConnectCmd = &cobra.Command{
	Use:   "connect <instance-id>",
	Short: "SSH into a cloud instance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		instanceID := args[0]

		client, err := getCloudClient()
		if err != nil {
			return err
		}

		// Get SSH config
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/instances/%s/ssh", cloudAPIURL, instanceID))
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var sshConfig map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&sshConfig)

		host := sshConfig["host"].(string)
		port := int(sshConfig["port"].(float64))
		user := "root"
		if u, ok := sshConfig["user"].(string); ok {
			user = u
		}

		fmt.Printf("🔌 Connecting to %s@%s:%d...\n", user, host, port)

		sshCmd := exec.Command("ssh", "-p", fmt.Sprintf("%d", port), fmt.Sprintf("%s@%s", user, host))
		sshCmd.Stdin = os.Stdin
		sshCmd.Stdout = os.Stdout
		sshCmd.Stderr = os.Stderr

		return sshCmd.Run()
	},
}

var cloudStopCmd = &cobra.Command{
	Use:   "stop <instance-id>",
	Short: "Stop a cloud instance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		instanceID := args[0]
		client, err := getCloudClient()
		if err != nil {
			return err
		}

		resp, err := client.Post(fmt.Sprintf("%s/api/v1/instances/%s/stop", cloudAPIURL, instanceID), "", nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		fmt.Printf("✅ Instance %s stopped\n", instanceID)
		return nil
	},
}

var cloudDeleteCmd = &cobra.Command{
	Use:   "delete <instance-id>",
	Short: "Delete a cloud instance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		instanceID := args[0]
		client, err := getCloudClient()
		if err != nil {
			return err
		}

		req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/instances/%s", cloudAPIURL, instanceID), nil)
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		fmt.Printf("✅ Instance %s deleted\n", instanceID)
		return nil
	},
}

var cloudProvidersCmd = &cobra.Command{
	Use:   "providers",
	Short: "List available cloud providers",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getCloudClient()
		if err != nil {
			return err
		}

		resp, err := client.Get(cloudAPIURL + "/api/v1/providers")
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var providers []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&providers)

		fmt.Println("☁️  Available Cloud Providers")
		fmt.Println()
		fmt.Printf("  %-15s %-25s %s\n", "Name", "Display Name", "Status")
		fmt.Printf("  %-15s %-25s %s\n", "───────────────", "─────────────────────────", "────────")

		for _, p := range providers {
			fmt.Printf("  %-15s %-25s %s\n", p["name"], p["display_name"], p["status"])
		}

		return nil
	},
}

var cloudBillingCmd = &cobra.Command{
	Use:   "billing",
	Short: "View billing and usage",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getCloudClient()
		if err != nil {
			return err
		}

		resp, err := client.Get(cloudAPIURL + "/api/v1/billing/usage")
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var usage map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&usage)

		currentMonth := usage["current_month"].(map[string]interface{})

		fmt.Println("💰 Billing & Usage")
		fmt.Println()
		fmt.Println("  Current Month:")
		fmt.Printf("    CPU Hours:    %.1f\n", currentMonth["cpu_hours"])
		fmt.Printf("    GPU Hours:    %.1f\n", currentMonth["gpu_hours"])
		fmt.Printf("    Total Cost:   $%.2f\n", currentMonth["total_cost"])
		fmt.Printf("    Instances:    %.0f\n", currentMonth["instances"])

		return nil
	},
}

func getCloudClient() (*http.Client, error) {
	cfg, err := userconfig.Load()
	if err != nil || (cfg.CloudAPIKey == "" && cfg.CloudToken == "") {
		return nil, fmt.Errorf("not logged in. Run: cm cloud login")
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &authTransport{
			apiKey: cfg.CloudAPIKey,
			token:  cfg.CloudToken,
		},
	}

	return client, nil
}

type authTransport struct {
	apiKey string
	token  string
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.apiKey != "" {
		req.Header.Set("X-API-Key", t.apiKey)
	} else if t.token != "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	return http.DefaultTransport.RoundTrip(req)
}

func init() {
	cloudLoginCmd.Flags().String("api-key", "", "API key for authentication")

	cloudCreateCmd.Flags().StringVar(&cloudCreateType, "type", "cpu-small", "Instance type")
	cloudCreateCmd.Flags().StringVar(&cloudCreateProvider, "provider", "aws", "Cloud provider")
	cloudCreateCmd.Flags().StringVar(&cloudCreateRegion, "region", "", "Cloud region")
	cloudCreateCmd.Flags().StringVar(&cloudCreateName, "name", "", "Instance name")

	cloudCmd.AddCommand(cloudLoginCmd)
	cloudCmd.AddCommand(cloudLogoutCmd)
	cloudCmd.AddCommand(cloudInstancesCmd)
	cloudCmd.AddCommand(cloudCreateCmd)
	cloudCmd.AddCommand(cloudConnectCmd)
	cloudCmd.AddCommand(cloudStopCmd)
	cloudCmd.AddCommand(cloudDeleteCmd)
	cloudCmd.AddCommand(cloudProvidersCmd)
	cloudCmd.AddCommand(cloudBillingCmd)
	rootCmd.AddCommand(cloudCmd)
}
