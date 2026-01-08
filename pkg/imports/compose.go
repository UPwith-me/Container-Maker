package imports

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/workspace"
	"gopkg.in/yaml.v3"
)

// ComposeImporter imports docker-compose.yml files
type ComposeImporter struct{}

// NewComposeImporter creates a new compose importer
func NewComposeImporter() *ComposeImporter {
	return &ComposeImporter{}
}

// CanHandle checks if this importer can handle the file
func (i *ComposeImporter) CanHandle(path string) bool {
	base := filepath.Base(path)
	return base == "docker-compose.yml" ||
		base == "docker-compose.yaml" ||
		base == "compose.yml" ||
		base == "compose.yaml" ||
		strings.HasPrefix(base, "docker-compose.")
}

// Validate checks if the source file is valid
func (i *ComposeImporter) Validate(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var compose ComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	if len(compose.Services) == 0 {
		return fmt.Errorf("no services found in compose file")
	}

	return nil
}

// Analyze analyzes a compose file without importing
func (i *ComposeImporter) Analyze(path string) (*AnalysisResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var compose ComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	result := &AnalysisResult{
		Source:     SourceDockerCompose,
		SourceFile: path,
		Valid:      true,
		Services:   make([]ServiceAnalysis, 0),
		Networks:   make([]string, 0),
		Volumes:    make([]string, 0),
	}

	// Analyze services
	fullySupported := 0
	partialSupport := 0
	notSupported := 0

	for name, svc := range compose.Services {
		analysis := i.analyzeService(name, svc)
		result.Services = append(result.Services, analysis)

		if len(analysis.Warnings) == 0 {
			fullySupported++
		} else if len(analysis.Warnings) < 3 {
			partialSupport++
		} else {
			notSupported++
		}
	}

	// Collect networks
	for name := range compose.Networks {
		result.Networks = append(result.Networks, name)
	}

	// Collect volumes
	for name := range compose.Volumes {
		result.Volumes = append(result.Volumes, name)
	}

	// Build compatibility report
	total := len(compose.Services)
	if total == 0 {
		total = 1
	}
	result.Compatibility = CompatibilityReport{
		Score:           (fullySupported*100 + partialSupport*70) / total,
		FullySupported:  make([]string, 0),
		PartialSupport:  make([]string, 0),
		NotSupported:    make([]string, 0),
		Recommendations: make([]string, 0),
	}

	for _, svc := range result.Services {
		if len(svc.Warnings) == 0 {
			result.Compatibility.FullySupported = append(result.Compatibility.FullySupported, svc.Name)
		} else if len(svc.Warnings) < 3 {
			result.Compatibility.PartialSupport = append(result.Compatibility.PartialSupport, svc.Name)
		} else {
			result.Compatibility.NotSupported = append(result.Compatibility.NotSupported, svc.Name)
		}
	}

	return result, nil
}

// analyzeService analyzes a single service
func (i *ComposeImporter) analyzeService(name string, svc *ComposeService) ServiceAnalysis {
	analysis := ServiceAnalysis{
		Name:  name,
		Image: svc.Image,
		Build: svc.Build != nil,
	}

	// Parse ports
	for _, p := range svc.Ports {
		analysis.Ports = append(analysis.Ports, fmt.Sprintf("%v", p))
	}

	// Parse depends_on
	switch deps := svc.DependsOn.(type) {
	case []interface{}:
		for _, d := range deps {
			analysis.Dependencies = append(analysis.Dependencies, fmt.Sprintf("%v", d))
		}
	case map[string]interface{}:
		for d := range deps {
			analysis.Dependencies = append(analysis.Dependencies, d)
		}
	}

	// Parse volumes
	for _, v := range svc.Volumes {
		analysis.Volumes = append(analysis.Volumes, fmt.Sprintf("%v", v))
	}

	// Count environment
	switch env := svc.Environment.(type) {
	case []interface{}:
		analysis.Environment = len(env)
	case map[string]interface{}:
		analysis.Environment = len(env)
	}

	analysis.HasHealthCheck = svc.HealthCheck != nil

	// Check for GPU
	if svc.Deploy != nil && svc.Deploy.Resources != nil {
		if svc.Deploy.Resources.Reservations != nil {
			analysis.HasGPU = len(svc.Deploy.Resources.Reservations.Devices) > 0
		}
	}

	// Warnings for unsupported features
	if svc.Privileged {
		analysis.Warnings = append(analysis.Warnings, "privileged mode not recommended")
	}
	if len(svc.CapAdd) > 0 {
		analysis.Warnings = append(analysis.Warnings, "cap_add requires manual review")
	}
	if len(svc.Devices) > 0 && !analysis.HasGPU {
		analysis.Warnings = append(analysis.Warnings, "devices may need manual configuration")
	}
	if svc.User != "" {
		analysis.Warnings = append(analysis.Warnings, "user setting needs verification")
	}

	return analysis
}

// Import imports a docker-compose file
func (i *ComposeImporter) Import(opts ImportOptions) (*ImportResult, error) {
	data, err := os.ReadFile(opts.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var compose ComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	result := &ImportResult{
		Source:     SourceDockerCompose,
		SourceFile: opts.SourcePath,
		Warnings:   make([]ImportWarning, 0),
		Errors:     make([]ImportError, 0),
		CreatedAt:  time.Now(),
	}

	// Create workspace
	wsName := opts.ProjectName
	if wsName == "" {
		wsName = filepath.Base(filepath.Dir(opts.SourcePath))
	}

	ws := workspace.CreateDefaultWorkspace(wsName)

	// Convert services
	for name, svc := range compose.Services {
		cmSvc, warnings := i.convertService(name, svc, opts)
		ws.Services[name] = cmSvc
		result.Warnings = append(result.Warnings, warnings...)
		result.Statistics.ServicesImported++
	}

	// Convert networks
	for name, net := range compose.Networks {
		cmNet := i.convertNetwork(name, net)
		if ws.Networks == nil {
			ws.Networks = make(map[string]*workspace.NetworkConfig)
		}
		ws.Networks[name] = cmNet
		result.Statistics.NetworksImported++
	}

	// Convert volumes
	for name, vol := range compose.Volumes {
		cmVol := i.convertVolume(name, vol)
		if ws.Volumes == nil {
			ws.Volumes = make(map[string]*workspace.VolumeConfig)
		}
		ws.Volumes[name] = cmVol
		result.Statistics.VolumesImported++
	}

	// Check for secrets
	result.Statistics.SecretsFound = len(compose.Secrets)
	if result.Statistics.SecretsFound > 0 {
		result.Warnings = append(result.Warnings, ImportWarning{
			Code:       "SECRETS_FOUND",
			Message:    fmt.Sprintf("%d secrets found - need manual migration", result.Statistics.SecretsFound),
			Suggestion: "Use environment variables or CM secret management",
		})
	}

	result.Workspace = ws

	// Write output if not dry run
	if !opts.DryRun {
		outputPath := opts.OutputPath
		if outputPath == "" {
			outputPath = filepath.Join(filepath.Dir(opts.SourcePath), "cm-workspace.yaml")
		}
		ws.ConfigFile = outputPath
		if err := workspace.Save(ws); err != nil {
			return result, fmt.Errorf("failed to write workspace: %w", err)
		}
	}

	return result, nil
}

// convertService converts a compose service to CM service
func (i *ComposeImporter) convertService(name string, svc *ComposeService, opts ImportOptions) (*workspace.Service, []ImportWarning) {
	var warnings []ImportWarning

	cmSvc := &workspace.Service{
		Name:          name,
		Image:         svc.Image,
		RestartPolicy: svc.Restart,
		WorkingDir:    svc.WorkingDir,
	}

	// Convert build
	if svc.Build != nil {
		cmSvc.Build = i.convertBuild(svc.Build)
	}

	// Convert command
	switch cmd := svc.Command.(type) {
	case string:
		cmSvc.Command = strings.Fields(cmd)
	case []interface{}:
		for _, c := range cmd {
			cmSvc.Command = append(cmSvc.Command, fmt.Sprintf("%v", c))
		}
	}

	// Convert entrypoint
	switch ep := svc.Entrypoint.(type) {
	case string:
		cmSvc.Entrypoint = strings.Fields(ep)
	case []interface{}:
		for _, e := range ep {
			cmSvc.Entrypoint = append(cmSvc.Entrypoint, fmt.Sprintf("%v", e))
		}
	}

	// Convert environment
	cmSvc.Environment = i.convertEnvironment(svc.Environment)

	// Convert ports
	for _, p := range svc.Ports {
		port := i.convertPort(p)
		if port != nil {
			cmSvc.Ports = append(cmSvc.Ports, *port)
		}
	}

	// Convert volumes
	for _, v := range svc.Volumes {
		cmSvc.Volumes = append(cmSvc.Volumes, fmt.Sprintf("%v", v))
	}

	// Convert depends_on
	switch deps := svc.DependsOn.(type) {
	case []interface{}:
		for _, d := range deps {
			cmSvc.DependsOn = append(cmSvc.DependsOn, fmt.Sprintf("%v", d))
		}
	case map[string]interface{}:
		for d := range deps {
			cmSvc.DependsOn = append(cmSvc.DependsOn, d)
		}
	}

	// Convert networks
	switch nets := svc.Networks.(type) {
	case []interface{}:
		for _, n := range nets {
			cmSvc.Networks = append(cmSvc.Networks, fmt.Sprintf("%v", n))
		}
	case map[string]interface{}:
		for n := range nets {
			cmSvc.Networks = append(cmSvc.Networks, n)
		}
	}

	// Convert healthcheck
	if svc.HealthCheck != nil && !svc.HealthCheck.Disable {
		cmSvc.HealthCheck = i.convertHealthCheck(svc.HealthCheck)
	}

	// Convert resources
	if svc.Deploy != nil && svc.Deploy.Resources != nil {
		cmSvc.Resources = i.convertResources(svc.Deploy.Resources)
		cmSvc.GPU = i.convertGPU(svc.Deploy.Resources)
	}

	// Convert labels
	cmSvc.Labels = i.convertLabels(svc.Labels)

	// Add warnings for unsupported features
	if svc.Privileged {
		warnings = append(warnings, ImportWarning{
			Code:       "PRIVILEGED_MODE",
			Message:    "privileged mode is not directly supported",
			Service:    name,
			Suggestion: "Remove privileged or use specific capabilities",
		})
	}

	return cmSvc, warnings
}

// convertBuild converts build configuration
func (i *ComposeImporter) convertBuild(build interface{}) *workspace.BuildConfig {
	switch b := build.(type) {
	case string:
		return &workspace.BuildConfig{Context: b}
	case map[string]interface{}:
		cfg := &workspace.BuildConfig{}
		if ctx, ok := b["context"].(string); ok {
			cfg.Context = ctx
		}
		if df, ok := b["dockerfile"].(string); ok {
			cfg.Dockerfile = df
		}
		if target, ok := b["target"].(string); ok {
			cfg.Target = target
		}
		if args, ok := b["args"].(map[string]interface{}); ok {
			cfg.Args = make(map[string]string)
			for k, v := range args {
				cfg.Args[k] = fmt.Sprintf("%v", v)
			}
		}
		return cfg
	}
	return nil
}

// convertEnvironment converts environment variables
func (i *ComposeImporter) convertEnvironment(env interface{}) map[string]string {
	result := make(map[string]string)

	switch e := env.(type) {
	case []interface{}:
		for _, item := range e {
			s := fmt.Sprintf("%v", item)
			parts := strings.SplitN(s, "=", 2)
			if len(parts) == 2 {
				result[parts[0]] = parts[1]
			} else {
				result[parts[0]] = ""
			}
		}
	case map[string]interface{}:
		for k, v := range e {
			if v == nil {
				result[k] = ""
			} else {
				result[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	return result
}

// convertPort converts port configuration
func (i *ComposeImporter) convertPort(port interface{}) *workspace.PortConfig {
	switch p := port.(type) {
	case int:
		return &workspace.PortConfig{Target: p, Published: p, Protocol: "tcp"}
	case float64:
		return &workspace.PortConfig{Target: int(p), Published: int(p), Protocol: "tcp"}
	case string:
		// Parse "host:container" or "host:container/protocol"
		parts := strings.Split(p, "/")
		protocol := "tcp"
		if len(parts) == 2 {
			protocol = parts[1]
		}

		portParts := strings.Split(parts[0], ":")
		if len(portParts) == 1 {
			port, _ := strconv.Atoi(portParts[0])
			return &workspace.PortConfig{Target: port, Published: port, Protocol: protocol}
		}

		host, _ := strconv.Atoi(portParts[0])
		container, _ := strconv.Atoi(portParts[1])
		return &workspace.PortConfig{Target: container, Published: host, Protocol: protocol}
	case map[string]interface{}:
		cfg := &workspace.PortConfig{Protocol: "tcp"}
		if target, ok := p["target"].(int); ok {
			cfg.Target = target
		}
		if published, ok := p["published"].(int); ok {
			cfg.Published = published
		}
		if protocol, ok := p["protocol"].(string); ok {
			cfg.Protocol = protocol
		}
		return cfg
	}
	return nil
}

// convertHealthCheck converts healthcheck configuration
func (i *ComposeImporter) convertHealthCheck(hc *ComposeHealthCheck) *workspace.HealthCheckConfig {
	cfg := &workspace.HealthCheckConfig{
		Retries: hc.Retries,
	}

	// Convert test
	switch t := hc.Test.(type) {
	case string:
		cfg.Test = strings.Fields(t)
	case []interface{}:
		for _, item := range t {
			cfg.Test = append(cfg.Test, fmt.Sprintf("%v", item))
		}
	}

	// Parse durations
	if hc.Interval != "" {
		cfg.Interval, _ = time.ParseDuration(hc.Interval)
	}
	if hc.Timeout != "" {
		cfg.Timeout, _ = time.ParseDuration(hc.Timeout)
	}
	if hc.StartPeriod != "" {
		cfg.StartPeriod, _ = time.ParseDuration(hc.StartPeriod)
	}

	return cfg
}

// convertResources converts resource limits
func (i *ComposeImporter) convertResources(res *ComposeResources) *workspace.ResourceConfig {
	cfg := &workspace.ResourceConfig{}

	if res.Limits != nil {
		cfg.CPUs, _ = strconv.ParseFloat(res.Limits.CPUs, 64)
		cfg.Memory = res.Limits.Memory
	}

	return cfg
}

// convertGPU extracts GPU configuration
func (i *ComposeImporter) convertGPU(res *ComposeResources) *workspace.GPUConfig {
	if res.Reservations == nil || len(res.Reservations.Devices) == 0 {
		return nil
	}

	for _, dev := range res.Reservations.Devices {
		if dev.Driver == "nvidia" || containsGPU(dev.Capabilities) {
			cfg := &workspace.GPUConfig{
				Driver:       dev.Driver,
				DeviceIDs:    dev.DeviceIDs,
				Capabilities: dev.Capabilities,
			}
			switch count := dev.Count.(type) {
			case int:
				cfg.Count = count
			case string:
				if count == "all" {
					cfg.Count = -1
				}
			}
			return cfg
		}
	}

	return nil
}

func containsGPU(caps []string) bool {
	for _, c := range caps {
		if c == "gpu" {
			return true
		}
	}
	return false
}

// convertLabels converts labels
func (i *ComposeImporter) convertLabels(labels interface{}) map[string]string {
	result := make(map[string]string)

	switch l := labels.(type) {
	case []interface{}:
		for _, item := range l {
			s := fmt.Sprintf("%v", item)
			parts := strings.SplitN(s, "=", 2)
			if len(parts) == 2 {
				result[parts[0]] = parts[1]
			}
		}
	case map[string]interface{}:
		for k, v := range l {
			result[k] = fmt.Sprintf("%v", v)
		}
	}

	return result
}

// convertNetwork converts network configuration
func (i *ComposeImporter) convertNetwork(name string, net *ComposeNetwork) *workspace.NetworkConfig {
	cfg := &workspace.NetworkConfig{
		Driver: net.Driver,
		Labels: net.Labels,
	}

	switch ext := net.External.(type) {
	case bool:
		cfg.External = ext
	case map[string]interface{}:
		cfg.External = true
	}

	return cfg
}

// convertVolume converts volume configuration
func (i *ComposeImporter) convertVolume(name string, vol *ComposeVolume) *workspace.VolumeConfig {
	cfg := &workspace.VolumeConfig{
		Driver: vol.Driver,
		Labels: vol.Labels,
	}

	switch ext := vol.External.(type) {
	case bool:
		cfg.External = ext
	case map[string]interface{}:
		cfg.External = true
	}

	return cfg
}
