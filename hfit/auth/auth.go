/*
 * Hot Fixture Tool CLI - Authentication
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

// Package auth provides authentication functionality for the hfit CLI tool
package auth

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
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
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type ChallengeRequest struct {
	Username string `json:"username"`
}

type ChallengeResponse struct {
	Challenge string `json:"challenge"`
}

type AuthRequest struct {
	Username  string `json:"username"`
	Challenge string `json:"challenge"`
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
	token, err := authenticate(serverURL, email, challenge, signature)
	if err != nil {
		return "", fmt.Errorf("failed to authenticate: %w", err)
	}

	return token, nil
}

func requestChallenge(serverURL, email string) (string, error) {
	challengeReq := ChallengeRequest{Username: email}
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

func signChallenge(challenge string, privateKey crypto.PrivateKey) (string, error) {
	// Decode the base64-encoded challenge to get the original bytes
	challengeBytes, err := base64.StdEncoding.DecodeString(challenge)
	if err != nil {
		return "", fmt.Errorf("failed to decode challenge: %w", err)
	}

	hash := sha256.Sum256(challengeBytes)

	switch key := privateKey.(type) {
	case *rsa.PrivateKey:
		signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
		if err != nil {
			return "", fmt.Errorf("failed to sign challenge with RSA: %w", err)
		}
		return base64.StdEncoding.EncodeToString(signature), nil

	case *ecdsa.PrivateKey:
		signature, err := ecdsa.SignASN1(rand.Reader, key, hash[:])
		if err != nil {
			return "", fmt.Errorf("failed to sign challenge with ECDSA: %w", err)
		}
		return base64.StdEncoding.EncodeToString(signature), nil

	case ed25519.PrivateKey:
		signature := ed25519.Sign(key, []byte(challenge))
		return base64.StdEncoding.EncodeToString(signature), nil

	default:
		return "", fmt.Errorf("unsupported private key type: %T", privateKey)
	}
}

func authenticate(serverURL, email, challenge, signature string) (string, error) {
	authReq := AuthRequest{
		Username:  email,
		Challenge: challenge,
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

func loadPrivateKey(keyPath string) (crypto.PrivateKey, error) {
	// If keyPath looks like a file path, read from file
	var keyData []byte

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

	// Check if it's OpenSSH format
	if bytes.Contains(keyData, []byte("-----BEGIN OPENSSH PRIVATE KEY-----")) {
		return parseOpenSSHPrivateKey(keyData)
	}

	// Parse as standard PEM format
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Try PKCS8 format first (supports multiple key types)
	if privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		return privateKey, nil
	}

	// Try PKCS1 format for RSA keys
	if privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return privateKey, nil
	}

	// Try EC private key format
	if privateKey, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return privateKey, nil
	}

	return nil, fmt.Errorf("failed to parse private key: unsupported format")
}

// parseOpenSSHPrivateKey parses OpenSSH format private keys
func parseOpenSSHPrivateKey(keyData []byte) (crypto.PrivateKey, error) {
	// Try parsing without passphrase first
	privateKey, err := ssh.ParseRawPrivateKey(keyData)
	if err != nil {
		// If it's a passphrase error, try with passphrase up to 3 times
		if strings.Contains(err.Error(), "passphrase protected") {
			maxAttempts := 3
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				fmt.Printf("Enter passphrase for private key (attempt %d/%d): ", attempt, maxAttempts)
				passphrase, readErr := term.ReadPassword(int(os.Stdin.Fd()))
				if readErr != nil {
					return nil, fmt.Errorf("failed to read passphrase: %w", readErr)
				}
				fmt.Println() // Print newline after password input

				// Parse with passphrase
				privateKey, err = ssh.ParseRawPrivateKeyWithPassphrase(keyData, passphrase)
				if err == nil {
					break // Success!
				}

				// Check if it's still a password error
				if attempt < maxAttempts && (strings.Contains(err.Error(), "password incorrect") || strings.Contains(err.Error(), "decryption password incorrect")) {
					fmt.Println("Incorrect passphrase, please try again.")
					continue
				}

				// If it's the last attempt or a different error, return the error
				if attempt == maxAttempts {
					return nil, fmt.Errorf("failed to parse OpenSSH private key after %d attempts: %w", maxAttempts, err)
				}
				return nil, fmt.Errorf("failed to parse OpenSSH private key with passphrase: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to parse OpenSSH private key: %w", err)
		}
	}

	// Convert to the appropriate crypto type
	switch key := privateKey.(type) {
	case *rsa.PrivateKey:
		return key, nil
	case *ecdsa.PrivateKey:
		return key, nil
	case *ed25519.PrivateKey:
		return *key, nil
	case ed25519.PrivateKey:
		return key, nil
	default:
		return nil, fmt.Errorf("unsupported OpenSSH private key type: %T", privateKey)
	}
}
