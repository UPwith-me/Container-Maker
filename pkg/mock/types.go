// Package mock provides service mocking capabilities for Container-Maker.
// It enables running mock versions of services for development and testing.
package mock

import (
	"net/http"
	"time"
)

// MockMode defines the type of mock to use
type MockMode string

const (
	ModeDynamic  MockMode = "dynamic"  // Dynamic responses based on rules
	ModeStatic   MockMode = "static"   // Static response files
	ModeRecord   MockMode = "record"   // Record real responses
	ModeReplay   MockMode = "replay"   // Replay recorded responses
	ModeContract MockMode = "contract" // Contract testing mode
)

// MockConfig defines mock service configuration
type MockConfig struct {
	Name    string   `yaml:"name" json:"name"`
	Service string   `yaml:"service" json:"service"` // Service to mock
	Mode    MockMode `yaml:"mode" json:"mode"`
	Port    int      `yaml:"port" json:"port"`
	Host    string   `yaml:"host,omitempty" json:"host,omitempty"`

	// Endpoint definitions
	Endpoints []EndpointConfig `yaml:"endpoints,omitempty" json:"endpoints,omitempty"`

	// Static file configuration
	StaticDir string `yaml:"static_dir,omitempty" json:"static_dir,omitempty"`

	// Recording configuration
	RecordDir  string `yaml:"record_dir,omitempty" json:"record_dir,omitempty"`
	RecordHost string `yaml:"record_host,omitempty" json:"record_host,omitempty"`

	// Contract configuration
	ContractFile string `yaml:"contract_file,omitempty" json:"contract_file,omitempty"`

	// General options
	Latency LatencyConfig     `yaml:"latency,omitempty" json:"latency,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	CORS    *CORSConfig       `yaml:"cors,omitempty" json:"cors,omitempty"`
}

// EndpointConfig defines a mock endpoint
type EndpointConfig struct {
	Path   string `yaml:"path" json:"path"`
	Method string `yaml:"method,omitempty" json:"method,omitempty"` // GET, POST, etc.

	// Response configuration
	Status   int               `yaml:"status,omitempty" json:"status,omitempty"`
	Body     string            `yaml:"body,omitempty" json:"body,omitempty"`
	BodyFile string            `yaml:"body_file,omitempty" json:"body_file,omitempty"`
	Headers  map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`

	// Dynamic response
	Template string         `yaml:"template,omitempty" json:"template,omitempty"`
	Rules    []ResponseRule `yaml:"rules,omitempty" json:"rules,omitempty"`

	// Latency override
	Latency *LatencyConfig `yaml:"latency,omitempty" json:"latency,omitempty"`

	// Behavior
	Passthrough bool `yaml:"passthrough,omitempty" json:"passthrough,omitempty"`
	Record      bool `yaml:"record,omitempty" json:"record,omitempty"`
}

// ResponseRule defines conditional response logic
type ResponseRule struct {
	Condition string            `yaml:"condition" json:"condition"` // Expression to evaluate
	Status    int               `yaml:"status,omitempty" json:"status,omitempty"`
	Body      string            `yaml:"body,omitempty" json:"body,omitempty"`
	Headers   map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
}

// LatencyConfig defines artificial latency
type LatencyConfig struct {
	Min    time.Duration `yaml:"min,omitempty" json:"min,omitempty"`
	Max    time.Duration `yaml:"max,omitempty" json:"max,omitempty"`
	Fixed  time.Duration `yaml:"fixed,omitempty" json:"fixed,omitempty"`
	Jitter float64       `yaml:"jitter,omitempty" json:"jitter,omitempty"` // 0-1 random factor
}

// CORSConfig defines CORS settings
type CORSConfig struct {
	Enabled          bool     `yaml:"enabled" json:"enabled"`
	AllowOrigins     []string `yaml:"allow_origins,omitempty" json:"allow_origins,omitempty"`
	AllowMethods     []string `yaml:"allow_methods,omitempty" json:"allow_methods,omitempty"`
	AllowHeaders     []string `yaml:"allow_headers,omitempty" json:"allow_headers,omitempty"`
	AllowCredentials bool     `yaml:"allow_credentials,omitempty" json:"allow_credentials,omitempty"`
}

// Contract defines a service contract for testing
type Contract struct {
	Name         string        `yaml:"name" json:"name"`
	Version      string        `yaml:"version,omitempty" json:"version,omitempty"`
	Provider     string        `yaml:"provider" json:"provider"`
	Consumer     string        `yaml:"consumer" json:"consumer"`
	Interactions []Interaction `yaml:"interactions" json:"interactions"`
}

// Interaction defines a request-response interaction
type Interaction struct {
	Description string   `yaml:"description" json:"description"`
	Request     Request  `yaml:"request" json:"request"`
	Response    Response `yaml:"response" json:"response"`
}

// Request defines an expected request
type Request struct {
	Method  string            `yaml:"method" json:"method"`
	Path    string            `yaml:"path" json:"path"`
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Query   map[string]string `yaml:"query,omitempty" json:"query,omitempty"`
	Body    interface{}       `yaml:"body,omitempty" json:"body,omitempty"`
}

// Response defines an expected response
type Response struct {
	Status  int               `yaml:"status" json:"status"`
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Body    interface{}       `yaml:"body,omitempty" json:"body,omitempty"`
}

// MockServer is the interface for mock servers
type MockServer interface {
	// Start starts the mock server
	Start() error

	// Stop stops the mock server
	Stop() error

	// IsRunning returns true if server is running
	IsRunning() bool

	// GetPort returns the server port
	GetPort() int

	// GetHandler returns the HTTP handler
	GetHandler() http.Handler

	// AddEndpoint adds a mock endpoint dynamically
	AddEndpoint(endpoint EndpointConfig) error

	// RemoveEndpoint removes a mock endpoint
	RemoveEndpoint(path, method string) error

	// GetStats returns mock statistics
	GetStats() *MockStats
}

// MockStats contains mock server statistics
type MockStats struct {
	Requests       int64            `json:"requests"`
	RequestsByPath map[string]int64 `json:"requests_by_path"`
	AverageLatency time.Duration    `json:"average_latency"`
	Errors         int64            `json:"errors"`
	LastRequest    time.Time        `json:"last_request,omitempty"`
}

// Recording represents a recorded request-response pair
type Recording struct {
	Timestamp time.Time        `json:"timestamp"`
	Request   RecordedRequest  `json:"request"`
	Response  RecordedResponse `json:"response"`
	Latency   time.Duration    `json:"latency"`
}

// RecordedRequest represents a recorded HTTP request
type RecordedRequest struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Headers map[string][]string `json:"headers"`
	Body    []byte              `json:"body,omitempty"`
}

// RecordedResponse represents a recorded HTTP response
type RecordedResponse struct {
	Status  int                 `json:"status"`
	Headers map[string][]string `json:"headers"`
	Body    []byte              `json:"body,omitempty"`
}

// ContractVerifier verifies service contracts
type ContractVerifier interface {
	// Verify verifies a service against a contract
	Verify(contract *Contract, serviceURL string) (*VerificationResult, error)

	// VerifyFile verifies using a contract file
	VerifyFile(contractPath, serviceURL string) (*VerificationResult, error)
}

// VerificationResult contains contract verification results
type VerificationResult struct {
	Passed       bool                `json:"passed"`
	Contract     string              `json:"contract"`
	Provider     string              `json:"provider"`
	Consumer     string              `json:"consumer"`
	Interactions []InteractionResult `json:"interactions"`
	Duration     time.Duration       `json:"duration"`
}

// InteractionResult contains the result of verifying an interaction
type InteractionResult struct {
	Description string `json:"description"`
	Passed      bool   `json:"passed"`
	Error       string `json:"error,omitempty"`
	Request     struct {
		Sent    bool `json:"sent"`
		Matched bool `json:"matched"`
	} `json:"request"`
	Response struct {
		Received bool   `json:"received"`
		Matched  bool   `json:"matched"`
		Diff     string `json:"diff,omitempty"`
	} `json:"response"`
}
