// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"database/sql"
	"fmt"
	"os/user"

	"github.com/toeirei/keymaster/internal/model"
)

// BaseStore provides common database operations that can be shared across
// different database implementations (SQLite, PostgreSQL, MySQL).
// It uses the SQLDialect interface to handle database-specific differences.
type BaseStore struct {
	db      *sql.DB
	dialect SQLDialect
}

// NewBaseStore creates a new BaseStore with the given database connection and dialect.
func NewBaseStore(db *sql.DB, dialect SQLDialect) *BaseStore {
	return &BaseStore{
		db:      db,
		dialect: dialect,
	}
}

// DB returns the underlying database connection.
// This allows concrete store implementations to access the raw connection when needed.
func (bs *BaseStore) DB() *sql.DB {
	return bs.db
}

// Dialect returns the SQL dialect used by this store.
func (bs *BaseStore) Dialect() SQLDialect {
	return bs.dialect
}

// LogAction adds an entry to the audit log with automatic username detection.
// This is a common operation used throughout the application.
func (bs *BaseStore) LogAction(action, details string) error {
	// Get the current OS user for audit logging.
	currentUser, err := user.Current()
	username := "unknown"
	if err == nil {
		username = currentUser.Username
	}

	query := fmt.Sprintf("INSERT INTO audit_log (username, action, details) VALUES (%s, %s, %s)",
		bs.dialect.Placeholder(1), bs.dialect.Placeholder(2), bs.dialect.Placeholder(3))

	_, err = bs.db.Exec(query, username, action, details)
	return err
}

// scanAccountRow scans a database row into an Account model.
// This is common logic used by multiple account-related queries.
func (bs *BaseStore) scanAccountRow(row *sql.Row) (*model.Account, error) {
	var acc model.Account
	var label sql.NullString
	var tags sql.NullString

	err := row.Scan(&acc.ID, &acc.Username, &acc.Hostname, &label, &tags, &acc.Serial, &acc.IsActive)
	if err != nil {
		return nil, err
	}

	if label.Valid {
		acc.Label = label.String
	}
	if tags.Valid {
		acc.Tags = tags.String
	}

	return &acc, nil
}

// scanAccountRows scans multiple database rows into Account models.
// This is common logic used by queries that return multiple accounts.
func (bs *BaseStore) scanAccountRows(rows *sql.Rows) ([]model.Account, error) {
	defer rows.Close()

	var accounts []model.Account
	for rows.Next() {
		var acc model.Account
		var label sql.NullString
		var tags sql.NullString

		if err := rows.Scan(&acc.ID, &acc.Username, &acc.Hostname, &label, &tags, &acc.Serial, &acc.IsActive); err != nil {
			return nil, err
		}

		if label.Valid {
			acc.Label = label.String
		}
		if tags.Valid {
			acc.Tags = tags.String
		}

		accounts = append(accounts, acc)
	}

	return accounts, nil
}

// scanPublicKeyRow scans a database row into a PublicKey model.
// This is common logic used by multiple public key queries.
func (bs *BaseStore) scanPublicKeyRow(row *sql.Row) (*model.PublicKey, error) {
	var key model.PublicKey
	err := row.Scan(&key.ID, &key.Algorithm, &key.KeyData, &key.Comment, &key.IsGlobal)
	if err != nil {
		return nil, err
	}
	return &key, nil
}

// scanPublicKeyRows scans multiple database rows into PublicKey models.
// This is common logic used by queries that return multiple public keys.
func (bs *BaseStore) scanPublicKeyRows(rows *sql.Rows) ([]model.PublicKey, error) {
	defer rows.Close()

	var keys []model.PublicKey
	for rows.Next() {
		var key model.PublicKey
		if err := rows.Scan(&key.ID, &key.Algorithm, &key.KeyData, &key.Comment, &key.IsGlobal); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}

	return keys, nil
}

// execWithLogging executes a query and logs the action if successful.
// This is a common pattern throughout the application.
func (bs *BaseStore) execWithLogging(query string, logAction, logDetails string, args ...interface{}) error {
	_, err := bs.db.Exec(query, args...)
	if err == nil && logAction != "" {
		// Log the action but don't fail the operation if logging fails
		_ = bs.LogAction(logAction, logDetails)
	}
	return err
}

// insertAndGetID executes an INSERT query and returns the generated ID.
// This handles the database-specific differences in getting the last insert ID.
func (bs *BaseStore) insertAndGetID(query string, args ...interface{}) (int, error) {
	// For PostgreSQL, we need to handle RETURNING clause differently
	if bs.dialect.Name() == "postgres" {
		// PostgreSQL requires RETURNING clause, this will be handled in concrete implementations
		return 0, fmt.Errorf("PostgreSQL insertAndGetID must be handled in concrete implementation")
	}

	result, err := bs.db.Exec(query, args...)
	if err != nil {
		return 0, err
	}

	id, err := bs.dialect.LastInsertID(result)
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	return int(id), nil
}