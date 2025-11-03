/*
 * Hot Fixture Tool Daemon (hfitd)
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

// Package redisclient provides Redis client and user key management
package redisclient

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"hfitd/config"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *redis.Client
}

// NewClient creates a new Redis client
func NewClient(cfg config.RedisConfig) (*Client, error) {
	opt, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %v", err)
	}

	rdb := redis.NewClient(opt)

	// Test connection
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	return &Client{rdb: rdb}, nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.rdb.Close()
}

// GetUserPublicKey retrieves the public key for a user
func (c *Client) GetUserPublicKey(ctx context.Context, userEmail string) (*rsa.PublicKey, error) {
	keyName := fmt.Sprintf("user__%s", userEmail)

	publicKeyPEM, err := c.rdb.Get(ctx, keyName).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("public key not found for user: %s", userEmail)
		}
		return nil, fmt.Errorf("failed to get public key from Redis: %v", err)
	}

	// Parse the PEM public key
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block for user: %s", userEmail)
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key for user %s: %v", userEmail, err)
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not RSA for user: %s", userEmail)
	}

	return rsaPublicKey, nil
}

// SetUserPublicKey stores a public key for a user
func (c *Client) SetUserPublicKey(ctx context.Context, userEmail, publicKeyPEM string) error {
	keyName := fmt.Sprintf("user__%s", userEmail)

	// Validate the public key before storing
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return fmt.Errorf("invalid PEM format for user: %s", userEmail)
	}

	_, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("invalid public key for user %s: %v", userEmail, err)
	}

	return c.rdb.Set(ctx, keyName, publicKeyPEM, 0).Err()
}

// DeleteUserPublicKey removes a user's public key
func (c *Client) DeleteUserPublicKey(ctx context.Context, userEmail string) error {
	keyName := fmt.Sprintf("user__%s", userEmail)
	return c.rdb.Del(ctx, keyName).Err()
}

// ListUsers returns all users who have public keys stored
func (c *Client) ListUsers(ctx context.Context) ([]string, error) {
	keys, err := c.rdb.Keys(ctx, "user__*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list user keys: %v", err)
	}

	users := make([]string, 0, len(keys))
	for _, key := range keys {
		// Extract email from key name (remove "user__" prefix)
		if len(key) > 6 {
			users = append(users, key[6:])
		}
	}

	return users, nil
}

// GetJWTPrivateKey retrieves the JWT private key
func (c *Client) GetJWTPrivateKey(ctx context.Context) (string, error) {
	privateKeyPEM, err := c.rdb.Get(ctx, "jwt_private_key").Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("JWT private key not found")
		}
		return "", fmt.Errorf("failed to get JWT private key from Redis: %v", err)
	}
	return privateKeyPEM, nil
}

// SetJWTPrivateKey stores the JWT private key
func (c *Client) SetJWTPrivateKey(ctx context.Context, privateKeyPEM string) error {
	return c.rdb.Set(ctx, "jwt_private_key", privateKeyPEM, 0).Err()
}

// Set stores a key-value pair in Redis
func (c *Client) Set(ctx context.Context, key, value string, expiration time.Duration) error {
	return c.rdb.Set(ctx, key, value, expiration).Err()
}

// Get retrieves a value from Redis by key
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	result, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("key not found: %s", key)
		}
		return "", fmt.Errorf("failed to get key %s from Redis: %v", key, err)
	}
	return result, nil
}

// Del deletes one or more keys from Redis
func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

// Exists checks if a key exists in Redis
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check key existence %s: %v", key, err)
	}
	return count > 0, nil
}
