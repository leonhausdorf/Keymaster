// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package repository provides the repository layer for Keymaster.
// This layer abstracts data access logic and provides a clean interface
// for the business logic layer, following the Repository pattern.
package repository

import (
	"time"

	"github.com/toeirei/keymaster/internal/model"
)

// QueryOption represents a functional option for customizing queries.
type QueryOption func(*QueryOptions)

// QueryOptions holds options for customizing repository queries.
type QueryOptions struct {
	Limit      int
	Offset     int
	OrderBy    []string
	Filters    map[string]interface{}
	ActiveOnly bool
}

// WithLimit sets the maximum number of results to return.
func WithLimit(limit int) QueryOption {
	return func(opts *QueryOptions) {
		opts.Limit = limit
	}
}

// WithOffset sets the number of results to skip.
func WithOffset(offset int) QueryOption {
	return func(opts *QueryOptions) {
		opts.Offset = offset
	}
}

// WithOrderBy sets the ordering for results.
func WithOrderBy(columns ...string) QueryOption {
	return func(opts *QueryOptions) {
		opts.OrderBy = append(opts.OrderBy, columns...)
	}
}

// WithFilter adds a filter condition.
func WithFilter(key string, value interface{}) QueryOption {
	return func(opts *QueryOptions) {
		if opts.Filters == nil {
			opts.Filters = make(map[string]interface{})
		}
		opts.Filters[key] = value
	}
}

// WithActive filters to only active records.
func WithActive(active bool) QueryOption {
	return func(opts *QueryOptions) {
		opts.ActiveOnly = active
	}
}

// WithTags filters accounts by tags (partial match).
func WithTags(tags string) QueryOption {
	return func(opts *QueryOptions) {
		if opts.Filters == nil {
			opts.Filters = make(map[string]interface{})
		}
		opts.Filters["tags"] = tags
	}
}

// newQueryOptions creates QueryOptions with default values and applies the given options.
func newQueryOptions(opts ...QueryOption) *QueryOptions {
	options := &QueryOptions{
		Filters: make(map[string]interface{}),
		OrderBy: []string{},
	}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

// AccountRepository defines the interface for account data access.
type AccountRepository interface {
	// Find retrieves a single account by ID.
	Find(id int) (*model.Account, error)

	// FindAll retrieves accounts with optional filtering and ordering.
	FindAll(opts ...QueryOption) ([]*model.Account, error)

	// FindByHostname retrieves accounts for a specific hostname.
	FindByHostname(hostname string) ([]*model.Account, error)

	// FindActive retrieves all active accounts.
	FindActive() ([]*model.Account, error)

	// Save creates or updates an account.
	Save(account *model.Account) error

	// Delete removes an account by ID.
	Delete(id int) error

	// UpdateSerial updates the system key serial for an account.
	UpdateSerial(id, serial int) error

	// ToggleActive toggles the active status of an account.
	ToggleActive(id int) error

	// Count returns the total number of accounts matching the criteria.
	Count(opts ...QueryOption) (int, error)
}

// PublicKeyRepository defines the interface for public key data access.
type PublicKeyRepository interface {
	// Find retrieves a single public key by ID.
	Find(id int) (*model.PublicKey, error)

	// FindAll retrieves public keys with optional filtering and ordering.
	FindAll(opts ...QueryOption) ([]*model.PublicKey, error)

	// FindByComment retrieves a public key by its comment.
	FindByComment(comment string) (*model.PublicKey, error)

	// FindGlobal retrieves all global public keys.
	FindGlobal() ([]*model.PublicKey, error)

	// FindForAccount retrieves all public keys assigned to an account.
	FindForAccount(accountID int) ([]*model.PublicKey, error)

	// Save creates or updates a public key.
	Save(key *model.PublicKey) error

	// Delete removes a public key by ID.
	Delete(id int) error

	// ToggleGlobal toggles the global status of a public key.
	ToggleGlobal(id int) error

	// AssignToAccount assigns a key to an account.
	AssignToAccount(keyID, accountID int) error

	// UnassignFromAccount removes a key assignment from an account.
	UnassignFromAccount(keyID, accountID int) error

	// Count returns the total number of public keys matching the criteria.
	Count(opts ...QueryOption) (int, error)
}

// SystemKeyRepository defines the interface for system key data access.
type SystemKeyRepository interface {
	// FindActive retrieves the currently active system key.
	FindActive() (*model.SystemKey, error)

	// FindBySerial retrieves a system key by its serial number.
	FindBySerial(serial int) (*model.SystemKey, error)

	// FindAll retrieves all system keys.
	FindAll() ([]*model.SystemKey, error)

	// Create creates a new system key and makes it active.
	Create(publicKey, privateKey string) (int, error)

	// Rotate deactivates all keys and creates a new active one.
	Rotate(publicKey, privateKey string) (int, error)

	// HasKeys checks if any system keys exist.
	HasKeys() (bool, error)
}

// AuditLogRepository defines the interface for audit log data access.
type AuditLogRepository interface {
	// FindAll retrieves audit log entries with optional filtering and ordering.
	FindAll(opts ...QueryOption) ([]*model.AuditLogEntry, error)

	// FindRecent retrieves the most recent audit log entries.
	FindRecent(limit int) ([]*model.AuditLogEntry, error)

	// FindByAction retrieves audit log entries for a specific action.
	FindByAction(action string, opts ...QueryOption) ([]*model.AuditLogEntry, error)

	// FindByDateRange retrieves audit log entries within a date range.
	FindByDateRange(start, end time.Time) ([]*model.AuditLogEntry, error)

	// Log creates a new audit log entry.
	Log(action, details string) error

	// Count returns the total number of audit log entries matching the criteria.
	Count(opts ...QueryOption) (int, error)
}

// BootstrapRepository defines the interface for bootstrap session data access.
type BootstrapRepository interface {
	// Find retrieves a bootstrap session by ID.
	Find(id string) (*model.BootstrapSession, error)

	// FindAll retrieves all bootstrap sessions.
	FindAll() ([]*model.BootstrapSession, error)

	// FindExpired retrieves all expired bootstrap sessions.
	FindExpired() ([]*model.BootstrapSession, error)

	// FindOrphaned retrieves all orphaned bootstrap sessions.
	FindOrphaned() ([]*model.BootstrapSession, error)

	// Save creates or updates a bootstrap session.
	Save(session *model.BootstrapSession) error

	// Delete removes a bootstrap session by ID.
	Delete(id string) error

	// UpdateStatus updates the status of a bootstrap session.
	UpdateStatus(id, status string) error
}

// HostKeyRepository defines the interface for known host key data access.
type HostKeyRepository interface {
	// Find retrieves a known host key by hostname.
	Find(hostname string) (string, error)

	// Save stores a host key for the given hostname.
	Save(hostname, key string) error

	// Delete removes a host key by hostname.
	Delete(hostname string) error

	// FindAll retrieves all known host keys.
	FindAll() (map[string]string, error)
}