package security

import (
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
