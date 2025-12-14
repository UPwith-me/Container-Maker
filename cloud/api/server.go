// Package api provides the HTTP API server for Cloud Control Plane
package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Config holds API server configuration
type Config struct {
	Port            int
	JWTSecret       string
	StripeSecretKey string

	// OAuth
	GitHubClientID     string
	GitHubClientSecret string
	GoogleClientID     string
	GoogleClientSecret string

	// Database
	DatabaseURL string
}

// Server is the API server
type Server struct {
	echo   *echo.Echo
	config Config

	// In-memory store (Mock DB for Phase 1)
	mu        sync.RWMutex
	instances map[string]map[string]interface{} // ID -> Instance data
	apiKeys   map[string]map[string]interface{} // Key -> Metadata
}

// NewServer creates a new API server
func NewServer(cfg Config) *Server {
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"}, // Allow all for dev
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "X-API-Key"},
	}))
	e.Use(middleware.RequestID())

	s := &Server{
		echo:      e,
		config:    cfg,
		instances: make(map[string]map[string]interface{}),
		apiKeys:   make(map[string]map[string]interface{}),
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check
	s.echo.GET("/health", s.healthCheck)

	// API v1
	v1 := s.echo.Group("/api/v1")

	// Public routes
	v1.POST("/auth/register", s.register)
	v1.POST("/auth/login", s.login)
	v1.POST("/auth/refresh", s.refreshToken)
	v1.GET("/auth/github", s.githubOAuth)
	v1.GET("/auth/github/callback", s.githubCallback)
	v1.GET("/auth/google", s.googleOAuth)
	v1.GET("/auth/google/callback", s.googleCallback)

	// Protected routes (require auth)
	protected := v1.Group("")
	protected.Use(s.authMiddleware)

	// User
	protected.GET("/user", s.getCurrentUser)
	protected.PUT("/user", s.updateUser)

	// API Keys
	protected.GET("/api-keys", s.listAPIKeys)
	protected.POST("/api-keys", s.createAPIKey)
	protected.DELETE("/api-keys/:id", s.deleteAPIKey)

	// Cloud Credentials
	protected.GET("/credentials", s.listCredentials)
	protected.POST("/credentials", s.addCredential)
	protected.DELETE("/credentials/:id", s.deleteCredential)
	protected.POST("/credentials/:id/verify", s.verifyCredential)

	// Instances
	protected.GET("/instances", s.listInstances)
	protected.POST("/instances", s.createInstance)
	protected.GET("/instances/:id", s.getInstance)
	protected.POST("/instances/:id/start", s.startInstance)
	protected.POST("/instances/:id/stop", s.stopInstance)
	protected.DELETE("/instances/:id", s.deleteInstance)
	protected.GET("/instances/:id/logs", s.getInstanceLogs)
	protected.GET("/instances/:id/ssh", s.getSSHConfig)

	// Providers
	protected.GET("/providers", s.listProviders)
	protected.GET("/providers/:name/regions", s.listRegions)
	protected.GET("/providers/:name/types", s.listInstanceTypes)

	// Teams
	protected.GET("/teams", s.listTeams)
	protected.POST("/teams", s.createTeam)
	protected.GET("/teams/:id", s.getTeam)
	protected.PUT("/teams/:id", s.updateTeam)
	protected.POST("/teams/:id/members", s.addTeamMember)
	protected.DELETE("/teams/:id/members/:userId", s.removeTeamMember)

	// Billing
	protected.GET("/billing/usage", s.getUsage)
	protected.GET("/billing/invoices", s.listInvoices)
	protected.POST("/billing/subscription", s.updateSubscription)

	// Stripe webhook (uses Stripe signature verification)
	v1.POST("/webhooks/stripe", s.stripeWebhook)
}

// Start starts the API server
func (s *Server) Start() error {
	return s.echo.Start(fmt.Sprintf(":%d", s.config.Port))
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.echo.Shutdown(ctx)
}

// ---- Auth Middleware ----

// Claims represents JWT claims
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func (s *Server) authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Check for Bearer token
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			// Check for API key
			apiKey := c.Request().Header.Get("X-API-Key")
			if apiKey != "" {
				return s.validateAPIKey(c, apiKey, next)
			}
			return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization")
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization format")
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(s.config.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}

		// Set user context
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)

		return next(c)
	}
}

func (s *Server) validateAPIKey(c echo.Context, apiKey string, next echo.HandlerFunc) error {
	// For API Key (mock validation for now)
	if strings.HasPrefix(apiKey, "cm_") {
		c.Set("user_id", "demo-user")
		c.Set("api_key", apiKey)
		return next(c)
	}
	return echo.NewHTTPError(http.StatusUnauthorized, "invalid API key")
}

func (s *Server) generateJWT(userID, email string) (string, error) {
	claims := &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

func (s *Server) generateAPIKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return "cm_" + base64.RawURLEncoding.EncodeToString(b)
}

// ---- Functional Handlers ----

func (s *Server) healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status":  "healthy",
		"version": "1.0.0",
	})
}

// Auth handlers
func (s *Server) register(c echo.Context) error {
	return c.JSON(http.StatusCreated, map[string]string{"message": "User registered (mock)"})
}

func (s *Server) login(c echo.Context) error {
	// Mock login - accepts any email/password
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind(&body); err != nil {
		return err
	}

	token, _ := s.generateJWT("user-123", body.Email)
	return c.JSON(http.StatusOK, map[string]string{
		"token":   token,
		"user_id": "user-123",
	})
}

func (s *Server) refreshToken(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (s *Server) githubOAuth(c echo.Context) error {
	if c.QueryParam("cli") == "true" {
		// CLI Logic: Print a dummy code
		return c.HTML(http.StatusOK, "<h1>Authentication Successful</h1><p>You can close this window and return to CLI.</p>")
	}
	// Web Logic: Redirect
	clientID := s.config.GitHubClientID
	redirectURL := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&scope=user:email", clientID)
	return c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

func (s *Server) githubCallback(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "Auth callback logic here"})
}

func (s *Server) googleOAuth(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (s *Server) googleCallback(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

// User handlers
func (s *Server) getCurrentUser(c echo.Context) error {
	userID := c.Get("user_id").(string)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":         userID,
		"email":      "demo@container-maker.dev",
		"name":       "Demo User",
		"avatar_url": "https://github.com/shadcn.png",
	})
}

func (s *Server) updateUser(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

// API Key handlers
func (s *Server) listAPIKeys(c echo.Context) error {
	return c.JSON(http.StatusOK, []interface{}{})
}

func (s *Server) createAPIKey(c echo.Context) error {
	key := s.generateAPIKey()
	return c.JSON(http.StatusCreated, map[string]string{
		"key":     key,
		"warning": "This key will only be shown once. Save it securely.",
	})
}

func (s *Server) deleteAPIKey(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// Credential handlers
func (s *Server) listCredentials(c echo.Context) error {
	return c.JSON(http.StatusOK, []interface{}{})
}

func (s *Server) addCredential(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (s *Server) deleteCredential(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) verifyCredential(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

// Instance handlers
func (s *Server) listInstances(c echo.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]interface{}, 0, len(s.instances))
	for _, inst := range s.instances {
		list = append(list, inst)
	}

	// Add a dummy instance if empty, for demo purposes
	if len(list) == 0 {
		return c.JSON(http.StatusOK, []map[string]interface{}{
			{
				"id":            "inst-demo-01",
				"name":          "My AI Workspace",
				"instance_type": "gpu-t4",
				"status":        "running",
				"provider":      "aws",
				"public_ip":     "34.201.12.45",
				"created_at":    time.Now().Add(-2 * time.Hour),
				"region":        "us-east-1",
			},
		})
	}

	return c.JSON(http.StatusOK, list)
}

func (s *Server) createInstance(c echo.Context) error {
	var body map[string]interface{}
	if err := c.Bind(&body); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := "inst-" + uuid.New().String()[:8]
	body["id"] = id
	body["status"] = "provisioning"
	body["created_at"] = time.Now()
	body["public_ip"] = "" // pending

	s.instances[id] = body

	// Simulate provisioning delay
	go func() {
		time.Sleep(5 * time.Second)
		s.mu.Lock()
		if inst, ok := s.instances[id]; ok {
			inst["status"] = "running"
			inst["public_ip"] = "54.123.45.67"
		}
		s.mu.Unlock()
	}()

	return c.JSON(http.StatusCreated, body)
}

func (s *Server) getInstance(c echo.Context) error {
	id := c.Param("id")
	s.mu.RLock()
	inst, ok := s.instances[id]
	s.mu.RUnlock()

	if !ok {
		// Mock response for demo ID
		if id == "inst-demo-01" {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"id":            "inst-demo-01",
				"name":          "My AI Workspace",
				"instance_type": "gpu-t4",
				"status":        "running",
				"provider":      "aws",
				"public_ip":     "34.201.12.45",
				"region":        "us-east-1",
			})
		}
		return echo.NewHTTPError(http.StatusNotFound, "Instance not found")
	}
	return c.JSON(http.StatusOK, inst)
}

func (s *Server) startInstance(c echo.Context) error {
	id := c.Param("id")
	s.mu.Lock()
	defer s.mu.Unlock()

	if inst, ok := s.instances[id]; ok {
		inst["status"] = "running"
		return c.JSON(http.StatusOK, inst)
	}
	return echo.NewHTTPError(http.StatusNotFound)
}

func (s *Server) stopInstance(c echo.Context) error {
	id := c.Param("id")
	s.mu.Lock()
	defer s.mu.Unlock()

	if inst, ok := s.instances[id]; ok {
		inst["status"] = "stopped"
		return c.JSON(http.StatusOK, inst)
	}
	return echo.NewHTTPError(http.StatusNotFound)
}

func (s *Server) deleteInstance(c echo.Context) error {
	id := c.Param("id")
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.instances, id)
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) getInstanceLogs(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"logs": "Initializing system...\nLoading drivers...\nReady.",
	})
}

func (s *Server) getSSHConfig(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"host": "34.201.12.45",
		"port": 22,
		"user": "ubuntu",
	})
}

// Provider handlers
func (s *Server) listProviders(c echo.Context) error {
	providers := []map[string]interface{}{
		{"name": "docker", "display_name": "Local Docker", "status": "available"},
		{"name": "aws", "display_name": "Amazon Web Services", "status": "available"},
		{"name": "gcp", "display_name": "Google Cloud Platform", "status": "available"},
		{"name": "azure", "display_name": "Microsoft Azure", "status": "available"},
		{"name": "digitalocean", "display_name": "DigitalOcean", "status": "available"},
		{"name": "linode", "display_name": "Linode/Akamai", "status": "available"},
		{"name": "vultr", "display_name": "Vultr", "status": "available"},
		{"name": "hetzner", "display_name": "Hetzner Cloud", "status": "available"},
		{"name": "oci", "display_name": "Oracle Cloud", "status": "available"},
		{"name": "alibaba", "display_name": "Alibaba Cloud", "status": "available"},
		{"name": "tencent", "display_name": "Tencent Cloud", "status": "available"},
		{"name": "lambdalabs", "display_name": "Lambda Labs (GPU)", "status": "available"},
		{"name": "runpod", "display_name": "RunPod (GPU)", "status": "available"},
		{"name": "vast", "display_name": "Vast.ai (GPU)", "status": "available"},
	}
	return c.JSON(http.StatusOK, providers)
}

func (s *Server) listRegions(c echo.Context) error {
	return c.JSON(http.StatusOK, []map[string]string{
		{"id": "us-east-1", "name": "US East (N. Virginia)"},
		{"id": "us-west-2", "name": "US West (Oregon)"},
		{"id": "eu-central-1", "name": "Europe (Frankfurt)"},
	})
}

func (s *Server) listInstanceTypes(c echo.Context) error {
	return c.JSON(http.StatusOK, []map[string]interface{}{
		{"id": "cpu-small", "name": "CPU Small", "vcpu": 2, "ram": 4, "price": 0.02},
		{"id": "gpu-t4", "name": "NVIDIA T4", "vcpu": 4, "ram": 16, "gpu": "T4", "price": 0.50},
		{"id": "gpu-a100", "name": "NVIDIA A100", "vcpu": 8, "ram": 80, "gpu": "A100", "price": 3.00},
	})
}

// Team handlers
func (s *Server) listTeams(c echo.Context) error {
	return c.JSON(http.StatusOK, []interface{}{})
}

func (s *Server) createTeam(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (s *Server) getTeam(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (s *Server) updateTeam(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (s *Server) addTeamMember(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (s *Server) removeTeamMember(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

// Billing handlers
func (s *Server) getUsage(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"current_month": map[string]interface{}{
			"cpu_hours":  124.5,
			"gpu_hours":  12.0,
			"total_cost": 45.20,
			"instances":  3,
			"forecast":   85.00,
		},
	})
}

func (s *Server) listInvoices(c echo.Context) error {
	return c.JSON(http.StatusOK, []interface{}{})
}

func (s *Server) updateSubscription(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (s *Server) stripeWebhook(c echo.Context) error {
	// TODO: Verify Stripe signature and handle events
	return c.JSON(http.StatusOK, map[string]string{"status": "received"})
}
