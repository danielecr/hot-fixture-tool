/*
 * Hot Fixture Tool Daemon - Database API Handlers
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

package dbapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"hfitd/apierrors"
	"hfitd/db"

	"github.com/gorilla/mux"
)

// DatabaseHandlers holds the database manager for handler methods
type DatabaseHandlers struct {
	databaseManager *db.DatabaseManager
}

// NewDatabaseHandlers creates a new instance of database handlers
func NewDatabaseHandlers(databaseManager *db.DatabaseManager) *DatabaseHandlers {
	return &DatabaseHandlers{
		databaseManager: databaseManager,
	}
}

// GetProviders godoc
//
//	@Summary		List DBMS providers
//	@Description	Get list of all available database management systems
//	@Tags			database
//	@Accept			json
//	@Produce		json
//	@Success		200	{array}		string	"List of DBMS providers"
//	@Failure		401	{object}	map[string]string	"Unauthorized"
//	@Failure		500	{object}	map[string]string	"Internal server error"
//	@Security		BearerAuth
//	@Router			/db/dbmss [get]
func (h *DatabaseHandlers) GetProviders(w http.ResponseWriter, r *http.Request) {
	providers := h.databaseManager.GetProviders()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(providers)
}

// GetDatabases godoc
//
//	@Summary		List databases
//	@Description	Get list of all databases for a specific DBMS
//	@Tags			database
//	@Accept			json
//	@Produce		json
//	@Param			dbms	path		string	true	"DBMS provider name"	example(mysql)
//	@Success		200		{array}		string	"List of database names"
//	@Failure		400		{object}	map[string]string	"Bad request - invalid DBMS"
//	@Failure		401		{object}	map[string]string	"Unauthorized"
//	@Failure		500		{object}	map[string]string	"Internal server error"
//	@Security		BearerAuth
//	@Router			/db/{dbms}/dbs [get]
func (h *DatabaseHandlers) GetDatabases(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dbms := vars["dbms"]

	conn, err := h.databaseManager.GetConnection(dbms)
	if err != nil {
		// Check if it's a provider not found or connection error
		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "unknown") {
			apiErr := apierrors.NewDBProviderNotFoundError(dbms)
			apierrors.WriteErrorResponse(w, apiErr)
		} else if strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "dial") {
			apiErr := apierrors.NewDBConnectionError(dbms, err)
			apierrors.WriteErrorResponse(w, apiErr)
		} else {
			apiErr := apierrors.NewDBConnectionError(dbms, err)
			apierrors.WriteErrorResponse(w, apiErr)
		}
		return
	}

	// Get databases for this DBMS provider
	databases, err := GetDatabases(conn, dbms, h.databaseManager)
	if err != nil {
		apiErr := apierrors.NewDBConnectionError(dbms, err)
		apierrors.WriteErrorResponse(w, apiErr)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(databases)
}

// GetTables godoc
//
//	@Summary		List tables
//	@Description	Get list of all tables in a specific database
//	@Tags			database
//	@Accept			json
//	@Produce		json
//	@Param			dbms	path		string	true	"DBMS provider name"	example(mysql)
//	@Param			dbid	path		string	true	"Database name"		example(myapp_db)
//	@Success		200		{array}		string	"List of table names"
//	@Failure		400		{object}	map[string]string	"Bad request - invalid DBMS or database"
//	@Failure		401		{object}	map[string]string	"Unauthorized"
//	@Failure		500		{object}	map[string]string	"Internal server error"
//	@Security		BearerAuth
//	@Router			/db/{dbms}/{dbid}/tables [get]
func (h *DatabaseHandlers) GetTables(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dbms := vars["dbms"]
	dbid := vars["dbid"]

	conn, err := h.databaseManager.GetConnection(dbms)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tables, err := GetTables(conn, dbms, dbid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tables)
}

// GetTableRows godoc
//
//	@Summary		Get table rows
//	@Description	Get rows from a specific table with optional filtering
//	@Tags			database
//	@Accept			json
//	@Produce		json
//	@Param			dbms		path		string	true	"DBMS provider name"	example(mysql)
//	@Param			dbid		path		string	true	"Database name"		example(myapp_db)
//	@Param			tableid		path		string	true	"Table name"		example(users)
//	@Param			filterpart	query		string	false	"SQL filter clause"	example("WHERE id > 100 ORDER BY name DESC")
//	@Success		200			{array}		object	"Table rows"
//	@Failure		400			{object}	map[string]string	"Bad request"
//	@Failure		401			{object}	map[string]string	"Unauthorized"
//	@Failure		500			{object}	map[string]string	"Internal server error"
//	@Security		BearerAuth
//	@Router			/db/{dbms}/{dbid}/table/{tableid}/rows [get]
func (h *DatabaseHandlers) GetTableRows(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dbms := vars["dbms"]
	dbid := vars["dbid"]
	tableid := vars["tableid"]

	// Get filterpart parameter for SQL filtering
	filterpart := r.URL.Query().Get("filterpart")

	conn, err := h.databaseManager.GetConnection(dbms)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check Accept header to determine response format
	acceptHeader := r.Header.Get("Accept")

	if acceptHeader == "application/x-json-stream" {
		// Stream as NDJSON
		if err := StreamTableRows(w, conn, dbms, dbid, tableid, filterpart); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// Traditional JSON array response (backward compatibility)
		rows, err := GetTableRowsWithFilter(conn, dbms, dbid, tableid, filterpart)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rows)
	}
}

// CheckDBMSExists godoc
//
//	@Summary		Check DBMS provider exists
//	@Description	Check if a DBMS provider connection is available
//	@Tags			database
//	@Param			dbms	path	string	true	"DBMS provider name"	example(mysql)
//	@Success		200
//	@Failure		404
//	@Security		BearerAuth
//	@Router			/db/{dbms} [head]
func (h *DatabaseHandlers) CheckDBMSExists(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dbms := vars["dbms"]

	_, err := h.databaseManager.GetConnection(dbms)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// CheckDatabaseExists godoc
//
//	@Summary		Check database exists
//	@Description	Check if a specific database exists in the DBMS
//	@Tags			database
//	@Param			dbms	path	string	true	"DBMS provider name"	example(mysql)
//	@Param			dbid	path	string	true	"Database name"		example(myapp_db)
//	@Success		200
//	@Failure		404
//	@Security		BearerAuth
//	@Router			/db/{dbms}/{dbid} [head]
func (h *DatabaseHandlers) CheckDatabaseExistsHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dbms := vars["dbms"]
	dbid := vars["dbid"]

	conn, err := h.databaseManager.GetConnection(dbms)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Check if database exists
	exists, err := CheckDatabaseExists(conn, dbms, dbid)
	if err != nil || !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// CheckTableExists godoc
//
//	@Summary		Check table exists
//	@Description	Check if a specific table exists in the database
//	@Tags			database
//	@Param			dbms		path	string	true	"DBMS provider name"	example(mysql)
//	@Param			dbid		path	string	true	"Database name"		example(myapp_db)
//	@Param			tableid		path	string	true	"Table name"		example(users)
//	@Success		200
//	@Failure		404
//	@Security		BearerAuth
//	@Router			/db/{dbms}/{dbid}/table/{tableid} [head]
func (h *DatabaseHandlers) CheckTableExistsHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dbms := vars["dbms"]
	dbid := vars["dbid"]
	tableid := vars["tableid"]

	conn, err := h.databaseManager.GetConnection(dbms)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Check if table exists
	exists, err := CheckTableExists(conn, dbms, dbid, tableid)
	if err != nil || !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// SetupDatabaseRoutes sets up all database-related routes on the provided router
func SetupDatabaseRoutes(router *mux.Router, databaseManager *db.DatabaseManager) {
	handlers := NewDatabaseHandlers(databaseManager)

	// DBMS provider routes
	router.HandleFunc("/db/dbmss", handlers.GetProviders).Methods("GET")

	// Database routes
	router.HandleFunc("/db/{dbms}/dbs", handlers.GetDatabases).Methods("GET")

	// Table routes
	router.HandleFunc("/db/{dbms}/{dbid}/tables", handlers.GetTables).Methods("GET")
	router.HandleFunc("/db/{dbms}/{dbid}/table/{tableid}/rows", handlers.GetTableRows).Methods("GET")

	// HEAD methods for resource existence checking
	router.HandleFunc("/db/{dbms}", handlers.CheckDBMSExists).Methods("HEAD")
	router.HandleFunc("/db/{dbms}/{dbid}", handlers.CheckDatabaseExistsHTTP).Methods("HEAD")
	router.HandleFunc("/db/{dbms}/{dbid}/table/{tableid}", handlers.CheckTableExistsHTTP).Methods("HEAD")
}
