// Package db provides database models for the Cloud Control Plane
package db

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User represents a registered user
type User struct {
	ID           string `gorm:"primaryKey;size:36" json:"id"`
	Email        string `gorm:"uniqueIndex;size:255" json:"email"`
	Name         string `gorm:"size:255" json:"name"`
	PasswordHash string `gorm:"size:255" json:"-"`
	AvatarURL    string `gorm:"size:500" json:"avatar_url,omitempty"`

	// OAuth
	GitHubID string `gorm:"size:50;index" json:"-"`
	GoogleID string `gorm:"size:50;index" json:"-"`

	// Stripe
	StripeCustomerID string `gorm:"size:50" json:"-"`

	// Status
	EmailVerified bool `gorm:"default:false" json:"email_verified"`
	IsActive      bool `gorm:"default:true" json:"is_active"`

	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Instances   []Instance        `gorm:"foreignKey:OwnerID" json:"-"`
	APIKeys     []APIKey          `gorm:"foreignKey:UserID" json:"-"`
	Credentials []CloudCredential `gorm:"foreignKey:UserID" json:"-"`
}

// SetPassword hashes and sets the user's password
func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

// CheckPassword verifies if the provided password matches
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// Team represents an organization/team
type Team struct {
	ID      string `gorm:"primaryKey;size:36" json:"id"`
	Name    string `gorm:"size:255" json:"name"`
	Slug    string `gorm:"uniqueIndex;size:100" json:"slug"`
	OwnerID string `gorm:"size:36;index" json:"owner_id"`

	// Stripe
	StripeCustomerID string `gorm:"size:50" json:"-"`

	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Members   []TeamMember `gorm:"foreignKey:TeamID" json:"-"`
	Instances []Instance   `gorm:"foreignKey:TeamID" json:"-"`
}

// TeamMember represents a user's membership in a team
type TeamMember struct {
	ID       string    `gorm:"primaryKey;size:36" json:"id"`
	TeamID   string    `gorm:"size:36;index" json:"team_id"`
	UserID   string    `gorm:"size:36;index" json:"user_id"`
	Role     string    `gorm:"size:50;default:'member'" json:"role"` // owner, admin, member
	JoinedAt time.Time `json:"joined_at"`

	// Relations
	Team Team `gorm:"foreignKey:TeamID" json:"-"`
	User User `gorm:"foreignKey:UserID" json:"-"`
}

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID        string `gorm:"primaryKey;size:36" json:"id"`
	UserID    string `gorm:"size:36;index" json:"user_id"`
	Name      string `gorm:"size:100" json:"name"`
	KeyPrefix string `gorm:"size:10" json:"key_prefix"` // First 8 chars for display
	KeyHash   string `gorm:"size:255;uniqueIndex" json:"-"`

	// Permissions
	Scopes string `gorm:"size:500" json:"scopes"` // Comma-separated: read,write,admin

	// Timestamps
	LastUsedAt *time.Time     `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time     `json:"expires_at,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	User User `gorm:"foreignKey:UserID" json:"-"`
}

// CloudCredential stores encrypted cloud provider credentials
type CloudCredential struct {
	ID       string `gorm:"primaryKey;size:36" json:"id"`
	UserID   string `gorm:"size:36;index" json:"user_id"`
	Provider string `gorm:"size:50" json:"provider"` // aws, gcp, azure, etc.
	Name     string `gorm:"size:100" json:"name"`

	// Encrypted credentials (JSON blob encrypted with user's key)
	EncryptedData string `gorm:"type:text" json:"-"`

	// Status
	IsVerified   bool       `gorm:"default:false" json:"is_verified"`
	LastVerified *time.Time `json:"last_verified,omitempty"`

	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	User User `gorm:"foreignKey:UserID" json:"-"`
}

// Instance represents a cloud compute instance
type Instance struct {
	ID      string  `gorm:"primaryKey;size:36" json:"id"`
	OwnerID string  `gorm:"size:36;index" json:"owner_id"`
	TeamID  *string `gorm:"size:36;index" json:"team_id,omitempty"`

	// Instance Info
	Name         string `gorm:"size:100" json:"name"`
	Provider     string `gorm:"size:50" json:"provider"`
	Region       string `gorm:"size:50" json:"region"`
	Zone         string `gorm:"size:50" json:"zone,omitempty"`
	InstanceType string `gorm:"size:50" json:"instance_type"`

	// Status
	Status       string `gorm:"size:50;default:'pending'" json:"status"` // pending, provisioning, running, stopped, terminated, error
	StatusReason string `gorm:"size:255" json:"status_reason,omitempty"`

	// Networking
	PublicIP  string `gorm:"size:50" json:"public_ip,omitempty"`
	PrivateIP string `gorm:"size:50" json:"private_ip,omitempty"`
	SSHPort   int    `gorm:"default:22" json:"ssh_port"`

	// Provider-specific
	ProviderID   string `gorm:"size:100" json:"provider_id,omitempty"` // EC2 instance ID, etc.
	ProviderData string `gorm:"type:text" json:"-"`                    // JSON blob for provider-specific data

	// Pricing
	HourlyRate float64 `gorm:"type:decimal(10,4)" json:"hourly_rate"`

	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	StartedAt *time.Time     `json:"started_at,omitempty"`
	StoppedAt *time.Time     `json:"stopped_at,omitempty"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Owner User  `gorm:"foreignKey:OwnerID" json:"-"`
	Team  *Team `gorm:"foreignKey:TeamID" json:"-"`
}

// UsageRecord tracks resource usage for billing
type UsageRecord struct {
	ID         string `gorm:"primaryKey;size:36" json:"id"`
	UserID     string `gorm:"size:36;index" json:"user_id"`
	InstanceID string `gorm:"size:36;index" json:"instance_id"`

	// Usage
	Type      string  `gorm:"size:50" json:"type"` // compute, storage, network
	Quantity  float64 `gorm:"type:decimal(20,6)" json:"quantity"`
	Unit      string  `gorm:"size:20" json:"unit"` // hours, gb, requests
	UnitPrice float64 `gorm:"type:decimal(10,6)" json:"unit_price"`
	TotalCost float64 `gorm:"type:decimal(10,4)" json:"total_cost"`

	// Period
	Timestamp   time.Time `gorm:"index" json:"timestamp"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`

	// Relations
	User     User     `gorm:"foreignKey:UserID" json:"-"`
	Instance Instance `gorm:"foreignKey:InstanceID" json:"-"`
}

// Invoice represents a billing invoice
type Invoice struct {
	ID     string `gorm:"primaryKey;size:36" json:"id"`
	UserID string `gorm:"size:36;index" json:"user_id"`

	// Invoice Details
	Number string `gorm:"size:50;uniqueIndex" json:"number"`
	Status string `gorm:"size:20" json:"status"` // draft, pending, paid, failed, void

	// Amounts (in cents)
	Subtotal   int64 `json:"subtotal"`
	Tax        int64 `json:"tax"`
	Total      int64 `json:"total"`
	AmountPaid int64 `json:"amount_paid"`
	AmountDue  int64 `json:"amount_due"`

	// Currency
	Currency string `gorm:"size:3;default:'USD'" json:"currency"`

	// Stripe
	StripeInvoiceID       string `gorm:"size:50" json:"-"`
	StripePaymentIntentID string `gorm:"size:50" json:"-"`
	InvoiceURL            string `gorm:"size:500" json:"invoice_url,omitempty"` // Stripe hosted invoice URL

	// Period
	PeriodStart time.Time  `json:"period_start"`
	PeriodEnd   time.Time  `json:"period_end"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations
	User User `gorm:"foreignKey:UserID" json:"-"`
}

// Session represents a user session for JWT refresh tokens
type Session struct {
	ID     string `gorm:"primaryKey;size:36" json:"id"`
	UserID string `gorm:"size:36;index" json:"user_id"`
	Token  string `gorm:"size:255;uniqueIndex" json:"-"`

	// Metadata
	UserAgent string `gorm:"size:500" json:"user_agent,omitempty"`
	IPAddress string `gorm:"size:50" json:"ip_address,omitempty"`

	// Timestamps
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	LastActiveAt time.Time `json:"last_active_at"`

	// Relations
	User User `gorm:"foreignKey:UserID" json:"-"`
}
