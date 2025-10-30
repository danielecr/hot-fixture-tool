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

	"hfit/api"
	"hfit/auth"
	"hfit/config"
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
	case "dbs":
		handleDbsCommand()
	case "tables":
		handleTablesCommand()
	case "rows":
		handleRowsCommand()
	case "files":
		handleFilesCommand()
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
	fmt.Println("  hfit dbs                                           List databases")
	fmt.Println("  hfit tables <db_id>                                List tables in database")
	fmt.Println("  hfit rows <db_id> <table_id>                       List rows in table")
	fmt.Println("  hfit files                                         List files")
	fmt.Println("  hfit download <file_path>                          Download file")
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

func handleDbsCommand() {
	client := getAuthenticatedClient()

	databases, err := client.ListDatabases()
	if err != nil {
		fatalError("Failed to list databases", err)
	}

	printJSON(databases)
}

func handleTablesCommand() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: hfit tables <db_id>")
		os.Exit(1)
	}

	client := getAuthenticatedClient()
	dbID := os.Args[2]

	tables, err := client.ListTables(dbID)
	if err != nil {
		fatalError("Failed to list tables", err)
	}

	printJSON(tables)
}

func handleRowsCommand() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: hfit rows <db_id> <table_id>")
		os.Exit(1)
	}

	client := getAuthenticatedClient()
	dbID := os.Args[2]
	tableID := os.Args[3]

	rows, err := client.ListRows(dbID, tableID)
	if err != nil {
		fatalError("Failed to list rows", err)
	}

	printJSON(rows)
}

func handleFilesCommand() {
	client := getAuthenticatedClient()

	files, err := client.ListFiles()
	if err != nil {
		fatalError("Failed to list files", err)
	}

	printJSON(files)
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

func printJSON(data interface{}) {
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fatalError("Failed to marshal JSON", err)
	}
	fmt.Println(string(output))
}
