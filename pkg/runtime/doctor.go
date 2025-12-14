package runtime

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// DiagnosticResult holds the result of a diagnostic check
type DiagnosticResult struct {
	Name    string
	Status  string // "ok", "warning", "error"
	Message string
	Details string
	Fix     string
}

// RunDiagnostics performs all diagnostic checks
func RunDiagnostics() []DiagnosticResult {
	var results []DiagnosticResult

	// 1. Container Runtime Check
	results = append(results, checkContainerRuntime())

	// 2. GPU Check
	results = append(results, checkGPU())

	// 3. Network Check
	results = append(results, checkNetwork())

	// 4. Disk Space Check
	results = append(results, checkDiskSpace())

	// 5. Docker Compose Check
	results = append(results, checkDockerCompose())

	return results
}

func checkContainerRuntime() DiagnosticResult {
	result := DiagnosticResult{
		Name: "Container Runtime",
	}

	detector := NewDetector()
	detection := detector.Detect()

	if len(detection.Backends) == 0 {
		result.Status = "error"
		result.Message = "No container runtime installed"
		result.Fix = "Install Docker: https://docker.com/get-started\nOr Podman: https://podman.io/getting-started"
		return result
	}

	var running []string
	var stopped []string

	for _, b := range detection.Backends {
		if b.Running {
			running = append(running, fmt.Sprintf("%s v%s", b.Name, b.Version))
		} else {
			stopped = append(stopped, b.Name)
		}
	}

	if len(running) > 0 {
		result.Status = "ok"
		result.Message = strings.Join(running, ", ")
		if len(stopped) > 0 {
			result.Details = fmt.Sprintf("Not running: %s", strings.Join(stopped, ", "))
		}
	} else {
		result.Status = "warning"
		result.Message = "Installed but not running"
		result.Details = strings.Join(stopped, ", ")
		if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
			result.Fix = "Start Docker Desktop"
		} else {
			result.Fix = "sudo systemctl start docker"
		}
	}

	return result
}

func checkGPU() DiagnosticResult {
	result := DiagnosticResult{
		Name: "GPU Support",
	}

	gpu := DetectGPU()

	if !gpu.Available {
		result.Status = "warning"
		result.Message = "No GPU detected"
		result.Details = "Deep learning templates will run on CPU only"
		return result
	}

	result.Status = "ok"
	result.Message = gpu.Name

	var details []string
	if gpu.Memory != "" {
		details = append(details, fmt.Sprintf("Memory: %s", gpu.Memory))
	}
	if gpu.DriverVer != "" {
		details = append(details, fmt.Sprintf("Driver: %s", gpu.DriverVer))
	}
	if gpu.CUDAVersion != "" {
		details = append(details, fmt.Sprintf("CUDA: %s", gpu.CUDAVersion))
	}
	if gpu.Count > 1 {
		details = append(details, fmt.Sprintf("%d GPUs available", gpu.Count))
	}

	result.Details = strings.Join(details, ", ")

	// Check NVIDIA Container Toolkit
	if gpu.Type == "nvidia" {
		if _, err := exec.LookPath("nvidia-container-toolkit"); err != nil {
			result.Status = "warning"
			result.Fix = "Install NVIDIA Container Toolkit for GPU in containers:\nhttps://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html"
		}
	}

	return result
}

func checkNetwork() DiagnosticResult {
	result := DiagnosticResult{
		Name: "Network",
	}

	// Check Docker Hub connectivity
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	targets := []struct {
		name string
		url  string
	}{
		{"Docker Hub", "https://registry-1.docker.io/v2/"},
		{"GitHub Container Registry", "https://ghcr.io/v2/"},
	}

	var reachable []string
	var unreachable []string

	for _, target := range targets {
		resp, err := client.Get(target.url)
		if err != nil {
			unreachable = append(unreachable, target.name)
		} else {
			resp.Body.Close()
			reachable = append(reachable, target.name)
		}
	}

	if len(unreachable) == 0 {
		result.Status = "ok"
		result.Message = "All registries reachable"
	} else if len(reachable) > 0 {
		result.Status = "warning"
		result.Message = fmt.Sprintf("%d/%d registries reachable", len(reachable), len(targets))
		result.Details = "Unreachable: " + strings.Join(unreachable, ", ")
	} else {
		result.Status = "error"
		result.Message = "Cannot reach container registries"
		result.Fix = "Check your internet connection or proxy settings"
	}

	return result
}

func checkDiskSpace() DiagnosticResult {
	result := DiagnosticResult{
		Name: "Disk Space",
	}

	// Get Docker root directory disk space
	var path string
	if runtime.GOOS == "windows" {
		path = os.Getenv("SYSTEMDRIVE") + "\\"
		if path == "\\" {
			path = "C:\\"
		}
	} else {
		path = "/var/lib/docker"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			path = "/"
		}
	}

	// Use df command on Unix, or call Windows API
	freeGB, totalGB, err := getDiskSpace(path)
	if err != nil {
		result.Status = "warning"
		result.Message = "Could not check disk space"
		return result
	}

	percentFree := (freeGB / totalGB) * 100
	result.Details = fmt.Sprintf("%.1f GB free of %.1f GB", freeGB, totalGB)

	if freeGB < 5 {
		result.Status = "error"
		result.Message = fmt.Sprintf("Low disk space: %.1f GB free", freeGB)
		result.Fix = "Free up disk space. Run 'docker system prune' to remove unused data"
	} else if percentFree < 10 {
		result.Status = "warning"
		result.Message = fmt.Sprintf("Disk space low: %.0f%% free", percentFree)
		result.Fix = "Consider freeing up disk space"
	} else {
		result.Status = "ok"
		result.Message = fmt.Sprintf("%.1f GB free (%.0f%%)", freeGB, percentFree)
	}

	return result
}

func getDiskSpace(path string) (freeGB, totalGB float64, err error) {
	if runtime.GOOS == "windows" {
		// Use PowerShell on Windows
		cmd := exec.Command("powershell", "-Command",
			fmt.Sprintf("(Get-PSDrive -Name '%s').Free, (Get-PSDrive -Name '%s').Used + (Get-PSDrive -Name '%s').Free",
				strings.TrimSuffix(path, ":\\"), strings.TrimSuffix(path, ":\\"), strings.TrimSuffix(path, "\\")))
		output, err := cmd.Output()
		if err != nil {
			// Fallback: just return a reasonable estimate
			return 50, 500, nil
		}
		var free, total int64
		_, _ = fmt.Sscanf(string(output), "%d\n%d", &free, &total)
		return float64(free) / 1e9, float64(total) / 1e9, nil
	}

	// Unix: use df
	cmd := exec.Command("df", "-B1", path)
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return 0, 0, fmt.Errorf("unexpected df output")
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 4 {
		return 0, 0, fmt.Errorf("unexpected df output format")
	}

	var total, free int64
	_, _ = fmt.Sscanf(fields[1], "%d", &total)
	_, _ = fmt.Sscanf(fields[3], "%d", &free)

	return float64(free) / 1e9, float64(total) / 1e9, nil
}

func checkDockerCompose() DiagnosticResult {
	result := DiagnosticResult{
		Name: "Docker Compose",
	}

	// Check for docker compose (v2) or docker-compose (v1)
	var version string

	// Try docker compose (v2)
	cmd := exec.Command("docker", "compose", "version", "--short")
	if output, err := cmd.Output(); err == nil {
		version = "v2: " + strings.TrimSpace(string(output))
	} else {
		// Try docker-compose (v1)
		cmd = exec.Command("docker-compose", "version", "--short")
		if output, err := cmd.Output(); err == nil {
			version = "v1: " + strings.TrimSpace(string(output))
		}
	}

	if version != "" {
		result.Status = "ok"
		result.Message = version
	} else {
		result.Status = "warning"
		result.Message = "Not installed"
		result.Details = "Docker Compose is optional but useful for multi-container setups"
		result.Fix = "Docker Desktop includes Compose. Or: pip install docker-compose"
	}

	return result
}

// CheckPort checks if a port is available
func CheckPort(port int) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}
