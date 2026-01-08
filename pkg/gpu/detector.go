package gpu

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// NVIDIADetector detects NVIDIA GPUs using nvidia-smi
type NVIDIADetector struct {
	smiPath string
}

// NewNVIDIADetector creates a new NVIDIA GPU detector
func NewNVIDIADetector() *NVIDIADetector {
	return &NVIDIADetector{
		smiPath: "nvidia-smi",
	}
}

// IsAvailable checks if NVIDIA drivers are available
func (d *NVIDIADetector) IsAvailable() bool {
	cmd := exec.Command(d.smiPath, "--version")
	return cmd.Run() == nil
}

// Detect detects all NVIDIA GPUs
func (d *NVIDIADetector) Detect() ([]GPU, error) {
	// Query GPU info with CSV output
	cmd := exec.Command(d.smiPath, "--query-gpu=index,uuid,name,memory.total,memory.used,temperature.gpu,power.draw,power.limit,utilization.gpu,utilization.memory,driver_version,compute_cap", "--format=csv,noheader,nounits")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run nvidia-smi: %w", err)
	}

	return d.parseNvidiaSmiOutput(output)
}

// parseNvidiaSmiOutput parses nvidia-smi CSV output
func (d *NVIDIADetector) parseNvidiaSmiOutput(output []byte) ([]GPU, error) {
	reader := csv.NewReader(bytes.NewReader(output))
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	var gpus []GPU
	for _, record := range records {
		if len(record) < 12 {
			continue
		}

		index, _ := strconv.Atoi(strings.TrimSpace(record[0]))
		vramTotal, _ := strconv.ParseInt(strings.TrimSpace(record[3]), 10, 64)
		vramUsed, _ := strconv.ParseInt(strings.TrimSpace(record[4]), 10, 64)
		temp, _ := strconv.Atoi(strings.TrimSpace(record[5]))
		powerDraw, _ := strconv.ParseFloat(strings.TrimSpace(record[6]), 64)
		powerLimit, _ := strconv.ParseFloat(strings.TrimSpace(record[7]), 64)
		gpuUtil, _ := strconv.Atoi(strings.TrimSpace(record[8]))
		memUtil, _ := strconv.Atoi(strings.TrimSpace(record[9]))

		gpu := GPU{
			ID:             strings.TrimSpace(record[1]),
			Index:          index,
			Name:           strings.TrimSpace(record[2]),
			Vendor:         VendorNVIDIA,
			Driver:         strings.TrimSpace(record[10]),
			VRAM:           vramTotal * 1024 * 1024, // MB to bytes
			VRAMUsed:       vramUsed * 1024 * 1024,
			ComputeCap:     strings.TrimSpace(record[11]),
			Temperature:    temp,
			PowerUsage:     int(powerDraw),
			PowerLimit:     int(powerLimit),
			Utilization:    gpuUtil,
			MemUtilization: memUtil,
		}

		gpus = append(gpus, gpu)
	}

	return gpus, nil
}

// DetectVendor for NVIDIA always returns NVIDIA GPUs
func (d *NVIDIADetector) DetectVendor(vendor GPUVendor) ([]GPU, error) {
	if vendor != VendorNVIDIA {
		return nil, nil
	}
	return d.Detect()
}

// GetGPU returns a specific GPU by ID
func (d *NVIDIADetector) GetGPU(id string) (*GPU, error) {
	gpus, err := d.Detect()
	if err != nil {
		return nil, err
	}

	for _, gpu := range gpus {
		if gpu.ID == id {
			return &gpu, nil
		}
	}

	return nil, fmt.Errorf("GPU %s not found", id)
}

// Refresh refreshes GPU state
func (d *NVIDIADetector) Refresh(gpu *GPU) error {
	cmd := exec.Command(d.smiPath, "--query-gpu=temperature.gpu,power.draw,utilization.gpu,utilization.memory,memory.used", "--format=csv,noheader,nounits", "-i", gpu.ID)

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to refresh GPU state: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(string(output)), ",")
	if len(parts) >= 5 {
		gpu.Temperature, _ = strconv.Atoi(strings.TrimSpace(parts[0]))
		powerDraw, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		gpu.PowerUsage = int(powerDraw)
		gpu.Utilization, _ = strconv.Atoi(strings.TrimSpace(parts[2]))
		gpu.MemUtilization, _ = strconv.Atoi(strings.TrimSpace(parts[3]))
		vramUsed, _ := strconv.ParseInt(strings.TrimSpace(parts[4]), 10, 64)
		gpu.VRAMUsed = vramUsed * 1024 * 1024
	}

	return nil
}

// SimpleScheduler implements a basic GPU scheduler
type SimpleScheduler struct {
	detector GPUDetector
	pool     *GPUPool
	config   GPUSchedulerConfig
}

// NewSimpleScheduler creates a new simple GPU scheduler
func NewSimpleScheduler(detector GPUDetector, config GPUSchedulerConfig) (*SimpleScheduler, error) {
	s := &SimpleScheduler{
		detector: detector,
		config:   config,
		pool: &GPUPool{
			Allocations: make([]GPUAllocation, 0),
		},
	}

	// Initial scan
	if err := s.scan(); err != nil {
		return nil, err
	}

	return s, nil
}

// scan scans for GPUs
func (s *SimpleScheduler) scan() error {
	gpus, err := s.detector.Detect()
	if err != nil {
		return err
	}

	s.pool.GPUs = gpus
	s.pool.Total = len(gpus)
	s.pool.LastScan = time.Now()
	s.updateAvailable()

	return nil
}

// updateAvailable updates available count
func (s *SimpleScheduler) updateAvailable() {
	allocated := 0
	for _, gpu := range s.pool.GPUs {
		if gpu.Allocated {
			allocated++
		}
	}
	s.pool.Allocated = allocated
	s.pool.Available = s.pool.Total - allocated
}

// Allocate allocates GPUs based on requirements
func (s *SimpleScheduler) Allocate(owner string, req GPURequirements) (*GPUAllocation, error) {
	// Refresh pool
	if err := s.scan(); err != nil {
		return nil, err
	}

	// Find matching GPUs
	var candidates []GPU
	for _, gpu := range s.pool.GPUs {
		if gpu.Allocated && req.Exclusive {
			continue
		}
		if req.Vendor != "" && gpu.Vendor != req.Vendor {
			continue
		}
		if req.MinVRAM > 0 && gpu.VRAM < req.MinVRAM {
			continue
		}
		if len(req.DeviceIDs) > 0 && !contains(req.DeviceIDs, gpu.ID) {
			continue
		}
		candidates = append(candidates, gpu)
	}

	// Check if we have enough
	count := req.Count
	if count <= 0 {
		count = 1
	}
	if len(candidates) < count {
		return nil, fmt.Errorf("not enough GPUs available: need %d, have %d", count, len(candidates))
	}

	// Allocate
	allocation := &GPUAllocation{
		ID:           fmt.Sprintf("alloc-%d", time.Now().UnixNano()),
		GPUs:         candidates[:count],
		Owner:        owner,
		Requirements: req,
		AllocatedAt:  time.Now(),
	}

	// Mark as allocated
	for i, gpu := range s.pool.GPUs {
		for _, allocated := range allocation.GPUs {
			if gpu.ID == allocated.ID {
				s.pool.GPUs[i].Allocated = true
				s.pool.GPUs[i].AllocatedTo = owner
				s.pool.GPUs[i].AllocatedAt = time.Now()
			}
		}
	}

	s.pool.Allocations = append(s.pool.Allocations, *allocation)
	s.updateAvailable()

	return allocation, nil
}

// Release releases a GPU allocation
func (s *SimpleScheduler) Release(allocationID string) error {
	for i, alloc := range s.pool.Allocations {
		if alloc.ID == allocationID {
			// Unmark GPUs
			for _, allocGPU := range alloc.GPUs {
				for j, gpu := range s.pool.GPUs {
					if gpu.ID == allocGPU.ID {
						s.pool.GPUs[j].Allocated = false
						s.pool.GPUs[j].AllocatedTo = ""
					}
				}
			}
			// Remove allocation
			s.pool.Allocations = append(s.pool.Allocations[:i], s.pool.Allocations[i+1:]...)
			s.updateAvailable()
			return nil
		}
	}
	return fmt.Errorf("allocation %s not found", allocationID)
}

// ReleaseByOwner releases all allocations for an owner
func (s *SimpleScheduler) ReleaseByOwner(owner string) error {
	var toRelease []string
	for _, alloc := range s.pool.Allocations {
		if alloc.Owner == owner {
			toRelease = append(toRelease, alloc.ID)
		}
	}
	for _, id := range toRelease {
		s.Release(id)
	}
	return nil
}

// GetAllocation returns an allocation by ID
func (s *SimpleScheduler) GetAllocation(id string) (*GPUAllocation, error) {
	for _, alloc := range s.pool.Allocations {
		if alloc.ID == id {
			return &alloc, nil
		}
	}
	return nil, fmt.Errorf("allocation %s not found", id)
}

// GetPool returns the current GPU pool state
func (s *SimpleScheduler) GetPool() *GPUPool {
	return s.pool
}

// CanSatisfy checks if requirements can be satisfied
func (s *SimpleScheduler) CanSatisfy(req GPURequirements) bool {
	count := 0
	for _, gpu := range s.pool.GPUs {
		if gpu.Allocated && req.Exclusive {
			continue
		}
		if req.Vendor != "" && gpu.Vendor != req.Vendor {
			continue
		}
		if req.MinVRAM > 0 && gpu.VRAM < req.MinVRAM {
			continue
		}
		count++
	}

	needed := req.Count
	if needed <= 0 {
		needed = 1
	}
	return count >= needed
}

// WaitForGPUs waits for GPUs to become available
func (s *SimpleScheduler) WaitForGPUs(req GPURequirements, timeout time.Duration) (*GPUAllocation, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		alloc, err := s.Allocate("waiting", req)
		if err == nil {
			return alloc, nil
		}
		time.Sleep(5 * time.Second)
		s.scan()
	}

	return nil, fmt.Errorf("timeout waiting for GPUs")
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
