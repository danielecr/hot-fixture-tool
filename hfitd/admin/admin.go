// Package admin provides JWT key management for HTTP API authentication
package admin

import (
	"context"
	"crypto/rsa"
	"fmt"

	redisclient "hfitd/redis"
	"hfitd/security"
)

// AdminServer provides JWT key management for HTTP API operations
type AdminServer struct {
	redisClient   *redisclient.Client
	jwtKeyPair    *JWTKeyPair
	cryptoManager *security.CryptoManager
}

type JWTKeyPair struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

// NewAdminServer creates a new admin server for JWT key management
func NewAdminServer(redisClient *redisclient.Client) *AdminServer {
	server := &AdminServer{
		redisClient:   redisClient,
		cryptoManager: security.NewCryptoManager(),
	}

	// Initialize JWT keys from Redis for HTTP API operations
	server.initializeJWTKeyPair()

	return server
}

// JWT Key Management Methods (for HTTP API)

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
