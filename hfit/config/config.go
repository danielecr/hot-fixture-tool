/*
 * Hot Fixture Tool CLI - Configuration Management
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 * 
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 * 
 * Licensed under the terms specified in the LICENSE file.
 */

// Package config provides configuration management for the hfit CLI tool
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	HfitdHost string
	Email     string
	PublicKey string
}

// GetConfigDir returns the config directory path
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".hfit"), nil
}

// GetConfigPath returns the config file path
func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config"), nil
}

// GetTokenPath returns the JWT token file path
func GetTokenPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "token"), nil
}

// LoadConfig loads configuration from ~/.hfit/config
func LoadConfig() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %s: %w", configPath, err)
	}
	defer file.Close()

	config := &Config{}
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		switch key {
		case "HFITD_HOST":
			config.HfitdHost = value
		case "EMAIL":
			config.Email = value
		case "PUBLIC_KEY":
			config.PublicKey = value
		}
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}
	
	return config, nil
}

// SaveConfig saves configuration to ~/.hfit/config
func SaveConfig(config *Config) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	
	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}
	
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()
	
	fmt.Fprintf(file, "HFITD_HOST=%s\n", config.HfitdHost)
	fmt.Fprintf(file, "EMAIL=%s\n", config.Email)
	fmt.Fprintf(file, "PUBLIC_KEY=%s\n", config.PublicKey)
	
	return nil
}

// LoadToken loads JWT token from ~/.hfit/token
func LoadToken() (string, error) {
	tokenPath, err := GetTokenPath()
	if err != nil {
		return "", err
	}
	
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return "", fmt.Errorf("failed to read token file: %w", err)
	}
	
	return strings.TrimSpace(string(data)), nil
}

// SaveToken saves JWT token to ~/.hfit/token
func SaveToken(token string) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	
	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	tokenPath, err := GetTokenPath()
	if err != nil {
		return err
	}
	
	return os.WriteFile(tokenPath, []byte(token), 0600)
}