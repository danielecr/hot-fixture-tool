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

// CheckConnection tests Redis connectivity by performing a simple get operation
func (c *Client) CheckConnection(ctx context.Context) error {
	// Try a simple get operation on a test key with prefix
	testKey := c.buildKey("connection_test")
	_, err := c.rdb.Get(ctx, testKey).Result()

	// If the error is "redis: nil" (key doesn't exist), that's actually a successful connection
	if err != nil && err.Error() != "redis: nil" {
		return err
	}

	return nil
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
func (c *Client) GetUserPublicKey(ctx context.Context, userEmail string) (string, error) {
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

// Template Storage Methods

// TmplSetTemplate stores a template YAML content
func (c *Client) TmplSetTemplate(ctx context.Context, userEmail, templateName, yamlContent string) error {
	templateKey := c.buildKey(fmt.Sprintf("pkg_template_%s_%s", userEmail, templateName))
	return c.rdb.Set(ctx, templateKey, yamlContent, 0).Err()
}

// TmplGetTemplate retrieves a template YAML content
func (c *Client) TmplGetTemplate(ctx context.Context, userEmail, templateName string) (string, error) {
	templateKey := c.buildKey(fmt.Sprintf("pkg_template_%s_%s", userEmail, templateName))
	return c.rdb.Get(ctx, templateKey).Result()
}

// TmplDelTemplate deletes a template
func (c *Client) TmplDelTemplate(ctx context.Context, userEmail, templateName string) error {
	templateKey := c.buildKey(fmt.Sprintf("pkg_template_%s_%s", userEmail, templateName))
	return c.rdb.Del(ctx, templateKey).Err()
}

// TmplExistsTemplate checks if a template exists
func (c *Client) TmplExistsTemplate(ctx context.Context, userEmail, templateName string) (bool, error) {
	templateKey := c.buildKey(fmt.Sprintf("pkg_template_%s_%s", userEmail, templateName))
	count, err := c.rdb.Exists(ctx, templateKey).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// TmplSetList stores a template list (JSON array of template names)
func (c *Client) TmplSetList(ctx context.Context, userEmail, jsonData string) error {
	listKey := c.buildKey(fmt.Sprintf("pkg_templatelst_%s", userEmail))
	return c.rdb.Set(ctx, listKey, jsonData, 0).Err()
}

// TmplGetList retrieves a template list (JSON array of template names)
func (c *Client) TmplGetList(ctx context.Context, userEmail string) (string, error) {
	listKey := c.buildKey(fmt.Sprintf("pkg_templatelst_%s", userEmail))
	return c.rdb.Get(ctx, listKey).Result()
}

// TmplSetLog stores download log (JSON array of timestamps)
func (c *Client) TmplSetLog(ctx context.Context, userEmail, templateName, jsonData string) error {
	logKey := c.buildKey(fmt.Sprintf("%s_pkg_%s_dwnld", userEmail, templateName))
	return c.rdb.Set(ctx, logKey, jsonData, 0).Err()
}

// TmplGetLog retrieves download log (JSON array of timestamps)
func (c *Client) TmplGetLog(ctx context.Context, userEmail, templateName string) (string, error) {
	logKey := c.buildKey(fmt.Sprintf("%s_pkg_%s_dwnld", userEmail, templateName))
	return c.rdb.Get(ctx, logKey).Result()
}

// TmplGetMetadata retrieves download metadata by timestamp
func (c *Client) TmplGetMetadata(ctx context.Context, userEmail, templateName string, timestamp int64) (string, error) {
	metadataKey := c.buildKey(fmt.Sprintf("%s_pkg_%s_dwnld_%d", userEmail, templateName, timestamp))
	return c.rdb.Get(ctx, metadataKey).Result()
}

// TmplSetMetadata stores download metadata with a specific timestamp
func (c *Client) TmplSetMetadata(ctx context.Context, userEmail, templateName string, timestamp int64, jsonData string) error {
	metadataKey := c.buildKey(fmt.Sprintf("%s_pkg_%s_dwnld_%d", userEmail, templateName, timestamp))
	return c.rdb.Set(ctx, metadataKey, jsonData, 0).Err()
}
