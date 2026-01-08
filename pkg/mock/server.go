package mock

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"
)

// HTTPServer implements MockServer
type HTTPServer struct {
	config    MockConfig
	server    *http.Server
	endpoints map[string]map[string]EndpointConfig // path -> method -> config
	mu        sync.RWMutex
	stats     *MockStats
}

// NewHTTPServer creates a new mock HTTP server
func NewHTTPServer(config MockConfig) *HTTPServer {
	s := &HTTPServer{
		config:    config,
		endpoints: make(map[string]map[string]EndpointConfig),
		stats: &MockStats{
			RequestsByPath: make(map[string]int64),
		},
	}

	// Register initial endpoints
	for _, ep := range config.Endpoints {
		s.addEndpoint(ep)
	}

	return s
}

// Start starts the mock server
func (s *HTTPServer) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.server = &http.Server{
		Addr:    addr,
		Handler: s.GetHandler(),
	}

	fmt.Printf("Mock server '%s' starting on %s\n", s.config.Name, addr)
	return s.server.ListenAndServe()
}

// Stop stops the mock server
func (s *HTTPServer) Stop() error {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}

// IsRunning returns true if server is running
func (s *HTTPServer) IsRunning() bool {
	return s.server != nil
}

// GetPort returns the server port
func (s *HTTPServer) GetPort() int {
	return s.config.Port
}

// GetHandler returns the HTTP handler
func (s *HTTPServer) GetHandler() http.Handler {
	return http.HandlerFunc(s.handleRequest)
}

// AddEndpoint adds a mock endpoint dynamically
func (s *HTTPServer) AddEndpoint(endpoint EndpointConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.addEndpoint(endpoint)
	return nil
}

func (s *HTTPServer) addEndpoint(ep EndpointConfig) {
	method := strings.ToUpper(ep.Method)
	if method == "" {
		method = "GET"
	}

	if s.endpoints[ep.Path] == nil {
		s.endpoints[ep.Path] = make(map[string]EndpointConfig)
	}
	s.endpoints[ep.Path][method] = ep
}

// RemoveEndpoint removes a mock endpoint
func (s *HTTPServer) RemoveEndpoint(path, method string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	method = strings.ToUpper(method)
	if s.endpoints[path] != nil {
		delete(s.endpoints[path], method)
		if len(s.endpoints[path]) == 0 {
			delete(s.endpoints, path)
		}
	}
	return nil
}

// GetStats returns mock statistics
func (s *HTTPServer) GetStats() *MockStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Return copy
	stats := *s.stats
	// Deep copy map needed if we want full isolation, but simple copy is ok for now
	return &stats
}

func (s *HTTPServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	s.mu.RLock()
	pathConfig, pathExists := s.endpoints[r.URL.Path]
	ep, methodExists := pathConfig[r.Method]
	s.mu.RUnlock()

	// Update stats
	s.mu.Lock()
	s.stats.Requests++
	s.stats.LastRequest = start
	s.stats.RequestsByPath[r.URL.Path]++
	s.mu.Unlock()

	// Handle CORS
	if s.config.CORS != nil && s.config.CORS.Enabled {
		if s.handleCORS(w, r) {
			return
		}
	}

	if !pathExists || !methodExists {
		// Try fallback to static files if enabled
		if s.config.StaticDir != "" {
			fs := http.FileServer(http.Dir(s.config.StaticDir))
			fs.ServeHTTP(w, r)
			return
		}

		http.NotFound(w, r)
		return
	}

	// Simulate latency
	s.simulateLatency(ep.Latency)

	// Determine response
	status := ep.Status
	if status == 0 {
		status = 200
	}

	body := ep.Body
	if ep.BodyFile != "" {
		if b, err := os.ReadFile(ep.BodyFile); err == nil {
			body = string(b)
		}
	}

	// Dynamic template processing
	if ep.Template != "" {
		tmpl, err := template.New("resp").Parse(ep.Template)
		if err == nil {
			var buf bytes.Buffer
			data := map[string]interface{}{
				"Query":   r.URL.Query(),
				"Headers": r.Header,
				"Path":    r.URL.Path,
			}
			if err := tmpl.Execute(&buf, data); err == nil {
				body = buf.String()
			}
		}
	}

	// Write headers
	for k, v := range s.config.Headers {
		w.Header().Set(k, v)
	}
	for k, v := range ep.Headers {
		w.Header().Set(k, v)
	}

	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))

	// Update latency stats
	duration := time.Since(start)
	s.mu.Lock()
	// Simple moving average for demo
	if s.stats.AverageLatency == 0 {
		s.stats.AverageLatency = duration
	} else {
		s.stats.AverageLatency = (s.stats.AverageLatency + duration) / 2
	}
	s.mu.Unlock()
}

func (s *HTTPServer) handleCORS(w http.ResponseWriter, r *http.Request) bool {
	// Add CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if len(s.config.CORS.AllowOrigins) > 0 {
		w.Header().Set("Access-Control-Allow-Origin", strings.Join(s.config.CORS.AllowOrigins, ","))
	}

	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(s.config.CORS.AllowMethods, ","))
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(s.config.CORS.AllowHeaders, ","))
		w.WriteHeader(http.StatusOK)
		return true
	}
	return false
}

func (s *HTTPServer) simulateLatency(l *LatencyConfig) {
	config := l
	if config == nil {
		config = &s.config.Latency
	}

	if config.Fixed > 0 {
		time.Sleep(config.Fixed)
		return
	}

	if config.Min > 0 && config.Max > config.Min {
		delta := int64(config.Max - config.Min)
		jitter := time.Duration(rand.Int63n(delta))
		time.Sleep(config.Min + jitter)
	}
}

// ContractVerifier implementation
type DefaultContractVerifier struct{}

func (v *DefaultContractVerifier) Verify(contract *Contract, serviceURL string) (*VerificationResult, error) {
	result := &VerificationResult{
		Contract: contract.Name,
		Provider: contract.Provider,
		Consumer: contract.Consumer,
		Passed:   true,
	}

	start := time.Now()

	for _, interaction := range contract.Interactions {
		res := v.verifyInteraction(interaction, serviceURL)
		result.Interactions = append(result.Interactions, res)
		if !res.Passed {
			result.Passed = false
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}

func (v *DefaultContractVerifier) VerifyFile(contractPath, serviceURL string) (*VerificationResult, error) {
	data, err := os.ReadFile(contractPath)
	if err != nil {
		return nil, err
	}

	var contract Contract
	if err := json.Unmarshal(data, &contract); err != nil {
		return nil, fmt.Errorf("failed to parse contract: %w", err)
	}

	return v.Verify(&contract, serviceURL)
}

func (v *DefaultContractVerifier) verifyInteraction(i Interaction, url string) InteractionResult {
	res := InteractionResult{
		Description: i.Description,
		Passed:      true,
	}

	target := url + i.Request.Path
	req, err := http.NewRequest(i.Request.Method, target, nil)
	if err != nil {
		res.Passed = false
		res.Error = err.Error()
		return res
	}

	// Set headers
	for k, v := range i.Request.Headers {
		req.Header.Set(k, v)
	}

	res.Request.Sent = true

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		res.Passed = false
		res.Error = err.Error()
		return res
	}
	defer resp.Body.Close()

	res.Response.Received = true

	// Verify status
	if resp.StatusCode != i.Response.Status {
		res.Passed = false
		res.Error = fmt.Sprintf("expected status %d, got %d", i.Response.Status, resp.StatusCode)
		return res
	}

	res.Response.Matched = true
	return res
}
