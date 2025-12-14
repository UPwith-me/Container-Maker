// Package api provides admin configuration API handlers
package api

import (
	"net/http"

	"github.com/UPwith-me/Container-Maker/cloud/db"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// AdminConfig represents the admin configuration response
type AdminConfig struct {
	// OAuth
	GitHubClientID   string `json:"github_client_id"`
	GitHubConfigured bool   `json:"github_configured"`
	GoogleClientID   string `json:"google_client_id"`
	GoogleConfigured bool   `json:"google_configured"`

	// Stripe
	StripePublishableKey string `json:"stripe_publishable_key"`
	StripeConfigured     bool   `json:"stripe_configured"`
}

// getAdminConfig returns current admin configuration
func (s *Server) getAdminConfig(c echo.Context) error {
	config := AdminConfig{}

	// Get GitHub config
	if cfg, err := s.db.GetConfig(db.ConfigGitHubClientID); err == nil {
		config.GitHubClientID = cfg.Value
		config.GitHubConfigured = cfg.Value != ""
	}

	// Get Google config
	if cfg, err := s.db.GetConfig(db.ConfigGoogleClientID); err == nil {
		config.GoogleClientID = cfg.Value
		config.GoogleConfigured = cfg.Value != ""
	}

	// Get Stripe config (only publishable key is shown)
	if cfg, err := s.db.GetConfig(db.ConfigStripePublishable); err == nil {
		config.StripePublishableKey = cfg.Value
		config.StripeConfigured = cfg.Value != ""
	}

	return c.JSON(http.StatusOK, config)
}

// updateAdminConfig updates admin configuration
func (s *Server) updateAdminConfig(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var req struct {
		GitHubClientID     string `json:"github_client_id"`
		GitHubClientSecret string `json:"github_client_secret"`
		GoogleClientID     string `json:"google_client_id"`
		GoogleClientSecret string `json:"google_client_secret"`
		StripePublishable  string `json:"stripe_publishable_key"`
		StripeSecret       string `json:"stripe_secret_key"`
		StripeWebhook      string `json:"stripe_webhook_secret"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	// Save GitHub OAuth
	if req.GitHubClientID != "" {
		s.db.SetConfig(db.ConfigGitHubClientID, req.GitHubClientID, false, "GitHub OAuth Client ID", userID)
	}
	if req.GitHubClientSecret != "" {
		s.db.SetConfig(db.ConfigGitHubClientSecret, req.GitHubClientSecret, true, "GitHub OAuth Client Secret", userID)
	}

	// Save Google OAuth
	if req.GoogleClientID != "" {
		s.db.SetConfig(db.ConfigGoogleClientID, req.GoogleClientID, false, "Google OAuth Client ID", userID)
	}
	if req.GoogleClientSecret != "" {
		s.db.SetConfig(db.ConfigGoogleClientSecret, req.GoogleClientSecret, true, "Google OAuth Client Secret", userID)
	}

	// Save Stripe
	if req.StripePublishable != "" {
		s.db.SetConfig(db.ConfigStripePublishable, req.StripePublishable, false, "Stripe Publishable Key", userID)
	}
	if req.StripeSecret != "" {
		s.db.SetConfig(db.ConfigStripeSecret, req.StripeSecret, true, "Stripe Secret Key", userID)
		// Also update in-memory config for immediate use
		s.config.StripeSecretKey = req.StripeSecret
	}
	if req.StripeWebhook != "" {
		s.db.SetConfig(db.ConfigStripeWebhook, req.StripeWebhook, true, "Stripe Webhook Secret", userID)
	}

	// Update OAuth configs in memory
	if req.GitHubClientID != "" && req.GitHubClientSecret != "" {
		s.config.GitHubClientID = req.GitHubClientID
		s.config.GitHubClientSecret = req.GitHubClientSecret
	}
	if req.GoogleClientID != "" && req.GoogleClientSecret != "" {
		s.config.GoogleClientID = req.GoogleClientID
		s.config.GoogleClientSecret = req.GoogleClientSecret
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Admin configuration updated successfully",
		"updated": true,
	})
}

// Unused but useful for generating IDs
var _ = uuid.New
