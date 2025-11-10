/*
 * Hot Fixture Tool Daemon - Template Storage Module
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

// Package tmplstorage provides Redis storage operations for package templates
package tmplstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	redisclient "hfitd/redis"
)

// TemplateStorage manages package template storage in Redis
type TemplateStorage struct {
	redisClient *redisclient.Client
}

// PackageMetadata represents the metadata stored for package downloads
type PackageMetadata struct {
	Input       []string      `json:"input"`
	Replacement []Replacement `json:"replacement"`
	Timestamps  Timestamps    `json:"timestamps"`
	Template    string        `json:"pkg-template"`
}

// Replacement tracks variable substitution for metadata
type Replacement struct {
	Variable string `json:"variable"`
	Value    string `json:"value"`
	Source   string `json:"source"`
}

// Timestamps represents timing information for package creation
type Timestamps struct {
	PackageCreation time.Time            `json:"package_creation"`
	FileMTimes      map[string]time.Time `json:"file_mtimes"`
}

// NewTemplateStorage creates a new template storage instance
func NewTemplateStorage(redisClient *redisclient.Client) *TemplateStorage {
	return &TemplateStorage{
		redisClient: redisClient,
	}
}

// StoreTemplate stores a package template in Redis
func (ts *TemplateStorage) StoreTemplate(ctx context.Context, userEmail, templateName, yamlContent string) error {
	// Store the template definition
	if err := ts.redisClient.TmplSetTemplate(ctx, userEmail, templateName, yamlContent); err != nil {
		return fmt.Errorf("failed to store template: %v", err)
	}

	// Update the template list
	if err := ts.addToTemplateList(ctx, userEmail, templateName); err != nil {
		return fmt.Errorf("failed to update template list: %v", err)
	}

	return nil
}

// GetTemplate retrieves a package template from Redis
func (ts *TemplateStorage) GetTemplate(ctx context.Context, userEmail, templateName string) (string, error) {
	yamlContent, err := ts.redisClient.TmplGetTemplate(ctx, userEmail, templateName)
	if err != nil {
		return "", fmt.Errorf("failed to get template: %v", err)
	}
	return yamlContent, nil
}

// ListTemplates returns all template names for a user
func (ts *TemplateStorage) ListTemplates(ctx context.Context, userEmail string) ([]string, error) {
	jsonData, err := ts.redisClient.TmplGetList(ctx, userEmail)
	if err != nil {
		// If key doesn't exist, return empty list
		return []string{}, nil
	}

	var templates []string
	if err := json.Unmarshal([]byte(jsonData), &templates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template list: %v", err)
	}

	return templates, nil
}

// DeleteTemplate removes a package template from Redis
func (ts *TemplateStorage) DeleteTemplate(ctx context.Context, userEmail, templateName string) error {
	// Remove the template definition
	if err := ts.redisClient.TmplDelTemplate(ctx, userEmail, templateName); err != nil {
		return fmt.Errorf("failed to delete template: %v", err)
	}

	// Update the template list
	if err := ts.removeFromTemplateList(ctx, userEmail, templateName); err != nil {
		return fmt.Errorf("failed to update template list: %v", err)
	}

	return nil
}

// TemplateExists checks if a template exists for a user
func (ts *TemplateStorage) TemplateExists(ctx context.Context, userEmail, templateName string) (bool, error) {
	exists, err := ts.redisClient.TmplExistsTemplate(ctx, userEmail, templateName)
	if err != nil {
		return false, fmt.Errorf("failed to check template existence: %v", err)
	}
	return exists, nil
}

// LogDownload logs a package download execution
func (ts *TemplateStorage) LogDownload(ctx context.Context, userEmail, templateName string, timestamp int64) error {
	// Get existing log
	existing, err := ts.redisClient.TmplGetLog(ctx, userEmail, templateName)
	var timestamps []int64
	if err == nil {
		// Parse existing timestamps
		if err := json.Unmarshal([]byte(existing), &timestamps); err != nil {
			return fmt.Errorf("failed to unmarshal existing log: %v", err)
		}
	}

	// Add new timestamp
	timestamps = append(timestamps, timestamp)

	// Store updated log
	jsonData, err := json.Marshal(timestamps)
	if err != nil {
		return fmt.Errorf("failed to marshal timestamps: %v", err)
	}

	if err := ts.redisClient.TmplSetLog(ctx, userEmail, templateName, string(jsonData)); err != nil {
		return fmt.Errorf("failed to store download log: %v", err)
	}

	return nil
}

// StoreDownloadMetadata stores detailed metadata for a package download
func (ts *TemplateStorage) StoreDownloadMetadata(ctx context.Context, userEmail, templateName string, timestamp int64, metadata PackageMetadata) error {
	jsonData, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %v", err)
	}

	if err := ts.redisClient.TmplSetMetadata(ctx, userEmail, templateName, timestamp, string(jsonData)); err != nil {
		return fmt.Errorf("failed to store download metadata: %v", err)
	}

	return nil
}

// GetDownloadMetadata retrieves metadata for a specific package download
func (ts *TemplateStorage) GetDownloadMetadata(ctx context.Context, userEmail, templateName string, timestamp int64) (PackageMetadata, error) {
	jsonData, err := ts.redisClient.TmplGetMetadata(ctx, userEmail, templateName, timestamp)
	if err != nil {
		return PackageMetadata{}, fmt.Errorf("failed to get download metadata: %v", err)
	}

	var metadata PackageMetadata
	if err := json.Unmarshal([]byte(jsonData), &metadata); err != nil {
		return PackageMetadata{}, fmt.Errorf("failed to unmarshal metadata: %v", err)
	}

	return metadata, nil
}

// addToTemplateList adds a template to the user's template list
func (ts *TemplateStorage) addToTemplateList(ctx context.Context, userEmail, templateName string) error {
	templates, err := ts.ListTemplates(ctx, userEmail)
	if err != nil {
		return err
	}

	// Check if template already exists in list
	for _, name := range templates {
		if name == templateName {
			return nil // Already exists
		}
	}

	// Add new template
	templates = append(templates, templateName)

	// Store updated list
	jsonData, err := json.Marshal(templates)
	if err != nil {
		return fmt.Errorf("failed to marshal template list: %v", err)
	}

	if err := ts.redisClient.TmplSetList(ctx, userEmail, string(jsonData)); err != nil {
		return fmt.Errorf("failed to store template list: %v", err)
	}

	return nil
}

// removeFromTemplateList removes a template from the user's template list
func (ts *TemplateStorage) removeFromTemplateList(ctx context.Context, userEmail, templateName string) error {
	templates, err := ts.ListTemplates(ctx, userEmail)
	if err != nil {
		return err
	}

	// Remove template from list
	var newTemplates []string
	for _, name := range templates {
		if name != templateName {
			newTemplates = append(newTemplates, name)
		}
	}

	// Store updated list
	jsonData, err := json.Marshal(newTemplates)
	if err != nil {
		return fmt.Errorf("failed to marshal template list: %v", err)
	}

	if err := ts.redisClient.TmplSetList(ctx, userEmail, string(jsonData)); err != nil {
		return fmt.Errorf("failed to store template list: %v", err)
	}

	return nil
}
