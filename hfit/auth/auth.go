// Package auth provides authentication functionality for the hfit CLI tool
package auth

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
)

type ChallengeRequest struct {
	Email string `json:"email"`
}

type ChallengeResponse struct {
	Challenge string `json:"challenge"`
}

type AuthRequest struct {
	Email     string `json:"email"`
	Signature string `json:"signature"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

// AuthenticateWithChallenge performs the public key challenge authentication
func AuthenticateWithChallenge(serverURL, email, publicKeyPath string) (string, error) {
	// Load the private key for signing
	privateKey, err := loadPrivateKey(publicKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to load private key: %w", err)
	}

	// Step 1: Request challenge
	challenge, err := requestChallenge(serverURL, email)
	if err != nil {
		return "", fmt.Errorf("failed to request challenge: %w", err)
	}

	// Step 2: Sign challenge
	signature, err := signChallenge(challenge, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign challenge: %w", err)
	}

	// Step 3: Send signed challenge and get JWT
	token, err := authenticate(serverURL, email, signature)
	if err != nil {
		return "", fmt.Errorf("failed to authenticate: %w", err)
	}

	return token, nil
}

func requestChallenge(serverURL, email string) (string, error) {
	challengeReq := ChallengeRequest{Email: email}
	reqBody, err := json.Marshal(challengeReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal challenge request: %w", err)
	}

	resp, err := http.Post(serverURL+"/auth/challenge", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to send challenge request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("challenge request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var challengeResp ChallengeResponse
	if err := json.NewDecoder(resp.Body).Decode(&challengeResp); err != nil {
		return "", fmt.Errorf("failed to decode challenge response: %w", err)
	}

	return challengeResp.Challenge, nil
}

func signChallenge(challenge string, privateKey *rsa.PrivateKey) (string, error) {
	hash := sha256.Sum256([]byte(challenge))
	signature, err := rsa.SignPKCS1v15(nil, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign challenge: %w", err)
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func authenticate(serverURL, email, signature string) (string, error) {
	authReq := AuthRequest{
		Email:     email,
		Signature: signature,
	}
	reqBody, err := json.Marshal(authReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal auth request: %w", err)
	}

	resp, err := http.Post(serverURL+"/auth/authenticate", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to send auth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(body))
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", fmt.Errorf("failed to decode auth response: %w", err)
	}

	return authResp.Token, nil
}

func loadPrivateKey(keyPath string) (*rsa.PrivateKey, error) {
	// If keyPath looks like a file path, read from file
	var keyData []byte
	var err error
	
	if _, err := os.Stat(keyPath); err == nil {
		// It's a file path
		keyData, err = os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key file: %w", err)
		}
	} else {
		// Assume it's the key content directly
		keyData = []byte(keyPath)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format
		key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("failed to parse private key (PKCS1: %v, PKCS8: %v)", err, err2)
		}
		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not an RSA private key")
		}
	}

	return privateKey, nil
}