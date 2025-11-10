package security

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func TestChallengeGenerator(t *testing.T) {
	cg := NewChallengeGenerator()

	// Test generating random bytes
	bytes, err := cg.GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed to generate random bytes: %v", err)
	}

	if len(bytes) != 32 {
		t.Errorf("Expected 32 bytes, got %d", len(bytes))
	}

	// Test that two calls generate different bytes
	bytes2, err := cg.GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed to generate second set of random bytes: %v", err)
	}

	// It's extremely unlikely (but not impossible) for two random 32-byte arrays to be identical
	equal := true
	for i := 0; i < len(bytes); i++ {
		if bytes[i] != bytes2[i] {
			equal = false
			break
		}
	}
	if equal {
		t.Error("Two consecutive random byte generations should not be identical")
	}
}

func TestCryptoManager(t *testing.T) {
	cm := NewCryptoManager()

	// Test that all components are initialized
	if cm.ChallengeGenerator == nil {
		t.Error("ChallengeGenerator not initialized")
	}
	if cm.KeyParser == nil {
		t.Error("KeyParser not initialized")
	}
	if cm.SignatureVerifier == nil {
		t.Error("SignatureVerifier not initialized")
	}
	if cm.HashGenerator == nil {
		t.Error("HashGenerator not initialized")
	}

	// Test hash generation
	data := []byte("test data")
	hash := cm.HashGenerator.SHA256(data)
	if len(hash) != 32 {
		t.Errorf("Expected SHA256 hash to be 32 bytes, got %d", len(hash))
	}

	// Test challenge generation through manager
	challenge, err := cm.ChallengeGenerator.GenerateRandomBytes(16)
	if err != nil {
		t.Fatalf("Failed to generate challenge: %v", err)
	}
	if len(challenge) != 16 {
		t.Errorf("Expected 16-byte challenge, got %d bytes", len(challenge))
	}
}

func TestSignatureVerifierEd25519(t *testing.T) {
	sv := NewSignatureVerifier()

	// generate a key pair for testing
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key pair: %v", err)
	}

	message := []byte("test message")
	signature := ed25519.Sign(privateKey, message)

	// Test valid signature
	err = sv.verifyOpenSSHEd25519Signature(string(publicKey), message, signature)
	if err != nil {
		t.Errorf("Valid signature verification failed: %v", err)
	}

	// Test invalid signature
	invalidSignature := make([]byte, len(signature))
	copy(invalidSignature, signature)
	invalidSignature[0] ^= 0xFF // Corrupt the signature

	err = sv.verifyOpenSSHEd25519Signature(string(publicKey), message, invalidSignature)
	if err == nil {
		t.Error("Invalid signature verification should have failed but passed")
	}

}

func TestSignatureVerifierECDSA(t *testing.T) {
	sv := NewSignatureVerifier()

	// Generate ECDSA key pair for testing
	ecdsaPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ECDSA key pair: %v", err)
	}

	// Convert public key to PEM format manually
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&ecdsaPrivateKey.PublicKey)
	if err != nil {
		t.Fatalf("Failed to marshal ECDSA public key: %v", err)
	}

	pubKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	}
	ecdsaPublicKeyPEM := string(pem.EncodeToMemory(pubKeyPEM))

	message := []byte("test message")

	// Hash the message with SHA256 (required for ECDSA)
	hashGen := NewHashGenerator()
	messageHash := hashGen.SHA256(message)

	// Sign the hash (convert [32]byte to []byte)
	signature, err := ecdsa.SignASN1(rand.Reader, ecdsaPrivateKey, messageHash[:])
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	// Test valid signature verification
	err = sv.VerifySignature(ecdsaPublicKeyPEM, message, signature)
	if err != nil {
		t.Errorf("Valid ECDSA signature verification failed: %v", err)
	}

	// Test invalid signature
	invalidSignature := make([]byte, len(signature))
	copy(invalidSignature, signature)
	invalidSignature[0] ^= 0xFF // Corrupt the signature

	err = sv.VerifySignature(ecdsaPublicKeyPEM, message, invalidSignature)
	if err == nil {
		t.Error("Invalid ECDSA signature verification should have failed but passed")
	}
}
