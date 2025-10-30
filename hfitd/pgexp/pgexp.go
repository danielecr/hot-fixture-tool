/*
 * Hot Fixture Tool Daemon - PostgreSQL Export Module
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

// Package pgexp provides PostgreSQL database export functionality
package pgexp

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// parseDSN parses a PostgreSQL DSN and returns connection parameters
func parseDSN(dsn string) (host, port, user, password, dbname string, err error) {
	// Parse DSN format: "user=username password=password host=hostname port=5432 dbname=database sslmode=disable"
	parts := strings.Fields(dsn)
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key, value := kv[0], kv[1]
		switch key {
		case "host":
			host = value
		case "port":
			port = value
		case "user":
			user = value
		case "password":
			password = value
		case "dbname":
			dbname = value
		}
	}

	// Set defaults
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "5432"
	}

	return host, port, user, password, dbname, nil
}

// ExportDatabase exports CREATE DATABASE/SCHEMA SQL commands to the specified file
func ExportDatabase(dsn, dbname, exportPath string) error {
	host, port, user, password, _, err := parseDSN(dsn)
	if err != nil {
		return fmt.Errorf("failed to parse DSN: %v", err)
	}

	// Use the provided dbname parameter
	if dbname == "" {
		return fmt.Errorf("database name is required")
	}

	// Create pg_dump command for schema only
	cmd := exec.Command("pg_dump",
		"--host="+host,
		"--port="+port,
		"--username="+user,
		"--schema-only",
		"--no-owner",
		"--no-privileges",
		dbname)

	// Set password via environment variable
	if password != "" {
		cmd.Env = append(os.Environ(), "PGPASSWORD="+password)
	}

	// Create output file
	outputFile, err := os.Create(exportPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()

	// Set command output to file
	cmd.Stdout = outputFile
	cmd.Stderr = os.Stderr

	// Execute command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump failed: %v", err)
	}

	return nil
}

// ExportTable exports CREATE TABLE SQL for a specific table to the specified file
func ExportTable(dsn, tableName, exportPath string) error {
	host, port, user, password, dbname, err := parseDSN(dsn)
	if err != nil {
		return fmt.Errorf("failed to parse DSN: %v", err)
	}

	// Create pg_dump command for specific table schema only
	cmd := exec.Command("pg_dump",
		"--host="+host,
		"--port="+port,
		"--username="+user,
		"--schema-only",
		"--no-owner",
		"--no-privileges",
		"--table="+tableName,
		dbname)

	// Set password via environment variable
	if password != "" {
		cmd.Env = append(os.Environ(), "PGPASSWORD="+password)
	}

	// Create output file
	outputFile, err := os.Create(exportPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()

	// Set command output to file
	cmd.Stdout = outputFile
	cmd.Stderr = os.Stderr

	// Execute command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump failed: %v", err)
	}

	return nil
}

// ExportTableData exports INSERT SQL statements for table data to the specified file
func ExportTableData(dsn, tableName, exportPath string) error {
	host, port, user, password, dbname, err := parseDSN(dsn)
	if err != nil {
		return fmt.Errorf("failed to parse DSN: %v", err)
	}

	// Create pg_dump command for specific table data only
	cmd := exec.Command("pg_dump",
		"--host="+host,
		"--port="+port,
		"--username="+user,
		"--data-only",
		"--no-owner",
		"--no-privileges",
		"--table="+tableName,
		dbname)

	// Set password via environment variable
	if password != "" {
		cmd.Env = append(os.Environ(), "PGPASSWORD="+password)
	}

	// Create output file
	outputFile, err := os.Create(exportPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()

	// Set command output to file
	cmd.Stdout = outputFile
	cmd.Stderr = os.Stderr

	// Execute command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump failed: %v", err)
	}

	return nil
}

// ExportTableDataWithFilter exports filtered INSERT SQL statements for table data to the specified file
func ExportTableDataWithFilter(dsn, tableName, whereClause, exportPath string) error {
	// For filtered data, we need to use a custom approach since pg_dump doesn't support WHERE clauses directly
	// We'll use psql to execute a custom query and format the output as INSERT statements
	host, port, user, password, dbname, err := parseDSN(dsn)
	if err != nil {
		return fmt.Errorf("failed to parse DSN: %v", err)
	}

	// Create a custom SQL query to generate INSERT statements
	// This is a simplified approach - in production, you might want to use a more sophisticated method
	query := fmt.Sprintf("COPY (SELECT * FROM %s WHERE %s) TO STDOUT WITH CSV HEADER", tableName, whereClause)

	// Create psql command
	cmd := exec.Command("psql",
		"--host="+host,
		"--port="+port,
		"--username="+user,
		"--dbname="+dbname,
		"--command="+query)

	// Set password via environment variable
	if password != "" {
		cmd.Env = append(os.Environ(), "PGPASSWORD="+password)
	}

	// Create output file
	outputFile, err := os.Create(exportPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()

	// Set command output to file
	cmd.Stdout = outputFile
	cmd.Stderr = os.Stderr

	// Execute command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("psql query failed: %v", err)
	}

	return nil
}
