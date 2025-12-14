// Package api provides Stripe integration for billing
package api

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/UPwith-me/Container-Maker/cloud/db"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// StripeEvent represents a Stripe webhook event (simplified)
type StripeEvent struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Data    StripeEventData `json:"data"`
	Created int64           `json:"created"`
}

type StripeEventData struct {
	Object json.RawMessage `json:"object"`
}

// StripeInvoice represents a Stripe invoice object (simplified)
type StripeInvoice struct {
	ID               string `json:"id"`
	CustomerID       string `json:"customer"`
	AmountDue        int64  `json:"amount_due"`
	AmountPaid       int64  `json:"amount_paid"`
	Currency         string `json:"currency"`
	Status           string `json:"status"`
	HostedInvoiceURL string `json:"hosted_invoice_url"`
	PeriodStart      int64  `json:"period_start"`
	PeriodEnd        int64  `json:"period_end"`
}

// StripeCustomer represents a Stripe customer object (simplified)
type StripeCustomer struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// stripeWebhook handles Stripe webhook events
func (s *Server) stripeWebhook(c echo.Context) error {
	// Read body
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to read body")
	}

	// In production, verify Stripe signature
	// sig := c.Request().Header.Get("Stripe-Signature")
	// event, err := webhook.ConstructEvent(body, sig, s.config.StripeWebhookSecret)

	// Parse event
	var event StripeEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event")
	}

	// Handle different event types
	switch event.Type {
	case "invoice.paid":
		return s.handleInvoicePaid(event)
	case "invoice.payment_failed":
		return s.handleInvoiceFailed(event)
	case "customer.subscription.updated":
		return s.handleSubscriptionUpdated(event)
	case "customer.subscription.deleted":
		return s.handleSubscriptionDeleted(event)
	case "checkout.session.completed":
		return s.handleCheckoutCompleted(event)
	default:
		// Acknowledge unknown events
		return c.JSON(http.StatusOK, map[string]string{"status": "received"})
	}
}

func (s *Server) handleInvoicePaid(event StripeEvent) error {
	var invoice StripeInvoice
	if err := json.Unmarshal(event.Data.Object, &invoice); err != nil {
		return err
	}

	// Find user by Stripe customer ID
	user, err := s.db.GetUserByStripeCustomerID(invoice.CustomerID)
	if err != nil {
		// Customer not found, might be a new customer
		return nil
	}

	// Create or update invoice record
	dbInvoice := &db.Invoice{
		ID:              uuid.New().String(),
		UserID:          user.ID,
		Number:          invoice.ID,
		Status:          "paid",
		Subtotal:        invoice.AmountPaid,
		Total:           invoice.AmountPaid,
		AmountPaid:      invoice.AmountPaid,
		Currency:        invoice.Currency,
		StripeInvoiceID: invoice.ID,
		InvoiceURL:      invoice.HostedInvoiceURL,
		PeriodStart:     time.Unix(invoice.PeriodStart, 0),
		PeriodEnd:       time.Unix(invoice.PeriodEnd, 0),
		PaidAt:          timePtr(time.Now()),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	s.db.CreateInvoice(dbInvoice)
	return nil
}

func (s *Server) handleInvoiceFailed(event StripeEvent) error {
	var invoice StripeInvoice
	if err := json.Unmarshal(event.Data.Object, &invoice); err != nil {
		return err
	}

	// Find user and update invoice status
	user, err := s.db.GetUserByStripeCustomerID(invoice.CustomerID)
	if err != nil {
		return nil
	}

	dbInvoice := &db.Invoice{
		ID:              uuid.New().String(),
		UserID:          user.ID,
		Number:          invoice.ID,
		Status:          "failed",
		Subtotal:        invoice.AmountDue,
		Total:           invoice.AmountDue,
		AmountDue:       invoice.AmountDue,
		Currency:        invoice.Currency,
		StripeInvoiceID: invoice.ID,
		InvoiceURL:      invoice.HostedInvoiceURL,
		PeriodStart:     time.Unix(invoice.PeriodStart, 0),
		PeriodEnd:       time.Unix(invoice.PeriodEnd, 0),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	s.db.CreateInvoice(dbInvoice)

	// TODO: Send notification to user about failed payment
	// TODO: Consider suspending instances if payment repeatedly fails

	return nil
}

func (s *Server) handleSubscriptionUpdated(event StripeEvent) error {
	// Handle subscription changes (plan upgrade/downgrade)
	// In production, update user's subscription tier in database
	return nil
}

func (s *Server) handleSubscriptionDeleted(event StripeEvent) error {
	// Handle subscription cancellation
	// In production, downgrade user to free tier
	return nil
}

func (s *Server) handleCheckoutCompleted(event StripeEvent) error {
	// Handle successful checkout (new subscription)
	// In production, link Stripe customer to user
	return nil
}

// createStripeCustomer creates a Stripe customer (would use Stripe SDK in production)
func (s *Server) createStripeCustomer(user *db.User) (string, error) {
	// In production with Stripe SDK:
	// params := &stripe.CustomerParams{
	//     Email: stripe.String(user.Email),
	//     Name:  stripe.String(user.Name),
	// }
	// customer, err := customer.New(params)
	// return customer.ID, err

	// Simulated customer ID
	return "cus_" + uuid.New().String()[:8], nil
}

// createCheckoutSession creates a Stripe Checkout session
func (s *Server) createCheckoutSession(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var req struct {
		PriceID    string `json:"price_id"` // Stripe price ID
		SuccessURL string `json:"success_url"`
		CancelURL  string `json:"cancel_url"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	// Get or create customer
	user, err := s.db.GetUserByID(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	customerID := user.StripeCustomerID
	if customerID == "" {
		customerID, err = s.createStripeCustomer(user)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create customer")
		}
		user.StripeCustomerID = customerID
		s.db.UpdateUser(user)
	}

	// In production with Stripe SDK:
	// params := &stripe.CheckoutSessionParams{
	//     Customer:   stripe.String(customerID),
	//     SuccessURL: stripe.String(req.SuccessURL),
	//     CancelURL:  stripe.String(req.CancelURL),
	//     Mode:       stripe.String("subscription"),
	//     LineItems: []*stripe.CheckoutSessionLineItemParams{
	//         {Price: stripe.String(req.PriceID), Quantity: stripe.Int64(1)},
	//     },
	// }
	// session, err := session.New(params)

	// Simulated checkout URL
	checkoutURL := "https://checkout.stripe.com/c/pay/demo_" + uuid.New().String()[:8]

	return c.JSON(http.StatusOK, map[string]string{
		"checkout_url": checkoutURL,
		"session_id":   "cs_demo_" + uuid.New().String()[:8],
	})
}

func timePtr(t time.Time) *time.Time {
	return &t
}
