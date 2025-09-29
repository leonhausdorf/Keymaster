// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// This file demonstrates how the QueryBuilder can be used in practice.
// It's an example implementation showing the new approach vs. the old one.

package db

import (
	"fmt"

	"github.com/toeirei/keymaster/internal/errors"
	"github.com/toeirei/keymaster/internal/model"
)

// AccountModel demonstrates the QueryBuilder usage for account operations.
// This is an example implementation to show how the new system would work.
type AccountModel struct {
	*BaseStore
}

// NewAccountModel creates a new AccountModel instance.
func NewAccountModel(baseStore *BaseStore) *AccountModel {
	return &AccountModel{BaseStore: baseStore}
}

// ====================
// EXAMPLE: OLD APPROACH (current implementation)
// ====================

func (am *AccountModel) GetAllAccountsOld() ([]model.Account, error) {
	// Hard-coded SQL string - not dialect-agnostic
	rows, err := am.DB().Query("SELECT id, username, hostname, label, tags, serial, is_active FROM accounts ORDER BY label, hostname, username")
	if err != nil {
		return nil, err
	}
	return am.scanAccountRows(rows)
}

func (am *AccountModel) GetActiveAccountsOld() ([]model.Account, error) {
	// Another hard-coded SQL string
	rows, err := am.DB().Query("SELECT id, username, hostname, label, tags, serial, is_active FROM accounts WHERE is_active = ? ORDER BY label, hostname, username", true)
	if err != nil {
		return nil, err
	}
	return am.scanAccountRows(rows)
}

// ====================
// EXAMPLE: NEW APPROACH (with QueryBuilder)
// ====================

func (am *AccountModel) GetAllAccounts() ([]model.Account, error) {
	qb := NewQueryBuilder(am.Dialect())
	query, args := qb.
		Select("id", "username", "hostname", "label", "tags", "serial", "is_active").
		From("accounts").
		OrderBy("label", "hostname", "username").
		Build()

	rows, err := am.DB().Query(query, args...)
	if err != nil {
		return nil, errors.WrapDatabaseError(err, "GetAllAccounts")
	}
	return am.scanAccountRows(rows)
}

func (am *AccountModel) GetActiveAccounts() ([]model.Account, error) {
	qb := NewQueryBuilder(am.Dialect())
	query, args := qb.
		Select("id", "username", "hostname", "label", "tags", "serial", "is_active").
		From("accounts").
		Where("is_active", true).
		OrderBy("label", "hostname", "username").
		Build()

	rows, err := am.DB().Query(query, args...)
	if err != nil {
		return nil, errors.WrapDatabaseError(err, "GetActiveAccounts")
	}
	return am.scanAccountRows(rows)
}

func (am *AccountModel) GetAccountsByTag(tag string) ([]model.Account, error) {
	qb := NewQueryBuilder(am.Dialect())
	query, args := qb.
		Select("id", "username", "hostname", "label", "tags", "serial", "is_active").
		From("accounts").
		WhereOp("tags", "LIKE", "%"+tag+"%"). // SQL LIKE for partial matching
		Where("is_active", true).
		OrderBy("label", "hostname", "username").
		Build()

	rows, err := am.DB().Query(query, args...)
	if err != nil {
		return nil, errors.WrapDatabaseError(err, "GetAccountsByTag")
	}
	return am.scanAccountRows(rows)
}

func (am *AccountModel) GetAccountsPaginated(limit, offset int) ([]model.Account, error) {
	qb := NewQueryBuilder(am.Dialect())
	query, args := qb.
		Select("id", "username", "hostname", "label", "tags", "serial", "is_active").
		From("accounts").
		OrderBy("label", "hostname", "username").
		Limit(limit).
		Offset(offset).
		Build()

	rows, err := am.DB().Query(query, args...)
	if err != nil {
		return nil, errors.WrapDatabaseError(err, "GetAccountsPaginated")
	}
	return am.scanAccountRows(rows)
}

func (am *AccountModel) AddAccount(username, hostname, label, tags string) (int, error) {
	qb := NewQueryBuilder(am.Dialect())

	// For INSERT queries, we need to handle dialects differently
	if am.Dialect().Name() == "postgres" {
		// PostgreSQL needs RETURNING clause
		query, args := qb.
			Insert("accounts").
			Set("username", username).
			Set("hostname", hostname).
			Set("label", label).
			Set("tags", tags).
			Build()

		// Add RETURNING clause for PostgreSQL
		query += " RETURNING id"

		var id int
		err := am.DB().QueryRow(query, args...).Scan(&id)
		if err != nil {
			return 0, errors.WrapDatabaseError(err, "AddAccount")
		}

		// Log the action
		_ = am.LogAction("ADD_ACCOUNT", fmt.Sprintf("account: %s@%s", username, hostname))
		return id, nil
	}

	// For SQLite and MySQL, use the standard approach
	query, args := qb.
		Insert("accounts").
		Set("username", username).
		Set("hostname", hostname).
		Set("label", label).
		Set("tags", tags).
		Build()

	id, err := am.insertAndGetID(query, args...)
	if err != nil {
		return 0, errors.WrapDatabaseError(err, "AddAccount")
	}

	// Log the action
	_ = am.LogAction("ADD_ACCOUNT", fmt.Sprintf("account: %s@%s", username, hostname))
	return id, nil
}

func (am *AccountModel) UpdateAccountLabel(id int, label string) error {
	qb := NewQueryBuilder(am.Dialect())
	query, args := qb.
		Update("accounts").
		Set("label", label).
		Where("id", id).
		Build()

	err := am.execWithLogging(query, "UPDATE_ACCOUNT_LABEL",
		fmt.Sprintf("account_id: %d, new_label: '%s'", id, label), args...)
	if err != nil {
		return errors.WrapDatabaseError(err, "UpdateAccountLabel")
	}
	return nil
}

func (am *AccountModel) DeleteAccount(id int) error {
	// Get account details before deleting for better logging
	var username, hostname string
	selectQb := NewQueryBuilder(am.Dialect())
	selectQuery, selectArgs := selectQb.
		Select("username", "hostname").
		From("accounts").
		Where("id", id).
		Build()

	err := am.DB().QueryRow(selectQuery, selectArgs...).Scan(&username, &hostname)
	details := fmt.Sprintf("id: %d", id)
	if err == nil {
		details = fmt.Sprintf("account: %s@%s", username, hostname)
	}

	// Delete the account
	deleteQb := NewQueryBuilder(am.Dialect())
	deleteQuery, deleteArgs := deleteQb.
		Delete().
		From("accounts").
		Where("id", id).
		Build()

	err = am.execWithLogging(deleteQuery, "DELETE_ACCOUNT", details, deleteArgs...)
	if err != nil {
		return errors.WrapDatabaseError(err, "DeleteAccount")
	}
	return nil
}

func (am *AccountModel) GetAccountsWithMultipleFilters(hostname string, isActive bool, tagFilter string) ([]model.Account, error) {
	qb := NewQueryBuilder(am.Dialect())
	query, args := qb.
		Select("id", "username", "hostname", "label", "tags", "serial", "is_active").
		From("accounts").
		Where("hostname", hostname).
		Where("is_active", isActive).
		WhereOp("tags", "LIKE", "%"+tagFilter+"%").
		OrderBy("label", "username").
		Build()

	rows, err := am.DB().Query(query, args...)
	if err != nil {
		return nil, errors.WrapDatabaseError(err, "GetAccountsWithMultipleFilters")
	}
	return am.scanAccountRows(rows)
}

// ====================
// COMPARISON: Generated SQL for different dialects
// ====================

/*
Example: GetActiveAccounts() generates different SQL based on dialect:

SQLite/MySQL:
	SELECT id, username, hostname, label, tags, serial, is_active
	FROM accounts
	WHERE is_active = ?
	ORDER BY label, hostname, username

PostgreSQL:
	SELECT id, username, hostname, label, tags, serial, is_active
	FROM accounts
	WHERE is_active = $1
	ORDER BY label, hostname, username

Example: GetAccountsByTag("prod") with LIKE:

SQLite/MySQL:
	SELECT id, username, hostname, label, tags, serial, is_active
	FROM accounts
	WHERE tags LIKE ? AND is_active = ?
	ORDER BY label, hostname, username
	Args: ["%prod%", true]

PostgreSQL:
	SELECT id, username, hostname, label, tags, serial, is_active
	FROM accounts
	WHERE tags LIKE $1 AND is_active = $2
	ORDER BY label, hostname, username
	Args: ["%prod%", true]

Example: AddAccount() for PostgreSQL:
	INSERT INTO accounts (username, hostname, label, tags)
	VALUES ($1, $2, $3, $4) RETURNING id
	Args: ["deploy", "server1", "Production Server", "env:prod"]

Example: AddAccount() for SQLite/MySQL:
	INSERT INTO accounts (username, hostname, label, tags)
	VALUES (?, ?, ?, ?)
	Args: ["deploy", "server1", "Production Server", "env:prod"]
*/
