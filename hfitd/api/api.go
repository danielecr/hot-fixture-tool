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
	"hfitd/pkgtmpl"
	redisclient "hfitd/redis"
	"hfitd/tmplstorage"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
)

/*
API:
 /db/dbmss
 /db/{dbms}/dbs
 /db/{dbms}/{dbid}/tables
 /db/{dbms}/{dbid}/table/{tableid}/rows
 /db/{dbms}/{dbid}/table/{tableid}/rows?filterpart="WHERE id > 100 ORDER BY name DESC"
 /files/{volume}/list
 /files/{volume}/download?path=
 /files/{volume}/download?folder=<folder>&filter[]=name:*.config
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

		// Get filterpart parameter for SQL filtering
		filterpart := r.URL.Query().Get("filterpart")

		conn, err := databaseManager.GetConnection(dbms)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Check Accept header to determine response format
		acceptHeader := r.Header.Get("Accept")

		if acceptHeader == "application/x-json-stream" {
			// Stream as NDJSON
			if err := streamTableRows(w, conn, dbms, dbid, tableid, filterpart); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			// Traditional JSON array response (backward compatibility)
			rows, err := getTableRowsWithFilter(conn, dbms, dbid, tableid, filterpart)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(rows)
		}
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

		// Set streaming headers for NDJSON
		w.Header().Set("Content-Type", "application/x-json-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// Use streaming file listing for performance
		err := streamFiles(w, volumeConfig.Path, r.URL.Query())
		if err != nil {
			// Error handling: if we haven't written anything yet, send HTTP error
			// If we've started streaming, we can only log the error
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
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

	// Package template routes
	templateStorage := tmplstorage.NewTemplateStorage(redisClient)

	// List package templates
	protected.HandleFunc("/packtmpl", func(w http.ResponseWriter, r *http.Request) {
		userEmail := r.Header.Get("X-User")
		if userEmail == "" {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		err := handleListTemplates(w, r, userEmail, templateStorage)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}).Methods("GET")

	// Get specific package template
	protected.HandleFunc("/packtmpl/{templatename}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		templateName := vars["templatename"]
		userEmail := r.Header.Get("X-User")
		if userEmail == "" {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		err := handleGetTemplate(w, r, userEmail, templateName, templateStorage)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}).Methods("GET")

	// Create/Update package template
	protected.HandleFunc("/packtmpl/{templatename}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		templateName := vars["templatename"]
		userEmail := r.Header.Get("X-User")
		if userEmail == "" {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		err := handleSetTemplate(w, r, userEmail, templateName, templateStorage)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}).Methods("POST")

	// Delete package template
	protected.HandleFunc("/packtmpl/{templatename}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		templateName := vars["templatename"]
		userEmail := r.Header.Get("X-User")
		if userEmail == "" {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		err := handleDeleteTemplate(w, r, userEmail, templateName, templateStorage)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}).Methods("DELETE")

	// Bulk upload package templates
	protected.HandleFunc("/packtmplpackupld", func(w http.ResponseWriter, r *http.Request) {
		userEmail := r.Header.Get("X-User")
		if userEmail == "" {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		err := handleBulkUploadTemplates(w, r, userEmail, templateStorage)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}).Methods("POST")

	// Package template download with parameters
	protected.HandleFunc("/packdownload/{templatename}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		templateName := vars["templatename"]
		userEmail := r.Header.Get("X-User")
		if userEmail == "" {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		err := handleTemplatePackageDownload(w, r, userEmail, templateName, cfg, databaseManager, redisClient, templateStorage)
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
* streamFiles streams files in a volume with optional filtering using NDJSON format.
* This provides performance similar to Unix find command for large directories.
 */
func streamFiles(w http.ResponseWriter, volumePath string, query url.Values) error {
	// Ensure the response is flushed immediately for streaming
	flusher, canFlush := w.(http.Flusher)

	// Validate volume path exists and is accessible
	if _, err := os.Stat(volumePath); os.IsNotExist(err) {
		return fmt.Errorf("volume path does not exist: %s", volumePath)
	}

	// Use filepath.Walk which is optimized like Unix find
	return filepath.Walk(volumePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Log error but continue processing other files (like Unix find does)
			// Don't return error unless it's a critical filesystem issue
			if os.IsPermission(err) {
				// Permission denied - skip this path and continue
				return nil
			}
			return nil
		}

		// Skip directories for now, only return files
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(volumePath, path)
		if err != nil {
			return nil // Skip this file and continue
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
			// Write each file as a separate JSON line (NDJSON)
			jsonData, err := json.Marshal(fileInfo)
			if err != nil {
				return nil // Skip this file and continue
			}

			// Write the JSON object followed by newline
			if _, err := w.Write(jsonData); err != nil {
				return err // Stop on write error (client disconnected)
			}
			if _, err := w.Write([]byte("\n")); err != nil {
				return err // Stop on write error
			}

			// Flush after every file for real-time streaming
			if canFlush {
				flusher.Flush()
			}
		}

		return nil
	})
}

/*
* passesFilters checks if a file passes the filter criteria.
* Supports name wildcards, time filtering, and sorting hints.
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
			// Enhanced wildcard matching with glob-like patterns
			fileName := fileInfo["name"].(string)
			if !matchesPattern(fileName, value) {
				return false
			}
		case "mtime":
			// Time-based filtering (days ago)
			if days, err := strconv.Atoi(value); err == nil {
				fileTime := time.Unix(fileInfo["modtime"].(int64), 0)
				if days < 0 {
					// Negative days means "modified before X days ago"
					cutoff := time.Now().AddDate(0, 0, days)
					if fileTime.After(cutoff) {
						return false
					}
				} else {
					// Positive days means "modified after X days ago"
					cutoff := time.Now().AddDate(0, 0, -days)
					if fileTime.Before(cutoff) {
						return false
					}
				}
			}
		case "size":
			// Size filtering (bytes)
			fileSize := fileInfo["size"].(int64)
			if strings.HasPrefix(value, ">") {
				if size, err := strconv.ParseInt(value[1:], 10, 64); err == nil {
					if fileSize <= size {
						return false
					}
				}
			} else if strings.HasPrefix(value, "<") {
				if size, err := strconv.ParseInt(value[1:], 10, 64); err == nil {
					if fileSize >= size {
						return false
					}
				}
			} else {
				if size, err := strconv.ParseInt(value, 10, 64); err == nil {
					if fileSize != size {
						return false
					}
				}
			}
		case "sort":
			// Sort is handled at a higher level, just pass through
			continue
		}
	}

	return true
}

/*
* matchesPattern performs glob-like pattern matching for file names.
 */
func matchesPattern(fileName, pattern string) bool {
	// Simple glob pattern matching
	// * matches any sequence of characters
	// ? matches any single character

	if pattern == "*" {
		return true
	}

	// Convert glob pattern to regex-like matching
	if strings.Contains(pattern, "*") {
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			// Simple case: prefix*suffix
			prefix, suffix := parts[0], parts[1]
			return strings.HasPrefix(fileName, prefix) && strings.HasSuffix(fileName, suffix)
		}
		// For more complex patterns, fall back to simple contains check
		cleanPattern := strings.ReplaceAll(pattern, "*", "")
		return strings.Contains(fileName, cleanPattern)
	}

	if strings.Contains(pattern, "?") {
		// For now, just treat ? as any character - could be enhanced with regex
		cleanPattern := strings.ReplaceAll(pattern, "?", "")
		return strings.Contains(fileName, cleanPattern)
	}

	// Exact match or contains
	return strings.Contains(fileName, pattern)
}

/*
* findBestFile finds the best matching file based on filters and sorting criteria in O(n) time.
* This streams through files, applies filters, and keeps track of the best candidate.
 */
func findBestFile(volumePath string, query url.Values) (map[string]interface{}, error) {
	filters := query["filter[]"]
	var sortCriteria string
	var sortOrder string = "asc" // default ascending

	// Extract sort criteria from filters
	for _, filter := range filters {
		parts := strings.SplitN(filter, ":", 2)
		if len(parts) == 2 && parts[0] == "sort" {
			sortSpec := parts[1]
			// Handle sort direction (e.g., "mtime:desc", "size:asc", "name")
			if strings.Contains(sortSpec, ":") {
				sortParts := strings.SplitN(sortSpec, ":", 2)
				sortCriteria = sortParts[0]
				sortOrder = sortParts[1]
			} else {
				sortCriteria = sortSpec
			}
			break
		}
	}

	var bestFile map[string]interface{}

	// Stream through files and find the best one in a single pass
	err := filepath.Walk(volumePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip errors and continue (like Unix find)
			if os.IsPermission(err) {
				return nil
			}
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(volumePath, path)
		if err != nil {
			return nil // Skip this file and continue
		}

		fileInfo := map[string]interface{}{
			"name":    info.Name(),
			"path":    relPath,
			"size":    info.Size(),
			"modtime": info.ModTime().Unix(),
			"isdir":   info.IsDir(),
		}

		// Apply filters - skip if doesn't match
		if !passesFilters(fileInfo, query) {
			return nil
		}

		// If no best file yet, this becomes the best
		if bestFile == nil {
			bestFile = fileInfo
			return nil
		}

		// Compare with current best based on sort criteria
		if isBetterFile(fileInfo, bestFile, sortCriteria, sortOrder) {
			bestFile = fileInfo
		}

		return nil
	})

	return bestFile, err
}

/*
* isBetterFile compares two files based on sorting criteria to determine which is "better".
* Returns true if newFile is better than currentBest.
 */
func isBetterFile(newFile, currentBest map[string]interface{}, sortCriteria, sortOrder string) bool {
	switch sortCriteria {
	case "mtime", "modtime":
		newTime := newFile["modtime"].(int64)
		bestTime := currentBest["modtime"].(int64)
		if sortOrder == "desc" {
			return newTime > bestTime // newer is better
		}
		return newTime < bestTime // older is better

	case "size":
		newSize := newFile["size"].(int64)
		bestSize := currentBest["size"].(int64)
		if sortOrder == "desc" {
			return newSize > bestSize // larger is better
		}
		return newSize < bestSize // smaller is better

	case "name":
		newName := newFile["name"].(string)
		bestName := currentBest["name"].(string)
		if sortOrder == "desc" {
			return newName > bestName // Z-A order
		}
		return newName < bestName // A-Z order

	case "path":
		newPath := newFile["path"].(string)
		bestPath := currentBest["path"].(string)
		if sortOrder == "desc" {
			return newPath > bestPath
		}
		return newPath < bestPath

	default:
		// No sorting criteria, keep the first one found
		return false
	}
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
		// Find the best matching file in folder with filters and sorting in O(n) time
		bestFile, err := findBestFile(filepath.Join(volumePath, folder), r.URL.Query())
		if err != nil {
			return err
		}

		if bestFile == nil {
			return fmt.Errorf("no matching files found")
		}

		// Download the best matching file
		return downloadFile(w, &http.Request{
			URL: &url.URL{
				RawQuery: fmt.Sprintf("path=%s", bestFile["path"]),
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
* validateFilterPart validates and sanitizes the filterpart parameter to prevent SQL injection.
* Returns the sanitized filter or an error if potentially dangerous content is detected.
 */
func validateFilterPart(filterpart string) (string, error) {
	if filterpart == "" {
		return "", nil
	}

	// Convert to lowercase for case-insensitive checking
	lower := strings.ToLower(strings.TrimSpace(filterpart))

	// List of dangerous SQL keywords/patterns that should be blocked
	dangerousPatterns := []string{
		"insert", "update", "delete", "drop", "create", "alter", "truncate",
		"grant", "revoke", "exec", "execute", "sp_", "xp_", "union",
		"information_schema", "sys.", "pg_", "mysql.", "--", "/*", "*/",
		";", "@@", "char(", "nchar(", "varchar(", "nvarchar(",
		"waitfor", "delay", "benchmark(", "sleep(", "load_file(",
		"into outfile", "into dumpfile", "script", "javascript", "vbscript",
	}

	// Check for dangerous patterns
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lower, pattern) {
			return "", fmt.Errorf("potentially dangerous SQL content detected: %s", pattern)
		}
	}

	// Only allow safe SQL clauses (WHERE, ORDER BY, LIMIT, HAVING, GROUP BY)
	allowedPrefixes := []string{"where ", "order by ", "limit ", "having ", "group by "}
	hasValidPrefix := false
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(lower, prefix) {
			hasValidPrefix = true
			break
		}
	}

	if !hasValidPrefix {
		return "", fmt.Errorf("filterpart must start with WHERE, ORDER BY, LIMIT, HAVING, or GROUP BY")
	}

	// Additional validation: check for balanced quotes
	singleQuotes := strings.Count(filterpart, "'")
	doubleQuotes := strings.Count(filterpart, "\"")
	if singleQuotes%2 != 0 || doubleQuotes%2 != 0 {
		return "", fmt.Errorf("unbalanced quotes in filterpart")
	}

	return strings.TrimSpace(filterpart), nil
}

/*
* getTableRowsWithFilter retrieves rows from a specific table with optional filtering.
* Maintains backward compatibility with traditional JSON array response.
 */
func getTableRowsWithFilter(conn *sql.DB, dbms string, dbid string, tableid string, filterpart string) ([]map[string]interface{}, error) {
	// Build the base query
	baseQuery := fmt.Sprintf("SELECT * FROM %s", tableid)

	// Add filter part if provided and validated
	var finalQuery string
	if filterpart != "" {
		validatedFilter, err := validateFilterPart(filterpart)
		if err != nil {
			return nil, fmt.Errorf("invalid filterpart: %v", err)
		}
		finalQuery = fmt.Sprintf("%s %s", baseQuery, validatedFilter)
	} else {
		// Default limit to prevent accidental massive queries
		finalQuery = fmt.Sprintf("%s LIMIT 100", baseQuery)
	}

	rows, err := conn.Query(finalQuery)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %v", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %v", err)
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

/*
* streamTableRows streams database table rows as NDJSON format with optional filtering.
* Uses O(1) memory by processing one row at a time.
 */
func streamTableRows(w http.ResponseWriter, conn *sql.DB, dbms string, dbid string, tableid string, filterpart string) error {
	// Build the base query
	baseQuery := fmt.Sprintf("SELECT * FROM %s", tableid)

	// Add filter part if provided and validated
	var finalQuery string
	if filterpart != "" {
		validatedFilter, err := validateFilterPart(filterpart)
		if err != nil {
			return fmt.Errorf("invalid filterpart: %v", err)
		}
		finalQuery = fmt.Sprintf("%s %s", baseQuery, validatedFilter)
	} else {
		// Default limit to prevent accidental massive queries
		finalQuery = fmt.Sprintf("%s LIMIT 1000", baseQuery)
	}

	rows, err := conn.Query(finalQuery)
	if err != nil {
		return fmt.Errorf("query execution failed: %v", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %v", err)
	}

	// Set response headers for NDJSON streaming
	w.Header().Set("Content-Type", "application/x-json-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Create a flusher for real-time streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}

	encoder := json.NewEncoder(w)

	// Stream each row as a separate JSON object
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			// Log error but continue with next row
			fmt.Fprintf(w, `{"error":"row scan failed: %v"}`+"\n", err)
			flusher.Flush()
			continue
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

		// Encode and stream this row
		if err := encoder.Encode(row); err != nil {
			return fmt.Errorf("failed to encode row: %v", err)
		}

		// Flush immediately for real-time streaming
		flusher.Flush()
	}

	// Check for any errors from iterating over rows
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error during row iteration: %v", err)
	}

	return nil
}

/*
* Template handler functions for package template operations
 */
func handleListTemplates(w http.ResponseWriter, r *http.Request, userEmail string, templateStorage *tmplstorage.TemplateStorage) error {
	templates, err := templateStorage.ListTemplates(r.Context(), userEmail)
	if err != nil {
		return fmt.Errorf("failed to list templates: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(templates)
}

// handleGetTemplate retrieves a specific package template
func handleGetTemplate(w http.ResponseWriter, r *http.Request, userEmail, templateName string, templateStorage *tmplstorage.TemplateStorage) error {
	yamlContent, err := templateStorage.GetTemplate(r.Context(), userEmail, templateName)
	if err != nil {
		return fmt.Errorf("failed to get template: %v", err)
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Write([]byte(yamlContent))
	return nil
}

// handleSetTemplate creates or updates a package template
func handleSetTemplate(w http.ResponseWriter, r *http.Request, userEmail, templateName string, templateStorage *tmplstorage.TemplateStorage) error {
	// Read YAML content from request body
	yamlContent, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %v", err)
	}

	// Validate YAML by attempting to parse it
	var template pkgtmpl.PackageTemplate
	if err := yaml.Unmarshal(yamlContent, &template); err != nil {
		return fmt.Errorf("invalid YAML content: %v", err)
	}

	// Store the template
	if err := templateStorage.StoreTemplate(r.Context(), userEmail, templateName, string(yamlContent)); err != nil {
		return fmt.Errorf("failed to store template: %v", err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Template stored successfully"))
	return nil
}

// handleDeleteTemplate deletes a package template
func handleDeleteTemplate(w http.ResponseWriter, r *http.Request, userEmail, templateName string, templateStorage *tmplstorage.TemplateStorage) error {
	// Check if template exists
	exists, err := templateStorage.TemplateExists(r.Context(), userEmail, templateName)
	if err != nil {
		return fmt.Errorf("failed to check template existence: %v", err)
	}

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Template not found"))
		return nil
	}

	// Delete the template
	if err := templateStorage.DeleteTemplate(r.Context(), userEmail, templateName); err != nil {
		return fmt.Errorf("failed to delete template: %v", err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Template deleted successfully"))
	return nil
}

// handleBulkUploadTemplates handles bulk upload of package templates from tar.gz
func handleBulkUploadTemplates(w http.ResponseWriter, r *http.Request, userEmail string, templateStorage *tmplstorage.TemplateStorage) error {
	// Parse multipart form
	err := r.ParseMultipartForm(32 << 20) // 32MB max memory
	if err != nil {
		return fmt.Errorf("failed to parse multipart form: %v", err)
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		return fmt.Errorf("failed to get uploaded file: %v", err)
	}
	defer file.Close()

	// Create gzip reader
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzipReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzipReader)

	var report []map[string]interface{}

	// Process each file in the archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %v", err)
		}

		// Skip directories and non-yaml files
		if header.Typeflag != tar.TypeReg || !strings.HasSuffix(header.Name, ".yaml") {
			continue
		}

		// Extract template name from filename
		templateName := strings.TrimSuffix(filepath.Base(header.Name), ".yaml")

		// Read file content
		content, err := io.ReadAll(tarReader)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %v", header.Name, err)
		}

		// Validate YAML
		var template pkgtmpl.PackageTemplate
		if err := yaml.Unmarshal(content, &template); err != nil {
			report = append(report, map[string]interface{}{
				"template": templateName,
				"status":   "error",
				"message":  fmt.Sprintf("Invalid YAML: %v", err),
			})
			continue
		}

		// Check if template exists
		exists, err := templateStorage.TemplateExists(r.Context(), userEmail, templateName)
		if err != nil {
			report = append(report, map[string]interface{}{
				"template": templateName,
				"status":   "error",
				"message":  fmt.Sprintf("Failed to check existence: %v", err),
			})
			continue
		}

		// Store the template
		if err := templateStorage.StoreTemplate(r.Context(), userEmail, templateName, string(content)); err != nil {
			report = append(report, map[string]interface{}{
				"template": templateName,
				"status":   "error",
				"message":  fmt.Sprintf("Failed to store: %v", err),
			})
			continue
		}

		action := "created"
		if exists {
			action = "updated"
		}

		report = append(report, map[string]interface{}{
			"template": templateName,
			"status":   "success",
			"action":   action,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(map[string]interface{}{
		"templates_processed": len(report),
		"report":              report,
	})
}

// handleTemplatePackageDownload processes template-based package downloads with parameters
func handleTemplatePackageDownload(w http.ResponseWriter, r *http.Request, userEmail, templateName string, cfg *config.Config, databaseManager *db.DatabaseManager, redisClient *redisclient.Client, templateStorage *tmplstorage.TemplateStorage) error {
	// Parse JSON parameters from request body
	var params []string
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		return fmt.Errorf("failed to parse parameters: %v", err)
	}

	// Get the template
	yamlContent, err := templateStorage.GetTemplate(r.Context(), userEmail, templateName)
	if err != nil {
		return fmt.Errorf("template not found: %v", err)
	}

	// Parse the template
	var template pkgtmpl.PackageTemplate
	if err := yaml.Unmarshal([]byte(yamlContent), &template); err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	// Create package template processor
	processor := pkgtmpl.NewPackageTemplateProcessor(cfg, databaseManager)

	// Process the template with parameters
	timestamp := time.Now().Unix()
	packagePath, metadata, err := processor.ProcessTemplate(r.Context(), &template, params, userEmail, templateName, timestamp)
	if err != nil {
		return fmt.Errorf("failed to process template: %v", err)
	}

	// Log the download
	if err := templateStorage.LogDownload(r.Context(), userEmail, templateName, timestamp); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: failed to log download: %v\n", err)
	}

	// Store download metadata
	if err := templateStorage.StoreDownloadMetadata(r.Context(), userEmail, templateName, timestamp, *metadata); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: failed to store download metadata: %v\n", err)
	}

	// Open the generated package file
	packageFile, err := os.Open(packagePath)
	if err != nil {
		return fmt.Errorf("failed to open package file: %v", err)
	}
	defer packageFile.Close()

	// Get file info for Content-Length header
	fileInfo, err := packageFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}

	// Set response headers
	filename := fmt.Sprintf("pkg-%s-%d.tar.gz", templateName, timestamp)
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// Stream the file to the client
	_, err = io.Copy(w, packageFile)
	return err
}
