// Package db provides database models for the Cloud Control Plane
package db

import (
	"time"
)

// User represents a registered user
type User struct {
	ID           string `json:"id" db:"id"`
	Email        string `json:"email" db:"email"`
	Name         string `json:"name" db:"name"`
	AvatarURL    string `json:"avatar_url,omitempty" db:"avatar_url"`
	PasswordHash string `json:"-" db:"password_hash"` // For email/password auth

	// OAuth identities
	GitHubID string `json:"-" db:"github_id"`
	GoogleID string `json:"-" db:"google_id"`

	// Stripe
	StripeCustomerID string `json:"-" db:"stripe_customer_id"`

	// Status
	IsActive      bool `json:"is_active" db:"is_active"`
	IsAdmin       bool `json:"is_admin" db:"is_admin"`
	EmailVerified bool `json:"email_verified" db:"email_verified"`

	// Timestamps
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	LastLoginAt time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
}

// Team represents an organization/team
type Team struct {
	ID      string `json:"id" db:"id"`
	Name    string `json:"name" db:"name"`
	Slug    string `json:"slug" db:"slug"` // URL-friendly name
	OwnerID string `json:"owner_id" db:"owner_id"`

	// Billing
	StripeCustomerID     string `json:"-" db:"stripe_customer_id"`
	StripeSubscriptionID string `json:"-" db:"stripe_subscription_id"`
	Plan                 string `json:"plan" db:"plan"` // free, pro, enterprise

	// Quotas
	MaxInstances  int     `json:"max_instances" db:"max_instances"`
	MaxGPUHours   int     `json:"max_gpu_hours" db:"max_gpu_hours"`
	MonthlyBudget float64 `json:"monthly_budget" db:"monthly_budget"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// TeamMember represents a user's membership in a team
type TeamMember struct {
	ID        string    `json:"id" db:"id"`
	TeamID    string    `json:"team_id" db:"team_id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Role      string    `json:"role" db:"role"` // owner, admin, member
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID        string `json:"id" db:"id"`
	UserID    string `json:"user_id" db:"user_id"`
	TeamID    string `json:"team_id,omitempty" db:"team_id"`
	Name      string `json:"name" db:"name"`
	KeyPrefix string `json:"key_prefix" db:"key_prefix"` // First 8 chars for display
	KeyHash   string `json:"-" db:"key_hash"`            // Hashed key

	// Permissions
	Scopes []string `json:"scopes" db:"scopes"` // read, write, admin

	// Status
	IsActive   bool      `json:"is_active" db:"is_active"`
	LastUsedAt time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
	ExpiresAt  time.Time `json:"expires_at,omitempty" db:"expires_at"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Instance represents a cloud development environment instance
type Instance struct {
	ID      string `json:"id" db:"id"`
	Name    string `json:"name" db:"name"`
	OwnerID string `json:"owner_id" db:"owner_id"`
	TeamID  string `json:"team_id,omitempty" db:"team_id"`

	// Provider
	Provider     string `json:"provider" db:"provider"`
	Region       string `json:"region" db:"region"`
	InstanceType string `json:"instance_type" db:"instance_type"`

	// Status
	Status    string `json:"status" db:"status"`
	PublicIP  string `json:"public_ip,omitempty" db:"public_ip"`
	PrivateIP string `json:"private_ip,omitempty" db:"private_ip"`
	SSHPort   int    `json:"ssh_port" db:"ssh_port"`

	// Configuration
	Image        string `json:"image" db:"image"`
	DevContainer string `json:"devcontainer,omitempty" db:"devcontainer"` // JSON

	// Cost tracking
	HourlyRate float64 `json:"hourly_rate" db:"hourly_rate"`
	TotalCost  float64 `json:"total_cost" db:"total_cost"`

	// Provider-specific
	ProviderID string `json:"-" db:"provider_id"` // ID in the cloud provider

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	StartedAt time.Time `json:"started_at,omitempty" db:"started_at"`
	StoppedAt time.Time `json:"stopped_at,omitempty" db:"stopped_at"`
}

// CloudCredential stores encrypted cloud provider credentials
type CloudCredential struct {
	ID       string `json:"id" db:"id"`
	UserID   string `json:"user_id" db:"user_id"`
	TeamID   string `json:"team_id,omitempty" db:"team_id"`
	Provider string `json:"provider" db:"provider"`
	Name     string `json:"name" db:"name"`

	// Encrypted credentials (use AES-256-GCM)
	EncryptedData string `json:"-" db:"encrypted_data"`

	// Status
	IsValid     bool      `json:"is_valid" db:"is_valid"`
	LastChecked time.Time `json:"last_checked,omitempty" db:"last_checked"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// UsageRecord tracks resource usage for billing
type UsageRecord struct {
	ID         string `json:"id" db:"id"`
	UserID     string `json:"user_id" db:"user_id"`
	TeamID     string `json:"team_id,omitempty" db:"team_id"`
	InstanceID string `json:"instance_id" db:"instance_id"`

	// Usage
	InstanceType string    `json:"instance_type" db:"instance_type"`
	Provider     string    `json:"provider" db:"provider"`
	StartTime    time.Time `json:"start_time" db:"start_time"`
	EndTime      time.Time `json:"end_time,omitempty" db:"end_time"`
	DurationSecs int64     `json:"duration_secs" db:"duration_secs"`

	// Cost
	HourlyRate float64 `json:"hourly_rate" db:"hourly_rate"`
	TotalCost  float64 `json:"total_cost" db:"total_cost"`

	// Billing
	Invoiced  bool   `json:"invoiced" db:"invoiced"`
	InvoiceID string `json:"invoice_id,omitempty" db:"invoice_id"`
}

// Session represents an active user session (for JWT refresh)
type Session struct {
	ID           string    `json:"id" db:"id"`
	UserID       string    `json:"user_id" db:"user_id"`
	RefreshToken string    `json:"-" db:"refresh_token"`
	UserAgent    string    `json:"user_agent" db:"user_agent"`
	IPAddress    string    `json:"ip_address" db:"ip_address"`
	ExpiresAt    time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}
