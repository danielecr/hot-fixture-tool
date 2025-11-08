/*
 * Hot Fixture Tool Daemon (hfitd)
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

// Package security provides cryptographic utilities and key management functions
package security

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// ChallengeGenerator provides methods for generating cryptographic challenges
type ChallengeGenerator struct{}

// NewChallengeGenerator creates a new challenge generator
func NewChallengeGenerator() *ChallengeGenerator {
	return &ChallengeGenerator{}
}

// GenerateRandomBytes generates random bytes of specified length
func (cg *ChallengeGenerator) GenerateRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %v", err)
	}
	return bytes, nil
}

// KeyGenerator provides methods for generating cryptographic key pairs
type KeyGenerator struct{}

// NewKeyGenerator creates a new key generator
func NewKeyGenerator() *KeyGenerator {
	return &KeyGenerator{}
}

// GenerateRSAKeyPair generates a new RSA key pair with the specified bit size
func (kg *KeyGenerator) GenerateRSAKeyPair(bits int) (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key pair: %v", err)
	}
	return privateKey, nil
}

// KeyParser provides methods for parsing different key formats
type KeyParser struct{}

// NewKeyParser creates a new key parser
func NewKeyParser() *KeyParser {
	return &KeyParser{}
}

// ParsePEMPublicKey parses a PEM-encoded public key and returns the crypto public key
func (kp *KeyParser) ParsePEMPublicKey(publicKeyPEM string) (crypto.PublicKey, error) {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %v", err)
	}

	return publicKey, nil
}

// ParseRSAPublicKeyFromPEM parses a PEM-encoded RSA public key
func (kp *KeyParser) ParseRSAPublicKeyFromPEM(publicKeyPEM string) (*rsa.PublicKey, error) {
	publicKey, err := kp.ParsePEMPublicKey(publicKeyPEM)
	if err != nil {
		return nil, err
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not RSA")
	}

	return rsaPublicKey, nil
}

// ParseRSAPrivateKeyFromPEM parses a PEM-encoded RSA private key
func (kp *KeyParser) ParseRSAPrivateKeyFromPEM(privateKeyPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	return privateKey, nil
}

// ValidatePublicKey validates public keys in multiple formats (PEM, OpenSSH, etc.)
func (kp *KeyParser) ValidatePublicKey(publicKey string) error {
	publicKey = strings.TrimSpace(publicKey)

	if publicKey == "" {
		return fmt.Errorf("empty public key")
	}

	// Try PEM format first
	if strings.Contains(publicKey, "-----BEGIN") {
		_, err := kp.ParsePEMPublicKey(publicKey)
		if err == nil {
			return nil // Valid PEM format
		}
	}

	// Try OpenSSH format (supports all SSH key types)
	if _, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey)); err == nil {
		return nil // Valid OpenSSH format
	}

	// If both formats fail, return error with helpful message
	return fmt.Errorf("unsupported public key format - expected PEM (-----BEGIN PUBLIC KEY-----) or OpenSSH format (ssh-rsa, ecdsa-sha2-*, ssh-ed25519, etc.)")
}

// DetectKeyFormat detects the format of a public key
func (kp *KeyParser) DetectKeyFormat(publicKey string) string {
	publicKey = strings.TrimSpace(publicKey)

	if strings.Contains(publicKey, "-----BEGIN") {
		return "pem"
	}

	if strings.HasPrefix(publicKey, "ssh-rsa") {
		return "ssh-rsa"
	}

	if strings.HasPrefix(publicKey, "ssh-ed25519") {
		return "ssh-ed25519"
	}

	if strings.HasPrefix(publicKey, "ecdsa-sha2-") {
		return "ssh-ecdsa"
	}

	return "unknown"
}

// SignatureVerifier provides methods for verifying digital signatures
type SignatureVerifier struct {
	keyParser *KeyParser
}

// NewSignatureVerifier creates a new signature verifier
func NewSignatureVerifier() *SignatureVerifier {
	return &SignatureVerifier{
		keyParser: NewKeyParser(),
	}
}

// VerifySignature verifies a signature against a message using the provided public key
func (sv *SignatureVerifier) VerifySignature(publicKeyStr string, message, signature []byte) error {
	format := sv.keyParser.DetectKeyFormat(publicKeyStr)

	switch format {
	case "pem":
		return sv.verifyPEMSignature(publicKeyStr, message, signature)
	case "ssh-rsa":
		return sv.verifyOpenSSHRSASignature(publicKeyStr, message, signature)
	case "ssh-ecdsa":
		return sv.verifyOpenSSHECDSASignature(publicKeyStr, message, signature)
	case "ssh-ed25519":
		return sv.verifyOpenSSHEd25519Signature(publicKeyStr, message, signature)
	default:
		return fmt.Errorf("unsupported public key format: %s", format)
	}
}

// verifyPEMSignature verifies signature for PEM format keys
func (sv *SignatureVerifier) verifyPEMSignature(publicKeyPEM string, message, signature []byte) error {
	publicKey, err := sv.keyParser.ParsePEMPublicKey(publicKeyPEM)
	if err != nil {
		return err
	}

	hashed := sha256.Sum256(message)

	switch key := publicKey.(type) {
	case *rsa.PublicKey:
		return sv.verifyRSASignature(key, hashed[:], signature)
	case *ecdsa.PublicKey:
		return sv.verifyECDSASignature(key, hashed[:], signature)
	default:
		return fmt.Errorf("unsupported key type in PEM: %T", publicKey)
	}
}

// verifyRSASignature verifies RSA signature
func (sv *SignatureVerifier) verifyRSASignature(publicKey *rsa.PublicKey, hash, signature []byte) error {
	return rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash, signature)
}

// verifyECDSASignature verifies ECDSA signature
func (sv *SignatureVerifier) verifyECDSASignature(publicKey *ecdsa.PublicKey, hash, signature []byte) error {
	if !ecdsa.VerifyASN1(publicKey, hash, signature) {
		return fmt.Errorf("ECDSA signature verification failed")
	}
	return nil
}

// verifyOpenSSHRSASignature verifies signature for OpenSSH RSA format
func (sv *SignatureVerifier) verifyOpenSSHRSASignature(publicKeyStr string, message, signature []byte) error {
	sshPublicKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKeyStr))
	if err != nil {
		return fmt.Errorf("failed to parse SSH public key: %v", err)
	}

	cryptoKey := sshPublicKey.(ssh.CryptoPublicKey)
	if rsaPubKey, ok := cryptoKey.CryptoPublicKey().(*rsa.PublicKey); ok {
		hashed := sha256.Sum256(message)
		return sv.verifyRSASignature(rsaPubKey, hashed[:], signature)
	}

	return fmt.Errorf("SSH key is not RSA")
}

// verifyOpenSSHECDSASignature verifies signature for OpenSSH ECDSA format
func (sv *SignatureVerifier) verifyOpenSSHECDSASignature(publicKeyStr string, message, signature []byte) error {
	sshPublicKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKeyStr))
	if err != nil {
		return fmt.Errorf("failed to parse SSH public key: %v", err)
	}

	cryptoKey := sshPublicKey.(ssh.CryptoPublicKey)
	if ecdsaPubKey, ok := cryptoKey.CryptoPublicKey().(*ecdsa.PublicKey); ok {
		hashed := sha256.Sum256(message)
		return sv.verifyECDSASignature(ecdsaPubKey, hashed[:], signature)
	}

	return fmt.Errorf("SSH key is not ECDSA")
}

// verifyOpenSSHEd25519Signature verifies signature for OpenSSH Ed25519 format
func (sv *SignatureVerifier) verifyOpenSSHEd25519Signature(publicKeyStr string, message, signature []byte) error {
	// TODO: Implement Ed25519 signature verification
	// This would use ed25519.Verify when fully implemented
	_ = ed25519.PublicKey(nil) // Reference to avoid import error
	return fmt.Errorf("Ed25519 signature verification not implemented")
}

// HashGenerator provides methods for generating cryptographic hashes
type HashGenerator struct{}

// NewHashGenerator creates a new hash generator
func NewHashGenerator() *HashGenerator {
	return &HashGenerator{}
}

// SHA256 computes SHA256 hash of the input data
func (hg *HashGenerator) SHA256(data []byte) [32]byte {
	return sha256.Sum256(data)
}

// PEMConverter provides methods for converting keys to/from PEM format
type PEMConverter struct{}

// NewPEMConverter creates a new PEM converter
func NewPEMConverter() *PEMConverter {
	return &PEMConverter{}
}

// RSAPrivateKeyToPEM converts an RSA private key to PEM format
func (pc *PEMConverter) RSAPrivateKeyToPEM(key *rsa.PrivateKey) string {
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return string(pem.EncodeToMemory(block))
}

// RSAPublicKeyToPEM converts an RSA public key to PEM format
func (pc *PEMConverter) RSAPublicKeyToPEM(key *rsa.PublicKey) (string, error) {
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

// JWTKeyPairPEM represents a JWT key pair in PEM format with generation time
type JWTKeyPairPEM struct {
	PrivateKeyPEM  string
	PublicKeyPEM   string
	GenerationTime string // RFC3339 format
}

// GenerateJWTKeyPairPEM generates a complete JWT key pair in PEM format with timestamp
func (kg *KeyGenerator) GenerateJWTKeyPairPEM(keySize int) (*JWTKeyPairPEM, error) {
	// Generate RSA key pair
	privateKey, err := kg.GenerateRSAKeyPair(keySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key pair: %v", err)
	}

	// Convert to PEM format
	pemConverter := NewPEMConverter()
	privateKeyPEM := pemConverter.RSAPrivateKeyToPEM(privateKey)

	publicKeyPEM, err := pemConverter.RSAPublicKeyToPEM(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to convert public key to PEM: %v", err)
	}

	return &JWTKeyPairPEM{
		PrivateKeyPEM:  privateKeyPEM,
		PublicKeyPEM:   publicKeyPEM,
		GenerationTime: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// ParseJWTPrivateKeyForSigning converts PEM private key to *rsa.PrivateKey for JWT signing
func (kp *KeyParser) ParseJWTPrivateKeyForSigning(privateKeyPEM string) (*rsa.PrivateKey, error) {
	return kp.ParseRSAPrivateKeyFromPEM(privateKeyPEM)
}

// CryptoManager provides a centralized interface to all crypto operations
type CryptoManager struct {
	ChallengeGenerator *ChallengeGenerator
	KeyParser          *KeyParser
	SignatureVerifier  *SignatureVerifier
	HashGenerator      *HashGenerator
	KeyGenerator       *KeyGenerator
	PEMConverter       *PEMConverter
}

// NewCryptoManager creates a new crypto manager with all components
func NewCryptoManager() *CryptoManager {
	return &CryptoManager{
		ChallengeGenerator: NewChallengeGenerator(),
		KeyParser:          NewKeyParser(),
		SignatureVerifier:  NewSignatureVerifier(),
		HashGenerator:      NewHashGenerator(),
		KeyGenerator:       NewKeyGenerator(),
		PEMConverter:       NewPEMConverter(),
	}
}
