package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// LocalProvider represents a local AI model provider
type LocalProvider interface {
	IsAvailable() bool
	Generate(ctx context.Context, prompt string) (string, error)
	GetModel() string
}

// OllamaProvider provides access to Ollama local models
type OllamaProvider struct {
	endpoint string
	model    string
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider() *OllamaProvider {
	return &OllamaProvider{
		endpoint: "http://localhost:11434",
		model:    "", // Will be auto-detected
	}
}

// NewOllamaProviderWithModel creates a new Ollama provider with a specific model
func NewOllamaProviderWithModel(model string) *OllamaProvider {
	return &OllamaProvider{
		endpoint: "http://localhost:11434",
		model:    model,
	}
}

// IsAvailable checks if Ollama is running
func (o *OllamaProvider) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", o.endpoint+"/api/tags", nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

// GetModel returns the current model name
func (o *OllamaProvider) GetModel() string {
	if o.model != "" {
		return o.model
	}
	return o.detectBestModel()
}

// detectBestModel finds the best available model for code generation
func (o *OllamaProvider) detectBestModel() string {
	models, err := o.ListModels()
	if err != nil {
		return "llama3.2"
	}

	// Priority list of models for code generation
	preferredModels := []string{
		"deepseek-coder",
		"codellama",
		"qwen2.5-coder",
		"starcoder2",
		"codegemma",
		"llama3.2",
		"mistral",
		"qwen2.5",
	}

	for _, preferred := range preferredModels {
		for _, available := range models {
			if strings.Contains(strings.ToLower(available), preferred) {
				o.model = available
				return available
			}
		}
	}

	// Return first available model
	if len(models) > 0 {
		o.model = models[0]
		return models[0]
	}

	return "llama3.2"
}

// ListModels returns all available models
func (o *OllamaProvider) ListModels() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", o.endpoint+"/api/tags", nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var models []string
	for _, m := range result.Models {
		models = append(models, m.Name)
	}

	return models, nil
}

// Generate generates text using Ollama
func (o *OllamaProvider) Generate(ctx context.Context, prompt string) (string, error) {
	model := o.GetModel()

	reqBody := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": 0.3,
			"num_predict": 4096,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.endpoint+"/api/generate", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("ollama error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Response string `json:"response"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.Response, nil
}

// Chat sends a chat-style message to Ollama
func (o *OllamaProvider) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	model := o.GetModel()

	var ollamaMessages []map[string]string
	for _, m := range messages {
		ollamaMessages = append(ollamaMessages, map[string]string{
			"role":    m.Role,
			"content": m.Content,
		})
	}

	reqBody := map[string]interface{}{
		"model":    model,
		"messages": ollamaMessages,
		"stream":   false,
		"options": map[string]interface{}{
			"temperature": 0.3,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.endpoint+"/api/chat", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama chat failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("ollama error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.Message.Content, nil
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string `json:"role"` // "system", "user", "assistant"
	Content string `json:"content"`
}

// PullModel pulls a model from Ollama registry
func (o *OllamaProvider) PullModel(ctx context.Context, modelName string) error {
	fmt.Printf("📥 Pulling model '%s' from Ollama...\n", modelName)

	reqBody := map[string]interface{}{
		"name":   modelName,
		"stream": false,
	}

	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", o.endpoint+"/api/pull", bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to pull model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pull failed: %s", string(body))
	}

	fmt.Println("✅ Model pulled successfully")
	return nil
}

// EnsureOllamaRunning starts Ollama if not running
func EnsureOllamaRunning() error {
	provider := NewOllamaProvider()
	if provider.IsAvailable() {
		return nil
	}

	fmt.Println("🔄 Starting Ollama...")

	// Try to start ollama serve in background
	cmd := exec.Command("ollama", "serve")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ollama: %w. Install from https://ollama.ai", err)
	}

	// Wait for it to become available
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		if provider.IsAvailable() {
			fmt.Println("✅ Ollama started")
			return nil
		}
	}

	return fmt.Errorf("ollama failed to start within timeout")
}

// AIBackend represents available AI backends
type AIBackend string

const (
	BackendOpenAI AIBackend = "openai"
	BackendOllama AIBackend = "ollama"
	BackendRules  AIBackend = "rules"
)

// SmartGenerator uses the best available AI backend
type SmartGenerator struct {
	backend AIBackend
	openai  *Generator
	ollama  *OllamaProvider
}

// NewSmartGenerator creates a generator that uses the best available backend
func NewSmartGenerator() (*SmartGenerator, error) {
	sg := &SmartGenerator{}

	// Try OpenAI first
	if gen, err := NewGenerator(); err == nil {
		sg.backend = BackendOpenAI
		sg.openai = gen
		return sg, nil
	}

	// Try Ollama
	ollama := NewOllamaProvider()
	if ollama.IsAvailable() {
		sg.backend = BackendOllama
		sg.ollama = ollama
		return sg, nil
	}

	// Fall back to rules engine
	sg.backend = BackendRules
	return sg, nil
}

// GetBackend returns the current backend
func (sg *SmartGenerator) GetBackend() AIBackend {
	return sg.backend
}

// GenerateConfig generates a devcontainer.json config
func (sg *SmartGenerator) GenerateConfig(ctx context.Context, projectDir string) (string, error) {
	switch sg.backend {
	case BackendOpenAI:
		return sg.openai.AnalyzeProject(ctx, projectDir)
	case BackendOllama:
		return sg.generateWithOllama(ctx, projectDir)
	case BackendRules:
		return sg.generateWithRules(projectDir)
	}
	return "", fmt.Errorf("no AI backend available")
}

// generateWithOllama uses Ollama for config generation
func (sg *SmartGenerator) generateWithOllama(ctx context.Context, projectDir string) (string, error) {
	// Collect project info (reusing generator logic)
	gen := &Generator{}
	info := gen.collectProjectInfo(projectDir)
	prompt := gen.buildPrompt(info)

	// Add system context
	systemPrompt := `You are an expert DevOps engineer. Generate valid devcontainer.json configurations.
Return ONLY valid JSON, no explanation or markdown code blocks.`

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: prompt},
	}

	response, err := sg.ollama.Chat(ctx, messages)
	if err != nil {
		return "", err
	}

	// Clean up response
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	// Validate JSON
	var js json.RawMessage
	if err := json.Unmarshal([]byte(response), &js); err != nil {
		return "", fmt.Errorf("invalid JSON from Ollama: %w", err)
	}

	return response, nil
}

// generateWithRules uses the rule engine without AI
func (sg *SmartGenerator) generateWithRules(projectDir string) (string, error) {
	engine := NewRuleEngine()
	gen := &Generator{}
	info := gen.collectProjectInfo(projectDir)

	return engine.Generate(info)
}
