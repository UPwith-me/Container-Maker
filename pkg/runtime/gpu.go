package runtime

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// GPUInfo holds GPU detection results
type GPUInfo struct {
	Available   bool
	Type        string // "nvidia", "amd", "intel", "none"
	Name        string
	Memory      string
	DriverVer   string
	CUDAVersion string
	Count       int
}

// GetActiveRuntime returns the currently active container runtime
func GetActiveRuntime() (ContainerRuntime, error) {
	detector := NewDetector()
	result := detector.Detect()

	if result.Active == nil {
		return nil, fmt.Errorf("no running container runtime found")
	}

	return CreateRuntime(result.Active.Name, result.Active.Path, result.Active.Type)
}

// CreateRuntime creates a runtime instance based on type
func CreateRuntime(name, path, typ string) (ContainerRuntime, error) {
	switch typ {
	case "docker":
		return NewDockerRuntime(name, path)
	case "podman":
		return NewPodmanRuntime(name, path)
	default:
		// Default to docker-compatible
		return NewDockerRuntime(name, path)
	}
}

// DetectGPU detects available GPU hardware
func DetectGPU() *GPUInfo {
	info := &GPUInfo{
		Available: false,
		Type:      "none",
	}

	// Try NVIDIA first (most common for deep learning)
	if nvidia := detectNVIDIA(); nvidia != nil {
		return nvidia
	}

	// Try AMD ROCm
	if amd := detectAMD(); amd != nil {
		return amd
	}

	// Try Intel (basic detection)
	if intel := detectIntel(); intel != nil {
		return intel
	}

	return info
}

func detectNVIDIA() *GPUInfo {
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,memory.total,driver_version", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return nil
	}

	// Parse first GPU
	parts := strings.Split(lines[0], ", ")
	info := &GPUInfo{
		Available: true,
		Type:      "nvidia",
		Count:     len(lines),
	}

	if len(parts) >= 1 {
		info.Name = strings.TrimSpace(parts[0])
	}
	if len(parts) >= 2 {
		info.Memory = strings.TrimSpace(parts[1]) + " MiB"
	}
	if len(parts) >= 3 {
		info.DriverVer = strings.TrimSpace(parts[2])
	}

	// Get CUDA version
	cudaCmd := exec.Command("nvidia-smi", "--query-gpu=cuda_version", "--format=csv,noheader")
	if cudaOutput, err := cudaCmd.Output(); err == nil {
		info.CUDAVersion = strings.TrimSpace(string(cudaOutput))
	}

	return info
}

func detectAMD() *GPUInfo {
	// Try rocm-smi for AMD GPUs
	cmd := exec.Command("rocm-smi", "--showproductname")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	if !strings.Contains(string(output), "GPU") {
		return nil
	}

	info := &GPUInfo{
		Available: true,
		Type:      "amd",
		Name:      "AMD GPU (ROCm)",
		Count:     1,
	}

	// Try to get more details
	if memCmd := exec.Command("rocm-smi", "--showmeminfo", "vram"); memCmd != nil {
		if memOutput, err := memCmd.Output(); err == nil {
			// Parse memory info
			info.Memory = strings.TrimSpace(string(memOutput))
		}
	}

	return info
}

func detectIntel() *GPUInfo {
	// Basic Intel GPU detection
	if runtime.GOOS == "linux" {
		// Check for Intel GPU via lspci
		cmd := exec.Command("lspci")
		output, err := cmd.Output()
		if err != nil {
			return nil
		}

		if strings.Contains(strings.ToLower(string(output)), "intel") &&
			strings.Contains(strings.ToLower(string(output)), "vga") {
			return &GPUInfo{
				Available: true,
				Type:      "intel",
				Name:      "Intel Integrated Graphics",
				Count:     1,
			}
		}
	}
	return nil
}

// GPUDockerArgs returns Docker/Podman args for GPU support
func GPUDockerArgs(gpu *GPUInfo) []string {
	if gpu == nil || !gpu.Available {
		return nil
	}

	switch gpu.Type {
	case "nvidia":
		return []string{"--gpus", "all"}
	case "amd":
		// ROCm uses device mapping
		return []string{"--device=/dev/kfd", "--device=/dev/dri"}
	default:
		return nil
	}
}

// FormatGPUInfo returns a formatted string of GPU info
func FormatGPUInfo(gpu *GPUInfo) string {
	if gpu == nil || !gpu.Available {
		return "No GPU detected"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("GPU: %s\n", gpu.Name))
	if gpu.Memory != "" {
		sb.WriteString(fmt.Sprintf("Memory: %s\n", gpu.Memory))
	}
	if gpu.DriverVer != "" {
		sb.WriteString(fmt.Sprintf("Driver: %s\n", gpu.DriverVer))
	}
	if gpu.CUDAVersion != "" {
		sb.WriteString(fmt.Sprintf("CUDA: %s\n", gpu.CUDAVersion))
	}
	if gpu.Count > 1 {
		sb.WriteString(fmt.Sprintf("Count: %d GPUs\n", gpu.Count))
	}

	return sb.String()
}
