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
	"os"
	"strings"

	"hfit/api"
	"hfit/auth"
	"hfit/config"
	"hfit/usage"
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
		usage.PrintUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "config":
		handleConfigCommand()
	case "login":
		handleLoginCommand()
	case "help", "-h", "--help":
		usage.PrintUsage()
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
	case "pkg-example":
		handlePkgExampleCommand()
	case "pkg-tmpl":
		handlePkgTmplCommand()
	case "pkg-download":
		handlePkgDownloadCommand()
	case "download":
		handleDownloadCommand()
	default:
		fmt.Printf("Error: Unknown command '%s'\n\n", command)
		fmt.Printf("Run 'hfit help' for usage information.\n")
		fmt.Printf("For support, contact: Daniele Cruciani <daniele@smartango.com>\n")
		os.Exit(1)
	}
}

// print the same content as hfit-pkg-tmpl.yaml and print it to stdout
func handlePkgExampleCommand() {
	fmt.Print(`## Example of hfit package template YAML file
hfitVersion: 1
templateName: usecase_data
projectName: project_name # typically the repository or project name
packageName: basedata_$1
prepare:
  - setVar: dataid
    from: input
    source: $1
  - setVar: usrId
    from: hot-data
    hdata:
        type: dbquery
        dbms: dbms_mysql1
        query: "SELECT usrId FROM dbname.datatable WHERE dataid=${dataid} ORDER BY utime LIMIT 1"
  - setVar: fBaseName
    from: hot-data
    hdata:
        type: volume
        volume: vol1
        glob: "*_{dataid}_{usrId}_*.txt"
        # take the first in mtime desc order:
        sort: "mtime|desc"
        # extract the first number of filename as fBaseName value
        regex_replace: "/([0-9]+).*/$1/"
exports:
  dbcreate.sql:
    type: dbcreate
    data:
      dbms: dbms_mysql1
      tablelist:
        - dbname1
        - dbname2
  tablegroup1.create.sql:
    type: table-create
    data:
      dbms: dbms_mysql1
      tablelist:
        - dbname1.table1
        - dbname2.table2
        - dbname1.tablex
      option: <dropcreate|ifnotexists>
  tabledata1.data.sql:
    type: table-data
    data:
      dbms: dbms_mysql1
      table: dbname1.table1
      filter: WHERE key<34 AND key>12
  tabledata2.data.sql:
    type: table-data
    data:
      dbms: dbms_mysql1
      table: dbname1.usertable
      filter: WHERE usrId=${usrId}
  target-filename.txt:
    type: file
    data:
      volume: datavol1
      path: relative/path/to/file_${dataid}_${usrId}_${fBaseName}.txt
`)
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
		fmt.Println("Usage: hfit rows <dbms> <db_id> <table_id> [filterpart]")
		fmt.Println("Examples:")
		fmt.Println("  hfit rows mysql mydb users")
		fmt.Println("  hfit rows mysql mydb users \"WHERE age > 25\"")
		fmt.Println("  hfit rows postgres mydb orders \"ORDER BY created_at DESC LIMIT 50\"")
		os.Exit(1)
	}

	client := getAuthenticatedClient()
	dbms := os.Args[2]
	dbID := os.Args[3]
	tableID := os.Args[4]

	// Optional filterpart parameter
	var filterpart string
	if len(os.Args) > 5 {
		filterpart = os.Args[5]
	}

	err := client.StreamRows(dbms, dbID, tableID, filterpart)
	if err != nil {
		fatalError("Failed to stream rows", err)
	}
}

func handleFilesCommand() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: hfit files <volume> [filters...]")
		fmt.Println("Examples:")
		fmt.Println("  hfit files volume1")
		fmt.Println("  hfit files volume1 \"name:*.log\"")
		fmt.Println("  hfit files volume1 \"name:*.log\" \"mtime:7\" \"size:>1024\"")
		fmt.Println("  hfit files volume1 \"name:backup_*\" \"size:>1048576\"")
		fmt.Println("")
		fmt.Println("Available filters:")
		fmt.Println("  name:pattern    - File name pattern with * and ? wildcards")
		fmt.Println("  mtime:days      - Modified time: 7 (last 7 days), -30 (older than 30 days)")
		fmt.Println("  size:condition  - File size: >1024 (larger than), <1048576 (smaller than)")
		os.Exit(1)
	}

	client := getAuthenticatedClient()
	volume := os.Args[2]

	// Collect filter arguments if provided
	var filters []string
	if len(os.Args) > 3 {
		filters = os.Args[3:]
		err := client.StreamFilesWithFilters(volume, filters)
		if err != nil {
			fatalError("Failed to stream files with filters", err)
		}
	} else {
		err := client.StreamFiles(volume)
		if err != nil {
			fatalError("Failed to stream files", err)
		}
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
		fmt.Println("Usage: hfit download <volume:/path/to/file>")
		fmt.Println("Example: hfit download datavol1:/logs/app.log")
		os.Exit(1)
	}

	client := getAuthenticatedClient()
	volumePath := os.Args[2]

	data, err := client.DownloadFile(volumePath)
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

// handlePkgTmplCommand handles package template management commands
func handlePkgTmplCommand() {
	if len(os.Args) < 3 {
		usage.PrintUsagePkgTmpl()
		os.Exit(1)
	}

	subcommand := os.Args[2]

	switch subcommand {
	case "list":
		handlePkgTmplList()
	case "show":
		handlePkgTmplShow()
	case "create":
		handlePkgTmplCreate()
	case "update":
		handlePkgTmplUpdate()
	case "patch":
		handlePkgTmplPatch()
	case "delete":
		handlePkgTmplDelete()
	default:
		fmt.Printf("Unknown pkg-tmpl subcommand: %s\n", subcommand)
		os.Exit(1)
	}
}

// handlePkgDownloadCommand handles package download using templates
func handlePkgDownloadCommand() {
	if len(os.Args) < 3 {
		fmt.Println("Usage:")
		fmt.Println("  hfit pkg-download <template_name> [param1] [param2] [param3] ...")
		fmt.Println("  Parameters are substituted for $1, $2, $3, etc. in the template")
		os.Exit(1)
	}

	templateName := os.Args[2]
	params := os.Args[3:] // Get all remaining parameters

	cfg, err := config.LoadConfig()
	if err != nil {
		fatalError("Failed to load configuration", err)
	}

	token, err := config.LoadToken()
	if err != nil {
		fatalError("Failed to load authentication token", err)
	}

	client := api.NewClient(cfg.HfitdHost, token)

	// Call the package generation API with template name and parameters
	err = client.GenerateAndDownloadPackage(templateName, params)
	if err != nil {
		fatalError("Failed to generate and download package", err)
	}

	fmt.Printf("Package generated and downloaded successfully from template '%s'\n", templateName)
}

// handlePkgTmplList lists all package templates for the authenticated user
func handlePkgTmplList() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fatalError("Failed to load configuration", err)
	}

	token, err := config.LoadToken()
	if err != nil {
		fatalError("Failed to load authentication token", err)
	}

	client := api.NewClient(cfg.HfitdHost, token)

	templates, err := client.ListTemplates()
	if err != nil {
		fatalError("Failed to list templates", err)
	}

	if len(templates) == 0 {
		fmt.Println("No templates found")
		return
	}

	fmt.Println("Package Templates:")
	for _, template := range templates {
		fmt.Printf("  %s\n", template)
	}
}

// handlePkgTmplShow shows the YAML content of a specific template
func handlePkgTmplShow() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: hfit pkg-tmpl show <template_name>")
		os.Exit(1)
	}

	templateName := os.Args[3]

	cfg, err := config.LoadConfig()
	if err != nil {
		fatalError("Failed to load configuration", err)
	}

	token, err := config.LoadToken()
	if err != nil {
		fatalError("Failed to load authentication token", err)
	}

	client := api.NewClient(cfg.HfitdHost, token)

	templateYAML, err := client.GetTemplate(templateName)
	if err != nil {
		fatalError("Failed to get template", err)
	}

	fmt.Print(templateYAML)
}

// handlePkgTmplCreate creates a new template from a local YAML file
func handlePkgTmplCreate() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: hfit pkg-tmpl create <template_file.yaml>")
		os.Exit(1)
	}

	templateFile := os.Args[3]

	// Read the local YAML file
	yamlContent, err := os.ReadFile(templateFile)
	if err != nil {
		fatalError("Failed to read template file", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		fatalError("Failed to load configuration", err)
	}

	token, err := config.LoadToken()
	if err != nil {
		fatalError("Failed to load authentication token", err)
	}

	client := api.NewClient(cfg.HfitdHost, token)

	err = client.CreateTemplate(yamlContent)
	if err != nil {
		fatalError("Failed to create template", err)
	}

	fmt.Printf("Template created successfully from file '%s'\n", templateFile)
}

// handlePkgTmplUpdate updates an existing template from a local YAML file
func handlePkgTmplUpdate() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: hfit pkg-tmpl update <template_file.yaml>")
		os.Exit(1)
	}

	templateFile := os.Args[3]

	// Read the local YAML file
	yamlContent, err := os.ReadFile(templateFile)
	if err != nil {
		fatalError("Failed to read template file", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		fatalError("Failed to load configuration", err)
	}

	token, err := config.LoadToken()
	if err != nil {
		fatalError("Failed to load authentication token", err)
	}

	client := api.NewClient(cfg.HfitdHost, token)

	err = client.UpdateTemplate(yamlContent)
	if err != nil {
		fatalError("Failed to update template", err)
	}

	fmt.Printf("Template updated successfully from file '%s'\n", templateFile)
}

// handlePkgTmplPatch partially updates a template and shows the diff
func handlePkgTmplPatch() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: hfit pkg-tmpl patch <template_file.yaml>")
		os.Exit(1)
	}

	templateFile := os.Args[3]

	// Read the local YAML file
	yamlContent, err := os.ReadFile(templateFile)
	if err != nil {
		fatalError("Failed to read template file", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		fatalError("Failed to load configuration", err)
	}

	token, err := config.LoadToken()
	if err != nil {
		fatalError("Failed to load authentication token", err)
	}

	client := api.NewClient(cfg.HfitdHost, token)

	diffOutput, err := client.PatchTemplate(yamlContent)
	if err != nil {
		fatalError("Failed to patch template", err)
	}

	fmt.Print(diffOutput)
}

// handlePkgTmplDelete deletes a specific template
func handlePkgTmplDelete() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: hfit pkg-tmpl delete <template_name>")
		os.Exit(1)
	}

	templateName := os.Args[3]

	cfg, err := config.LoadConfig()
	if err != nil {
		fatalError("Failed to load configuration", err)
	}

	token, err := config.LoadToken()
	if err != nil {
		fatalError("Failed to load authentication token", err)
	}

	client := api.NewClient(cfg.HfitdHost, token)

	err = client.DeleteTemplate(templateName)
	if err != nil {
		fatalError("Failed to delete template", err)
	}

	fmt.Printf("Template '%s' deleted successfully\n", templateName)
}
