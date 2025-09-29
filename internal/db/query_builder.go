// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"fmt"
	"strings"
)

// QueryBuilder provides a fluent interface for building SQL queries.
// It handles database-specific syntax differences through the SQLDialect interface.
type QueryBuilder struct {
	dialect    SQLDialect
	table      string
	queryType  queryType
	columns    []string
	values     []interface{}
	conditions []condition
	orderBy    []string
	limit      int
	offset     int
}

type queryType int

const (
	selectQuery queryType = iota
	insertQuery
	updateQuery
	deleteQuery
)

type condition struct {
	field    string
	operator string
	value    interface{}
}

// NewQueryBuilder creates a new QueryBuilder with the given dialect.
func NewQueryBuilder(dialect SQLDialect) *QueryBuilder {
	return &QueryBuilder{
		dialect:    dialect,
		columns:    make([]string, 0),
		values:     make([]interface{}, 0),
		conditions: make([]condition, 0),
		orderBy:    make([]string, 0),
	}
}

// Select starts a SELECT query builder.
func (qb *QueryBuilder) Select(columns ...string) *QueryBuilder {
	qb.queryType = selectQuery
	qb.columns = columns
	return qb
}

// Insert starts an INSERT query builder.
func (qb *QueryBuilder) Insert(table string) *QueryBuilder {
	qb.queryType = insertQuery
	qb.table = table
	return qb
}

// Update starts an UPDATE query builder.
func (qb *QueryBuilder) Update(table string) *QueryBuilder {
	qb.queryType = updateQuery
	qb.table = table
	return qb
}

// Delete starts a DELETE query builder.
func (qb *QueryBuilder) Delete() *QueryBuilder {
	qb.queryType = deleteQuery
	return qb
}

// From sets the table for SELECT queries.
func (qb *QueryBuilder) From(table string) *QueryBuilder {
	qb.table = table
	return qb
}

// Values sets the values for INSERT queries.
func (qb *QueryBuilder) Values(values ...interface{}) *QueryBuilder {
	qb.values = append(qb.values, values...)
	return qb
}

// Set adds a column=value pair for UPDATE queries.
func (qb *QueryBuilder) Set(column string, value interface{}) *QueryBuilder {
	qb.columns = append(qb.columns, column)
	qb.values = append(qb.values, value)
	return qb
}

// Where adds a WHERE condition.
func (qb *QueryBuilder) Where(field string, value interface{}) *QueryBuilder {
	return qb.WhereOp(field, "=", value)
}

// WhereOp adds a WHERE condition with a custom operator.
func (qb *QueryBuilder) WhereOp(field, operator string, value interface{}) *QueryBuilder {
	qb.conditions = append(qb.conditions, condition{
		field:    field,
		operator: operator,
		value:    value,
	})
	return qb
}

// WhereIn adds a WHERE field IN (...) condition.
func (qb *QueryBuilder) WhereIn(field string, values ...interface{}) *QueryBuilder {
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = qb.dialect.Placeholder(len(qb.values) + i + 1)
	}

	inClause := fmt.Sprintf("(%s)", strings.Join(placeholders, ", "))
	qb.conditions = append(qb.conditions, condition{
		field:    field,
		operator: "IN",
		value:    inClause,
	})
	qb.values = append(qb.values, values...)
	return qb
}

// OrderBy adds an ORDER BY clause.
func (qb *QueryBuilder) OrderBy(columns ...string) *QueryBuilder {
	qb.orderBy = append(qb.orderBy, columns...)
	return qb
}

// Limit sets the LIMIT clause.
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limit = limit
	return qb
}

// Offset sets the OFFSET clause.
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offset = offset
	return qb
}

// Build constructs the final SQL query and returns it with the parameter values.
func (qb *QueryBuilder) Build() (string, []interface{}) {
	switch qb.queryType {
	case selectQuery:
		return qb.buildSelect()
	case insertQuery:
		return qb.buildInsert()
	case updateQuery:
		return qb.buildUpdate()
	case deleteQuery:
		return qb.buildDelete()
	default:
		return "", nil
	}
}

func (qb *QueryBuilder) buildSelect() (string, []interface{}) {
	var query strings.Builder
	var args []interface{}

	// SELECT columns
	query.WriteString("SELECT ")
	if len(qb.columns) == 0 {
		query.WriteString("*")
	} else {
		query.WriteString(strings.Join(qb.columns, ", "))
	}

	// FROM table
	query.WriteString(" FROM ")
	query.WriteString(qb.table)

	// WHERE conditions
	if len(qb.conditions) > 0 {
		query.WriteString(" WHERE ")
		conditions := make([]string, len(qb.conditions))
		for i, cond := range qb.conditions {
			if cond.operator == "IN" {
				conditions[i] = fmt.Sprintf("%s %s %v", cond.field, cond.operator, cond.value)
			} else {
				conditions[i] = fmt.Sprintf("%s %s %s", cond.field, cond.operator, qb.dialect.Placeholder(len(args)+1))
				args = append(args, cond.value)
			}
		}
		query.WriteString(strings.Join(conditions, " AND "))
	}

	// ORDER BY
	if len(qb.orderBy) > 0 {
		query.WriteString(" ORDER BY ")
		query.WriteString(strings.Join(qb.orderBy, ", "))
	}

	// LIMIT
	if qb.limit > 0 {
		query.WriteString(fmt.Sprintf(" LIMIT %d", qb.limit))
	}

	// OFFSET
	if qb.offset > 0 {
		query.WriteString(fmt.Sprintf(" OFFSET %d", qb.offset))
	}

	// Add condition values that weren't handled in the IN clause
	for _, cond := range qb.conditions {
		if cond.operator != "IN" {
			// Already added to args above
		}
	}

	return query.String(), args
}

func (qb *QueryBuilder) buildInsert() (string, []interface{}) {
	var query strings.Builder

	query.WriteString("INSERT INTO ")
	query.WriteString(qb.table)

	if len(qb.columns) > 0 {
		// INSERT INTO table (col1, col2) VALUES (?, ?)
		query.WriteString(" (")
		query.WriteString(strings.Join(qb.columns, ", "))
		query.WriteString(") VALUES (")
		query.WriteString(qb.dialect.PlaceholderString(len(qb.columns)))
		query.WriteString(")")
		return query.String(), qb.values[:len(qb.columns)]
	}

	// Simple VALUES clause
	query.WriteString(" VALUES (")
	query.WriteString(qb.dialect.PlaceholderString(len(qb.values)))
	query.WriteString(")")

	return query.String(), qb.values
}

func (qb *QueryBuilder) buildUpdate() (string, []interface{}) {
	var query strings.Builder
	var args []interface{}

	query.WriteString("UPDATE ")
	query.WriteString(qb.table)
	query.WriteString(" SET ")

	// SET clauses
	setClauses := make([]string, len(qb.columns))
	for i, col := range qb.columns {
		setClauses[i] = fmt.Sprintf("%s = %s", col, qb.dialect.Placeholder(i+1))
		args = append(args, qb.values[i])
	}
	query.WriteString(strings.Join(setClauses, ", "))

	// WHERE conditions
	if len(qb.conditions) > 0 {
		query.WriteString(" WHERE ")
		conditions := make([]string, len(qb.conditions))
		for i, cond := range qb.conditions {
			conditions[i] = fmt.Sprintf("%s %s %s", cond.field, cond.operator, qb.dialect.Placeholder(len(args)+1))
			args = append(args, cond.value)
		}
		query.WriteString(strings.Join(conditions, " AND "))
	}

	return query.String(), args
}

func (qb *QueryBuilder) buildDelete() (string, []interface{}) {
	var query strings.Builder
	var args []interface{}

	query.WriteString("DELETE FROM ")
	query.WriteString(qb.table)

	// WHERE conditions
	if len(qb.conditions) > 0 {
		query.WriteString(" WHERE ")
		conditions := make([]string, len(qb.conditions))
		for i, cond := range qb.conditions {
			conditions[i] = fmt.Sprintf("%s %s %s", cond.field, cond.operator, qb.dialect.Placeholder(i+1))
			args = append(args, cond.value)
		}
		query.WriteString(strings.Join(conditions, " AND "))
	}

	return query.String(), args
}