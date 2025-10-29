// Package main provides the hfit CLI tool for interacting with hfitd
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"hfit/api"
	"hfit/auth"
	"hfit/config"
)

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
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("hfit - Hot Fixture Tool CLI")
	fmt.Println()
	fmt.Println("Usage:")
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
		log.Fatalf("Failed to save config: %v", err)
	}

	fmt.Println("Configuration saved successfully")
}

func handleLoginCommand() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	token, err := auth.AuthenticateWithChallenge(cfg.HfitdHost, cfg.Email, cfg.PublicKey)
	if err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}

	if err := config.SaveToken(token); err != nil {
		log.Fatalf("Failed to save token: %v", err)
	}

	fmt.Println("Authentication successful, token saved")
}

func handleDbsCommand() {
	client := getAuthenticatedClient()

	databases, err := client.ListDatabases()
	if err != nil {
		log.Fatalf("Failed to list databases: %v", err)
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
		log.Fatalf("Failed to list tables: %v", err)
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
		log.Fatalf("Failed to list rows: %v", err)
	}

	printJSON(rows)
}

func handleFilesCommand() {
	client := getAuthenticatedClient()

	files, err := client.ListFiles()
	if err != nil {
		log.Fatalf("Failed to list files: %v", err)
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
		log.Fatalf("Failed to download file: %v", err)
	}

	fmt.Print(string(data))
}

func getAuthenticatedClient() *api.Client {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	token, err := config.LoadToken()
	if err != nil {
		log.Fatalf("Failed to load token (please run 'hfit login' first): %v", err)
	}

	return api.NewClient(cfg.HfitdHost, token)
}

func printJSON(data interface{}) {
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}
	fmt.Println(string(output))
}
