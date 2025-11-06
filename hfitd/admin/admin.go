// Package admin provides Unix socket server for administrative commands
package admin

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	redisclient "hfitd/redis"
)

type AdminServer struct {
	socketPath  string
	redisClient *redisclient.Client
	jwtKeyPair  *JWTKeyPair
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
		socketPath:  socketPath,
		redisClient: redisClient,
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

	// Check if JWT private key exists in Redis
	privateKeyPEM, err := s.redisClient.GetJWTPrivateKey(ctx)
	if err == nil {
		// Load existing key pair
		return s.loadJWTKeyPair(privateKeyPEM)
	}

	// Generate new key pair
	return s.renewJWTKeyPair()
}

// renewJWTKeyPair generates a new JWT key pair
func (s *AdminServer) renewJWTKeyPair() error {
	// Generate new RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %v", err)
	}

	s.jwtKeyPair = &JWTKeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}

	// Store private key in Redis
	privateKeyPEM := s.privateKeyToPEM(privateKey)
	ctx := context.Background()

	if err := s.redisClient.SetJWTPrivateKey(ctx, privateKeyPEM); err != nil {
		return fmt.Errorf("failed to store JWT private key: %v", err)
	}

	// Store public key in Redis
	publicKeyPEM, err := s.publicKeyToPEM(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to convert public key to PEM: %v", err)
	}

	if err := s.redisClient.SetJWTPublicKey(ctx, publicKeyPEM); err != nil {
		return fmt.Errorf("failed to store JWT public key: %v", err)
	}

	// Store generation timestamp in RFC3339 format
	generationTime := time.Now().Format(time.RFC3339)
	if err := s.redisClient.SetJWTKeyGenerationTime(ctx, generationTime); err != nil {
		return fmt.Errorf("failed to store JWT key generation time: %v", err)
	}

	return nil
}

// loadJWTKeyPair loads JWT key pair from PEM
func (s *AdminServer) loadJWTKeyPair(privateKeyPEM string) error {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return fmt.Errorf("failed to decode PEM block")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
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

// getJWTPublicKeyPEM returns the JWT public key in PEM format
func (s *AdminServer) getJWTPublicKeyPEM() (string, error) {
	ctx := context.Background()

	// Try to get public key from Redis first
	publicKeyPEM, err := s.redisClient.GetJWTPublicKey(ctx)
	if err == nil {
		return publicKeyPEM, nil
	}

	// Fallback to in-memory key pair if Redis doesn't have it
	if s.jwtKeyPair == nil {
		return "", fmt.Errorf("JWT key pair not initialized and not found in Redis")
	}

	publicKeyDER, err := x509.MarshalPKIXPublicKey(s.jwtKeyPair.PublicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %v", err)
	}

	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyDER,
	}

	return string(pem.EncodeToMemory(block)), nil
}

// GetJWTPublicKeyPEM returns the JWT public key in PEM format (public method)
func (s *AdminServer) GetJWTPublicKeyPEM() (string, error) {
	return s.getJWTPublicKeyPEM()
}

// GetJWTKeyGenerationTime returns the JWT key generation timestamp
func (s *AdminServer) GetJWTKeyGenerationTime() (string, error) {
	ctx := context.Background()
	return s.redisClient.GetJWTKeyGenerationTime(ctx)
}

// privateKeyToPEM converts private key to PEM format
func (s *AdminServer) privateKeyToPEM(key *rsa.PrivateKey) string {
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return string(pem.EncodeToMemory(block))
}

// publicKeyToPEM converts public key to PEM format
func (s *AdminServer) publicKeyToPEM(key *rsa.PublicKey) (string, error) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %v", err)
	}

	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	return string(pem.EncodeToMemory(block)), nil
}
