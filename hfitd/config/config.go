/*
 * Hot Fixture Tool Daemon (hfitd)
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

// Package config provides configuration loading and management for the application.
package config

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Server        ServerConfig
	DBMSProviders map[string]DatabaseConfig
	Volumes       map[string]VolumeConfig
	Auth          AuthConfig
	Redis         RedisConfig
}

type ServerConfig struct {
	Address string
	Port    int
}

type DatabaseConfig struct {
	Type     string
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

type VolumeConfig struct {
	Path string
}

type RedisConfig struct {
	URL string
}

type AuthConfig struct {
	// JWT authentication using RSA key pairs managed by admin server
	// Keys are stored in Redis and generated via hfitd-cli renew-jwt
}

// LoadConfigFromEnv loads configuration from environment variables.
// It first attempts to load from .env file, then falls back to system environment variables.
func LoadConfigFromEnv() (*Config, error) {
	// Try to load .env file (ignore error if file doesn't exist)
	_ = godotenv.Load()

	serverAddress := os.Getenv("SERVER_ADDRESS")
	if serverAddress == "" {
		return nil, errors.New("SERVER_ADDRESS environment variable is required")
	}

	// Parse DBMS providers
	dbmsProviders := os.Getenv("DBMS_PROVIDERS")
	if dbmsProviders == "" {
		return nil, errors.New("DBMS_PROVIDERS environment variable is required")
	}

	providers := strings.Split(dbmsProviders, ",")
	dbConfigs := make(map[string]DatabaseConfig)

	for _, provider := range providers {
		provider = strings.TrimSpace(provider)

		dbType := os.Getenv(provider + ".DB_TYPE")
		if dbType == "" {
			return nil, errors.New(provider + ".DB_TYPE environment variable is required")
		}

		dbHost := os.Getenv(provider + ".DB_HOST")
		if dbHost == "" {
			return nil, errors.New(provider + ".DB_HOST environment variable is required")
		}

		dbUser := os.Getenv(provider + ".DB_USER")
		if dbUser == "" {
			return nil, errors.New(provider + ".DB_USER environment variable is required")
		}

		dbPassword := os.Getenv(provider + ".DB_PASSWORD")
		if dbPassword == "" {
			return nil, errors.New(provider + ".DB_PASSWORD environment variable is required")
		}

		dbName := os.Getenv(provider + ".DB_NAME")
		if dbName == "" {
			return nil, errors.New(provider + ".DB_NAME environment variable is required")
		}

		dbPortStr := os.Getenv(provider + ".DB_PORT")
		dbPort := 5432 // default port
		if dbPortStr != "" {
			if port, err := strconv.Atoi(dbPortStr); err == nil {
				dbPort = port
			}
		}

		dbConfigs[provider] = DatabaseConfig{
			Type:     dbType,
			Host:     dbHost,
			Port:     dbPort,
			User:     dbUser,
			Password: dbPassword,
			Name:     dbName,
		}
	}

	// Parse Volume providers
	volumes := os.Getenv("VOLUMES")
	volumeConfigs := make(map[string]VolumeConfig)

	if volumes != "" {
		volumeList := strings.Split(volumes, ",")
		for _, volume := range volumeList {
			volume = strings.TrimSpace(volume)

			volumePath := os.Getenv(volume + ".PATH")
			if volumePath == "" {
				return nil, errors.New(volume + ".PATH environment variable is required")
			}

			volumeConfigs[volume] = VolumeConfig{
				Path: volumePath,
			}
		}
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return nil, errors.New("REDIS_URL environment variable is required")
	}

	config := &Config{
		Server: ServerConfig{
			Address: serverAddress,
			Port:    80, // default port
		},
		DBMSProviders: dbConfigs,
		Volumes:       volumeConfigs,
		Redis: RedisConfig{
			URL: redisURL,
		},
		Auth: AuthConfig{
			// JWT keys managed by admin server and stored in Redis
		},
	}

	return config, nil
}
