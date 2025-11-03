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

	"hfitd/db"

	"github.com/gorilla/mux"
)

// SetupDatabaseRoutes sets up all database-related routes on the provided router
func SetupDatabaseRoutes(router *mux.Router, databaseManager *db.DatabaseManager) {
	// DBMS provider routes

	// GetProviders godoc
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
	router.HandleFunc("/db/dbmss", func(w http.ResponseWriter, r *http.Request) {
		providers := databaseManager.GetProviders()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(providers)
	}).Methods("GET")

	// Database routes

	// GetDatabases godoc
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
	router.HandleFunc("/db/{dbms}/dbs", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		dbms := vars["dbms"]

		conn, err := databaseManager.GetConnection(dbms)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Get databases for this DBMS provider
		databases, err := GetDatabases(conn, dbms, databaseManager)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(databases)
	}).Methods("GET")

	// GetTables godoc
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
	router.HandleFunc("/db/{dbms}/{dbid}/tables", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		dbms := vars["dbms"]
		dbid := vars["dbid"]

		conn, err := databaseManager.GetConnection(dbms)
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
	}).Methods("GET")

	router.HandleFunc("/db/{dbms}/{dbid}/table/{tableid}/rows", func(w http.ResponseWriter, r *http.Request) {
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
	}).Methods("GET")

	// HEAD methods for resource existence checking
	router.HandleFunc("/db/{dbms}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		dbms := vars["dbms"]

		_, err := databaseManager.GetConnection(dbms)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}).Methods("HEAD")

	router.HandleFunc("/db/{dbms}/{dbid}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		dbms := vars["dbms"]
		dbid := vars["dbid"]

		conn, err := databaseManager.GetConnection(dbms)
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
	}).Methods("HEAD")

	router.HandleFunc("/db/{dbms}/{dbid}/table/{tableid}", func(w http.ResponseWriter, r *http.Request) {
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
		exists, err := CheckTableExists(conn, dbms, dbid, tableid)
		if err != nil || !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}).Methods("HEAD")
}
