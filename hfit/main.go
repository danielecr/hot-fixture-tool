/*
 * Hot Fixture Tool CLI
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

// Package main provides the hfit CLI tool for interacting with hfitd
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"hfit/api"
	"hfit/auth"
	"hfit/config"

	"gopkg.in/yaml.v2"
)

// fatalError prints an error message with support contact information and exits
func fatalError(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s: %v\n", msg, err)
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	}
	fmt.Fprintf(os.Stderr, "\nFor support, contact: Daniele Cruciani <daniele@smartango.com>\n")
	fmt.Fprintf(os.Stderr, "Project repository: https://github.com/danielecr/hot-fixture-tool\n")
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "config":
		handleConfigCommand()
	case "login":
		handleLoginCommand()
	case "help", "-h", "--help":
		printUsage()
	case "dbmss":
		handleDbmssCommand()
	case "dbs":
		handleDbsCommand()
	case "tables":
		handleTablesCommand()
	case "rows":
		handleRowsCommand()
	case "files":
		handleFilesCommand()
	case "pkg":
		handlePkgCommand()
	case "download":
		handleDownloadCommand()
	default:
		fmt.Printf("Error: Unknown command '%s'\n\n", command)
		fmt.Printf("Run 'hfit help' for usage information.\n")
		fmt.Printf("For support, contact: Daniele Cruciani <daniele@smartango.com>\n")
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("hfit - Hot Fixture Tool CLI")
	fmt.Println("Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>")
	fmt.Println("GitHub: https://github.com/danielecr/hot-fixture-tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  hfit help                                           Show this help message")
	fmt.Println("  hfit config <hfitd_host> <email> <public_key_path>  Configure connection")
	fmt.Println("  hfit login                                          Authenticate and get JWT token")
	fmt.Println("  hfit dbmss                                          List available DBMS providers")
	fmt.Println("  hfit dbs <dbms>                                     List databases for DBMS provider")
	fmt.Println("  hfit tables <dbms> <db_id>                          List tables in database")
	fmt.Println("  hfit rows <dbms> <db_id> <table_id>                 List rows in table")
	fmt.Println("  hfit files <volume>                                 List files in volume (streaming)")
	fmt.Println("  hfit pkg create <package.yaml> <name>               Create new package definition")
	fmt.Println("  hfit pkg add <package.yaml> <name> <type> <data>    Add resource to package")
	fmt.Println("  hfit pkg rm <package.yaml> <name>                   Remove resource from package")
	fmt.Println("  hfit pkg downpack <package.yaml>                    Download and unpack package")
	fmt.Println("  hfit download <file_path>                           Download file")
	fmt.Println()
	fmt.Println("Configuration is stored in ~/.hfit/config")
	fmt.Println("JWT token is stored in ~/.hfit/token")
	fmt.Println()
	fmt.Println("For support, contact: Daniele Cruciani <daniele@smartango.com>")
	fmt.Println("Project repository: https://github.com/danielecr/hot-fixture-tool")
}

func handleConfigCommand() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: hfit config <hfitd_host> <email> <public_key_path>")
		os.Exit(1)
	}

	cfg := &config.Config{
		HfitdHost: os.Args[2],
		Email:     os.Args[3],
		PublicKey: os.Args[4],
	}

	if err := config.SaveConfig(cfg); err != nil {
		fatalError("Failed to save config", err)
	}

	fmt.Println("Configuration saved successfully")
}

func handleLoginCommand() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fatalError("Failed to load config", err)
	}

	token, err := auth.AuthenticateWithChallenge(cfg.HfitdHost, cfg.Email, cfg.PublicKey)
	if err != nil {
		fatalError("Authentication failed", err)
	}

	if err := config.SaveToken(token); err != nil {
		fatalError("Failed to save token", err)
	}

	fmt.Println("Authentication successful, token saved")
}

func handleDbmssCommand() {
	client := getAuthenticatedClient()

	providers, err := client.ListDBMSProviders()
	if err != nil {
		fatalError("Failed to list DBMS providers", err)
	}

	printJSON(providers)
}

func handleDbsCommand() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: hfit dbs <dbms>")
		os.Exit(1)
	}

	client := getAuthenticatedClient()
	dbms := os.Args[2]

	databases, err := client.ListDatabases(dbms)
	if err != nil {
		fatalError("Failed to list databases", err)
	}

	printJSON(databases)
}

func handleTablesCommand() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: hfit tables <dbms> <db_id>")
		os.Exit(1)
	}

	client := getAuthenticatedClient()
	dbms := os.Args[2]
	dbID := os.Args[3]

	tables, err := client.ListTables(dbms, dbID)
	if err != nil {
		fatalError("Failed to list tables", err)
	}

	printJSON(tables)
}

func handleRowsCommand() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: hfit rows <dbms> <db_id> <table_id>")
		os.Exit(1)
	}

	client := getAuthenticatedClient()
	dbms := os.Args[2]
	dbID := os.Args[3]
	tableID := os.Args[4]

	rows, err := client.ListRows(dbms, dbID, tableID)
	if err != nil {
		fatalError("Failed to list rows", err)
	}

	printJSON(rows)
}

func handleFilesCommand() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: hfit files <volume>")
		fmt.Println("Example: hfit files volume1")
		os.Exit(1)
	}

	client := getAuthenticatedClient()
	volume := os.Args[2]

	err := client.StreamFiles(volume)
	if err != nil {
		fatalError("Failed to list files", err)
	}
}

// PackageDefinition represents the YAML package structure
type PackageDefinition struct {
	Name    string                      `yaml:"name"`
	Exports map[string]ExportDefinition `yaml:"exports"`
}

type ExportDefinition struct {
	Type string                 `yaml:"type"`
	Data map[string]interface{} `yaml:"data"`
}

func handlePkgCommand() {
	if len(os.Args) < 3 {
		fmt.Println("Usage:")
		fmt.Println("  hfit pkg create <package.yaml> <name>")
		fmt.Println("  hfit pkg add <package.yaml> <name> <type> <data>")
		fmt.Println("  hfit pkg rm <package.yaml> <name>")
		fmt.Println("  hfit pkg downpack <package.yaml>")
		os.Exit(1)
	}

	subcommand := os.Args[2]

	switch subcommand {
	case "create":
		handlePkgCreate()
	case "add":
		handlePkgAdd()
	case "rm":
		handlePkgRemove()
	case "downpack":
		handlePkgDownpack()
	default:
		fmt.Printf("Unknown pkg subcommand: %s\n", subcommand)
		os.Exit(1)
	}
}

func handlePkgCreate() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: hfit pkg create <package.yaml> <name>")
		os.Exit(1)
	}

	packageFile := os.Args[3]
	packageName := os.Args[4]

	pkg := PackageDefinition{
		Name:    packageName,
		Exports: make(map[string]ExportDefinition),
	}

	data, err := yaml.Marshal(&pkg)
	if err != nil {
		fatalError("Failed to marshal YAML", err)
	}

	err = os.WriteFile(packageFile, data, 0644)
	if err != nil {
		fatalError("Failed to write package file", err)
	}

	fmt.Printf("Created package definition: %s\n", packageFile)
}

func handlePkgAdd() {
	if len(os.Args) < 7 {
		fmt.Println("Usage: hfit pkg add <package.yaml> <name> <type> <data>")
		fmt.Println("Example: hfit pkg add pkg.yaml db1.sql table-data '{\"dbms\":\"mysql1\",\"table\":\"users\",\"filter\":\"WHERE id < 100\"}'")
		os.Exit(1)
	}

	packageFile := os.Args[3]
	exportName := os.Args[4]
	exportType := os.Args[5]
	exportDataStr := os.Args[6]

	// Parse export data JSON
	var exportData map[string]interface{}
	if err := json.Unmarshal([]byte(exportDataStr), &exportData); err != nil {
		fatalError("Failed to parse export data JSON", err)
	}

	// Validate resource existence
	client := getAuthenticatedClient()
	if err := validateResourceExists(client, exportType, exportData); err != nil {
		fatalError("Resource validation failed", err)
	}

	// Load existing package
	var pkg PackageDefinition
	if data, err := os.ReadFile(packageFile); err == nil {
		if err := yaml.Unmarshal(data, &pkg); err != nil {
			fatalError("Failed to parse existing package file", err)
		}
	} else {
		// Create new package if file doesn't exist
		pkg = PackageDefinition{
			Name:    "package",
			Exports: make(map[string]ExportDefinition),
		}
	}

	// Add new export
	pkg.Exports[exportName] = ExportDefinition{
		Type: exportType,
		Data: exportData,
	}

	// Save package
	data, err := yaml.Marshal(&pkg)
	if err != nil {
		fatalError("Failed to marshal YAML", err)
	}

	err = os.WriteFile(packageFile, data, 0644)
	if err != nil {
		fatalError("Failed to write package file", err)
	}

	fmt.Printf("Added export '%s' to package: %s\n", exportName, packageFile)
}

func handlePkgRemove() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: hfit pkg rm <package.yaml> <name>")
		os.Exit(1)
	}

	packageFile := os.Args[3]
	exportName := os.Args[4]

	// Load existing package
	data, err := os.ReadFile(packageFile)
	if err != nil {
		fatalError("Failed to read package file", err)
	}

	var pkg PackageDefinition
	if err := yaml.Unmarshal(data, &pkg); err != nil {
		fatalError("Failed to parse package file", err)
	}

	// Remove export
	if _, exists := pkg.Exports[exportName]; !exists {
		fatalError("Export not found in package", nil)
	}

	delete(pkg.Exports, exportName)

	// Save package
	data, err = yaml.Marshal(&pkg)
	if err != nil {
		fatalError("Failed to marshal YAML", err)
	}

	err = os.WriteFile(packageFile, data, 0644)
	if err != nil {
		fatalError("Failed to write package file", err)
	}

	fmt.Printf("Removed export '%s' from package: %s\n", exportName, packageFile)
}

func handlePkgDownpack() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: hfit pkg downpack <package.yaml>")
		os.Exit(1)
	}

	packageFile := os.Args[3]

	// Load package definition
	data, err := os.ReadFile(packageFile)
	if err != nil {
		fatalError("Failed to read package file", err)
	}

	var pkg PackageDefinition
	if err := yaml.Unmarshal(data, &pkg); err != nil {
		fatalError("Failed to parse package file", err)
	}

	client := getAuthenticatedClient()

	// Call package download API
	resp, err := client.DownloadPackage(pkg.Name, data)
	if err != nil {
		fatalError("Failed to download package", err)
	}
	defer resp.Close()

	// Save package file
	packageTarFile := pkg.Name + ".tar.gz"
	outFile, err := os.Create(packageTarFile)
	if err != nil {
		fatalError("Failed to create output file", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp)
	if err != nil {
		fatalError("Failed to save package", err)
	}

	fmt.Printf("Package downloaded: %s\n", packageTarFile)
}

func validateResourceExists(client *api.Client, exportType string, exportData map[string]interface{}) error {
	switch exportType {
	case "dbcreate", "table-create", "table-data":
		dbms, ok := exportData["dbms"].(string)
		if !ok {
			return fmt.Errorf("dbms field required for %s export", exportType)
		}

		exists, err := client.CheckDBMSExists(dbms)
		if err != nil {
			return fmt.Errorf("failed to check DBMS existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("DBMS provider '%s' does not exist", dbms)
		}

		if exportType == "table-create" || exportType == "table-data" {
			if tablelist, ok := exportData["tablelist"].([]interface{}); ok {
				// Handle table list for table-create
				for _, tableEntry := range tablelist {
					if tableStr, ok := tableEntry.(string); ok {
						parts := strings.SplitN(tableStr, ".", 2)
						if len(parts) == 2 {
							dbid, tableid := parts[0], parts[1]

							dbExists, err := client.CheckDatabaseExists(dbms, dbid)
							if err != nil {
								return fmt.Errorf("failed to check database existence: %w", err)
							}
							if !dbExists {
								return fmt.Errorf("database '%s' does not exist in DBMS '%s'", dbid, dbms)
							}

							tableExists, err := client.CheckTableExists(dbms, dbid, tableid)
							if err != nil {
								return fmt.Errorf("failed to check table existence: %w", err)
							}
							if !tableExists {
								return fmt.Errorf("table '%s' does not exist in database '%s'", tableid, dbid)
							}
						}
					}
				}
			} else if table, ok := exportData["table"].(string); ok {
				// Handle single table for table-data
				parts := strings.SplitN(table, ".", 2)
				if len(parts) == 2 {
					dbid, tableid := parts[0], parts[1]

					dbExists, err := client.CheckDatabaseExists(dbms, dbid)
					if err != nil {
						return fmt.Errorf("failed to check database existence: %w", err)
					}
					if !dbExists {
						return fmt.Errorf("database '%s' does not exist in DBMS '%s'", dbid, dbms)
					}

					tableExists, err := client.CheckTableExists(dbms, dbid, tableid)
					if err != nil {
						return fmt.Errorf("failed to check table existence: %w", err)
					}
					if !tableExists {
						return fmt.Errorf("table '%s' does not exist in database '%s'", tableid, dbid)
					}
				}
			}
		}

	case "file":
		volume, ok := exportData["volume"].(string)
		if !ok {
			return fmt.Errorf("volume field required for file export")
		}

		volumeExists, err := client.CheckVolumeExists(volume)
		if err != nil {
			return fmt.Errorf("failed to check volume existence: %w", err)
		}
		if !volumeExists {
			return fmt.Errorf("volume '%s' does not exist", volume)
		}

		if path, ok := exportData["path"].(string); ok {
			fileExists, err := client.CheckFileExists(volume, path)
			if err != nil {
				return fmt.Errorf("failed to check file existence: %w", err)
			}
			if !fileExists {
				return fmt.Errorf("file '%s' does not exist in volume '%s'", path, volume)
			}
		}

	default:
		return fmt.Errorf("unsupported export type: %s", exportType)
	}

	return nil
}

func getAuthenticatedClient() *api.Client {
	cfg, err := config.LoadConfig()
	if err != nil {
		fatalError("Failed to load config", err)
	}

	token, err := config.LoadToken()
	if err != nil {
		fatalError("Failed to load token (please run 'hfit login' first)", err)
	}

	return api.NewClient(cfg.HfitdHost, token)
}

func handleDownloadCommand() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: hfit download <file_path>")
		os.Exit(1)
	}

	client := getAuthenticatedClient()
	filePath := os.Args[2]

	data, err := client.DownloadFile(filePath)
	if err != nil {
		fatalError("Failed to download file", err)
	}

	fmt.Print(string(data))
}

func printJSON(data interface{}) {
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fatalError("Failed to marshal JSON", err)
	}
	fmt.Println(string(output))
}
