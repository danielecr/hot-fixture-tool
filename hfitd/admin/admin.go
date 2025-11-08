// Package admin provides Unix socket server for administrative commands
package admin

import (
	"bufio"
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"

	redisclient "hfitd/redis"
	"hfitd/security"
)

type AdminServer struct {
	socketPath    string
	redisClient   *redisclient.Client
	jwtKeyPair    *JWTKeyPair
	cryptoManager *security.CryptoManager
}

type JWTKeyPair struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

type AdminCommand struct {
	Action string   `json:"action"`
	Args   []string `json:"args"`
}

type AdminResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// NewAdminServer creates a new admin server
func NewAdminServer(socketPath string, redisClient *redisclient.Client) *AdminServer {
	return &AdminServer{
		socketPath:    socketPath,
		redisClient:   redisClient,
		cryptoManager: security.NewCryptoManager(),
	}
}

// Start starts the Unix socket server
func (s *AdminServer) Start(ctx context.Context) error {
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

	log.Printf("Admin server listening on Unix socket: %s", s.socketPath)

	// Initialize JWT key pair if not exists
	if err := s.initializeJWTKeyPair(); err != nil {
		return fmt.Errorf("failed to initialize JWT key pair: %v", err)
	}

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
				log.Printf("Failed to accept connection: %v", err)
				continue
			}
		}

		go s.handleConnection(conn)
	}
}

// handleConnection handles a single client connection
func (s *AdminServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()

		var cmd AdminCommand
		if err := json.Unmarshal([]byte(line), &cmd); err != nil {
			s.sendResponse(conn, AdminResponse{
				Success: false,
				Message: fmt.Sprintf("Invalid command format: %v", err),
			})
			continue
		}

		response := s.executeCommand(cmd)
		s.sendResponse(conn, response)
	}
}

// executeCommand executes an admin command
func (s *AdminServer) executeCommand(cmd AdminCommand) AdminResponse {
	ctx := context.Background()

	switch cmd.Action {
	case "adduser":
		if len(cmd.Args) != 2 {
			return AdminResponse{
				Success: false,
				Message: "Usage: adduser <email> <public_key_pem>",
			}
		}
		email, publicKeyPEM := cmd.Args[0], cmd.Args[1]

		// Validate public key using security module before storing
		if err := s.cryptoManager.KeyParser.ValidatePublicKey(publicKeyPEM); err != nil {
			return AdminResponse{
				Success: false,
				Message: fmt.Sprintf("Invalid public key format: %v", err),
			}
		}

		// Store validated key in Redis
		if err := s.redisClient.SetUserPublicKey(ctx, email, publicKeyPEM); err != nil {
			return AdminResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to add user: %v", err),
			}
		}

		return AdminResponse{
			Success: true,
			Message: fmt.Sprintf("User %s added successfully", email),
		}

	case "renew-jwt":
		if err := s.renewJWTKeyPair(); err != nil {
			return AdminResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to renew JWT key pair: %v", err),
			}
		}

		return AdminResponse{
			Success: true,
			Message: "JWT key pair renewed successfully",
		}

	case "get-jwt-public-key":
		publicKeyPEM, err := s.getJWTPublicKeyPEM()
		if err != nil {
			return AdminResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to get JWT public key: %v", err),
			}
		}

		return AdminResponse{
			Success: true,
			Message: "JWT public key retrieved",
			Data:    publicKeyPEM,
		}

	case "get-jwt-generation-time":
		generationTime, err := s.GetJWTKeyGenerationTime()
		if err != nil {
			return AdminResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to get JWT generation time: %v", err),
			}
		}

		return AdminResponse{
			Success: true,
			Message: "JWT generation time retrieved",
			Data:    generationTime,
		}

	default:
		return AdminResponse{
			Success: false,
			Message: fmt.Sprintf("Unknown command: %s", cmd.Action),
		}
	}
}

// sendResponse sends a response to the client
func (s *AdminServer) sendResponse(conn net.Conn, response AdminResponse) {
	data, _ := json.Marshal(response)
	conn.Write(append(data, '\n'))
}

// initializeJWTKeyPair initializes JWT key pair if not exists
func (s *AdminServer) initializeJWTKeyPair() error {
	ctx := context.Background()
	privateKeyPEM, err := s.redisClient.GetJWTPrivateKeyForSigning(ctx)
	if err == nil {
		return s.loadJWTKeyPairFromPEM(privateKeyPEM)
	}
	return s.renewJWTKeyPair()
}

// renewJWTKeyPair generates and stores new JWT key pair
func (s *AdminServer) renewJWTKeyPair() error {
	// Generate complete key pair in PEM format with timestamp - security module responsibility
	keyPairPEM, err := s.cryptoManager.KeyGenerator.GenerateJWTKeyPairPEM(2048)
	if err != nil {
		return fmt.Errorf("failed to generate JWT key pair: %v", err)
	}

	// Store in Redis - pure storage, no crypto validation
	ctx := context.Background()
	if err := s.redisClient.SetJWTKeyPairPEM(ctx, keyPairPEM.PrivateKeyPEM, keyPairPEM.PublicKeyPEM, keyPairPEM.GenerationTime); err != nil {
		return fmt.Errorf("failed to store JWT key pair: %v", err)
	}

	// Update in-memory keys for JWT operations - security module handles parsing
	return s.loadJWTKeyPairFromPEM(keyPairPEM.PrivateKeyPEM)
}

// loadJWTKeyPairFromPEM loads keys into memory for JWT signing
func (s *AdminServer) loadJWTKeyPairFromPEM(privateKeyPEM string) error {
	privateKey, err := s.cryptoManager.KeyParser.ParseJWTPrivateKeyForSigning(privateKeyPEM)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %v", err)
	}

	s.jwtKeyPair = &JWTKeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}
	return nil
}

// GetJWTPrivateKey returns the JWT private key for signing
func (s *AdminServer) GetJWTPrivateKey() *rsa.PrivateKey {
	if s.jwtKeyPair == nil {
		return nil
	}
	return s.jwtKeyPair.PrivateKey
}

// GetJWTPublicKey returns the JWT public key for verification
func (s *AdminServer) GetJWTPublicKey() *rsa.PublicKey {
	if s.jwtKeyPair == nil {
		return nil
	}
	return s.jwtKeyPair.PublicKey
}

// getJWTPublicKeyPEM returns public key PEM for export (no conversion needed)
func (s *AdminServer) getJWTPublicKeyPEM() (string, error) {
	ctx := context.Background()
	publicKeyPEM, _, err := s.redisClient.GetJWTPublicKeyInfo(ctx)
	return publicKeyPEM, err
}

// GetJWTPublicKeyPEM returns the JWT public key in PEM format (public method)
func (s *AdminServer) GetJWTPublicKeyPEM() (string, error) {
	return s.getJWTPublicKeyPEM()
}

// GetJWTKeyGenerationTime returns generation timestamp
func (s *AdminServer) GetJWTKeyGenerationTime() (string, error) {
	ctx := context.Background()
	_, generationTime, err := s.redisClient.GetJWTPublicKeyInfo(ctx)
	return generationTime, err
}
