/*
 * Hot Fixture Tool Daemon (hfitd)
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

package mysqlexp

// MySQLExport provides functions to export MySQL database structure and data
// it uses https://pkg.go.dev/github.com/aliakseiz/go-mysqldump for mysqldump

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliakseiz/go-mysqldump"
	_ "github.com/go-sql-driver/mysql"
)

// ExportDatabase exports the entire database creation SQL to the specified file
func ExportDatabase(dsn, exportPath string) error {
	// Open database connection
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %v", err)
	}
	defer db.Close()

	// Create temporary directory for the dump
	tempDir := filepath.Dir(exportPath)
	filename := filepath.Base(exportPath)
	// Remove .sql extension if present for the format
	if filepath.Ext(filename) == ".sql" {
		filename = filename[:len(filename)-4]
	}

	// Register the dumper - it will create the file with .sql extension
	dumper, err := mysqldump.Register(db, tempDir, filename, "")
	if err != nil {
		return fmt.Errorf("failed to register mysqldump: %v", err)
	}
	defer dumper.Close()

	// Perform the dump
	if err = dumper.Dump(); err != nil {
		return fmt.Errorf("failed to dump database: %v", err)
	}

	return nil
}

// ExportTable exports the table creation SQL for a specific table to the specified file
func ExportTable(dsn, tableName, exportPath string) error {
	// Open database connection
	log.Println("ExportTable: opening database connection", dsn)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %v", err)
	}
	defer db.Close()

	// split tableName into database and table if it contains a dot
	var dbName, tblName string
	parts := strings.SplitN(tableName, ".", 2)
	if len(parts) == 2 {
		dbName = parts[0]
		tblName = parts[1]
	} else {
		tblName = tableName
	}
	log.Println("ExportTable:", dbName, tblName)
	// For individual table export, we need to use the CreateTable method
	tempDir := filepath.Dir(exportPath)
	//filename := fmt.Sprintf("table_%s_%d", tableName, time.Now().Unix())
	filename := fmt.Sprintf("table_%s_20060102T150405", tableName)

	dumper, err := mysqldump.Register(db, tempDir, filename, dbName)
	if err != nil {
		return fmt.Errorf("failed to register mysqldump: %v", err)
	}
	defer dumper.Close()

	// Create table struct and get CREATE TABLE statement
	table, err := dumper.CreateTable(dbName, tblName)
	if err != nil {
		return fmt.Errorf("failed to create table struct: %v", err)
	}

	createSQL, err := table.CreateSQL()
	if err != nil {
		return fmt.Errorf("failed to get CREATE TABLE SQL: %v", err)
	}

	// Write the CREATE TABLE statement to file
	if err := os.WriteFile(exportPath, []byte(createSQL), 0644); err != nil {
		return fmt.Errorf("failed to write export file: %v", err)
	}

	return nil
}

// ExportTableData exports the data for a specific table to the specified file
func ExportTableData(dsn, tableName, exportPath string) error {
	// Open database connection
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %v", err)
	}
	defer db.Close()

	// Create a file to write the dump to
	outputFile, err := os.Create(exportPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()

	// Create a Data struct to configure the dump
	dumper := &mysqldump.Data{
		Out:        outputFile,
		Connection: db,
	}

	// Get the specific table
	table, err := dumper.CreateTable("", tableName)
	if err != nil {
		return fmt.Errorf("failed to create table struct: %v", err)
	}

	// Initialize the table
	if err = table.Init(); err != nil {
		return fmt.Errorf("failed to initialize table: %v", err)
	}

	// Write only the data (INSERT statements) for this table
	if err = dumper.WriteTable(table); err != nil {
		return fmt.Errorf("failed to write table data: %v", err)
	}

	return nil
}
