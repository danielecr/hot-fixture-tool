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
	"fmt"
	"time"

	"hfitd/config"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	rdb    *redis.Client
	prefix string
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

	return &Client{
		rdb:    rdb,
		prefix: cfg.Prefix,
	}, nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.rdb.Close()
}

// buildKey builds a Redis key with the configured prefix
func (c *Client) buildKey(key string) string {
	return c.prefix + key
}

// extractUserEmail extracts email from a prefixed user key
func (c *Client) extractUserEmail(key string) string {
	userPrefix := c.prefix + "user__"
	if len(key) > len(userPrefix) {
		return key[len(userPrefix):]
	}
	return ""
}

// GetUserPublicKey retrieves the public key for a user by email
func (c *Client) GetUserPublicKeyString(ctx context.Context, userEmail string) (string, error) {
	keyName := c.buildKey(fmt.Sprintf("user__%s", userEmail))

	publicKeyStr, err := c.rdb.Get(ctx, keyName).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("public key not found for user: %s", userEmail)
		}
		return "", fmt.Errorf("failed to get public key from Redis: %v", err)
	}

	return publicKeyStr, nil
}

// SetUserPublicKey sets the public key for a user by email
func (c *Client) SetUserPublicKey(ctx context.Context, userEmail, publicKey string) error {
	keyName := c.buildKey(fmt.Sprintf("user__%s", userEmail))
	return c.rdb.Set(ctx, keyName, publicKey, 0).Err()
}

// DeleteUserPublicKey removes a user's public key
func (c *Client) DeleteUserPublicKey(ctx context.Context, userEmail string) error {
	keyName := c.buildKey(fmt.Sprintf("user__%s", userEmail))
	return c.rdb.Del(ctx, keyName).Err()
}

// ListUsers returns all users who have public keys stored
func (c *Client) ListUsers(ctx context.Context) ([]string, error) {
	keys, err := c.rdb.Keys(ctx, c.buildKey("user__*")).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list user keys: %v", err)
	}

	users := make([]string, 0, len(keys))
	for _, key := range keys {
		// Extract email from key name (remove prefix and "user__")
		email := c.extractUserEmail(key)
		if email != "" {
			users = append(users, email)
		}
	}

	return users, nil
}

// GetJWTPrivateKey retrieves the JWT private key
func (c *Client) GetJWTPrivateKey(ctx context.Context) (string, error) {
	privateKeyPEM, err := c.rdb.Get(ctx, c.buildKey("jwt_private_key")).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("JWT private key not found")
		}
		return "", fmt.Errorf("failed to get JWT private key from Redis: %v", err)
	}
	return privateKeyPEM, nil
}

// GetJWTPublicKey retrieves the JWT public key
func (c *Client) GetJWTPublicKey(ctx context.Context) (string, error) {
	publicKeyPEM, err := c.rdb.Get(ctx, c.buildKey("jwt_public_key")).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("JWT public key not found")
		}
		return "", fmt.Errorf("failed to get JWT public key from Redis: %v", err)
	}
	return publicKeyPEM, nil
}

// SetJWTKeyPairPEM stores JWT key pair data (private key, public key, generation time)
func (c *Client) SetJWTKeyPairPEM(ctx context.Context, privateKeyPEM, publicKeyPEM, generationTime string) error {
	pipe := c.rdb.Pipeline()
	pipe.Set(ctx, c.buildKey("jwt_private_key"), privateKeyPEM, 0)
	pipe.Set(ctx, c.buildKey("jwt_public_key"), publicKeyPEM, 0)
	pipe.Set(ctx, c.buildKey("jwt_key_generation_time"), generationTime, 0)
	_, err := pipe.Exec(ctx)
	return err
}

// GetJWTPublicKeyInfo returns public key PEM and generation time for info/export
func (c *Client) GetJWTPublicKeyInfo(ctx context.Context) (publicKeyPEM, generationTime string, err error) {
	pipe := c.rdb.Pipeline()
	pubKeyCmd := pipe.Get(ctx, c.buildKey("jwt_public_key"))
	genTimeCmd := pipe.Get(ctx, c.buildKey("jwt_key_generation_time"))
	_, err = pipe.Exec(ctx)
	if err != nil {
		return "", "", err
	}

	publicKeyPEM, err = pubKeyCmd.Result()
	if err != nil {
		return "", "", err
	}

	generationTime, err = genTimeCmd.Result()
	return publicKeyPEM, generationTime, err
}

// GetJWTPrivateKeyForSigning returns private key PEM for JWT signing operations
func (c *Client) GetJWTPrivateKeyForSigning(ctx context.Context) (string, error) {
	return c.rdb.Get(ctx, c.buildKey("jwt_private_key")).Result()
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
