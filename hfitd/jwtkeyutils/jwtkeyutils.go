// Package jwtkeyutils provides JWT key management utilities for HTTP API authentication
package jwtkeyutils

import (
	"context"
	"crypto/rsa"
	"fmt"

	redisclient "hfitd/redis"
	"hfitd/security"
)

// JwtData manages JWT key pairs for HTTP API operations
type JwtData struct {
	redisClient   *redisclient.Client
	jwtKeyPair    *JWTKeyPair
	cryptoManager *security.CryptoManager
}

type JWTKeyPair struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

// NewJwtData creates a new JWT key manager
func NewJwtData(redisClient *redisclient.Client) *JwtData {
	server := &JwtData{
		redisClient:   redisClient,
		cryptoManager: security.NewCryptoManager(),
	}

	// Initialize JWT keys from Redis for HTTP API operations
	server.initializeJWTKeyPair()

	return server
}

// JWT Key Management Methods (for HTTP API)

// initializeJWTKeyPair initializes JWT key pair if not exists
func (j *JwtData) initializeJWTKeyPair() error {
	ctx := context.Background()
	privateKeyPEM, err := j.redisClient.GetJWTPrivateKeyForSigning(ctx)
	if err == nil {
		return j.loadJWTKeyPairFromPEM(privateKeyPEM)
	}
	return j.renewJWTKeyPair()
}

// RenewJWTKeyPair generates and stores new JWT key pair (public method for external calls)
func (j *JwtData) RenewJWTKeyPair() error {
	return j.renewJWTKeyPair()
}

// renewJWTKeyPair generates and stores new JWT key pair
func (j *JwtData) renewJWTKeyPair() error {
	// Generate complete key pair in PEM format with timestamp - security module responsibility
	keyPairPEM, err := j.cryptoManager.KeyGenerator.GenerateJWTKeyPairPEM(2048)
	if err != nil {
		return fmt.Errorf("failed to generate JWT key pair: %v", err)
	}

	// Store in Redis - pure storage, no crypto validation
	ctx := context.Background()
	if err := j.redisClient.SetJWTKeyPairPEM(ctx, keyPairPEM.PrivateKeyPEM, keyPairPEM.PublicKeyPEM, keyPairPEM.GenerationTime); err != nil {
		return fmt.Errorf("failed to store JWT key pair: %v", err)
	}

	// Update in-memory keys for JWT operations - security module handles parsing
	// This ensures the server instance gets updated with new keys
	return j.loadJWTKeyPairFromPEM(keyPairPEM.PrivateKeyPEM)
}

// loadJWTKeyPairFromPEM loads keys into memory for JWT signing
func (j *JwtData) loadJWTKeyPairFromPEM(privateKeyPEM string) error {
	privateKey, err := j.cryptoManager.KeyParser.ParseJWTPrivateKeyForSigning(privateKeyPEM)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %v", err)
	}

	j.jwtKeyPair = &JWTKeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}
	return nil
}

// GetJWTPrivateKey returns the JWT private key for signing
func (j *JwtData) GetJWTPrivateKey() *rsa.PrivateKey {
	if j.jwtKeyPair == nil {
		return nil
	}
	return j.jwtKeyPair.PrivateKey
}

// GetJWTPublicKey returns the JWT public key for verification
func (j *JwtData) GetJWTPublicKey() *rsa.PublicKey {
	if j.jwtKeyPair == nil {
		return nil
	}
	return j.jwtKeyPair.PublicKey
}

// getJWTPublicKeyPEM returns public key PEM for export (no conversion needed)
func (j *JwtData) getJWTPublicKeyPEM() (string, error) {
	ctx := context.Background()
	publicKeyPEM, _, err := j.redisClient.GetJWTPublicKeyInfo(ctx)
	return publicKeyPEM, err
}

// GetJWTPublicKeyPEM returns the JWT public key in PEM format (public method)
func (j *JwtData) GetJWTPublicKeyPEM() (string, error) {
	return j.getJWTPublicKeyPEM()
}
