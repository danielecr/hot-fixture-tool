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
	"fmt"
	"os"

	"github.com/aliakseiz/go-mysqldump"
)

// ExportDatabase exports the entire database creation SQL to the specified file
func ExportDatabase(dsn, exportPath string) error {
	dumper, err := mysqldump.New(dsn)
	if err != nil {
		return fmt.Errorf("failed to create mysqldump instance: %v", err)
	}

	sql, err := dumper.Dump()
	if err != nil {
		return fmt.Errorf("failed to dump database: %v", err)
	}

	if err := os.WriteFile(exportPath, []byte(sql), 0644); err != nil {
		return fmt.Errorf("failed to write export file: %v", err)
	}

	return nil
}

// ExportTable exports the table creation SQL for a specific table to the specified file
func ExportTable(dsn, tableName, exportPath string) error {
	dumper, err := mysqldump.New(dsn)
	if err != nil {
		return fmt.Errorf("failed to create mysqldump instance: %v", err)
	}

	sql, err := dumper.DumpTable(tableName)
	if err != nil {
		return fmt.Errorf("failed to dump table %s: %v", tableName, err)
	}

	if err := os.WriteFile(exportPath, []byte(sql), 0644); err != nil {
		return fmt.Errorf("failed to write export file: %v", err)
	}

	return nil
}

// ExportTableData exports the data for a specific table to the specified file
func ExportTableData(dsn, tableName, exportPath string) error {
	dumper, err := mysqldump.New(dsn)
	if err != nil {
		return fmt.Errorf("failed to create mysqldump instance: %v", err)
	}

	sql, err := dumper.DumpTableData(tableName)
	if err != nil {
		return fmt.Errorf("failed to dump data for table %s: %v", tableName, err)
	}

	if err := os.WriteFile(exportPath, []byte(sql), 0644); err != nil {
		return fmt.Errorf("failed to write export file: %v", err)
	}

	return nil
}
