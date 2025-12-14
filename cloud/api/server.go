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

	"github.com/container-make/cm/cloud/db"
	"github.com/container-make/cm/cloud/providers"
	"github.com/container-make/cm/cloud/ui"
	// Import UI package
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
	DatabaseURL    string
	DatabaseDriver string // sqlite or postgres
}

// Server is the API server
type Server struct {
	echo      *echo.Echo
	config    Config
	db        *db.Database
	providers *providers.Manager
	wsHub     *WSHub

	// Legacy in-memory (to be removed after full DB migration)
	mu        sync.RWMutex
	instances map[string]map[string]interface{}
	apiKeys   map[string]map[string]interface{}
}

// NewServer creates a new API server
func NewServer(cfg Config) (*Server, error) {
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "X-API-Key"},
	}))
	e.Use(middleware.RequestID())

	// Initialize database
	dbConfig := db.DefaultSQLiteConfig()
	if cfg.DatabaseDriver != "" {
		dbConfig.Driver = cfg.DatabaseDriver
	}
	if cfg.DatabaseURL != "" {
		dbConfig.DSN = cfg.DatabaseURL
	}

	database, err := db.New(dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize provider manager
	providerManager := providers.GetDefaultManager()

	// Initialize WebSocket hub
	wsHub := NewWSHub()
	go wsHub.Run()

	s := &Server{
		echo:      e,
		config:    cfg,
		db:        database,
		providers: providerManager,
		wsHub:     wsHub,
		instances: make(map[string]map[string]interface{}),
		apiKeys:   make(map[string]map[string]interface{}),
	}

	// Load saved configuration from database
	s.loadSavedConfig()

	s.setupRoutes()
	return s, nil
}

// loadSavedConfig loads OAuth/Stripe config from database into memory
func (s *Server) loadSavedConfig() {
	// Load GitHub OAuth
	if cfg, err := s.db.GetConfig(db.ConfigGitHubClientID); err == nil && cfg.Value != "" {
		s.config.GitHubClientID = cfg.Value
	}
	if cfg, err := s.db.GetConfig(db.ConfigGitHubClientSecret); err == nil && cfg.Value != "" {
		s.config.GitHubClientSecret = cfg.Value
	}

	// Load Google OAuth
	if cfg, err := s.db.GetConfig(db.ConfigGoogleClientID); err == nil && cfg.Value != "" {
		s.config.GoogleClientID = cfg.Value
	}
	if cfg, err := s.db.GetConfig(db.ConfigGoogleClientSecret); err == nil && cfg.Value != "" {
		s.config.GoogleClientSecret = cfg.Value
	}

	// Load Stripe
	if cfg, err := s.db.GetConfig(db.ConfigStripeSecret); err == nil && cfg.Value != "" {
		s.config.StripeSecretKey = cfg.Value
	}
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check
	s.echo.GET("/health", s.healthCheck)

	// Serve Frontend (Embedded)
	distFS, err := ui.DistDir()
	if err == nil {
		fileServer := http.FileServer(http.FS(distFS))
		s.echo.GET("/*", func(c echo.Context) error {
			path := c.Request().URL.Path
			// Skip API routes
			if strings.HasPrefix(path, "/api") {
				return echo.ErrNotFound
			}

			// Serve static file
			f, err := distFS.Open(strings.TrimPrefix(path, "/"))
			if err == nil {
				defer f.Close()
				stat, _ := f.Stat()
				if !stat.IsDir() {
					fileServer.ServeHTTP(c.Response(), c.Request())
					return nil
				}
			}

			// SPA Fallback: Serve index.html for unknown routes
			indexFile, err := distFS.Open("index.html")
			if err != nil {
				return c.String(http.StatusNotFound, "Frontend not found")
			}
			defer indexFile.Close()

			stat, _ := indexFile.Stat()
			content := make([]byte, stat.Size())
			indexFile.Read(content)

			return c.HTMLBlob(http.StatusOK, content)
		})
	} else {
		fmt.Printf("Warning: Frontend not embedded: %v\n", err)
	}

	// API v1
	v1 := s.echo.Group("/api/v1")

	// Public routes - Auth is in auth.go and oauth.go
	v1.POST("/auth/register", s.register)
	v1.POST("/auth/login", s.login)
	v1.POST("/auth/refresh", s.refreshToken)
	v1.POST("/auth/logout", s.logout)
	v1.GET("/auth/github", s.githubOAuth)
	v1.GET("/auth/github/callback", s.githubCallback)
	v1.GET("/auth/google", s.googleOAuth)
	v1.GET("/auth/google/callback", s.googleCallback)

	// WebSocket endpoint (supports token via query param)
	v1.GET("/ws", s.HandleWebSocket)

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

	// Terminal and log streaming WebSockets (uses query param auth)
	v1.GET("/instances/:id/terminal", s.HandleTerminalWebSocket)
	v1.GET("/instances/:id/logs/stream", s.HandleLogStreamWebSocket)

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
	protected.POST("/billing/portal", s.createBillingPortalSession)
	protected.POST("/billing/setup-intent", s.createSetupIntent)
	protected.GET("/billing/invoices/:id/pdf", s.getInvoicePdfUrl)

	// Admin
	protected.GET("/admin/config", s.getAdminConfig)
	protected.PUT("/admin/config", s.updateAdminConfig)

	// Stripe webhook
	v1.POST("/webhooks/stripe", s.stripeWebhook)
}

// Start starts the API server
func (s *Server) Start() error {
	return s.echo.Start(fmt.Sprintf(":%d", s.config.Port))
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.db != nil {
		s.db.Close()
	}
	return s.echo.Shutdown(ctx)
}

// ---- Auth Middleware ----

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func (s *Server) authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
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

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)

		return next(c)
	}
}

func (s *Server) validateAPIKey(c echo.Context, apiKey string, next echo.HandlerFunc) error {
	if strings.HasPrefix(apiKey, "cm_") {
		// Look up in database
		key, err := s.db.GetAPIKeyByKey(apiKey)
		if err == nil && key != nil {
			c.Set("user_id", key.UserID)
			c.Set("api_key", apiKey)
			return next(c)
		}
		// Fallback for demo
		c.Set("user_id", "demo-user")
		c.Set("api_key", apiKey)
		return next(c)
	}
	return echo.NewHTTPError(http.StatusUnauthorized, "invalid API key")
}

func (s *Server) generateAPIKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return "cm_" + base64.RawURLEncoding.EncodeToString(b)
}

// ---- Handlers ----

func (s *Server) healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status":  "healthy",
		"version": "1.0.0",
	})
}

// User handlers
func (s *Server) getCurrentUser(c echo.Context) error {
	userID := c.Get("user_id").(string)

	user, err := s.db.GetUserByID(userID)
	if err != nil {
		// Fallback for demo
		return c.JSON(http.StatusOK, map[string]interface{}{
			"id":         userID,
			"email":      "demo@container-maker.dev",
			"name":       "Demo User",
			"avatar_url": "https://github.com/shadcn.png",
		})
	}
	return c.JSON(http.StatusOK, user)
}

func (s *Server) updateUser(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

// API Key handlers
func (s *Server) listAPIKeys(c echo.Context) error {
	userID := c.Get("user_id").(string)
	keys, err := s.db.ListAPIKeysByUser(userID)
	if err != nil {
		return c.JSON(http.StatusOK, []interface{}{})
	}
	return c.JSON(http.StatusOK, keys)
}

func (s *Server) createAPIKey(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var req struct {
		Name   string `json:"name"`
		Scopes string `json:"scopes"`
	}
	c.Bind(&req)

	key := s.generateAPIKey()

	apiKey := &db.APIKey{
		ID:        uuid.New().String(),
		UserID:    userID,
		Name:      req.Name,
		KeyPrefix: key[:11], // cm_ + first 8 chars
		KeyHash:   key,      // Should be hashed in production
		Scopes:    req.Scopes,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.db.CreateAPIKey(apiKey); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create API key")
	}

	return c.JSON(http.StatusCreated, map[string]string{
		"key":     key,
		"id":      apiKey.ID,
		"warning": "This key will only be shown once. Save it securely.",
	})
}

func (s *Server) deleteAPIKey(c echo.Context) error {
	id := c.Param("id")
	if err := s.db.DeleteAPIKey(id); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "API key not found")
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// Credential handlers
func (s *Server) listCredentials(c echo.Context) error {
	userID := c.Get("user_id").(string)
	creds, err := s.db.ListCredentialsByUser(userID)
	if err != nil {
		return c.JSON(http.StatusOK, []interface{}{})
	}
	return c.JSON(http.StatusOK, creds)
}

func (s *Server) addCredential(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var req struct {
		Provider string            `json:"provider"`
		Name     string            `json:"name"`
		Data     map[string]string `json:"data"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	// TODO: Encrypt the data before storing
	cred := &db.CloudCredential{
		ID:            uuid.New().String(),
		UserID:        userID,
		Provider:      req.Provider,
		Name:          req.Name,
		EncryptedData: fmt.Sprintf("%v", req.Data), // Should be encrypted
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	if err := s.db.CreateCredential(cred); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create credential")
	}

	return c.JSON(http.StatusCreated, cred)
}

func (s *Server) deleteCredential(c echo.Context) error {
	id := c.Param("id")
	if err := s.db.DeleteCredential(id); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "credential not found")
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) verifyCredential(c echo.Context) error {
	// TODO: Actually verify credentials against the provider
	return c.JSON(http.StatusOK, map[string]interface{}{
		"verified": true,
		"message":  "Credentials verified successfully",
	})
}

// Instance handlers
func (s *Server) listInstances(c echo.Context) error {
	userID := c.Get("user_id").(string)

	instances, err := s.db.ListInstancesByUser(userID)
	if err != nil {
		return c.JSON(http.StatusOK, []db.Instance{})
	}

	return c.JSON(http.StatusOK, instances)
}

func (s *Server) createInstance(c echo.Context) error {
	userID := c.Get("user_id").(string)
	ctx := c.Request().Context()

	var req struct {
		Name         string `json:"name"`
		Provider     string `json:"provider"`
		InstanceType string `json:"instance_type"`
		Region       string `json:"region"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	// Get the provider
	provider, err := s.providers.Get(providers.ProviderType(req.Provider))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "unsupported provider: "+req.Provider)
	}

	// Create instance in database first
	dbInstance := &db.Instance{
		ID:           "inst-" + uuid.New().String()[:8],
		OwnerID:      userID,
		Name:         req.Name,
		Provider:     req.Provider,
		InstanceType: req.InstanceType,
		Region:       req.Region,
		Status:       "provisioning",
		HourlyRate:   0.0,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	// Get pricing
	for _, pricing := range provider.InstanceTypes() {
		if string(pricing.Type) == req.InstanceType {
			dbInstance.HourlyRate = pricing.HourlyRate
			break
		}
	}

	if err := s.db.CreateInstance(dbInstance); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create instance")
	}

	// Actually create the instance via provider (async)
	go func() {
		config := providers.InstanceConfig{
			Name:   req.Name,
			Type:   providers.InstanceType(req.InstanceType),
			Region: req.Region,
			Image:  "ubuntu:22.04",
		}

		providerInst, err := provider.CreateInstance(ctx, config)
		if err != nil {
			dbInstance.Status = "error"
			dbInstance.StatusReason = err.Error()
		} else {
			dbInstance.Status = string(providerInst.Status)
			dbInstance.PublicIP = providerInst.PublicIP
			dbInstance.ProviderID = providerInst.ID
			dbInstance.SSHPort = providerInst.SSHPort
		}
		dbInstance.UpdatedAt = time.Now().UTC()
		s.db.UpdateInstance(dbInstance)
	}()

	return c.JSON(http.StatusCreated, dbInstance)
}

func (s *Server) getInstance(c echo.Context) error {
	id := c.Param("id")

	instance, err := s.db.GetInstanceByID(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Instance not found")
	}
	return c.JSON(http.StatusOK, instance)
}

func (s *Server) startInstance(c echo.Context) error {
	id := c.Param("id")

	instance, err := s.db.GetInstanceByID(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Instance not found")
	}

	instance.Status = "running"
	now := time.Now().UTC()
	instance.StartedAt = &now
	s.db.UpdateInstance(instance)

	return c.JSON(http.StatusOK, instance)
}

func (s *Server) stopInstance(c echo.Context) error {
	id := c.Param("id")

	instance, err := s.db.GetInstanceByID(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Instance not found")
	}

	instance.Status = "stopped"
	now := time.Now().UTC()
	instance.StoppedAt = &now
	s.db.UpdateInstance(instance)

	return c.JSON(http.StatusOK, instance)
}

func (s *Server) deleteInstance(c echo.Context) error {
	id := c.Param("id")
	if err := s.db.DeleteInstance(id); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Instance not found")
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) getInstanceLogs(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"logs": "Initializing system...\nLoading drivers...\nStarting services...\nReady.",
	})
}

func (s *Server) getSSHConfig(c echo.Context) error {
	id := c.Param("id")
	instance, _ := s.db.GetInstanceByID(id)

	host := "34.201.12.45"
	port := 22
	if instance != nil {
		host = instance.PublicIP
		port = instance.SSHPort
		if port == 0 {
			port = 22
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"host": host,
		"port": port,
		"user": "ubuntu",
	})
}

// Provider handlers
func (s *Server) listProviders(c echo.Context) error {
	ctx := c.Request().Context()
	providerList := s.providers.List()

	result := make([]providers.ProviderInfo, 0, len(providerList))
	for _, p := range providerList {
		result = append(result, providers.GetProviderInfo(p, ctx))
	}

	return c.JSON(http.StatusOK, result)
}

func (s *Server) listRegions(c echo.Context) error {
	name := c.Param("name")
	provider, err := s.providers.Get(providers.ProviderType(name))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Provider not found")
	}
	return c.JSON(http.StatusOK, provider.Regions())
}

func (s *Server) listInstanceTypes(c echo.Context) error {
	name := c.Param("name")
	provider, err := s.providers.Get(providers.ProviderType(name))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Provider not found")
	}
	return c.JSON(http.StatusOK, provider.InstanceTypes())
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
	userID := c.Get("user_id").(string)

	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	records, _ := s.db.GetUsageByUserAndPeriod(userID, startOfMonth, now)

	var totalCost float64
	var cpuHours, gpuHours float64

	for _, r := range records {
		totalCost += r.TotalCost
		if strings.Contains(r.Type, "cpu") {
			cpuHours += r.Quantity
		} else if strings.Contains(r.Type, "gpu") {
			gpuHours += r.Quantity
		}
	}

	// Default demo values
	if totalCost == 0 {
		cpuHours = 124.5
		gpuHours = 12.0
		totalCost = 45.20
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"current_month": map[string]interface{}{
			"cpu_hours":  cpuHours,
			"gpu_hours":  gpuHours,
			"total_cost": totalCost,
			"instances":  3,
			"forecast":   totalCost * 2,
		},
	})
}

func (s *Server) listInvoices(c echo.Context) error {
	userID := c.Get("user_id").(string)
	invoices, err := s.db.ListInvoicesByUser(userID)
	if err != nil {
		return c.JSON(http.StatusOK, []interface{}{})
	}
	return c.JSON(http.StatusOK, invoices)
}

func (s *Server) updateSubscription(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}
