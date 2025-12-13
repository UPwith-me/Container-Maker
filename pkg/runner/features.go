package runner

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

// Feature represents a DevContainer Feature
type Feature struct {
	ID      string                 `json:"id"`
	Version string                 `json:"version,omitempty"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// FeatureInstaller handles DevContainer Features installation
type FeatureInstaller struct {
	containerID string
	backend     string
}

// NewFeatureInstaller creates a new feature installer
func NewFeatureInstaller(containerID, backend string) *FeatureInstaller {
	return &FeatureInstaller{
		containerID: containerID,
		backend:     backend,
	}
}

// InstallFeatures installs features into a container
func (f *FeatureInstaller) InstallFeatures(ctx context.Context, features map[string]interface{}) error {
	if len(features) == 0 {
		return nil
	}

	fmt.Printf("🔧 Installing %d DevContainer feature(s)...\n", len(features))

	for featureID, options := range features {
		if err := f.installFeature(ctx, featureID, options); err != nil {
			fmt.Printf("⚠️  Feature '%s' failed: %v\n", featureID, err)
			continue
		}
		fmt.Printf("  ✓ Installed: %s\n", featureID)
	}

	fmt.Println("✅ Features installation complete")
	return nil
}

// installFeature installs a single feature
func (f *FeatureInstaller) installFeature(ctx context.Context, featureID string, options interface{}) error {
	// Try built-in command first (faster)
	if installCmd := f.getFeatureInstallCommand(featureID, options); installCmd != "" {
		cmd := exec.CommandContext(ctx, f.backend, "exec", f.containerID, "sh", "-c", installCmd)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// Try to download from OCI registry for ghcr.io features
	if strings.HasPrefix(featureID, "ghcr.io/") {
		if err := f.installFromOCI(ctx, featureID, options); err == nil {
			return nil
		}
	}

	return fmt.Errorf("unsupported feature: %s", featureID)
}

// installFromOCI downloads and installs a feature from OCI registry
func (f *FeatureInstaller) installFromOCI(ctx context.Context, featureID string, options interface{}) error {
	// Parse ghcr.io/owner/repo/feature:version
	parts := strings.Split(strings.TrimPrefix(featureID, "ghcr.io/"), "/")
	if len(parts) < 2 {
		return fmt.Errorf("invalid feature ID: %s", featureID)
	}

	// Extract version if present
	featureName := parts[len(parts)-1]
	version := "latest"
	if idx := strings.Index(featureName, ":"); idx != -1 {
		version = featureName[idx+1:]
		featureName = featureName[:idx]
	}

	// Try to get install script from devcontainers CDN
	cdnURL := fmt.Sprintf("https://github.com/devcontainers/features/raw/main/src/%s/install.sh", featureName)

	resp, err := http.Get(cdnURL)
	if err != nil || resp.StatusCode != 200 {
		return fmt.Errorf("feature not found in CDN")
	}
	defer resp.Body.Close()

	script, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Create temp file in container
	tmpScript := fmt.Sprintf("/tmp/feature-%s-install.sh", featureName)

	// Write script to container
	echoCmd := exec.CommandContext(ctx, f.backend, "exec", "-i", f.containerID, "sh", "-c",
		fmt.Sprintf("cat > %s && chmod +x %s", tmpScript, tmpScript))
	echoCmd.Stdin = strings.NewReader(string(script))
	if err := echoCmd.Run(); err != nil {
		return err
	}

	// Build environment variables from options
	envVars := []string{}
	if opts, ok := options.(map[string]interface{}); ok {
		for k, v := range opts {
			envVars = append(envVars, fmt.Sprintf("%s=%v", strings.ToUpper(k), v))
		}
	}
	envVars = append(envVars, fmt.Sprintf("VERSION=%s", version))

	// Execute install script
	args := []string{"exec"}
	for _, env := range envVars {
		args = append(args, "-e", env)
	}
	args = append(args, f.containerID, tmpScript)

	installCmd := exec.CommandContext(ctx, f.backend, args...)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr

	return installCmd.Run()
}

// getFeatureInstallCommand returns the install command for a feature
func (f *FeatureInstaller) getFeatureInstallCommand(featureID string, options interface{}) string {
	// Extract base feature name
	baseName := featureID
	if idx := strings.LastIndex(featureID, "/"); idx != -1 {
		baseName = featureID[idx+1:]
	}
	if idx := strings.Index(baseName, ":"); idx != -1 {
		baseName = baseName[:idx]
	}

	// Comprehensive feature install commands (Alpine/Debian/RHEL compatible)
	featureCommands := map[string]string{
		"git": `
			if command -v apk >/dev/null 2>&1; then
				apk add --no-cache git
			elif command -v apt-get >/dev/null 2>&1; then
				apt-get update && apt-get install -y git
			elif command -v yum >/dev/null 2>&1; then
				yum install -y git
			fi
		`,
		"docker-in-docker": `
			if command -v apk >/dev/null 2>&1; then
				apk add --no-cache docker-cli
			elif command -v apt-get >/dev/null 2>&1; then
				apt-get update && apt-get install -y docker.io
			fi
		`,
		"node": `
			if command -v apk >/dev/null 2>&1; then
				apk add --no-cache nodejs npm
			elif command -v apt-get >/dev/null 2>&1; then
				apt-get update && apt-get install -y nodejs npm
			fi
		`,
		"python": `
			if command -v apk >/dev/null 2>&1; then
				apk add --no-cache python3 py3-pip
			elif command -v apt-get >/dev/null 2>&1; then
				apt-get update && apt-get install -y python3 python3-pip
			fi
		`,
		"go": `
			if ! command -v go >/dev/null 2>&1; then
				if command -v apk >/dev/null 2>&1; then
					apk add --no-cache go
				elif command -v apt-get >/dev/null 2>&1; then
					apt-get update && apt-get install -y golang
				fi
			fi
		`,
		"rust": `
			if ! command -v rustc >/dev/null 2>&1; then
				curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
			fi
		`,
		"common-utils": `
			if command -v apk >/dev/null 2>&1; then
				apk add --no-cache curl wget vim nano less htop jq
			elif command -v apt-get >/dev/null 2>&1; then
				apt-get update && apt-get install -y curl wget vim nano less htop jq
			fi
		`,
		"github-cli": `
			if command -v apk >/dev/null 2>&1; then
				apk add --no-cache github-cli
			elif command -v apt-get >/dev/null 2>&1; then
				curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
				echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null
				apt-get update && apt-get install -y gh
			fi
		`,
		"kubectl-helm-minikube": `
			curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
			chmod +x kubectl && mv kubectl /usr/local/bin/
			curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
		`,
		"aws-cli": `
			if command -v apt-get >/dev/null 2>&1; then
				curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
				unzip -q awscliv2.zip && ./aws/install && rm -rf awscliv2.zip aws
			elif command -v apk >/dev/null 2>&1; then
				apk add --no-cache aws-cli
			fi
		`,
		"azure-cli": `
			if command -v apt-get >/dev/null 2>&1; then
				curl -sL https://aka.ms/InstallAzureCLIDeb | bash
			fi
		`,
		"terraform": `
			if command -v apk >/dev/null 2>&1; then
				apk add --no-cache terraform
			elif command -v apt-get >/dev/null 2>&1; then
				apt-get update && apt-get install -y gnupg software-properties-common
				wget -O- https://apt.releases.hashicorp.com/gpg | gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
				apt-get update && apt-get install -y terraform
			fi
		`,
		"java": `
			if command -v apk >/dev/null 2>&1; then
				apk add --no-cache openjdk17
			elif command -v apt-get >/dev/null 2>&1; then
				apt-get update && apt-get install -y openjdk-17-jdk
			fi
		`,
		"dotnet": `
			if command -v apt-get >/dev/null 2>&1; then
				apt-get update && apt-get install -y wget
				wget https://dot.net/v1/dotnet-install.sh -O dotnet-install.sh
				chmod +x dotnet-install.sh && ./dotnet-install.sh --channel 8.0
			fi
		`,
		"powershell": `
			if command -v apt-get >/dev/null 2>&1; then
				apt-get update && apt-get install -y wget apt-transport-https software-properties-common
				wget -q "https://packages.microsoft.com/config/ubuntu/$(lsb_release -rs)/packages-microsoft-prod.deb"
				dpkg -i packages-microsoft-prod.deb
				apt-get update && apt-get install -y powershell
			fi
		`,
		"sshd": `
			if command -v apk >/dev/null 2>&1; then
				apk add --no-cache openssh-server
			elif command -v apt-get >/dev/null 2>&1; then
				apt-get update && apt-get install -y openssh-server
			fi
		`,
	}

	if cmd, ok := featureCommands[baseName]; ok {
		return cmd
	}

	return ""
}
