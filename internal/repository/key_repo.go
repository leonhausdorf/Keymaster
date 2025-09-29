// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package repository

import (
	"fmt"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/errors"
	"github.com/toeirei/keymaster/internal/model"
)

// publicKeyRepository implements PublicKeyRepository using the database store.
type publicKeyRepository struct {
	store db.Store
}

// NewPublicKeyRepository creates a new public key repository.
func NewPublicKeyRepository(store db.Store) PublicKeyRepository {
	return &publicKeyRepository{store: store}
}

// Find retrieves a single public key by ID.
func (r *publicKeyRepository) Find(id int) (*model.PublicKey, error) {
	keys, err := r.store.GetAllPublicKeys()
	if err != nil {
		return nil, errors.WrapDatabaseError(err, "Find")
	}

	for _, key := range keys {
		if key.ID == id {
			return &key, nil
		}
	}

	return nil, errors.New("Find", errors.ErrNotFound, fmt.Errorf("public key with ID %d not found", id))
}

// FindAll retrieves public keys with optional filtering and ordering.
func (r *publicKeyRepository) FindAll(opts ...QueryOption) ([]*model.PublicKey, error) {
	options := newQueryOptions(opts...)

	keys, err := r.store.GetAllPublicKeys()
	if err != nil {
		return nil, errors.WrapDatabaseError(err, "FindAll")
	}

	// Apply filters
	filtered := r.applyFilters(keys, options)

	// Apply pagination
	result := r.applyPagination(filtered, options)

	// Convert to pointer slice
	pointers := make([]*model.PublicKey, len(result))
	for i := range result {
		pointers[i] = &result[i]
	}

	return pointers, nil
}

// FindByComment retrieves a public key by its comment.
func (r *publicKeyRepository) FindByComment(comment string) (*model.PublicKey, error) {
	key, err := r.store.GetPublicKeyByComment(comment)
	if err != nil {
		return nil, errors.WrapDatabaseError(err, "FindByComment")
	}
	return key, nil
}

// FindGlobal retrieves all global public keys.
func (r *publicKeyRepository) FindGlobal() ([]*model.PublicKey, error) {
	keys, err := r.store.GetGlobalPublicKeys()
	if err != nil {
		return nil, errors.WrapDatabaseError(err, "FindGlobal")
	}

	// Convert to pointer slice
	pointers := make([]*model.PublicKey, len(keys))
	for i := range keys {
		pointers[i] = &keys[i]
	}

	return pointers, nil
}

// FindForAccount retrieves all public keys assigned to an account.
func (r *publicKeyRepository) FindForAccount(accountID int) ([]*model.PublicKey, error) {
	keys, err := r.store.GetKeysForAccount(accountID)
	if err != nil {
		return nil, errors.WrapDatabaseError(err, "FindForAccount")
	}

	// Convert to pointer slice
	pointers := make([]*model.PublicKey, len(keys))
	for i := range keys {
		pointers[i] = &keys[i]
	}

	return pointers, nil
}

// Save creates or updates a public key.
func (r *publicKeyRepository) Save(key *model.PublicKey) error {
	if key.ID == 0 {
		// Create new key
		err := r.store.AddPublicKey(key.Algorithm, key.KeyData, key.Comment, key.IsGlobal)
		if err != nil {
			return errors.WrapDatabaseError(err, "Save")
		}

		// Retrieve the created key to get the ID
		created, err := r.store.GetPublicKeyByComment(key.Comment)
		if err != nil {
			return errors.WrapDatabaseError(err, "Save")
		}
		key.ID = created.ID
		return nil
	}

	// Update existing key - this would require additional methods in the store
	return errors.New("Save", errors.ErrInternal, fmt.Errorf("updating existing public keys not yet implemented"))
}

// Delete removes a public key by ID.
func (r *publicKeyRepository) Delete(id int) error {
	err := r.store.DeletePublicKey(id)
	if err != nil {
		return errors.WrapDatabaseError(err, "Delete")
	}
	return nil
}

// ToggleGlobal toggles the global status of a public key.
func (r *publicKeyRepository) ToggleGlobal(id int) error {
	err := r.store.TogglePublicKeyGlobal(id)
	if err != nil {
		return errors.WrapDatabaseError(err, "ToggleGlobal")
	}
	return nil
}

// AssignToAccount assigns a key to an account.
func (r *publicKeyRepository) AssignToAccount(keyID, accountID int) error {
	err := r.store.AssignKeyToAccount(keyID, accountID)
	if err != nil {
		return errors.WrapDatabaseError(err, "AssignToAccount")
	}
	return nil
}

// UnassignFromAccount removes a key assignment from an account.
func (r *publicKeyRepository) UnassignFromAccount(keyID, accountID int) error {
	err := r.store.UnassignKeyFromAccount(keyID, accountID)
	if err != nil {
		return errors.WrapDatabaseError(err, "UnassignFromAccount")
	}
	return nil
}

// Count returns the total number of public keys matching the criteria.
func (r *publicKeyRepository) Count(opts ...QueryOption) (int, error) {
	keys, err := r.FindAll(opts...)
	if err != nil {
		return 0, err
	}
	return len(keys), nil
}

// applyFilters applies filter conditions to the public key list.
func (r *publicKeyRepository) applyFilters(keys []model.PublicKey, options *QueryOptions) []model.PublicKey {
	if len(options.Filters) == 0 {
		return keys
	}

	var filtered []model.PublicKey
	for _, key := range keys {
		if r.matchesFilters(key, options.Filters) {
			filtered = append(filtered, key)
		}
	}

	return filtered
}

// matchesFilters checks if a public key matches the given filter conditions.
func (r *publicKeyRepository) matchesFilters(key model.PublicKey, filters map[string]interface{}) bool {
	for filterKey, value := range filters {
		switch filterKey {
		case "algorithm":
			if v, ok := value.(string); ok && key.Algorithm != v {
				return false
			}
		case "comment":
			if v, ok := value.(string); ok && key.Comment != v {
				return false
			}
		case "is_global":
			if v, ok := value.(bool); ok && key.IsGlobal != v {
				return false
			}
		}
	}
	return true
}

// applyPagination applies limit and offset to the public key list.
func (r *publicKeyRepository) applyPagination(keys []model.PublicKey, options *QueryOptions) []model.PublicKey {
	total := len(keys)

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
		return []model.PublicKey{}
	}

	return keys[start:end]
}