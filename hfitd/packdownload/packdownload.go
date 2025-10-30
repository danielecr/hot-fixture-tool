/*
 * Hot Fixture Tool Daemon (hfitd)
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

// Package packdownload handles package creation and download functionality
package packdownload

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"hfitd/config"
	"hfitd/db"
	"hfitd/mysqlexp"
	"hfitd/pgexp"
	redisclient "hfitd/redis"

	"gopkg.in/yaml.v2"
)

// PackageDefinition represents the YAML package structure
type PackageDefinition struct {
	HfitVersion int                         `yaml:"hfitVersion"`
	Name        string                      `yaml:"name"`
	Exports     map[string]ExportDefinition `yaml:"exports"`
}

// ExportDefinition represents an individual export item
type ExportDefinition struct {
	Type string                 `yaml:"type"`
	Data map[string]interface{} `yaml:"data"`
}

// PackageProcessor handles package creation and processing
type PackageProcessor struct {
	cfg         *config.Config
	dbManager   *db.DatabaseManager
	redisClient *redisclient.Client
}

// NewPackageProcessor creates a new package processor
func NewPackageProcessor(cfg *config.Config, dbManager *db.DatabaseManager, redisClient *redisclient.Client) *PackageProcessor {
	return &PackageProcessor{
		cfg:         cfg,
		dbManager:   dbManager,
		redisClient: redisClient,
	}
}

// ProcessPackage processes a package YAML and creates a tar.gz file
func (p *PackageProcessor) ProcessPackage(userEmail, packname string, yamlData []byte) (string, error) {
	// Store package request in Redis
	if err := p.storePackageInRedis(userEmail, packname, yamlData); err != nil {
		return "", fmt.Errorf("failed to store package in Redis: %v", err)
	}

	// Parse YAML
	var packageDef PackageDefinition
	if err := yaml.Unmarshal(yamlData, &packageDef); err != nil {
		return "", fmt.Errorf("failed to parse YAML: %v", err)
	}

	// Create temporary directory for package files
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("hfit_package_%s_", packname))
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create package directory inside temp
	packageDir := filepath.Join(tempDir, packname)
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create package directory: %v", err)
	}

	// Process each export
	for filename, exportDef := range packageDef.Exports {
		exportPath := filepath.Join(packageDir, filename)
		if err := p.processExport(exportDef, exportPath); err != nil {
			return "", fmt.Errorf("failed to process export %s: %v", filename, err)
		}
	}

	// Create tar.gz file
	tarPath := filepath.Join(tempDir, fmt.Sprintf("%s.tar.gz", packname))
	if err := p.createTarGz(packageDir, tarPath); err != nil {
		return "", fmt.Errorf("failed to create tar.gz: %v", err)
	}

	return tarPath, nil
}

// storePackageInRedis stores package info in Redis
func (p *PackageProcessor) storePackageInRedis(userEmail, packname string, yamlData []byte) error {
	ctx := context.Background()

	// Store YAML definition
	yamlKey := fmt.Sprintf("%s_pkg_%s", userEmail, packname)
	if err := p.redisClient.Set(ctx, yamlKey, string(yamlData), 0); err != nil {
		return fmt.Errorf("failed to store YAML in Redis: %v", err)
	}

	// Store timestamp
	timestampKey := fmt.Sprintf("%s_pkg_%s_timestamp", userEmail, packname)
	timestamp := time.Now().Unix()
	if err := p.redisClient.Set(ctx, timestampKey, fmt.Sprintf("%d", timestamp), 0); err != nil {
		return fmt.Errorf("failed to store timestamp in Redis: %v", err)
	}

	return nil
}

// buildDSN builds a DSN string from database config
func (p *PackageProcessor) buildDSN(dbms string) (string, error) {
	cfg, err := p.dbManager.GetConfig(dbms)
	if err != nil {
		return "", fmt.Errorf("failed to get config for DBMS %s: %v", dbms, err)
	}

	switch strings.ToLower(cfg.Type) {
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name), nil
	case "postgres", "postgresql":
		return fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=%s sslmode=disable",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name), nil
	default:
		return "", fmt.Errorf("unsupported database type: %s", cfg.Type)
	}
}

// processExport processes a single export definition
func (p *PackageProcessor) processExport(exportDef ExportDefinition, exportPath string) error {
	switch exportDef.Type {
	case "dbcreate":
		return p.processDBCreate(exportDef, exportPath)
	case "create-table":
		return p.processTableCreate(exportDef, exportPath)
	case "table-data":
		return p.processTableData(exportDef, exportPath)
	case "file":
		return p.processFile(exportDef, exportPath)
	default:
		return fmt.Errorf("unsupported export type: %s", exportDef.Type)
	}
}

// processDBCreate handles database creation export
func (p *PackageProcessor) processDBCreate(exportDef ExportDefinition, exportPath string) error {
	dbms, ok := exportDef.Data["dbms"].(string)
	if !ok {
		return fmt.Errorf("dbms not specified or invalid")
	}

	// Get DSN for export tools
	dsn, err := p.buildDSN(dbms)
	if err != nil {
		return fmt.Errorf("failed to build DSN: %v", err)
	}

	switch strings.ToLower(dbms) {
	case "mysql":
		return mysqlexp.ExportDatabase(dsn, exportPath)
	case "postgres", "postgresql":
		// For PostgreSQL, we need the database name
		dbname, ok := exportDef.Data["dbname"].(string)
		if !ok {
			dbname = "postgres" // default
		}
		return pgexp.ExportDatabase(dsn, dbname, exportPath)
	default:
		return fmt.Errorf("unsupported DBMS for database export: %s", dbms)
	}
}

// processTableCreate handles table creation export
func (p *PackageProcessor) processTableCreate(exportDef ExportDefinition, exportPath string) error {
	dbms, ok := exportDef.Data["dbms"].(string)
	if !ok {
		return fmt.Errorf("dbms not specified or invalid")
	}

	tablelistRaw, ok := exportDef.Data["tablelist"]
	if !ok {
		return fmt.Errorf("tablelist not specified")
	}

	// Convert interface{} to []string
	var tablelist []string
	switch v := tablelistRaw.(type) {
	case []interface{}:
		for _, item := range v {
			if str, ok := item.(string); ok {
				tablelist = append(tablelist, str)
			}
		}
	case []string:
		tablelist = v
	default:
		return fmt.Errorf("invalid tablelist format")
	}

	// Get DSN for export tools
	dsn, err := p.buildDSN(dbms)
	if err != nil {
		return fmt.Errorf("failed to build DSN: %v", err)
	}

	// Create combined SQL file with all table definitions
	var sqlContent strings.Builder

	for _, table := range tablelist {
		switch strings.ToLower(dbms) {
		case "mysql":
			tempFile := exportPath + "_temp_" + table
			if err := mysqlexp.ExportTable(dsn, table, tempFile); err != nil {
				return fmt.Errorf("failed to export table %s: %v", table, err)
			}
			content, err := os.ReadFile(tempFile)
			if err != nil {
				return fmt.Errorf("failed to read temp file for table %s: %v", table, err)
			}
			sqlContent.WriteString(string(content))
			sqlContent.WriteString("\n")
			os.Remove(tempFile)
		case "postgres", "postgresql":
			tempFile := exportPath + "_temp_" + table
			if err := pgexp.ExportTable(dsn, table, tempFile); err != nil {
				return fmt.Errorf("failed to export table %s: %v", table, err)
			}
			content, err := os.ReadFile(tempFile)
			if err != nil {
				return fmt.Errorf("failed to read temp file for table %s: %v", table, err)
			}
			sqlContent.WriteString(string(content))
			sqlContent.WriteString("\n")
			os.Remove(tempFile)
		default:
			return fmt.Errorf("unsupported DBMS for table export: %s", dbms)
		}
	}

	return os.WriteFile(exportPath, []byte(sqlContent.String()), 0644)
}

// processTableData handles table data export
func (p *PackageProcessor) processTableData(exportDef ExportDefinition, exportPath string) error {
	dbms, ok := exportDef.Data["dbms"].(string)
	if !ok {
		return fmt.Errorf("dbms not specified or invalid")
	}

	table, ok := exportDef.Data["table"].(string)
	if !ok {
		return fmt.Errorf("table not specified")
	}

	// Get DSN for export tools
	dsn, err := p.buildDSN(dbms)
	if err != nil {
		return fmt.Errorf("failed to build DSN: %v", err)
	}

	// Check for filter
	filter, hasFilter := exportDef.Data["filter"].(string)

	switch strings.ToLower(dbms) {
	case "mysql":
		if hasFilter {
			// For MySQL, we'd need to enhance mysqlexp to support filters
			return mysqlexp.ExportTableData(dsn, table, exportPath)
		} else {
			return mysqlexp.ExportTableData(dsn, table, exportPath)
		}
	case "postgres", "postgresql":
		if hasFilter {
			return pgexp.ExportTableDataWithFilter(dsn, table, filter, exportPath)
		} else {
			return pgexp.ExportTableData(dsn, table, exportPath)
		}
	default:
		return fmt.Errorf("unsupported DBMS for table data export: %s", dbms)
	}
}

// processFile handles file export
func (p *PackageProcessor) processFile(exportDef ExportDefinition, exportPath string) error {
	volume, ok := exportDef.Data["volume"].(string)
	if !ok {
		return fmt.Errorf("volume not specified or invalid")
	}

	path, ok := exportDef.Data["path"].(string)
	if !ok {
		return fmt.Errorf("path not specified or invalid")
	}

	// Get volume configuration
	volumeConfig, exists := p.cfg.Volumes[volume]
	if !exists {
		return fmt.Errorf("volume %s not found", volume)
	}

	sourcePath := filepath.Join(volumeConfig.Path, path)

	// Copy file
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(exportPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// createTarGz creates a tar.gz archive from the package directory
func (p *PackageProcessor) createTarGz(packageDir, tarPath string) error {
	tarFile, err := os.Create(tarPath)
	if err != nil {
		return fmt.Errorf("failed to create tar file: %v", err)
	}
	defer tarFile.Close()

	gzWriter := gzip.NewWriter(tarFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	return filepath.Walk(packageDir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		// Update header name to be relative to package directory
		relPath, err := filepath.Rel(filepath.Dir(packageDir), file)
		if err != nil {
			return err
		}
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// If it's a file, write its content
		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return err
			}
			defer data.Close()

			_, err = io.Copy(tarWriter, data)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
