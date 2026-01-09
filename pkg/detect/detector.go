package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ProjectInfo holds comprehensive information about a detected project
type ProjectInfo struct {
	// Basic info
	Name    string `json:"name"`
	RootDir string `json:"rootDir"`

	// Languages detected
	Languages       []LanguageInfo `json:"languages"`
	PrimaryLanguage string         `json:"primaryLanguage"`

	// Environment details
	Frameworks      []string `json:"frameworks,omitempty"`
	BuildTools      []string `json:"buildTools,omitempty"`
	PackageManagers []string `json:"packageManagers,omitempty"`

	// Version info
	Versions map[string]string `json:"versions,omitempty"`

	// Hardware requirements
	NeedsGPU      bool     `json:"needsGPU"`
	GPUFrameworks []string `json:"gpuFrameworks,omitempty"`
	CUDAVersion   string   `json:"cudaVersion,omitempty"`

	// Project structure
	IsMonorepo   bool          `json:"isMonorepo"`
	MonorepoType string        `json:"monorepoType,omitempty"` // npm, pnpm, yarn, lerna, turborepo, nx
	Services     []ServiceInfo `json:"services,omitempty"`

	// Existing config
	HasDockerfile    bool `json:"hasDockerfile"`
	HasDockerCompose bool `json:"hasDockerCompose"`
	HasDevcontainer  bool `json:"hasDevcontainer"`
	HasMakefile      bool `json:"hasMakefile"`

	// Dependencies (top 20)
	Dependencies []string `json:"dependencies,omitempty"`

	// Files in root
	RootFiles []string `json:"rootFiles,omitempty"`
}

// LanguageInfo holds information about a detected language
type LanguageInfo struct {
	Name       string   `json:"name"`
	Confidence float64  `json:"confidence"` // 0-1
	Version    string   `json:"version,omitempty"`
	Indicators []string `json:"indicators"` // files/patterns that led to detection
}

// ServiceInfo holds information about a service in a monorepo
type ServiceInfo struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Language string `json:"language"`
	Template string `json:"template"`
}

// TemplateRecommendation holds a template suggestion with confidence
type TemplateRecommendation struct {
	Template   string   `json:"template"`
	Score      float64  `json:"score"`      // 0-1
	Confidence string   `json:"confidence"` // "high", "medium", "low"
	Reasons    []string `json:"reasons"`
}

// Detector is the main project detection engine
type Detector struct {
	projectDir string
	info       *ProjectInfo
}

// NewDetector creates a new project detector
func NewDetector(projectDir string) *Detector {
	return &Detector{
		projectDir: projectDir,
		info: &ProjectInfo{
			Name:     filepath.Base(projectDir),
			RootDir:  projectDir,
			Versions: make(map[string]string),
		},
	}
}

// Detect runs the full detection pipeline
func (d *Detector) Detect() (*ProjectInfo, error) {
	// Layer 1: File signature detection
	d.detectByFileSignature()

	// Layer 2: Content analysis (frameworks, dependencies)
	d.analyzeContent()

	// Layer 3: Version detection
	d.detectVersions()

	// Layer 4: GPU/Hardware detection
	d.detectGPURequirements()

	// Layer 5: Monorepo detection
	d.detectMonorepo()

	// Layer 6: Existing config detection
	d.detectExistingConfigs()

	// Set primary language
	d.setPrimaryLanguage()

	// Collect root files
	d.collectRootFiles()

	return d.info, nil
}

// detectByFileSignature detects languages by presence of signature files
func (d *Detector) detectByFileSignature() {
	signatures := []struct {
		file     string
		language string
		weight   float64
	}{
		// Go
		{"go.mod", "Go", 0.9},
		{"go.sum", "Go", 0.7},

		// JavaScript/TypeScript
		{"package.json", "JavaScript", 0.8},
		{"tsconfig.json", "TypeScript", 0.9},
		{"bun.lockb", "TypeScript", 0.8},

		// Python
		{"requirements.txt", "Python", 0.7},
		{"pyproject.toml", "Python", 0.9},
		{"setup.py", "Python", 0.8},
		{"Pipfile", "Python", 0.8},
		{"poetry.lock", "Python", 0.9},
		{"environment.yml", "Python", 0.8},
		{"conda.yaml", "Python", 0.8},

		// Rust
		{"Cargo.toml", "Rust", 0.95},
		{"Cargo.lock", "Rust", 0.8},

		// Java
		{"pom.xml", "Java", 0.9},
		{"build.gradle", "Java", 0.9},
		{"build.gradle.kts", "Kotlin", 0.9},
		{"settings.gradle", "Java", 0.7},

		// .NET
		{"*.csproj", ".NET/C#", 0.9},
		{"*.fsproj", ".NET/F#", 0.9},
		{"*.vbproj", ".NET/VB", 0.9},
		{"*.sln", ".NET", 0.8},
		{"global.json", ".NET", 0.7},

		// Ruby
		{"Gemfile", "Ruby", 0.9},
		{"Gemfile.lock", "Ruby", 0.8},
		{"*.gemspec", "Ruby", 0.8},

		// PHP
		{"composer.json", "PHP", 0.9},
		{"composer.lock", "PHP", 0.8},

		// C/C++
		{"CMakeLists.txt", "C++", 0.8},
		{"Makefile", "C/C++", 0.5},
		{"meson.build", "C/C++", 0.8},
		{"conanfile.txt", "C++", 0.8},
		{"conanfile.py", "C++", 0.8},
		{"vcpkg.json", "C++", 0.8},

		// Scala
		{"build.sbt", "Scala", 0.95},
		{"project/build.properties", "Scala", 0.7},

		// Elixir
		{"mix.exs", "Elixir", 0.95},
		{"mix.lock", "Elixir", 0.8},

		// Haskell
		{"stack.yaml", "Haskell", 0.9},
		{"*.cabal", "Haskell", 0.9},
		{"cabal.project", "Haskell", 0.8},

		// Clojure
		{"project.clj", "Clojure", 0.95},
		{"deps.edn", "Clojure", 0.9},

		// Swift
		{"Package.swift", "Swift", 0.95},
		{"*.xcodeproj", "Swift", 0.8},
		{"*.xcworkspace", "Swift", 0.8},

		// Dart/Flutter
		{"pubspec.yaml", "Dart", 0.95},

		// Lua
		{"*.rockspec", "Lua", 0.8},

		// Zig
		{"build.zig", "Zig", 0.95},

		// Nim
		{"*.nimble", "Nim", 0.9},

		// OCaml
		{"dune-project", "OCaml", 0.9},
		{"*.opam", "OCaml", 0.8},

		// Erlang
		{"rebar.config", "Erlang", 0.9},
	}

	langScores := make(map[string]float64)
	langIndicators := make(map[string][]string)

	for _, sig := range signatures {
		if d.fileExists(sig.file) {
			langScores[sig.language] += sig.weight
			langIndicators[sig.language] = append(langIndicators[sig.language], sig.file)
		}
	}

	// Convert to LanguageInfo
	for lang, score := range langScores {
		// Normalize score
		confidence := score
		if confidence > 1 {
			confidence = 1
		}

		d.info.Languages = append(d.info.Languages, LanguageInfo{
			Name:       lang,
			Confidence: confidence,
			Indicators: langIndicators[lang],
		})
	}
}

// analyzeContent performs deep content analysis
func (d *Detector) analyzeContent() {
	// Analyze package.json for frameworks
	if d.fileExists("package.json") {
		d.analyzePackageJSON()
	}

	// Analyze Python dependencies
	if d.fileExists("requirements.txt") {
		d.analyzeRequirementsTxt()
	}
	if d.fileExists("pyproject.toml") {
		d.analyzePyprojectToml()
	}

	// Analyze Go modules
	if d.fileExists("go.mod") {
		d.analyzeGoMod()
	}

	// Analyze Cargo.toml
	if d.fileExists("Cargo.toml") {
		d.analyzeCargoToml()
	}
}

// analyzePackageJSON extracts framework and dependency info
func (d *Detector) analyzePackageJSON() {
	data, err := os.ReadFile(filepath.Join(d.projectDir, "package.json"))
	if err != nil {
		return
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
		Workspaces      interface{}       `json:"workspaces"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return
	}

	allDeps := make(map[string]string)
	for k, v := range pkg.Dependencies {
		allDeps[k] = v
	}
	for k, v := range pkg.DevDependencies {
		allDeps[k] = v
	}

	// Framework detection
	frameworks := map[string]string{
		"react":            "React",
		"next":             "Next.js",
		"vue":              "Vue",
		"nuxt":             "Nuxt",
		"@angular/core":    "Angular",
		"svelte":           "Svelte",
		"@sveltejs/kit":    "SvelteKit",
		"express":          "Express",
		"@nestjs/core":     "NestJS",
		"fastify":          "Fastify",
		"koa":              "Koa",
		"hono":             "Hono",
		"electron":         "Electron",
		"@remix-run/react": "Remix",
		"gatsby":           "Gatsby",
		"astro":            "Astro",
	}

	for dep, framework := range frameworks {
		if _, ok := allDeps[dep]; ok {
			d.info.Frameworks = append(d.info.Frameworks, framework)
		}
	}

	// Package manager detection
	if d.fileExists("pnpm-lock.yaml") {
		d.info.PackageManagers = append(d.info.PackageManagers, "pnpm")
	} else if d.fileExists("yarn.lock") {
		d.info.PackageManagers = append(d.info.PackageManagers, "yarn")
	} else if d.fileExists("bun.lockb") {
		d.info.PackageManagers = append(d.info.PackageManagers, "bun")
	} else if d.fileExists("package-lock.json") {
		d.info.PackageManagers = append(d.info.PackageManagers, "npm")
	}

	// Workspaces detection (monorepo)
	if pkg.Workspaces != nil {
		d.info.IsMonorepo = true
		d.info.MonorepoType = "npm-workspaces"
	}

	// Collect dependencies
	for dep := range pkg.Dependencies {
		d.info.Dependencies = append(d.info.Dependencies, dep)
		if len(d.info.Dependencies) >= 20 {
			break
		}
	}
}

// analyzeRequirementsTxt extracts Python dependency info
func (d *Detector) analyzeRequirementsTxt() {
	data, err := os.ReadFile(filepath.Join(d.projectDir, "requirements.txt"))
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")

	frameworks := map[string]string{
		"django":    "Django",
		"flask":     "Flask",
		"fastapi":   "FastAPI",
		"starlette": "Starlette",
		"tornado":   "Tornado",
		"pyramid":   "Pyramid",
		"bottle":    "Bottle",
		"streamlit": "Streamlit",
		"gradio":    "Gradio",
	}

	gpuPackages := map[string]string{
		"torch":        "PyTorch",
		"tensorflow":   "TensorFlow",
		"jax":          "JAX",
		"cupy":         "CuPy",
		"rapids":       "RAPIDS",
		"mxnet":        "MXNet",
		"paddlepaddle": "PaddlePaddle",
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Extract package name
		pkg := strings.Split(line, "==")[0]
		pkg = strings.Split(pkg, ">=")[0]
		pkg = strings.Split(pkg, "<=")[0]
		pkg = strings.Split(pkg, "[")[0]
		pkg = strings.ToLower(strings.TrimSpace(pkg))

		// Framework detection
		for key, framework := range frameworks {
			if strings.Contains(pkg, key) {
				d.info.Frameworks = append(d.info.Frameworks, framework)
			}
		}

		// GPU detection
		for key, gpuFramework := range gpuPackages {
			if strings.Contains(pkg, key) {
				d.info.NeedsGPU = true
				d.info.GPUFrameworks = append(d.info.GPUFrameworks, gpuFramework)
			}
		}

		if len(d.info.Dependencies) < 20 {
			d.info.Dependencies = append(d.info.Dependencies, pkg)
		}
	}
}

// analyzePyprojectToml analyzes Python pyproject.toml
func (d *Detector) analyzePyprojectToml() {
	data, err := os.ReadFile(filepath.Join(d.projectDir, "pyproject.toml"))
	if err != nil {
		return
	}

	content := string(data)

	// Detect build system
	if strings.Contains(content, "[tool.poetry]") {
		d.info.PackageManagers = append(d.info.PackageManagers, "poetry")
	}
	if strings.Contains(content, "[tool.pdm]") {
		d.info.PackageManagers = append(d.info.PackageManagers, "pdm")
	}
	if strings.Contains(content, "[tool.hatch]") {
		d.info.PackageManagers = append(d.info.PackageManagers, "hatch")
	}

	// GPU detection via dependencies
	gpuPatterns := []string{"torch", "tensorflow", "jax", "cuda", "cupy"}
	for _, pattern := range gpuPatterns {
		if strings.Contains(strings.ToLower(content), pattern) {
			d.info.NeedsGPU = true
			break
		}
	}
}

// analyzeGoMod analyzes Go module
func (d *Detector) analyzeGoMod() {
	data, err := os.ReadFile(filepath.Join(d.projectDir, "go.mod"))
	if err != nil {
		return
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	// Extract Go version
	for _, line := range lines {
		if strings.HasPrefix(line, "go ") {
			d.info.Versions["go"] = strings.TrimSpace(strings.TrimPrefix(line, "go "))
			break
		}
	}

	// Framework detection
	frameworks := map[string]string{
		"github.com/gin-gonic/gin": "Gin",
		"github.com/labstack/echo": "Echo",
		"github.com/gofiber/fiber": "Fiber",
		"github.com/go-chi/chi":    "Chi",
		"github.com/gorilla/mux":   "Gorilla",
		"github.com/beego/beego":   "Beego",
		"github.com/revel/revel":   "Revel",
	}

	for dep, framework := range frameworks {
		if strings.Contains(content, dep) {
			d.info.Frameworks = append(d.info.Frameworks, framework)
		}
	}
}

// analyzeCargoToml analyzes Rust Cargo.toml
func (d *Detector) analyzeCargoToml() {
	data, err := os.ReadFile(filepath.Join(d.projectDir, "Cargo.toml"))
	if err != nil {
		return
	}

	content := string(data)

	// Check for workspace (monorepo)
	if strings.Contains(content, "[workspace]") {
		d.info.IsMonorepo = true
		d.info.MonorepoType = "cargo-workspace"
	}

	// Framework detection
	frameworks := map[string]string{
		"actix-web": "Actix",
		"axum":      "Axum",
		"rocket":    "Rocket",
		"warp":      "Warp",
		"tide":      "Tide",
		"tauri":     "Tauri",
		"yew":       "Yew",
		"leptos":    "Leptos",
	}

	for dep, framework := range frameworks {
		if strings.Contains(content, dep) {
			d.info.Frameworks = append(d.info.Frameworks, framework)
		}
	}

	// GPU detection
	if strings.Contains(content, "cuda") || strings.Contains(content, "cudarc") {
		d.info.NeedsGPU = true
	}
}

// detectVersions detects specific language versions
func (d *Detector) detectVersions() {
	// Node.js version
	if data, err := os.ReadFile(filepath.Join(d.projectDir, ".nvmrc")); err == nil {
		d.info.Versions["node"] = strings.TrimSpace(string(data))
	}
	if data, err := os.ReadFile(filepath.Join(d.projectDir, ".node-version")); err == nil {
		d.info.Versions["node"] = strings.TrimSpace(string(data))
	}

	// Python version
	if data, err := os.ReadFile(filepath.Join(d.projectDir, ".python-version")); err == nil {
		d.info.Versions["python"] = strings.TrimSpace(string(data))
	}

	// Ruby version
	if data, err := os.ReadFile(filepath.Join(d.projectDir, ".ruby-version")); err == nil {
		d.info.Versions["ruby"] = strings.TrimSpace(string(data))
	}

	// Rust version
	if data, err := os.ReadFile(filepath.Join(d.projectDir, "rust-toolchain.toml")); err == nil {
		content := string(data)
		re := regexp.MustCompile(`channel\s*=\s*["']([^"']+)["']`)
		if matches := re.FindStringSubmatch(content); len(matches) > 1 {
			d.info.Versions["rust"] = matches[1]
		}
	}
	if data, err := os.ReadFile(filepath.Join(d.projectDir, "rust-toolchain")); err == nil {
		d.info.Versions["rust"] = strings.TrimSpace(string(data))
	}

	// Java version from .java-version or pom.xml
	if data, err := os.ReadFile(filepath.Join(d.projectDir, ".java-version")); err == nil {
		d.info.Versions["java"] = strings.TrimSpace(string(data))
	}
}

// detectGPURequirements checks for GPU-related dependencies
func (d *Detector) detectGPURequirements() {
	// Already partially done in content analysis
	// Additional checks:

	// Check for .cuda_version file
	if data, err := os.ReadFile(filepath.Join(d.projectDir, ".cuda_version")); err == nil {
		d.info.CUDAVersion = strings.TrimSpace(string(data))
		d.info.NeedsGPU = true
	}

	// Check environment.yml for CUDA
	if data, err := os.ReadFile(filepath.Join(d.projectDir, "environment.yml")); err == nil {
		content := strings.ToLower(string(data))
		if strings.Contains(content, "cuda") || strings.Contains(content, "pytorch") ||
			strings.Contains(content, "tensorflow") {
			d.info.NeedsGPU = true
		}
	}

	// Check for nvidia-docker or GPU-related docker-compose
	if data, err := os.ReadFile(filepath.Join(d.projectDir, "docker-compose.yml")); err == nil {
		content := strings.ToLower(string(data))
		if strings.Contains(content, "nvidia") || strings.Contains(content, "gpu") {
			d.info.NeedsGPU = true
		}
	}
}

// detectMonorepo detects monorepo structures
func (d *Detector) detectMonorepo() {
	// Already partially detected in content analysis
	// Additional monorepo tool detection

	// Turborepo
	if d.fileExists("turbo.json") {
		d.info.IsMonorepo = true
		d.info.MonorepoType = "turborepo"
	}

	// Nx
	if d.fileExists("nx.json") {
		d.info.IsMonorepo = true
		d.info.MonorepoType = "nx"
	}

	// Lerna
	if d.fileExists("lerna.json") {
		d.info.IsMonorepo = true
		d.info.MonorepoType = "lerna"
	}

	// Rush
	if d.fileExists("rush.json") {
		d.info.IsMonorepo = true
		d.info.MonorepoType = "rush"
	}

	// pnpm workspace
	if d.fileExists("pnpm-workspace.yaml") {
		d.info.IsMonorepo = true
		d.info.MonorepoType = "pnpm"
	}

	// If monorepo, detect services
	if d.info.IsMonorepo {
		d.detectServices()
	}
}

// detectServices detects services in a monorepo
func (d *Detector) detectServices() {
	// Common monorepo directories
	serviceDirs := []string{"apps", "packages", "services", "libs", "modules"}

	for _, dir := range serviceDirs {
		fullPath := filepath.Join(d.projectDir, dir)
		if _, err := os.Stat(fullPath); err == nil {
			entries, _ := os.ReadDir(fullPath)
			for _, entry := range entries {
				if entry.IsDir() {
					servicePath := filepath.Join(dir, entry.Name())
					serviceFullPath := filepath.Join(d.projectDir, servicePath)

					// Detect language for this service
					serviceDetector := NewDetector(serviceFullPath)
					serviceInfo, _ := serviceDetector.Detect()

					if len(serviceInfo.Languages) > 0 {
						d.info.Services = append(d.info.Services, ServiceInfo{
							Name:     entry.Name(),
							Path:     servicePath,
							Language: serviceInfo.PrimaryLanguage,
							Template: suggestTemplate(serviceInfo),
						})
					}
				}
			}
		}
	}
}

// detectExistingConfigs checks for existing configuration files
func (d *Detector) detectExistingConfigs() {
	d.info.HasDockerfile = d.fileExists("Dockerfile")
	d.info.HasDockerCompose = d.fileExists("docker-compose.yml") || d.fileExists("docker-compose.yaml")
	d.info.HasDevcontainer = d.fileExists(".devcontainer/devcontainer.json") || d.fileExists("devcontainer.json")
	d.info.HasMakefile = d.fileExists("Makefile")
}

// setPrimaryLanguage determines the primary language
func (d *Detector) setPrimaryLanguage() {
	if len(d.info.Languages) == 0 {
		d.info.PrimaryLanguage = "unknown"
		return
	}

	// Sort by confidence
	primary := d.info.Languages[0]
	for _, lang := range d.info.Languages {
		if lang.Confidence > primary.Confidence {
			primary = lang
		}
	}

	d.info.PrimaryLanguage = primary.Name
}

// collectRootFiles lists files in root directory
func (d *Detector) collectRootFiles() {
	entries, _ := os.ReadDir(d.projectDir)
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), ".") {
			d.info.RootFiles = append(d.info.RootFiles, entry.Name())
			if len(d.info.RootFiles) >= 30 {
				break
			}
		}
	}
}

// fileExists checks if a file exists (supports glob patterns)
func (d *Detector) fileExists(pattern string) bool {
	if strings.Contains(pattern, "*") {
		matches, _ := filepath.Glob(filepath.Join(d.projectDir, pattern))
		return len(matches) > 0
	}
	_, err := os.Stat(filepath.Join(d.projectDir, pattern))
	return err == nil
}

// suggestTemplate suggests a template based on project info
func suggestTemplate(info *ProjectInfo) string {
	// GPU templates
	if info.NeedsGPU {
		for _, fw := range info.GPUFrameworks {
			switch fw {
			case "PyTorch":
				return "pytorch"
			case "TensorFlow":
				return "tensorflow"
			case "JAX":
				return "jax-flax"
			}
		}
		return "pytorch" // default GPU template
	}

	// Framework-based
	for _, fw := range info.Frameworks {
		switch fw {
		case "Next.js":
			return "nextjs"
		case "React":
			return "react"
		case "Vue", "Nuxt":
			return "vue"
		case "Django":
			return "python-django"
		case "FastAPI":
			return "python-fastapi"
		case "Flask":
			return "python-flask"
		case "NestJS":
			return "nestjs"
		case "Express":
			return "node-express"
		}
	}

	// Language-based fallback
	switch info.PrimaryLanguage {
	case "Go":
		return "go-basic"
	case "Python":
		if containsAny(info.PackageManagers, "poetry") {
			return "python-poetry"
		}
		if containsAny(info.PackageManagers, "pdm") {
			return "python-pdm"
		}
		return "python-basic"
	case "JavaScript", "TypeScript":
		return "node-basic"
	case "Rust":
		return "rust-basic"
	case "Java":
		if containsAny(info.BuildTools, "gradle") {
			return "java-gradle"
		}
		return "java-maven"
	case ".NET/C#", ".NET/F#", ".NET":
		return "dotnet"
	case "Ruby":
		return "ruby-basic"
	case "PHP":
		return "php-composer"
	case "C++":
		if containsAny(info.BuildTools, "cmake") {
			return "cpp-cmake"
		}
		return "cpp-makefile"
	default:
		return "python-basic" // safe default
	}
}

// containsAny checks if slice contains any of the items
func containsAny(slice []string, items ...string) bool {
	for _, s := range slice {
		for _, item := range items {
			if strings.EqualFold(s, item) {
				return true
			}
		}
	}
	return false
}

// RecommendTemplates returns ranked template recommendations
// Uses the advanced TemplateScorer for weighted multi-factor matching
func (d *Detector) RecommendTemplates() []TemplateRecommendation {
	info := d.info
	if info == nil {
		return nil
	}

	// Use the advanced scorer
	scorer := NewTemplateScorer()
	scoredTemplates := scorer.ScoreTemplates(info)

	// Convert to TemplateRecommendation format
	var result []TemplateRecommendation
	for _, scored := range scoredTemplates {
		result = append(result, TemplateRecommendation{
			Template:   scored.Name,
			Score:      scored.Score,
			Confidence: scored.Confidence,
			Reasons:    scored.Reasons,
		})
	}

	// Fallback if no matches
	if len(result) == 0 {
		primaryTemplate := suggestTemplate(info)
		result = append(result, TemplateRecommendation{
			Template:   primaryTemplate,
			Score:      0.5,
			Confidence: "low",
			Reasons:    []string{"Fallback based on primary language"},
		})
	}

	return result
}
