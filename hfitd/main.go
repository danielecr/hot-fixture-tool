package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"hfitd/admin"
	"hfitd/api"
	"hfitd/config"
	"hfitd/db"
	redisclient "hfitd/redis"
)

func main() {
	// Load configuration from environment variables
	cfg, err := config.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("Failed to load config from environment: %v", err)
	}

	// Initialize Redis client
	redisClient, err := redisclient.NewClient(cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to initialize Redis client: %v", err)
	}
	defer redisClient.Close()

	// Initialize database
	database, err := db.NewDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start admin Unix socket server
	socketPath := getSocketPath()
	adminServer := admin.NewAdminServer(socketPath, redisClient)
	go func() {
		if err := adminServer.Start(ctx); err != nil {
			log.Printf("Admin server error: %v", err)
		}
	}()

	// Set up API routes
	apiHandler := api.NewHandler(database, cfg, adminServer)

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		cancel()
		os.Remove(socketPath)
		os.Exit(0)
	}()

	// Start HTTP server
	log.Printf("Starting server on %s", cfg.Server.Address)
	log.Printf("Admin socket: %s", socketPath)
	if err := http.ListenAndServe(cfg.Server.Address, apiHandler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getSocketPath() string {
	if path := os.Getenv("HFITD_SOCKET_PATH"); path != "" {
		return path
	}
	return "/tmp/hfitd.sock"
}
