/*
 * Hot Fixture Tool Daemon (hfitd)
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

package main

import (
	"context"
	"fmt"
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

// fatalError prints an error message with support contact information and exits
func fatalError(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "hfitd error: %s: %v | Support: daniele@smartango.com | GitHub: github.com/danielecr/hot-fixture-tool\n", msg, err)
	} else {
		fmt.Fprintf(os.Stderr, "hfitd error: %s | Support: daniele@smartango.com | GitHub: github.com/danielecr/hot-fixture-tool\n", msg)
	}
	os.Exit(1)
}

func main() {
	// Load configuration from environment variables
	cfg, err := config.LoadConfigFromEnv()
	if err != nil {
		fatalError("Failed to load config from environment", err)
	}

	// Initialize Redis client
	redisClient, err := redisclient.NewClient(cfg.Redis)
	if err != nil {
		fatalError("Failed to initialize Redis client", err)
	}
	defer redisClient.Close()

	// Initialize database manager
	databaseManager, err := db.NewDatabaseManager(cfg.DBMSProviders)
	if err != nil {
		fatalError("Failed to initialize database manager", err)
	}
	defer databaseManager.Close()

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
	apiHandler, err := api.NewHandler(databaseManager, cfg, adminServer)
	if err != nil {
		fatalError("Failed to initialize API handler", err)
	}

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
		fatalError("Failed to start server", err)
	}
}

func getSocketPath() string {
	if path := os.Getenv("HFITD_SOCKET_PATH"); path != "" {
		return path
	}
	return "/tmp/hfitd.sock"
}
