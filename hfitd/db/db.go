/*
 * Hot Fixture Tool Daemon (hfitd)
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */
// Package db provides database connection and management
package db

import (
	"database/sql"
	"fmt"

	"hfitd/config"

	_ "github.com/lib/pq"
)

type Database struct {
	conn *sql.DB
}

// NewDatabase creates a new database connection
func NewDatabase(cfg config.DatabaseConfig) (*Database, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name)

	conn, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	return &Database{conn: conn}, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.conn.Close()
}

// GetConn returns the underlying database connection
func (d *Database) GetConn() *sql.DB {
	return d.conn
}
