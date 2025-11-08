/*package socketapi

 * Hot Fixture Tool Daemon (hfitd)
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

// Package socketapi provides Unix socket API for administration and maintenance commands
package socketapi

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"

	"hfitd/admin"
	redisclient "hfitd/redis"
	"hfitd/security"
)

// Command represents a command received via Unix socket
type Command struct {
	Action string   `json:"action"`
	Args   []string `json:"args"`
}

// Response represents a response sent via Unix socket
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// SocketServer handles Unix socket communication for maintenance and administration
type SocketServer struct {
	socketPath    string
	redisClient   *redisclient.Client
	adminServer   *admin.AdminServer
	cryptoManager *security.CryptoManager
}

// NewSocketServer creates a new Unix socket server for maintenance operations
func NewSocketServer(socketPath string, redisClient *redisclient.Client, adminServer *admin.AdminServer) *SocketServer {
	return &SocketServer{
		socketPath:    socketPath,
		redisClient:   redisClient,
		adminServer:   adminServer,
		cryptoManager: security.NewCryptoManager(),
	}
}

// Start starts the Unix socket server for maintenance commands
func (s *SocketServer) Start(ctx context.Context) error {
	// Remove existing socket file
	if err := os.RemoveAll(s.socketPath); err != nil {
		return fmt.Errorf("failed to remove existing socket: %v", err)
	}

	// Create Unix socket listener
	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create Unix socket: %v", err)
	}
	defer listener.Close()

	// Set socket permissions (owner read/write only)
	if err := os.Chmod(s.socketPath, 0600); err != nil {
		return fmt.Errorf("failed to set socket permissions: %v", err)
	}

	log.Printf("Socket API server listening on Unix socket: %s", s.socketPath)

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				log.Printf("Failed to accept socket connection: %v", err)
				continue
			}
		}

		go s.handleConnection(conn)
	}
}

// handleConnection handles a single client connection
func (s *SocketServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()

		var cmd Command
		if err := json.Unmarshal([]byte(line), &cmd); err != nil {
			s.sendResponse(conn, Response{
				Success: false,
				Message: fmt.Sprintf("Invalid command format: %v", err),
			})
			continue
		}

		response := s.executeCommand(cmd)
		s.sendResponse(conn, response)
	}
}

// executeCommand executes a maintenance/administration command
func (s *SocketServer) executeCommand(cmd Command) Response {
	ctx := context.Background()

	switch cmd.Action {
	case "status":
		return s.handleStatus()

	case "adduser":
		if len(cmd.Args) != 2 {
			return Response{
				Success: false,
				Message: "Usage: adduser <email> <public_key_pem>",
			}
		}
		return s.handleAddUser(ctx, cmd.Args[0], cmd.Args[1])

	case "renew-jwt":
		return s.handleRenewJWT()

	case "get-jwt-public-key":
		return s.handleGetJWTPublicKey()

	case "get-jwt-generation-time":
		return s.handleGetJWTGenerationTime()

	case "ping":
		return Response{
			Success: true,
			Message: "pong",
		}

	default:
		return Response{
			Success: false,
			Message: fmt.Sprintf("Unknown command: %s", cmd.Action),
		}
	}
}

// handleStatus returns server status information
func (s *SocketServer) handleStatus() Response {
	ctx := context.Background()

	// Check Redis connection (try a simple get operation)
	redisStatus := "connected"
	if _, err := s.redisClient.Get(ctx, "test_connection"); err != nil && err.Error() != "redis: nil" {
		redisStatus = fmt.Sprintf("error: %v", err)
	}

	// Check JWT key status
	jwtStatus := "not initialized"
	if _, _, err := s.redisClient.GetJWTPublicKeyInfo(ctx); err == nil {
		jwtStatus = "initialized"
	}

	status := fmt.Sprintf("Server Status:\n- Redis: %s\n- JWT Keys: %s", redisStatus, jwtStatus)

	return Response{
		Success: true,
		Message: "Server status retrieved",
		Data:    status,
	}
}

// handleAddUser adds a new user with public key validation
func (s *SocketServer) handleAddUser(ctx context.Context, email, publicKeyPEM string) Response {
	log.Printf("Adding user %s with public key (first 50 chars): %.50s...", email, publicKeyPEM)

	// Validate public key using security module
	if err := s.cryptoManager.KeyParser.ValidatePublicKey(publicKeyPEM); err != nil {
		log.Printf("Public key validation failed for user %s: %v", email, err)
		return Response{
			Success: false,
			Message: fmt.Sprintf("Invalid public key format: %v", err),
		}
	}

	// Store validated key in Redis
	if err := s.redisClient.SetUserPublicKey(ctx, email, publicKeyPEM); err != nil {
		return Response{
			Success: false,
			Message: fmt.Sprintf("Failed to add user: %v", err),
		}
	}

	return Response{
		Success: true,
		Message: fmt.Sprintf("User %s added successfully", email),
	}
}

// handleRenewJWT renews the JWT key pair
func (s *SocketServer) handleRenewJWT() Response {
	// Generate complete key pair in PEM format with timestamp
	keyPairPEM, err := s.cryptoManager.KeyGenerator.GenerateJWTKeyPairPEM(2048)
	if err != nil {
		return Response{
			Success: false,
			Message: fmt.Sprintf("Failed to generate JWT key pair: %v", err),
		}
	}

	// Store in Redis
	ctx := context.Background()
	if err := s.redisClient.SetJWTKeyPairPEM(ctx, keyPairPEM.PrivateKeyPEM, keyPairPEM.PublicKeyPEM, keyPairPEM.GenerationTime); err != nil {
		return Response{
			Success: false,
			Message: fmt.Sprintf("Failed to store JWT key pair: %v", err),
		}
	}

	return Response{
		Success: true,
		Message: "JWT key pair renewed successfully",
	}
}

// handleGetJWTPublicKey returns the JWT public key in PEM format
func (s *SocketServer) handleGetJWTPublicKey() Response {
	ctx := context.Background()
	publicKeyPEM, _, err := s.redisClient.GetJWTPublicKeyInfo(ctx)
	if err != nil {
		return Response{
			Success: false,
			Message: fmt.Sprintf("Failed to get JWT public key: %v", err),
		}
	}

	return Response{
		Success: true,
		Message: "JWT public key retrieved",
		Data:    publicKeyPEM,
	}
}

// handleGetJWTGenerationTime returns the JWT key generation timestamp
func (s *SocketServer) handleGetJWTGenerationTime() Response {
	ctx := context.Background()
	_, generationTime, err := s.redisClient.GetJWTPublicKeyInfo(ctx)
	if err != nil {
		return Response{
			Success: false,
			Message: fmt.Sprintf("Failed to get JWT generation time: %v", err),
		}
	}

	return Response{
		Success: true,
		Message: "JWT generation time retrieved",
		Data:    generationTime,
	}
}

// sendResponse sends a JSON response to the client
func (s *SocketServer) sendResponse(conn net.Conn, response Response) {
	data, _ := json.Marshal(response)
	conn.Write(append(data, '\n'))
}
