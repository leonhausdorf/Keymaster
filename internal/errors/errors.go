// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package errors provides structured error handling for Keymaster.
// It defines error types, wrapping functionality, and context-aware error messages
// that integrate with the internationalization system.
package errors

import (
	"fmt"
)

// ErrorKind represents the category of an error.
// This allows for better error handling and user-friendly messages.
type ErrorKind int

const (
	// ErrDatabase represents database-related errors (connection, queries, etc.)
	ErrDatabase ErrorKind = iota
	// ErrSSHConnection represents SSH connection and authentication errors
	ErrSSHConnection
	// ErrValidation represents input validation errors
	ErrValidation
	// ErrNotFound represents resource not found errors
	ErrNotFound
	// ErrDuplicate represents duplicate resource errors
	ErrDuplicate
	// ErrPermission represents permission/authorization errors
	ErrPermission
	// ErrConfiguration represents configuration-related errors
	ErrConfiguration
	// ErrInternal represents internal/system errors
	ErrInternal
)

// String returns a human-readable name for the error kind.
func (k ErrorKind) String() string {
	switch k {
	case ErrDatabase:
		return "Database"
	case ErrSSHConnection:
		return "SSH Connection"
	case ErrValidation:
		return "Validation"
	case ErrNotFound:
		return "Not Found"
	case ErrDuplicate:
		return "Duplicate"
	case ErrPermission:
		return "Permission"
	case ErrConfiguration:
		return "Configuration"
	case ErrInternal:
		return "Internal"
	default:
		return "Unknown"
	}
}

// KeymasterError represents a structured error with context and categorization.
// It provides better error handling and debugging capabilities throughout the application.
type KeymasterError struct {
	// Op is the operation that failed (e.g., "AddAccount", "ConnectSSH")
	Op string
	// Kind is the category of error
	Kind ErrorKind
	// Err is the underlying error that caused this error
	Err error
	// Context provides additional information about the error
	Context map[string]interface{}
}

// Error implements the error interface.
func (e *KeymasterError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Kind, e.Op, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Op)
}

// Unwrap returns the underlying error for error unwrapping.
func (e *KeymasterError) Unwrap() error {
	return e.Err
}

// Is checks if the error matches a target error kind.
func (e *KeymasterError) Is(target error) bool {
	if t, ok := target.(*KeymasterError); ok {
		return e.Kind == t.Kind
	}
	return false
}

// New creates a new KeymasterError with the given operation, kind, and underlying error.
func New(op string, kind ErrorKind, err error) *KeymasterError {
	return &KeymasterError{
		Op:      op,
		Kind:    kind,
		Err:     err,
		Context: make(map[string]interface{}),
	}
}

// Newf creates a new KeymasterError with a formatted message.
func Newf(op string, kind ErrorKind, format string, args ...interface{}) *KeymasterError {
	return &KeymasterError{
		Op:      op,
		Kind:    kind,
		Err:     fmt.Errorf(format, args...),
		Context: make(map[string]interface{}),
	}
}

// WithContext adds context information to the error.
func (e *KeymasterError) WithContext(key string, value interface{}) *KeymasterError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// GetContext retrieves context information from the error.
func (e *KeymasterError) GetContext(key string) (interface{}, bool) {
	if e.Context == nil {
		return nil, false
	}
	value, exists := e.Context[key]
	return value, exists
}

// Wrap wraps an existing error with KeymasterError context.
func Wrap(err error, op string, kind ErrorKind) *KeymasterError {
	if err == nil {
		return nil
	}
	return New(op, kind, err)
}

// WrapDatabaseError is a convenience function for wrapping database errors.
func WrapDatabaseError(err error, op string) *KeymasterError {
	return Wrap(err, op, ErrDatabase)
}

// WrapSSHError is a convenience function for wrapping SSH errors.
func WrapSSHError(err error, op string) *KeymasterError {
	return Wrap(err, op, ErrSSHConnection)
}

// WrapValidationError is a convenience function for wrapping validation errors.
func WrapValidationError(err error, op string) *KeymasterError {
	return Wrap(err, op, ErrValidation)
}

// IsKind checks if an error is of a specific kind.
func IsKind(err error, kind ErrorKind) bool {
	if ke, ok := err.(*KeymasterError); ok {
		return ke.Kind == kind
	}
	return false
}

// GetKind returns the error kind if the error is a KeymasterError, otherwise returns ErrInternal.
func GetKind(err error) ErrorKind {
	if ke, ok := err.(*KeymasterError); ok {
		return ke.Kind
	}
	return ErrInternal
}

// GetOperation returns the operation that failed if the error is a KeymasterError.
func GetOperation(err error) string {
	if ke, ok := err.(*KeymasterError); ok {
		return ke.Op
	}
	return "unknown"
}