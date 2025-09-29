// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"database/sql"
	"fmt"
)

// SQLDialect defines database-specific behaviors for different SQL databases.
// This abstraction allows the BaseStore to work with SQLite, PostgreSQL, and MySQL
// while handling their differences in syntax and behavior.
type SQLDialect interface {
	// Placeholder returns the appropriate placeholder for the given parameter position.
	// SQLite/MySQL use "?", PostgreSQL uses "$1", "$2", etc.
	Placeholder(position int) string

	// PlaceholderString generates a string of placeholders separated by commas.
	// Useful for INSERT statements with multiple values.
	PlaceholderString(count int) string

	// LastInsertID retrieves the ID of the last inserted row.
	// Different databases handle this differently.
	LastInsertID(result sql.Result) (int64, error)

	// Name returns the name of the dialect (for debugging/logging).
	Name() string
}

// SqliteDialect implements SQLDialect for SQLite databases.
type SqliteDialect struct{}

func (d SqliteDialect) Placeholder(position int) string {
	return "?"
}

func (d SqliteDialect) PlaceholderString(count int) string {
	if count == 0 {
		return ""
	}
	result := "?"
	for i := 1; i < count; i++ {
		result += ", ?"
	}
	return result
}

func (d SqliteDialect) LastInsertID(result sql.Result) (int64, error) {
	return result.LastInsertId()
}

func (d SqliteDialect) Name() string {
	return "sqlite"
}

// PostgresDialect implements SQLDialect for PostgreSQL databases.
type PostgresDialect struct{}

func (d PostgresDialect) Placeholder(position int) string {
	return fmt.Sprintf("$%d", position)
}

func (d PostgresDialect) PlaceholderString(count int) string {
	if count == 0 {
		return ""
	}
	result := "$1"
	for i := 2; i <= count; i++ {
		result += fmt.Sprintf(", $%d", i)
	}
	return result
}

func (d PostgresDialect) LastInsertID(result sql.Result) (int64, error) {
	// PostgreSQL doesn't support LastInsertId() in the same way.
	// We'll need to use RETURNING clause in the actual implementation.
	return 0, fmt.Errorf("PostgreSQL requires RETURNING clause for last insert ID")
}

func (d PostgresDialect) Name() string {
	return "postgres"
}

// MySQLDialect implements SQLDialect for MySQL databases.
type MySQLDialect struct{}

func (d MySQLDialect) Placeholder(position int) string {
	return "?"
}

func (d MySQLDialect) PlaceholderString(count int) string {
	if count == 0 {
		return ""
	}
	result := "?"
	for i := 1; i < count; i++ {
		result += ", ?"
	}
	return result
}

func (d MySQLDialect) LastInsertID(result sql.Result) (int64, error) {
	return result.LastInsertId()
}

func (d MySQLDialect) Name() string {
	return "mysql"
}