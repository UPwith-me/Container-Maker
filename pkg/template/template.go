package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Template represents a devcontainer template
type Template struct {
	Name        string                 `json:"name"`
	Category    string                 `json:"category"`
	Description string                 `json:"description"`
	Image       string                 `json:"image"`
	Features    map[string]interface{} `json:"features,omitempty"`
	RunArgs     []string               `json:"runArgs,omitempty"`
	Mounts      []string               `json:"mounts,omitempty"`
	Extensions  []string               `json:"extensions,omitempty"`
	PostCreate  string                 `json:"postCreateCommand,omitempty"`
	IsCustom    bool                   `json:"isCustom,omitempty"`
}

// BuiltInTemplates returns all built-in templates
func BuiltInTemplates() map[string]*Template {
	return map[string]*Template{
		// Go templates
		"go-basic": {
			Name:        "go-basic",
			Category:    "Go",
			Description: "Go 基础开发环境",
			Image:       "golang:1.21-alpine",
			PostCreate:  "go mod download",
		},
		"go-api": {
			Name:        "go-api",
			Category:    "Go",
			Description: "Go API 服务开发 (含 hot-reload)",
			Image:       "golang:1.21",
			Features: map[string]interface{}{
				"ghcr.io/devcontainers/features/go:1": map[string]string{"version": "1.21"},
			},
			PostCreate: "go install github.com/cosmtrek/air@latest && go mod download",
		},

		// Python templates
		"python-basic": {
			Name:        "python-basic",
			Category:    "Python",
			Description: "Python 基础环境",
			Image:       "python:3.11-slim",
			PostCreate:  "pip install --upgrade pip",
		},
		"python-ml": {
			Name:        "python-ml",
			Category:    "Python",
			Description: "Python 机器学习 (含 Jupyter)",
			Image:       "python:3.11",
			PostCreate:  "pip install numpy pandas matplotlib scikit-learn jupyter",
		},

		// Node templates
		"node-basic": {
			Name:        "node-basic",
			Category:    "Node.js",
			Description: "Node.js 基础环境",
			Image:       "node:20-alpine",
			PostCreate:  "npm install",
		},
		"node-fullstack": {
			Name:        "node-fullstack",
			Category:    "Node.js",
			Description: "全栈开发环境",
			Image:       "node:20",
			PostCreate:  "npm install",
		},

		// Rust template
		"rust-basic": {
			Name:        "rust-basic",
			Category:    "Rust",
			Description: "Rust 开发环境",
			Image:       "rust:alpine",
			PostCreate:  "cargo fetch",
		},

		// C++ template
		"cpp-cmake": {
			Name:        "cpp-cmake",
			Category:    "C++",
			Description: "C++ CMake 项目",
			Image:       "gcc:latest",
			PostCreate:  "apt-get update && apt-get install -y cmake",
		},

		// Deep Learning / AI templates
		"pytorch": {
			Name:        "pytorch",
			Category:    "Deep Learning",
			Description: "PyTorch 深度学习 (GPU支持)",
			Image:       "pytorch/pytorch:2.1.0-cuda12.1-cudnn8-runtime",
			RunArgs:     []string{"--gpus", "all"},
			PostCreate:  "pip install transformers datasets accelerate wandb",
		},
		"tensorflow": {
			Name:        "tensorflow",
			Category:    "Deep Learning",
			Description: "TensorFlow 深度学习 (GPU支持)",
			Image:       "tensorflow/tensorflow:2.15.0-gpu",
			RunArgs:     []string{"--gpus", "all"},
			PostCreate:  "pip install keras tensorboard",
		},
		"huggingface": {
			Name:        "huggingface",
			Category:    "Deep Learning",
			Description: "HuggingFace 模型微调环境",
			Image:       "pytorch/pytorch:2.1.0-cuda12.1-cudnn8-runtime",
			RunArgs:     []string{"--gpus", "all"},
			PostCreate:  "pip install transformers datasets peft accelerate bitsandbytes trl wandb",
		},
		"llm-finetune": {
			Name:        "llm-finetune",
			Category:    "Deep Learning",
			Description: "大语言模型微调 (LoRA/QLoRA)",
			Image:       "pytorch/pytorch:2.1.0-cuda12.1-cudnn8-runtime",
			RunArgs:     []string{"--gpus", "all", "--shm-size=8g"},
			PostCreate:  "pip install transformers datasets peft accelerate bitsandbytes trl wandb deepspeed",
		},

		// Reinforcement Learning template
		"rl-gym": {
			Name:        "rl-gym",
			Category:    "Deep Learning",
			Description: "强化学习 (Gymnasium + Stable-Baselines3)",
			Image:       "pytorch/pytorch:2.1.0-cuda12.1-cudnn8-runtime",
			RunArgs:     []string{"--gpus", "all"},
			PostCreate:  "pip install gymnasium stable-baselines3 sb3-contrib tensorboard wandb pygame",
		},

		// JAX/Flax for ML research
		"jax-flax": {
			Name:        "jax-flax",
			Category:    "Deep Learning",
			Description: "JAX/Flax ML研究环境",
			Image:       "nvidia/cuda:12.1.0-cudnn8-devel-ubuntu22.04",
			RunArgs:     []string{"--gpus", "all"},
			PostCreate:  "pip install jax[cuda12_pip] flax optax orbax-checkpoint chex wandb -f https://storage.googleapis.com/jax-releases/jax_cuda_releases.html",
		},

		// Computer Vision with Detectron2
		"cv-detectron": {
			Name:        "cv-detectron",
			Category:    "Deep Learning",
			Description: "计算机视觉 (Detectron2)",
			Image:       "pytorch/pytorch:2.1.0-cuda12.1-cudnn8-devel",
			RunArgs:     []string{"--gpus", "all", "--shm-size=8g"},
			PostCreate:  "pip install opencv-python-headless albumentations timm && pip install 'git+https://github.com/facebookresearch/detectron2.git'",
		},

		// Diffusion Models (Stable Diffusion)
		"diffusion": {
			Name:        "diffusion",
			Category:    "Deep Learning",
			Description: "扩散模型 (Stable Diffusion/SDXL)",
			Image:       "pytorch/pytorch:2.1.0-cuda12.1-cudnn8-runtime",
			RunArgs:     []string{"--gpus", "all", "--shm-size=16g"},
			PostCreate:  "pip install diffusers transformers accelerate safetensors xformers wandb",
		},

		// NLP with spaCy
		"nlp-spacy": {
			Name:        "nlp-spacy",
			Category:    "Python",
			Description: "NLP开发 (spaCy + transformers)",
			Image:       "python:3.11",
			PostCreate:  "pip install spacy transformers datasets nltk gensim sentence-transformers && python -m spacy download en_core_web_sm",
		},

		// MLOps environment
		"mlops": {
			Name:        "mlops",
			Category:    "Python",
			Description: "MLOps工具链 (MLflow + DVC)",
			Image:       "python:3.11",
			PostCreate:  "pip install mlflow dvc boto3 hydra-core omegaconf pytorch-lightning wandb",
		},

		// === Complex Python Environments ===
		"miniconda": {
			Name:        "miniconda",
			Category:    "Python",
			Description: "Miniconda 数据科学环境",
			Image:       "mcr.microsoft.com/devcontainers/miniconda:3",
			PostCreate:  "if [ -f environment.yml ]; then conda env update -f environment.yml; elif [ -f requirements.txt ]; then pip install -r requirements.txt; fi",
		},
		"python-poetry": {
			Name:        "python-poetry",
			Category:    "Python",
			Description: "Poetry 现代Python包管理",
			Image:       "mcr.microsoft.com/devcontainers/python:3.11",
			PostCreate:  "pip install poetry && poetry install --no-interaction",
		},
		"python-pipenv": {
			Name:        "python-pipenv",
			Category:    "Python",
			Description: "Pipenv 虚拟环境管理",
			Image:       "mcr.microsoft.com/devcontainers/python:3.11",
			PostCreate:  "pip install pipenv && pipenv install --dev",
		},

		// === C/C++ Advanced Build Systems ===
		"cpp-conan": {
			Name:        "cpp-conan",
			Category:    "C++",
			Description: "C++ Conan 包管理器",
			Image:       "mcr.microsoft.com/devcontainers/cpp:ubuntu",
			PostCreate:  "pip install conan && conan profile detect --force && if [ -f conanfile.txt ]; then conan install . --build=missing; fi",
		},
		"cpp-vcpkg": {
			Name:        "cpp-vcpkg",
			Category:    "C++",
			Description: "C++ Vcpkg 包管理器",
			Image:       "mcr.microsoft.com/devcontainers/cpp:ubuntu",
			Features: map[string]interface{}{
				"ghcr.io/devcontainers/features/vcpkg:1": map[string]string{},
			},
			PostCreate: "if [ -f vcpkg.json ]; then vcpkg install; fi",
		},
		"cpp-makefile": {
			Name:        "cpp-makefile",
			Category:    "C++",
			Description: "C++ Makefile 项目",
			Image:       "gcc:latest",
			PostCreate:  "apt-get update && apt-get install -y build-essential gdb",
		},

		// === Java Build Systems ===
		"java-maven": {
			Name:        "java-maven",
			Category:    "Java",
			Description: "Java Maven 项目",
			Image:       "mcr.microsoft.com/devcontainers/java:17",
			Features: map[string]interface{}{
				"ghcr.io/devcontainers/features/java:1": map[string]string{"version": "17", "installMaven": "true"},
			},
			PostCreate: "if [ -f pom.xml ]; then mvn dependency:resolve; fi",
		},
		"java-gradle": {
			Name:        "java-gradle",
			Category:    "Java",
			Description: "Java Gradle 项目",
			Image:       "mcr.microsoft.com/devcontainers/java:17",
			Features: map[string]interface{}{
				"ghcr.io/devcontainers/features/java:1": map[string]string{"version": "17", "installGradle": "true"},
			},
			PostCreate: "if [ -f build.gradle ]; then gradle dependencies; fi",
		},

		// === .NET ===
		"dotnet": {
			Name:        "dotnet",
			Category:    ".NET",
			Description: ".NET 8.0 开发环境",
			Image:       "mcr.microsoft.com/devcontainers/dotnet:8.0",
			PostCreate:  "dotnet restore",
		},

		// === PHP ===
		"php-composer": {
			Name:        "php-composer",
			Category:    "PHP",
			Description: "PHP Composer 项目",
			Image:       "mcr.microsoft.com/devcontainers/php:8.2",
			PostCreate:  "if [ -f composer.json ]; then composer install; fi",
		},

		// === Ruby ===
		"ruby-basic": {
			Name:        "ruby-basic",
			Category:    "Ruby",
			Description: "Ruby Bundler 项目",
			Image:       "ruby:3.2-slim",
			PostCreate:  "if [ -f Gemfile ]; then bundle install; fi",
		},
	}
}

// GetTemplatesDir returns the path to custom templates directory
func GetTemplatesDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cm", "templates")
}

// LoadCustomTemplates loads user's custom templates
func LoadCustomTemplates() (map[string]*Template, error) {
	templatesDir := GetTemplatesDir()
	templates := make(map[string]*Template)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return templates, nil
	}

	// Read all JSON files
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return templates, nil
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(templatesDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var t Template
		if err := json.Unmarshal(data, &t); err != nil {
			continue
		}

		t.IsCustom = true
		name := strings.TrimSuffix(entry.Name(), ".json")
		templates[name] = &t
	}

	return templates, nil
}

// GetAllTemplates returns both built-in and custom templates
func GetAllTemplates() map[string]*Template {
	templates := BuiltInTemplates()
	custom, _ := LoadCustomTemplates()

	for name, t := range custom {
		templates[name] = t
	}

	return templates
}

// GetTemplate returns a template by name
func GetTemplate(name string) (*Template, bool) {
	templates := GetAllTemplates()
	t, ok := templates[name]
	return t, ok
}

// ListTemplates returns a formatted list of all templates
func ListTemplates() string {
	templates := GetAllTemplates()

	// Group by category
	categories := make(map[string][]*Template)
	for _, t := range templates {
		cat := t.Category
		if t.IsCustom {
			cat = "Custom"
		}
		categories[cat] = append(categories[cat], t)
	}

	// Sort categories
	var cats []string
	for cat := range categories {
		cats = append(cats, cat)
	}
	sort.Strings(cats)

	// Move "Custom" to end
	for i, cat := range cats {
		if cat == "Custom" {
			cats = append(cats[:i], cats[i+1:]...)
			cats = append(cats, "Custom")
			break
		}
	}

	var sb strings.Builder
	sb.WriteString("📦 Container-Make Templates\n\n")

	for _, cat := range cats {
		sb.WriteString(fmt.Sprintf("  %s:\n", cat))

		// Sort templates by name
		ts := categories[cat]
		sort.Slice(ts, func(i, j int) bool {
			return ts[i].Name < ts[j].Name
		})

		for _, t := range ts {
			sb.WriteString(fmt.Sprintf("    %-15s %s\n", t.Name, t.Description))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Usage: cm template use <name>\n")

	return sb.String()
}

// ApplyTemplate creates devcontainer.json from a template
func ApplyTemplate(name, targetDir string) error {
	t, ok := GetTemplate(name)
	if !ok {
		return fmt.Errorf("template '%s' not found", name)
	}

	// Create .devcontainer directory
	devcontainerDir := filepath.Join(targetDir, ".devcontainer")
	if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
		return err
	}

	// Build devcontainer.json content
	config := map[string]interface{}{
		"name":  t.Name,
		"image": t.Image,
	}

	if len(t.Features) > 0 {
		config["features"] = t.Features
	}
	if len(t.RunArgs) > 0 {
		config["runArgs"] = t.RunArgs
	}
	if len(t.Mounts) > 0 {
		config["mounts"] = t.Mounts
	}
	if t.PostCreate != "" {
		config["postCreateCommand"] = t.PostCreate
	}

	// Write JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	configPath := filepath.Join(devcontainerDir, "devcontainer.json")
	return os.WriteFile(configPath, data, 0644)
}

// SaveTemplate saves the current devcontainer.json as a custom template
func SaveTemplate(name, sourceDir string) error {
	// Read current devcontainer.json
	configPath := filepath.Join(sourceDir, ".devcontainer", "devcontainer.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("no devcontainer.json found: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("invalid devcontainer.json: %w", err)
	}

	// Create template
	t := &Template{
		Name:     name,
		Category: "Custom",
		IsCustom: true,
	}

	if img, ok := config["image"].(string); ok {
		t.Image = img
	}
	if desc, ok := config["name"].(string); ok {
		t.Description = desc
	} else {
		t.Description = "Custom template"
	}
	if features, ok := config["features"].(map[string]interface{}); ok {
		t.Features = features
	}
	if postCreate, ok := config["postCreateCommand"].(string); ok {
		t.PostCreate = postCreate
	}

	// Save to templates directory
	templatesDir := GetTemplatesDir()
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return err
	}

	templateData, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}

	templatePath := filepath.Join(templatesDir, name+".json")
	return os.WriteFile(templatePath, templateData, 0644)
}

// RemoveTemplate removes a custom template
func RemoveTemplate(name string) error {
	// Check if built-in
	builtIn := BuiltInTemplates()
	if _, ok := builtIn[name]; ok {
		return fmt.Errorf("'%s' is a built-in template and cannot be removed", name)
	}

	templatePath := filepath.Join(GetTemplatesDir(), name+".json")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return fmt.Errorf("template '%s' not found", name)
	}

	return os.Remove(templatePath)
}

// TemplateInfo returns detailed info about a template
func TemplateInfo(name string) (string, error) {
	t, ok := GetTemplate(name)
	if !ok {
		return "", fmt.Errorf("template '%s' not found", name)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 Template: %s\n", t.Name))
	sb.WriteString(fmt.Sprintf("   Category: %s\n", t.Category))
	sb.WriteString(fmt.Sprintf("   Description: %s\n", t.Description))
	sb.WriteString(fmt.Sprintf("   Image: %s\n", t.Image))

	if t.PostCreate != "" {
		sb.WriteString(fmt.Sprintf("   PostCreate: %s\n", t.PostCreate))
	}
	if len(t.Features) > 0 {
		sb.WriteString("   Features:\n")
		for f := range t.Features {
			sb.WriteString(fmt.Sprintf("     • %s\n", f))
		}
	}

	return sb.String(), nil
}

// RequiresGPU checks if a template requires GPU
func (t *Template) RequiresGPU() bool {
	// Check runArgs for GPU flags
	for _, arg := range t.RunArgs {
		if arg == "--gpus" || arg == "all" || strings.Contains(arg, "/dev/dri") {
			return true
		}
	}
	// Check category
	if t.Category == "Deep Learning" {
		return true
	}
	return false
}

// SearchOptions holds search filter options
type SearchOptions struct {
	Query    string
	Category string
	GPUOnly  bool
}

// SearchTemplates searches templates with filters
func SearchTemplates(opts SearchOptions) []*Template {
	templates := GetAllTemplates()
	var results []*Template

	query := strings.ToLower(opts.Query)
	category := strings.ToLower(opts.Category)

	for _, t := range templates {
		// Category filter
		if category != "" && strings.ToLower(t.Category) != category {
			continue
		}

		// GPU filter
		if opts.GPUOnly && !t.RequiresGPU() {
			continue
		}

		// Query filter (search in name, description, category)
		if query != "" {
			searchText := strings.ToLower(t.Name + " " + t.Description + " " + t.Category)
			if !strings.Contains(searchText, query) {
				continue
			}
		}

		results = append(results, t)
	}

	// Sort by name
	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	return results
}

// FormatSearchResults formats search results for display
func FormatSearchResults(results []*Template, query string) string {
	if len(results) == 0 {
		return fmt.Sprintf("No templates found matching '%s'\n", query)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📦 Found %d template(s)", len(results)))
	if query != "" {
		sb.WriteString(fmt.Sprintf(" matching '%s'", query))
	}
	sb.WriteString("\n\n")

	for _, t := range results {
		gpu := ""
		if t.RequiresGPU() {
			gpu = " 🎮"
		}
		sb.WriteString(fmt.Sprintf("  %-15s %s%s\n", t.Name, t.Description, gpu))
	}

	sb.WriteString("\nUsage: cm template use <name>\n")

	return sb.String()
}

// GetCategories returns all unique categories
func GetCategories() []string {
	templates := GetAllTemplates()
	catMap := make(map[string]bool)

	for _, t := range templates {
		catMap[t.Category] = true
	}

	var categories []string
	for cat := range catMap {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	return categories
}
