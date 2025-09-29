// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package repository

import (
	"fmt"
	"strings"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/errors"
	"github.com/toeirei/keymaster/internal/model"
)

// accountRepository implements AccountRepository using the database store.
type accountRepository struct {
	store db.Store
}

// NewAccountRepository creates a new account repository.
func NewAccountRepository(store db.Store) AccountRepository {
	return &accountRepository{store: store}
}

// Find retrieves a single account by ID.
func (r *accountRepository) Find(id int) (*model.Account, error) {
	accounts, err := r.store.GetAllAccounts()
	if err != nil {
		return nil, errors.WrapDatabaseError(err, "Find")
	}

	for _, account := range accounts {
		if account.ID == id {
			return &account, nil
		}
	}

	return nil, errors.New("Find", errors.ErrNotFound, fmt.Errorf("account with ID %d not found", id))
}

// FindAll retrieves accounts with optional filtering and ordering.
func (r *accountRepository) FindAll(opts ...QueryOption) ([]*model.Account, error) {
	options := newQueryOptions(opts...)

	var accounts []model.Account
	var err error

	if options.ActiveOnly {
		accounts, err = r.store.GetAllActiveAccounts()
	} else {
		accounts, err = r.store.GetAllAccounts()
	}

	if err != nil {
		return nil, errors.WrapDatabaseError(err, "FindAll")
	}

	// Apply filters
	filtered := r.applyFilters(accounts, options)

	// Apply ordering (already handled by the database query, but could be extended)
	// Apply limit and offset
	result := r.applyPagination(filtered, options)

	// Convert to pointer slice
	pointers := make([]*model.Account, len(result))
	for i := range result {
		pointers[i] = &result[i]
	}

	return pointers, nil
}

// FindByHostname retrieves accounts for a specific hostname.
func (r *accountRepository) FindByHostname(hostname string) ([]*model.Account, error) {
	return r.FindAll(WithFilter("hostname", hostname))
}

// FindActive retrieves all active accounts.
func (r *accountRepository) FindActive() ([]*model.Account, error) {
	return r.FindAll(WithActive(true))
}

// Save creates or updates an account.
func (r *accountRepository) Save(account *model.Account) error {
	if account.ID == 0 {
		// Create new account
		id, err := r.store.AddAccount(account.Username, account.Hostname, account.Label, account.Tags)
		if err != nil {
			return errors.WrapDatabaseError(err, "Save")
		}
		account.ID = id
		return nil
	}

	// Update existing account - this would require additional methods in the store
	// For now, we'll return an error indicating this needs to be implemented
	return errors.New("Save", errors.ErrInternal, fmt.Errorf("updating existing accounts not yet implemented"))
}

// Delete removes an account by ID.
func (r *accountRepository) Delete(id int) error {
	err := r.store.DeleteAccount(id)
	if err != nil {
		return errors.WrapDatabaseError(err, "Delete")
	}
	return nil
}

// UpdateSerial updates the system key serial for an account.
func (r *accountRepository) UpdateSerial(id, serial int) error {
	err := r.store.UpdateAccountSerial(id, serial)
	if err != nil {
		return errors.WrapDatabaseError(err, "UpdateSerial")
	}
	return nil
}

// ToggleActive toggles the active status of an account.
func (r *accountRepository) ToggleActive(id int) error {
	err := r.store.ToggleAccountStatus(id)
	if err != nil {
		return errors.WrapDatabaseError(err, "ToggleActive")
	}
	return nil
}

// Count returns the total number of accounts matching the criteria.
func (r *accountRepository) Count(opts ...QueryOption) (int, error) {
	accounts, err := r.FindAll(opts...)
	if err != nil {
		return 0, err
	}
	return len(accounts), nil
}

// applyFilters applies filter conditions to the account list.
func (r *accountRepository) applyFilters(accounts []model.Account, options *QueryOptions) []model.Account {
	if len(options.Filters) == 0 {
		return accounts
	}

	var filtered []model.Account
	for _, account := range accounts {
		if r.matchesFilters(account, options.Filters) {
			filtered = append(filtered, account)
		}
	}

	return filtered
}

// matchesFilters checks if an account matches the given filter conditions.
func (r *accountRepository) matchesFilters(account model.Account, filters map[string]interface{}) bool {
	for key, value := range filters {
		switch key {
		case "hostname":
			if v, ok := value.(string); ok && account.Hostname != v {
				return false
			}
		case "username":
			if v, ok := value.(string); ok && account.Username != v {
				return false
			}
		case "tags":
			if v, ok := value.(string); ok && !strings.Contains(account.Tags, v) {
				return false
			}
		case "label":
			if v, ok := value.(string); ok && !strings.Contains(account.Label, v) {
				return false
			}
		case "serial":
			if v, ok := value.(int); ok && account.Serial != v {
				return false
			}
		case "is_active":
			if v, ok := value.(bool); ok && account.IsActive != v {
				return false
			}
		}
	}
	return true
}

// applyPagination applies limit and offset to the account list.
func (r *accountRepository) applyPagination(accounts []model.Account, options *QueryOptions) []model.Account {
	total := len(accounts)

	// Apply offset
	start := 0
	if options.Offset > 0 && options.Offset < total {
		start = options.Offset
	}

	// Apply limit
	end := total
	if options.Limit > 0 {
		requestedEnd := start + options.Limit
		if requestedEnd < total {
			end = requestedEnd
		}
	}

	if start >= total {
		return []model.Account{}
	}

	return accounts[start:end]
}