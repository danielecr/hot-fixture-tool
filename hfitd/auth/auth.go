// Package auth provides authentication middleware and JWT token management
package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	redisclient "hfitd/redis"

	"github.com/golang-jwt/jwt/v5"
)

type AuthManager struct {
	jwtSecret   []byte
	redisClient *redisclient.Client
}

type ChallengeRequest struct {
	Username string `json:"username"`
}

type ChallengeResponse struct {
	Challenge string `json:"challenge"`
	ExpiresAt int64  `json:"expires_at"`
}

type AuthRequest struct {
	Username  string `json:"username"`
	Challenge string `json:"challenge"`
	Signature string `json:"signature"`
}

type AuthResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(jwtSecret []byte, redisClient *redisclient.Client) (*AuthManager, error) {
	return &AuthManager{
		jwtSecret:   jwtSecret,
		redisClient: redisClient,
	}, nil
}

// GenerateChallenge creates a random challenge for the client to sign
func (am *AuthManager) GenerateChallenge(w http.ResponseWriter, r *http.Request) {
	var req ChallengeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	// Generate a random 32-byte challenge
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		http.Error(w, "Failed to generate challenge", http.StatusInternalServerError)
		return
	}

	challengeStr := base64.StdEncoding.EncodeToString(challenge)
	expiresAt := time.Now().Add(5 * time.Minute).Unix()

	response := ChallengeResponse{
		Challenge: challengeStr,
		ExpiresAt: expiresAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Authenticate verifies the signed challenge and issues a JWT token
func (am *AuthManager) Authenticate(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Challenge == "" || req.Signature == "" {
		http.Error(w, "Username, challenge, and signature are required", http.StatusBadRequest)
		return
	}

	// Decode the challenge and signature
	challengeBytes, err := base64.StdEncoding.DecodeString(req.Challenge)
	if err != nil {
		http.Error(w, "Invalid challenge encoding", http.StatusBadRequest)
		return
	}

	signatureBytes, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil {
		http.Error(w, "Invalid signature encoding", http.StatusBadRequest)
		return
	}

	// Get user's public key from Redis
	ctx := context.Background()
	userPublicKey, err := am.redisClient.GetUserPublicKey(ctx, req.Username)
	if err != nil {
		http.Error(w, "User not found or invalid", http.StatusUnauthorized)
		return
	}

	// Verify the signature
	hashed := sha256.Sum256(challengeBytes)
	if err := rsa.VerifyPKCS1v15(userPublicKey, crypto.SHA256, hashed[:], signatureBytes); err != nil {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	expiresAt := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Username: req.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   req.Username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(am.jwtSecret)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	response := AuthResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt.Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// JWTMiddleware validates JWT tokens in requests
func (am *AuthManager) JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for auth endpoints
		if strings.HasPrefix(r.URL.Path, "/auth/") {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Check for Bearer token
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := tokenParts[1]

		// Parse and validate the token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return am.jwtSecret, nil
		})

		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		if claims, ok := token.Claims.(*Claims); ok && token.Valid {
			// Add user info to request context if needed
			r.Header.Set("X-User", claims.Username)
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}
	})
}
