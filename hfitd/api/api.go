// provide the REST API for Hot Fixture Tool
package api

import (
	"encoding/json"
	"net/http"

	"hfitd/admin"
	"hfitd/auth"
	"hfitd/config"
	"hfitd/db"
	redisclient "hfitd/redis"

	"github.com/gorilla/mux"
)

/*
API:
 /db/dbs
 /db/{dbid}/tables
 /db/{dbid}/table/{tableid}/rows
 /files/list
 /files/download?path=
*/

/*
* NewHandler creates a new HTTP handler for the API.
 */
func NewHandler(database *db.Database, cfg *config.Config, adminServer *admin.AdminServer) http.Handler {
	router := mux.NewRouter()

	// Initialize Redis client
	redisClient, err := redisclient.NewClient(cfg.Redis)
	if err != nil {
		panic("Failed to initialize Redis client: " + err.Error())
	}

	// Initialize authentication manager
	authManager, err := auth.NewAuthManager([]byte(cfg.Auth.JWTSecret), redisClient)
	if err != nil {
		panic("Failed to initialize auth manager: " + err.Error())
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

	// Database routes
	protected.HandleFunc("/db/dbs", Dbs).Methods("GET")

	protected.HandleFunc("/db/{dbid}/tables", func(w http.ResponseWriter, r *http.Request) {
		// Handler logic to list tables in a database
	}).Methods("GET")

	protected.HandleFunc("/db/{dbid}/table/{tableid}/rows", func(w http.ResponseWriter, r *http.Request) {
		// Handler logic to list rows in a table
	}).Methods("GET")

	// File routes
	protected.HandleFunc("/files/list", func(w http.ResponseWriter, r *http.Request) {
		// Handler logic to list files
	}).Methods("GET")

	protected.HandleFunc("/files/download", func(w http.ResponseWriter, r *http.Request) {
		// Handler logic to download a file
	}).Methods("GET")

	// Health check (unprotected)
	router.HandleFunc("/health", healthCheckHandler).Methods("GET")

	return router
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
