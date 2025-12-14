// Package db provides database connectivity and models for the Cloud Control Plane
package db

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite" // Pure Go SQLite (no CGO required)
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database wraps the GORM database connection
type Database struct {
	*gorm.DB
}

// Config holds database configuration
type Config struct {
	Driver string // "sqlite" or "postgres"
	DSN    string // Data Source Name
	Debug  bool   // Enable query logging
}

// DefaultSQLiteConfig returns config for local SQLite database
func DefaultSQLiteConfig() Config {
	homeDir, _ := os.UserHomeDir()
	dbPath := filepath.Join(homeDir, ".cm", "cloud.db")
	os.MkdirAll(filepath.Dir(dbPath), 0755)

	return Config{
		Driver: "sqlite",
		DSN:    dbPath,
		Debug:  false,
	}
}

// New creates a new database connection
func New(cfg Config) (*Database, error) {
	var dialector gorm.Dialector

	switch cfg.Driver {
	case "sqlite":
		dialector = sqlite.Open(cfg.DSN)
	case "postgres":
		dialector = postgres.Open(cfg.DSN)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	logLevel := logger.Silent
	if cfg.Debug {
		logLevel = logger.Info
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run auto migrations
	if err := db.AutoMigrate(
		&User{},
		&Team{},
		&TeamMember{},
		&APIKey{},
		&CloudCredential{},
		&Instance{},
		&UsageRecord{},
		&Invoice{},
		&Session{},
	); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &Database{db}, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// ---- User Operations ----

func (d *Database) CreateUser(user *User) error {
	return d.Create(user).Error
}

func (d *Database) GetUserByID(id string) (*User, error) {
	var user User
	if err := d.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *Database) GetUserByEmail(email string) (*User, error) {
	var user User
	if err := d.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *Database) UpdateUser(user *User) error {
	return d.Save(user).Error
}

// ---- API Key Operations ----

func (d *Database) CreateAPIKey(key *APIKey) error {
	return d.Create(key).Error
}

func (d *Database) GetAPIKeyByKey(key string) (*APIKey, error) {
	var apiKey APIKey
	if err := d.Where("key_hash = ?", key).First(&apiKey).Error; err != nil {
		return nil, err
	}
	return &apiKey, nil
}

func (d *Database) ListAPIKeysByUser(userID string) ([]APIKey, error) {
	var keys []APIKey
	if err := d.Where("user_id = ?", userID).Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

func (d *Database) DeleteAPIKey(id string) error {
	return d.Where("id = ?", id).Delete(&APIKey{}).Error
}

// ---- Instance Operations ----

func (d *Database) CreateInstance(instance *Instance) error {
	return d.Create(instance).Error
}

func (d *Database) GetInstanceByID(id string) (*Instance, error) {
	var instance Instance
	if err := d.Where("id = ?", id).First(&instance).Error; err != nil {
		return nil, err
	}
	return &instance, nil
}

func (d *Database) ListInstancesByUser(userID string) ([]Instance, error) {
	var instances []Instance
	if err := d.Where("owner_id = ?", userID).Find(&instances).Error; err != nil {
		return nil, err
	}
	return instances, nil
}

func (d *Database) UpdateInstance(instance *Instance) error {
	return d.Save(instance).Error
}

func (d *Database) DeleteInstance(id string) error {
	return d.Where("id = ?", id).Delete(&Instance{}).Error
}

// ---- Cloud Credential Operations ----

func (d *Database) CreateCredential(cred *CloudCredential) error {
	return d.Create(cred).Error
}

func (d *Database) ListCredentialsByUser(userID string) ([]CloudCredential, error) {
	var creds []CloudCredential
	if err := d.Where("user_id = ?", userID).Find(&creds).Error; err != nil {
		return nil, err
	}
	return creds, nil
}

func (d *Database) GetCredentialByID(id string) (*CloudCredential, error) {
	var cred CloudCredential
	if err := d.Where("id = ?", id).First(&cred).Error; err != nil {
		return nil, err
	}
	return &cred, nil
}

func (d *Database) DeleteCredential(id string) error {
	return d.Where("id = ?", id).Delete(&CloudCredential{}).Error
}

// ---- Usage & Billing Operations ----

func (d *Database) CreateUsageRecord(record *UsageRecord) error {
	return d.Create(record).Error
}

func (d *Database) GetUsageByUserAndPeriod(userID string, start, end time.Time) ([]UsageRecord, error) {
	var records []UsageRecord
	if err := d.Where("user_id = ? AND timestamp >= ? AND timestamp <= ?", userID, start, end).Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (d *Database) CreateInvoice(invoice *Invoice) error {
	return d.Create(invoice).Error
}

func (d *Database) ListInvoicesByUser(userID string) ([]Invoice, error) {
	var invoices []Invoice
	if err := d.Where("user_id = ?", userID).Order("created_at DESC").Find(&invoices).Error; err != nil {
		return nil, err
	}
	return invoices, nil
}

// ---- Session Operations ----

func (d *Database) CreateSession(session *Session) error {
	return d.Create(session).Error
}

func (d *Database) GetSessionByToken(token string) (*Session, error) {
	var session Session
	if err := d.Where("token = ? AND expires_at > ?", token, time.Now()).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (d *Database) DeleteSession(token string) error {
	return d.Where("token = ?", token).Delete(&Session{}).Error
}

func (d *Database) DeleteExpiredSessions() error {
	return d.Where("expires_at < ?", time.Now()).Delete(&Session{}).Error
}
