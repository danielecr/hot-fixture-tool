/*
 * Hot Fixture Tool Daemon - Database API Operations
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

// Package dbapi provides database API operations and utilities
package dbapi

import (
	"database/sql"
	"fmt"
	"strings"

	"hfitd/db"
)

/*
* getDatabases retrieves the list of databases for a specific DBMS provider.
 */
func GetDatabases(conn *sql.DB, dbms string, databaseManager *db.DatabaseManager) ([]string, error) {
	var query string

	switch strings.ToLower(dbms) {
	case "mysql":
		query = "SHOW DATABASES"
	case "postgres":
		query = "SELECT datname FROM pg_database WHERE datistemplate = false"
	default:
		return nil, fmt.Errorf("unsupported DBMS: %s", dbms)
	}

	rows, err := conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			return nil, err
		}
		databases = append(databases, dbName)
	}

	return databases, nil
}

/*
* checkDatabaseExists checks if a database exists for a specific DBMS provider.
 */
func CheckDatabaseExists(conn *sql.DB, dbms string, dbid string) (bool, error) {
	databases, err := GetDatabases(conn, dbms, nil)
	if err != nil {
		return false, err
	}

	for _, db := range databases {
		if db == dbid {
			return true, nil
		}
	}
	return false, nil
}

/*
* checkTableExists checks if a table exists in a specific database.
 */
func CheckTableExists(conn *sql.DB, dbms string, dbid string, tableid string) (bool, error) {
	tables, err := GetTables(conn, dbms, dbid)
	if err != nil {
		return false, err
	}

	for _, table := range tables {
		if table == tableid {
			return true, nil
		}
	}
	return false, nil
}

/*
* getTables retrieves the list of tables for a specific database.
 */
func GetTables(conn *sql.DB, dbms string, dbid string) ([]string, error) {
	var query string

	switch strings.ToLower(dbms) {
	case "mysql":
		query = fmt.Sprintf("SELECT table_name FROM information_schema.tables WHERE table_schema = '%s'", dbid)
	case "postgres":
		// For PostgreSQL, we need to connect to the specific database first
		// This is a simplified version - in practice you might need a separate connection
		query = "SELECT tablename FROM pg_tables WHERE schemaname = 'public'"
	default:
		return nil, fmt.Errorf("unsupported DBMS: %s", dbms)
	}

	rows, err := conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	return tables, nil
}

/*
* getTableRowsWithFilter retrieves rows from a specific table with optional filtering.
* Maintains backward compatibility with traditional JSON array response.
 */
func GetTableRowsWithFilter(conn *sql.DB, dbms string, dbid string, tableid string, filterpart string) ([]map[string]interface{}, error) {
	// Build the base query
	baseQuery := fmt.Sprintf("SELECT * FROM %s", tableid)

	// Add filter part if provided and validated
	var finalQuery string
	if filterpart != "" {
		validatedFilter, err := ValidateFilterPart(filterpart)
		if err != nil {
			return nil, fmt.Errorf("invalid filterpart: %v", err)
		}
		finalQuery = fmt.Sprintf("%s %s", baseQuery, validatedFilter)
	} else {
		// Default limit to prevent accidental massive queries
		finalQuery = fmt.Sprintf("%s LIMIT 100", baseQuery)
	}

	rows, err := conn.Query(finalQuery)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %v", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %v", err)
	}

	var result []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}

		result = append(result, row)
	}

	return result, nil
}
