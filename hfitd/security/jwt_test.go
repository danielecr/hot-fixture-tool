package security

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateJWTKeyPairPEM(t *testing.T) {
	// Create key generator
	kg := NewKeyGenerator()

	// Generate JWT key pair
	keyPair, err := kg.GenerateJWTKeyPairPEM(2048)
	if err != nil {
		t.Fatalf("Failed to generate JWT key pair: %v", err)
	}

	// Validate structure
	if keyPair == nil {
		t.Fatal("Key pair is nil")
	}

	if keyPair.PrivateKeyPEM == "" {
		t.Fatal("Private key PEM is empty")
	}

	if keyPair.PublicKeyPEM == "" {
		t.Fatal("Public key PEM is empty")
	}

	if keyPair.GenerationTime == "" {
		t.Fatal("Generation time is empty")
	}

	// Validate PEM format
	if !strings.Contains(keyPair.PrivateKeyPEM, "-----BEGIN RSA PRIVATE KEY-----") {
		t.Fatal("Private key PEM format is invalid")
	}

	if !strings.Contains(keyPair.PublicKeyPEM, "-----BEGIN PUBLIC KEY-----") {
		t.Fatal("Public key PEM format is invalid")
	}

	// Validate timestamp format (RFC3339)
	_, err = time.Parse(time.RFC3339, keyPair.GenerationTime)
	if err != nil {
		t.Fatalf("Invalid generation time format: %v", err)
	}

	// Test that we can parse the generated PEM keys
	kp := NewKeyParser()

	// Parse private key for signing
	privateKey, err := kp.ParseJWTPrivateKeyForSigning(keyPair.PrivateKeyPEM)
	if err != nil {
		t.Fatalf("Failed to parse generated private key: %v", err)
	}

	// Verify key size (2048 bits = 256 bytes)
	if privateKey.Size() != 256 {
		t.Fatalf("Expected key size 256, got %d", privateKey.Size())
	}

	// Test round-trip: parse the PEM back to ensure it's valid
	parsedPrivateKey, err := kp.ParseRSAPrivateKeyFromPEM(keyPair.PrivateKeyPEM)
	if err != nil {
		t.Fatalf("Failed to parse generated private key PEM: %v", err)
	}

	// Verify parsed key matches original
	if parsedPrivateKey.Size() != privateKey.Size() {
		t.Fatalf("Parsed key size mismatch: expected %d, got %d", privateKey.Size(), parsedPrivateKey.Size())
	}
}
