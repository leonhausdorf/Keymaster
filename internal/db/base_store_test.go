// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"testing"
)

func TestSQLDialects(t *testing.T) {
	tests := []struct {
		name     string
		dialect  SQLDialect
		expected map[string]string
	}{
		{
			name:    "SQLite",
			dialect: SqliteDialect{},
			expected: map[string]string{
				"placeholder1": "?",
				"placeholder2": "?",
				"name":         "sqlite",
			},
		},
		{
			name:    "PostgreSQL",
			dialect: PostgresDialect{},
			expected: map[string]string{
				"placeholder1": "$1",
				"placeholder2": "$2",
				"name":         "postgres",
			},
		},
		{
			name:    "MySQL",
			dialect: MySQLDialect{},
			expected: map[string]string{
				"placeholder1": "?",
				"placeholder2": "?",
				"name":         "mysql",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Placeholder
			if got := tt.dialect.Placeholder(1); got != tt.expected["placeholder1"] {
				t.Errorf("Placeholder(1) = %v, want %v", got, tt.expected["placeholder1"])
			}
			if got := tt.dialect.Placeholder(2); got != tt.expected["placeholder2"] {
				t.Errorf("Placeholder(2) = %v, want %v", got, tt.expected["placeholder2"])
			}

			// Test Name
			if got := tt.dialect.Name(); got != tt.expected["name"] {
				t.Errorf("Name() = %v, want %v", got, tt.expected["name"])
			}
		})
	}
}

func TestPlaceholderString(t *testing.T) {
	tests := []struct {
		name     string
		dialect  SQLDialect
		count    int
		expected string
	}{
		{"SQLite 0", SqliteDialect{}, 0, ""},
		{"SQLite 1", SqliteDialect{}, 1, "?"},
		{"SQLite 3", SqliteDialect{}, 3, "?, ?, ?"},
		{"PostgreSQL 0", PostgresDialect{}, 0, ""},
		{"PostgreSQL 1", PostgresDialect{}, 1, "$1"},
		{"PostgreSQL 3", PostgresDialect{}, 3, "$1, $2, $3"},
		{"MySQL 0", MySQLDialect{}, 0, ""},
		{"MySQL 1", MySQLDialect{}, 1, "?"},
		{"MySQL 3", MySQLDialect{}, 3, "?, ?, ?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.dialect.PlaceholderString(tt.count); got != tt.expected {
				t.Errorf("PlaceholderString(%d) = %v, want %v", tt.count, got, tt.expected)
			}
		})
	}
}