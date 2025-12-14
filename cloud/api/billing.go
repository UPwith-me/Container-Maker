// Package api provides billing-related API handlers
package api

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// Billing portal session - opens Stripe Customer Portal
func (s *Server) createBillingPortalSession(c echo.Context) error {
	// Check if Stripe is configured
	if s.config.StripeSecretKey == "" {
		return echo.NewHTTPError(http.StatusServiceUnavailable,
			"Stripe is not configured. Please add your Stripe API keys in Settings > Admin.")
	}

	// In production, this would:
	// 1. Get the user's Stripe customer ID from database
	// 2. Create a Stripe Billing Portal session
	// 3. Return the URL

	// For now, return a helpful message
	return c.JSON(http.StatusOK, map[string]interface{}{
		"url":     "https://billing.stripe.com/p/login/test",
		"message": "Stripe Customer Portal would open here",
	})
}

// Setup intent for adding new payment method
func (s *Server) createSetupIntent(c echo.Context) error {
	if s.config.StripeSecretKey == "" {
		return echo.NewHTTPError(http.StatusServiceUnavailable,
			"Stripe is not configured. Please add your Stripe API keys in Settings > Admin.")
	}

	// In production, this would create a Stripe SetupIntent
	// and return the client_secret for Stripe.js

	return c.JSON(http.StatusOK, map[string]interface{}{
		"client_secret": "seti_xxxxxxxxxxxxx_secret_xxxxxxxxxxxxx",
		"message":       "Setup intent would be created here",
	})
}

// Get invoice PDF URL
func (s *Server) getInvoicePdfUrl(c echo.Context) error {
	invoiceID := c.Param("id")

	// In production, this would:
	// 1. Look up the invoice in database
	// 2. Get the Stripe invoice URL
	// 3. Return or redirect to the PDF

	return c.JSON(http.StatusOK, map[string]interface{}{
		"url":        "https://pay.stripe.com/invoice/" + invoiceID + "/pdf",
		"invoice_id": invoiceID,
	})
}

// Updated getUsage with real calculation
func (s *Server) getUsageDetailed(c echo.Context) error {
	userID := c.Get("user_id").(string)

	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	records, _ := s.db.GetUsageByUserAndPeriod(userID, startOfMonth, now)

	var totalCost float64
	var cpuHours, gpuHours float64

	for _, r := range records {
		totalCost += r.TotalCost
		if r.Type == "cpu" {
			cpuHours += r.Quantity
		} else if r.Type == "gpu" {
			gpuHours += r.Quantity
		}
	}

	// Get active instances count
	instances, _ := s.db.ListInstancesByUser(userID)
	activeCount := 0
	for _, inst := range instances {
		if inst.Status == "running" {
			activeCount++
		}
	}

	// Calculate forecast based on current usage rate
	dayOfMonth := float64(now.Day())
	daysInMonth := float64(time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day())
	forecast := totalCost
	if dayOfMonth > 0 {
		forecast = (totalCost / dayOfMonth) * daysInMonth
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"current_month": map[string]interface{}{
			"cpu_hours":  cpuHours,
			"gpu_hours":  gpuHours,
			"total_cost": totalCost,
			"instances":  activeCount,
			"forecast":   forecast,
		},
	})
}

// List invoices from database (or Stripe)
func (s *Server) listInvoicesDetailed(c echo.Context) error {
	userID := c.Get("user_id").(string)

	invoices, err := s.db.ListInvoicesByUser(userID)
	if err != nil || len(invoices) == 0 {
		// Return empty array instead of error
		return c.JSON(http.StatusOK, []interface{}{})
	}

	// Transform to API format
	result := make([]map[string]interface{}, 0, len(invoices))
	for _, inv := range invoices {
		result = append(result, map[string]interface{}{
			"id":                inv.ID,
			"stripe_invoice_id": inv.StripeInvoiceID,
			"amount":            float64(inv.Total) / 100.0, // Convert cents to dollars
			"currency":          inv.Currency,
			"status":            inv.Status,
			"invoice_url":       inv.InvoiceURL,
			"period_start":      inv.PeriodStart,
			"period_end":        inv.PeriodEnd,
			"created_at":        inv.CreatedAt,
		})
	}

	return c.JSON(http.StatusOK, result)
}
