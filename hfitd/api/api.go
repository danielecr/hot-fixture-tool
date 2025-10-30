/*
 * Hot Fixture Tool Daemon - REST API
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

// provide the REST API for Hot Fixture Tool
package api

import (
	"archive/tar"
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"hfitd/admin"
	"hfitd/auth"
	"hfitd/config"
	"hfitd/db"
	redisclient "hfitd/redis"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
)

/*
API:
 /db/dbmss
 /db/{dbms}/dbs
 /db/{dbms}/{dbid}/tables
 /db/{dbms}/{dbid}/table/{tableid}/rows
 /files/list
 /files/download?path=
*/

/*
* NewHandler creates a new HTTP handler for the API.
 */
func NewHandler(databaseManager *db.DatabaseManager, cfg *config.Config, adminServer *admin.AdminServer) (http.Handler, error) {
	router := mux.NewRouter()

	// Initialize Redis client
	redisClient, err := redisclient.NewClient(cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Redis client: %w", err)
	}

	// Initialize authentication manager
	authManager, err := auth.NewAuthManager([]byte(cfg.Auth.JWTSecret), redisClient)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize auth manager: %w", err)
	}

	// Well-known JWT public key endpoint (RFC 7517 style)
	router.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		jwtPublicKeyPEM, err := adminServer.GetJWTPublicKeyPEM()
		if err != nil {
			http.Error(w, "JWT public key not available", http.StatusInternalServerError)
			return
		}

		// Return in JWKS format
		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
				{
					"kty": "RSA",
					"use": "sig",
					"key": jwtPublicKeyPEM,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}).Methods("GET")

	// Authentication routes (unprotected)
	router.HandleFunc("/auth/challenge", authManager.GenerateChallenge).Methods("POST")
	router.HandleFunc("/auth/authenticate", authManager.Authenticate).Methods("POST")

	// Protected routes - apply JWT middleware
	protected := router.PathPrefix("/").Subrouter()
	protected.Use(authManager.JWTMiddleware)

	// DBMS provider routes
	protected.HandleFunc("/db/dbmss", func(w http.ResponseWriter, r *http.Request) {
		providers := databaseManager.GetProviders()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(providers)
	}).Methods("GET")

	// Database routes
	protected.HandleFunc("/db/{dbms}/dbs", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		dbms := vars["dbms"]

		conn, err := databaseManager.GetConnection(dbms)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Get databases for this DBMS provider
		databases, err := getDatabases(conn, dbms, databaseManager)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(databases)
	}).Methods("GET")

	protected.HandleFunc("/db/{dbms}/{dbid}/tables", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		dbms := vars["dbms"]
		dbid := vars["dbid"]

		conn, err := databaseManager.GetConnection(dbms)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		tables, err := getTables(conn, dbms, dbid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tables)
	}).Methods("GET")

	protected.HandleFunc("/db/{dbms}/{dbid}/table/{tableid}/rows", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		dbms := vars["dbms"]
		dbid := vars["dbid"]
		tableid := vars["tableid"]

		conn, err := databaseManager.GetConnection(dbms)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		rows, err := getTableRows(conn, dbms, dbid, tableid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rows)
	}).Methods("GET")

	// HEAD methods for resource existence checking
	protected.HandleFunc("/db/{dbms}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		dbms := vars["dbms"]

		_, err := databaseManager.GetConnection(dbms)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}).Methods("HEAD")

	protected.HandleFunc("/db/{dbms}/{dbid}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		dbms := vars["dbms"]
		dbid := vars["dbid"]

		conn, err := databaseManager.GetConnection(dbms)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Check if database exists
		exists, err := checkDatabaseExists(conn, dbms, dbid)
		if err != nil || !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}).Methods("HEAD")

	protected.HandleFunc("/db/{dbms}/{dbid}/table/{tableid}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		dbms := vars["dbms"]
		dbid := vars["dbid"]
		tableid := vars["tableid"]

		conn, err := databaseManager.GetConnection(dbms)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Check if table exists
		exists, err := checkTableExists(conn, dbms, dbid, tableid)
		if err != nil || !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}).Methods("HEAD")

	// Volume routes
	protected.HandleFunc("/volumes", func(w http.ResponseWriter, r *http.Request) {
		var volumes []string
		for volumeName := range cfg.Volumes {
			volumes = append(volumes, volumeName)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(volumes)
	}).Methods("GET")

	protected.HandleFunc("/volumes/{volume}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		volume := vars["volume"]

		if _, exists := cfg.Volumes[volume]; !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}).Methods("HEAD")

	// File routes
	protected.HandleFunc("/files/{volume}/list", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		volume := vars["volume"]

		volumeConfig, exists := cfg.Volumes[volume]
		if !exists {
			http.Error(w, "Volume not found", http.StatusNotFound)
			return
		}

		files, err := listFiles(volumeConfig.Path, r.URL.Query())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(files)
	}).Methods("GET")

	protected.HandleFunc("/files/{volume}/download", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		volume := vars["volume"]

		volumeConfig, exists := cfg.Volumes[volume]
		if !exists {
			http.Error(w, "Volume not found", http.StatusNotFound)
			return
		}

		err := downloadFile(w, r, volumeConfig.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}).Methods("GET")

	protected.HandleFunc("/files/{volume}/{filepath:.*}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		volume := vars["volume"]
		filepath := vars["filepath"]

		volumeConfig, exists := cfg.Volumes[volume]
		if !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		exists, err := checkFileExists(volumeConfig.Path, filepath)
		if err != nil || !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}).Methods("HEAD")

	// Package download route
	protected.HandleFunc("/packdownload/{packname}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		packname := vars["packname"]

		err := handlePackageDownload(w, r, packname, cfg, databaseManager)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}).Methods("POST")

	// Health check (unprotected)
	router.HandleFunc("/health", healthCheckHandler).Methods("GET")

	return router, nil
}

/*
* Dbs handles the /db/dbs endpoint to list databases.
 */
func Dbs(w http.ResponseWriter, r *http.Request) {
	// Implementation to list databases
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("List of databases"))
}

/*
* healthCheckHandler provides a simple health check endpoint.
 */
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

/*
* registerHealthCheckRoute registers the health check route.
 */
func registerHealthCheckRoute(router *mux.Router) {
	router.HandleFunc("/health", healthCheckHandler).Methods("GET")
}

/*
* getDatabases retrieves the list of databases for a specific DBMS provider.
 */
func getDatabases(conn *sql.DB, dbms string, databaseManager *db.DatabaseManager) ([]string, error) {
	var query string

	switch strings.ToLower(dbms) {
	case "mysql":
		query = "SHOW DATABASES"
	case "postgres":
		query = "SELECT datname FROM pg_database WHERE datistemplate = false"
	default:
		return nil, fmt.Errorf("unsupported DBMS: %s", dbms)
	}

	rows, err := conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			return nil, err
		}
		databases = append(databases, dbName)
	}

	return databases, nil
}

/*
* checkDatabaseExists checks if a database exists for a specific DBMS provider.
 */
func checkDatabaseExists(conn *sql.DB, dbms string, dbid string) (bool, error) {
	databases, err := getDatabases(conn, dbms, nil)
	if err != nil {
		return false, err
	}

	for _, db := range databases {
		if db == dbid {
			return true, nil
		}
	}
	return false, nil
}

/*
* checkTableExists checks if a table exists in a specific database.
 */
func checkTableExists(conn *sql.DB, dbms string, dbid string, tableid string) (bool, error) {
	tables, err := getTables(conn, dbms, dbid)
	if err != nil {
		return false, err
	}

	for _, table := range tables {
		if table == tableid {
			return true, nil
		}
	}
	return false, nil
}

/*
* listFiles lists files in a volume with optional filtering.
 */
func listFiles(volumePath string, query url.Values) ([]map[string]interface{}, error) {
	var files []map[string]interface{}

	err := filepath.Walk(volumePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories for now, only return files
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(volumePath, path)
		if err != nil {
			return err
		}

		fileInfo := map[string]interface{}{
			"name":    info.Name(),
			"path":    relPath,
			"size":    info.Size(),
			"modtime": info.ModTime().Unix(),
			"isdir":   info.IsDir(),
		}

		// Apply filters from query parameters
		if passesFilters(fileInfo, query) {
			files = append(files, fileInfo)
		}

		return nil
	})

	return files, err
}

/*
* passesFilters checks if a file passes the filter criteria.
 */
func passesFilters(fileInfo map[string]interface{}, query url.Values) bool {
	filters := query["filter[]"]

	for _, filter := range filters {
		parts := strings.SplitN(filter, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		switch key {
		case "name":
			// Simple wildcard matching
			fileName := fileInfo["name"].(string)
			if !strings.Contains(fileName, strings.ReplaceAll(value, "*", "")) {
				return false
			}
		case "mtime":
			// Time-based filtering (days ago)
			if days, err := strconv.Atoi(value); err == nil {
				fileTime := time.Unix(fileInfo["modtime"].(int64), 0)
				cutoff := time.Now().AddDate(0, 0, days)
				if days < 0 && fileTime.Before(cutoff) {
					return false
				}
			}
		}
	}

	return true
}

/*
* downloadFile handles file download requests.
 */
func downloadFile(w http.ResponseWriter, r *http.Request, volumePath string) error {
	query := r.URL.Query()
	filePath := query.Get("path")
	folder := query.Get("folder")

	if filePath != "" {
		// Direct file download
		fullPath := filepath.Join(volumePath, filePath)

		// Security check - ensure path is within volume
		if !strings.HasPrefix(fullPath, volumePath) {
			return fmt.Errorf("invalid file path")
		}

		file, err := os.Open(fullPath)
		if err != nil {
			return err
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			return err
		}

		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(filePath)))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))

		_, err = io.Copy(w, file)
		return err
	} else if folder != "" {
		// Find first matching file in folder with filters
		files, err := listFiles(filepath.Join(volumePath, folder), r.URL.Query())
		if err != nil {
			return err
		}

		if len(files) == 0 {
			return fmt.Errorf("no matching files found")
		}

		// Download first matching file
		firstFile := files[0]
		return downloadFile(w, &http.Request{
			URL: &url.URL{
				RawQuery: fmt.Sprintf("path=%s", firstFile["path"]),
			},
		}, volumePath)
	}

	return fmt.Errorf("either path or folder parameter is required")
}

/*
* checkFileExists checks if a file exists in a volume.
 */
func checkFileExists(volumePath string, filePath string) (bool, error) {
	fullPath := filepath.Join(volumePath, filePath)

	// Security check - ensure path is within volume
	if !strings.HasPrefix(fullPath, volumePath) {
		return false, fmt.Errorf("invalid file path")
	}

	_, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
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

/*
* handlePackageDownload processes package download requests with YAML definition.
 */
func handlePackageDownload(w http.ResponseWriter, r *http.Request, packname string, cfg *config.Config, databaseManager *db.DatabaseManager) error {
	var packageDef PackageDefinition

	decoder := yaml.NewDecoder(r.Body)
	if err := decoder.Decode(&packageDef); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Create temporary directory for package assembly
	tmpDir, err := os.MkdirTemp("", "hfit-package-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// Process each export in the package
	for filename, export := range packageDef.Exports {
		exportPath := filepath.Join(tmpDir, filename)

		switch export.Type {
		case "dbcreate":
			err = handleDBCreateExport(exportPath, export.Data, databaseManager)
		case "table-create":
			err = handleTableCreateExport(exportPath, export.Data, databaseManager)
		case "table-data":
			err = handleTableDataExport(exportPath, export.Data, databaseManager)
		case "file":
			err = handleFileExport(exportPath, export.Data, cfg)
		default:
			err = fmt.Errorf("unsupported export type: %s", export.Type)
		}

		if err != nil {
			return fmt.Errorf("failed to process export %s: %w", filename, err)
		}
	}

	// Create tar.gz archive
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.tar.gz", packname))

	gzipWriter := gzip.NewWriter(w)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Add files to archive
	return filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		relPath, err := filepath.Rel(tmpDir, path)
		if err != nil {
			return err
		}

		header := &tar.Header{
			Name:    relPath,
			Size:    info.Size(),
			Mode:    int64(info.Mode()),
			ModTime: info.ModTime(),
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tarWriter, file)
		return err
	})
}

/*
* handleDBCreateExport creates database creation SQL.
 */
func handleDBCreateExport(exportPath string, data map[string]interface{}, databaseManager *db.DatabaseManager) error {
	// Implementation for database creation export
	return os.WriteFile(exportPath, []byte("-- Database creation SQL placeholder\n"), 0644)
}

/*
* handleTableCreateExport creates table creation SQL.
 */
func handleTableCreateExport(exportPath string, data map[string]interface{}, databaseManager *db.DatabaseManager) error {
	// Implementation for table creation export
	return os.WriteFile(exportPath, []byte("-- Table creation SQL placeholder\n"), 0644)
}

/*
* handleTableDataExport creates table data SQL.
 */
func handleTableDataExport(exportPath string, data map[string]interface{}, databaseManager *db.DatabaseManager) error {
	// Implementation for table data export
	return os.WriteFile(exportPath, []byte("-- Table data SQL placeholder\n"), 0644)
}

/*
* handleFileExport copies files from volumes.
 */
func handleFileExport(exportPath string, data map[string]interface{}, cfg *config.Config) error {
	volume, ok := data["volume"].(string)
	if !ok {
		return fmt.Errorf("volume not specified")
	}

	path, ok := data["path"].(string)
	if !ok {
		return fmt.Errorf("path not specified")
	}

	volumeConfig, exists := cfg.Volumes[volume]
	if !exists {
		return fmt.Errorf("volume %s not found", volume)
	}

	sourcePath := filepath.Join(volumeConfig.Path, path)

	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(exportPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

/*
* getTables retrieves the list of tables for a specific database.
 */
func getTables(conn *sql.DB, dbms string, dbid string) ([]string, error) {
	var query string

	switch strings.ToLower(dbms) {
	case "mysql":
		query = fmt.Sprintf("SELECT table_name FROM information_schema.tables WHERE table_schema = '%s'", dbid)
	case "postgres":
		// For PostgreSQL, we need to connect to the specific database first
		// This is a simplified version - in practice you might need a separate connection
		query = "SELECT tablename FROM pg_tables WHERE schemaname = 'public'"
	default:
		return nil, fmt.Errorf("unsupported DBMS: %s", dbms)
	}

	rows, err := conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	return tables, nil
}

/*
* getTableRows retrieves the rows from a specific table.
 */
func getTableRows(conn *sql.DB, dbms string, dbid string, tableid string) ([]map[string]interface{}, error) {
	// Simple SELECT * query - in production you might want to limit rows
	query := fmt.Sprintf("SELECT * FROM %s LIMIT 100", tableid)

	rows, err := conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}

		result = append(result, row)
	}

	return result, nil
}
