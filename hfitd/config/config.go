// Package config provides configuration loading and management for the application.
package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Redis    RedisConfig
}

type ServerConfig struct {
	Address string
	Port    int
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

type RedisConfig struct {
	URL string
}

type AuthConfig struct {
	JWTSecret string
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

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		return nil, errors.New("DB_HOST environment variable is required")
	}

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		return nil, errors.New("DB_USER environment variable is required")
	}

	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		return nil, errors.New("DB_PASSWORD environment variable is required")
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		return nil, errors.New("DB_NAME environment variable is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, errors.New("JWT_SECRET environment variable is required")
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
		Database: DatabaseConfig{
			Host:     dbHost,
			Port:     5432, // default port
			User:     dbUser,
			Password: dbPassword,
			Name:     dbName,
		},
		Redis: RedisConfig{
			URL: redisURL,
		},
		Auth: AuthConfig{
			JWTSecret: jwtSecret,
		},
	}

	return config, nil
}
