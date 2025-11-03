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
	"encoding/json"
	"fmt"
	"net/http"

	"hfitd/admin"
	"hfitd/auth"
	"hfitd/config"
	"hfitd/db"
	"hfitd/dbapi"
	"hfitd/fileapi"
	redisclient "hfitd/redis"
	"hfitd/templateapi"

	"github.com/gorilla/mux"
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

	// Database routes (from dbapi module)
	dbapi.SetupDatabaseRoutes(protected, databaseManager)
	/*
	 * Database API endpoints:
	 * GET    /db/dbmss                             - List available DBMS providers
	 * GET    /db/{dbms}/dbs                        - List databases for a DBMS
	 * GET    /db/{dbms}/{dbid}/tables              - List tables in a database
	 * GET    /db/{dbms}/{dbid}/table/{tableid}/rows - Get table rows (JSON or NDJSON stream)
	 * HEAD   /db/{dbms}                            - Check DBMS availability
	 * HEAD   /db/{dbms}/{dbid}                     - Check database existence
	 * HEAD   /db/{dbms}/{dbid}/table/{tableid}     - Check table existence
	 */

	// File and volume routes (from fileapi module)
	fileapi.SetupFileRoutes(protected, cfg)
	/*
	 * File API endpoints:
	 * GET    /volumes                              - List available volumes
	 * HEAD   /volumes/{volume}                     - Check volume existence
	 * GET    /files/{volume}/list                  - List files in volume (NDJSON stream)
	 * GET    /files/{volume}/download              - Download file(s) with filtering
	 * HEAD   /files/{volume}/{filepath:.*}         - Check file existence
	 */

	// Template routes (from templateapi module)
	templateapi.SetupTemplateRoutes(protected, cfg, databaseManager, redisClient)
	/*
	 * Template API endpoints:
	 * GET    /packtmpl                             - List user's package templates
	 * GET    /packtmpl/{templatename}              - Get specific template (YAML)
	 * POST   /packtmpl/{templatename}              - Create/update template
	 * DELETE /packtmpl/{templatename}              - Delete template
	 * POST   /packtmplpackupld                     - Bulk upload templates (tar.gz)
	 * POST   /packdownload/{templatename}          - Generate package from template
	 */

	// Health check (unprotected)
	router.HandleFunc("/health", healthCheckHandler).Methods("GET")

	return router, nil
}

/*
* healthCheckHandler provides a simple health check endpoint.
 */
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
