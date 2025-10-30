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

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

type DatabaseManager struct {
	connections map[string]*sql.DB
	configs     map[string]config.DatabaseConfig
}

// NewDatabaseManager creates a new database manager with multiple DBMS connections
func NewDatabaseManager(dbConfigs map[string]config.DatabaseConfig) (*DatabaseManager, error) {
	manager := &DatabaseManager{
		connections: make(map[string]*sql.DB),
		configs:     dbConfigs,
	}

	for providerName, cfg := range dbConfigs {
		conn, err := createConnection(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to %s: %v", providerName, err)
		}
		manager.connections[providerName] = conn
	}

	return manager, nil
}

// createConnection creates a database connection based on the database type
func createConnection(cfg config.DatabaseConfig) (*sql.DB, error) {
	var connStr string
	var driverName string

	switch cfg.Type {
	case "postgres":
		driverName = "postgres"
		connStr = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name)
	case "mysql":
		driverName = "mysql"
		connStr = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	conn, err := sql.Open(driverName, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	return conn, nil
}

// Close closes all database connections
func (dm *DatabaseManager) Close() error {
	for _, conn := range dm.connections {
		if err := conn.Close(); err != nil {
			return err
		}
	}
	return nil
}

// GetConnection returns the database connection for a specific provider
func (dm *DatabaseManager) GetConnection(providerName string) (*sql.DB, error) {
	conn, exists := dm.connections[providerName]
	if !exists {
		return nil, fmt.Errorf("database provider '%s' not found", providerName)
	}
	return conn, nil
}

// GetProviders returns a list of available database providers
func (dm *DatabaseManager) GetProviders() []string {
	providers := make([]string, 0, len(dm.connections))
	for provider := range dm.connections {
		providers = append(providers, provider)
	}
	return providers
}

// GetConfig returns the configuration for a specific provider
func (dm *DatabaseManager) GetConfig(providerName string) (config.DatabaseConfig, error) {
	cfg, exists := dm.configs[providerName]
	if !exists {
		return config.DatabaseConfig{}, fmt.Errorf("database provider '%s' not found", providerName)
	}
	return cfg, nil
}
