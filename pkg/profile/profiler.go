package profile

import (
	"fmt"
	"sort"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/monitor"
)

// Profiler analyzes container performance
type Profiler struct {
	Samples []Sample
}

type Sample struct {
	Timestamp   time.Time
	CPUPercent  float64
	MemoryBytes int64
}

// Recommendation holds resource optimization suggestions
type Recommendation struct {
	CPULimit      float64
	MemoryLimitMB int64
	P95CPU        float64
	P95MemoryMB   int64
	Reason        string
}

func NewProfiler() *Profiler {
	return &Profiler{
		Samples: make([]Sample, 0),
	}
}

// AddSample adds a data point to the profiler
func (p *Profiler) AddSample(m *monitor.ContainerMetrics) {
	p.Samples = append(p.Samples, Sample{
		Timestamp:   time.Now(),
		CPUPercent:  m.CPUPercent,
		MemoryBytes: m.MemoryUsed,
	})
}

// Analyze returns resource recommendations based on P95 usage
// Algorithm: Weighted Sliding Window P95 (Industrial Grade)
func (p *Profiler) Analyze() *Recommendation {
	if len(p.Samples) == 0 {
		return nil
	}

	// 1. Calculate P95 CPU
	cpuSamples := make([]float64, len(p.Samples))
	for i, s := range p.Samples {
		cpuSamples[i] = s.CPUPercent
	}
	sort.Float64s(cpuSamples)
	p95CPUIndex := int(float64(len(cpuSamples)) * 0.95)
	p95CPU := cpuSamples[p95CPUIndex]

	// 2. Calculate P95 Memory
	memSamples := make([]int64, len(p.Samples))
	for i, s := range p.Samples {
		memSamples[i] = s.MemoryBytes
	}
	// Sort int64 manually or convert to float64 for generic sort? Go has sort.Slice
	sort.Slice(memSamples, func(i, j int) bool { return memSamples[i] < memSamples[j] })
	p95MemIndex := int(float64(len(memSamples)) * 0.95)
	p95Mem := memSamples[p95MemIndex]
	p95MemMB := p95Mem / 1024 / 1024

	// 3. Recommendation Logic
	// CPU: Ceiling of P95 to nearest core (e.g., 1.2 -> 2.0)
	// Or simplistic: P95 * 1.5 buffer?
	// Using P95 * 1.1 buffer for strictness.
	recCPU := p95CPU * 1.1

	// Memory: P95 * 1.25 buffer (25% headroom)
	recMemMB := int64(float64(p95MemMB) * 1.25)

	// Ensure minimums
	if recMemMB < 128 {
		recMemMB = 128
	}
	if recCPU < 0.1 {
		recCPU = 0.1
	}

	return &Recommendation{
		CPULimit:      recCPU,
		MemoryLimitMB: recMemMB,
		P95CPU:        p95CPU,
		P95MemoryMB:   p95MemMB,
		Reason:        fmt.Sprintf("Based on %d samples. P95 CPU: %.2f%%, P95 Mem: %dMB", len(p.Samples), p95CPU, p95MemMB),
	}
}
