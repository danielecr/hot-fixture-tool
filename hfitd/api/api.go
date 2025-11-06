/*
 * Hot Fixture Tool Daemon - REST API
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

// Hot Fixture Tool Daemon API
//
//	@title			Hot Fixture Tool Daemon API
//	@version		1.0.0
//	@description	API for managing database fixtures, file operations, and package templates
//	@termsOfService	https://github.com/danielecr/hot-fixture-tool
//
//	@contact.name	Daniele Cruciani
//	@contact.email	daniele@smartango.com
//
//	@license.name	MIT
//	@license.url	https://github.com/danielecr/hot-fixture-tool/blob/main/LICENSE
//
//	@host		localhost:8080
//	@BasePath	/api
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				JWT Bearer token authentication
//
// provide the REST API for Hot Fixture Tool
package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	_ "hfitd/docs" // Import swagger docs

	"hfitd/admin"
	"hfitd/auth"
	"hfitd/config"
	"hfitd/db"
	"hfitd/dbapi"
	"hfitd/fileapi"
	redisclient "hfitd/redis"
	"hfitd/templateapi"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
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
	authManager, err := auth.NewAuthManager(redisClient, adminServer)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize auth manager: %w", err)
	}

	// Well-known JWT public key endpoint (RFC 7517 style)
	router.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		handleJWKS(w, r, adminServer)
	}).Methods("GET")

	// Authentication routes (unprotected)
	router.HandleFunc("/auth/challenge", authManager.GenerateChallenge).Methods("POST")
	router.HandleFunc("/auth/authenticate", authManager.Authenticate).Methods("POST")

	// Swagger documentation (unprotected)
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// Standard swagger.json alias for convenience
	router.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/doc.json", http.StatusMovedPermanently)
	}).Methods("GET")

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

// healthCheckHandler godoc
//
//	@Summary		Health check
//	@Description	Simple health check endpoint to verify service availability
//	@Tags			system
//	@Accept			json
//	@Produce		plain
//	@Success		200		{string}	string	"OK"
//	@Router			/health [get]
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleJWKS godoc
//
//	@Summary		Get JSON Web Key Set
//	@Description	Get the JSON Web Key Set (JWKS) containing the public key for JWT verification
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Success		200		{object}	object	"JWKS with public keys"
//	@Failure		500		{object}	map[string]string	"Internal server error"
//	@Router			/.well-known/jwks.json [get]
func handleJWKS(w http.ResponseWriter, r *http.Request, adminServer *admin.AdminServer) {
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
}
