/*
 * Hot Fixture Tool Daemon - Package Template Module
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

// Package pkgtmpl provides package template processing and variable substitution
package pkgtmpl

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"hfitd/config"
	"hfitd/db"
	"hfitd/tmplstorage"
)

// PackageTemplate represents the complete package template structure
type PackageTemplate struct {
	HfitVersion  int                         `yaml:"hfitVersion"`
	TemplateName string                      `yaml:"templateName"`
	ProjectName  string                      `yaml:"projectName"`
	PackageName  string                      `yaml:"packageName"` // Can contain variables like $1
	Prepare      []PrepareStep               `yaml:"prepare"`
	Exports      map[string]ExportDefinition `yaml:"exports"`
}

// PrepareStep represents a single variable preparation step
type PrepareStep struct {
	SetVar string         `yaml:"setVar"`
	From   string         `yaml:"from"`   // "input" or "hot-data"
	Source string         `yaml:"source"` // For input: "$1", "$2", etc.
	HData  *HotDataSource `yaml:"hdata"`  // For hot-data
}

// HotDataSource represents hot data source configuration
type HotDataSource struct {
	Type         string `yaml:"type"`          // "dbquery" or "volume"
	DBMS         string `yaml:"dbms"`          // For dbquery
	Query        string `yaml:"query"`         // For dbquery
	Volume       string `yaml:"volume"`        // For volume
	Glob         string `yaml:"glob"`          // For volume
	Sort         string `yaml:"sort"`          // For volume
	RegexReplace string `yaml:"regex_replace"` // For volume
}

// ExportDefinition represents an export item (same as packdownload)
type ExportDefinition struct {
	Type string                 `yaml:"type"`
	Data map[string]interface{} `yaml:"data"`
}

// VariableContext holds the execution context for variable processing
type VariableContext struct {
	InputParams  []string            // Command line parameters
	Variables    map[string]string   // Calculated variables
	Replacements []Replacement       // Track all replacements for metadata
	DBManager    *db.DatabaseManager // Database manager for queries
	Config       *config.Config      // Configuration for volume access
}

// Replacement tracks variable substitution for metadata
type Replacement struct {
	Variable string `json:"variable"`
	Value    string `json:"value"`
	Source   string `json:"source"` // "input" or "hot-data"
}

// NewVariableContext creates a new variable processing context
func NewVariableContext(inputParams []string, dbManager *db.DatabaseManager, cfg *config.Config) *VariableContext {
	return &VariableContext{
		InputParams:  inputParams,
		Variables:    make(map[string]string),
		Replacements: make([]Replacement, 0),
		DBManager:    dbManager,
		Config:       cfg,
	}
}

// ProcessTemplate processes a package template with given input parameters
func (ctx *VariableContext) ProcessTemplate(template *PackageTemplate) error {
	// Execute prepare steps in order
	for _, step := range template.Prepare {
		if err := ctx.ExecutePrepareStep(step); err != nil {
			return fmt.Errorf("failed to execute prepare step for variable %s: %v", step.SetVar, err)
		}
	}

	// Substitute variables in package name
	packageName, err := ctx.SubstituteVariables(template.PackageName)
	if err != nil {
		return fmt.Errorf("failed to substitute variables in package name: %v", err)
	}
	template.PackageName = packageName

	// Substitute variables in exports
	for filename, exportDef := range template.Exports {
		if err := ctx.SubstituteExportDefinition(&exportDef); err != nil {
			return fmt.Errorf("failed to substitute variables in export %s: %v", filename, err)
		}
		template.Exports[filename] = exportDef
	}

	return nil
}

// ExecutePrepareStep executes a single prepare step
func (ctx *VariableContext) ExecutePrepareStep(step PrepareStep) error {
	var value string
	var source string
	var err error

	switch step.From {
	case "input":
		value, source, err = ctx.processInputVariable(step)
	case "hot-data":
		value, source, err = ctx.processHotDataVariable(step)
	default:
		return fmt.Errorf("unsupported variable source: %s", step.From)
	}

	if err != nil {
		return err
	}

	// Store the variable
	ctx.Variables[step.SetVar] = value
	ctx.Replacements = append(ctx.Replacements, Replacement{
		Variable: step.SetVar,
		Value:    value,
		Source:   source,
	})

	return nil
}

// processInputVariable processes input parameter variables
func (ctx *VariableContext) processInputVariable(step PrepareStep) (string, string, error) {
	// Parse input parameter index (e.g., "$1" -> index 0)
	if !strings.HasPrefix(step.Source, "$") {
		return "", "", fmt.Errorf("invalid input source format: %s", step.Source)
	}

	indexStr := strings.TrimPrefix(step.Source, "$")
	var index int
	if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
		return "", "", fmt.Errorf("invalid input parameter index: %s", indexStr)
	}

	// Convert to 0-based index
	index--
	if index < 0 || index >= len(ctx.InputParams) {
		return "", "", fmt.Errorf("input parameter index %d out of range (have %d parameters)", index+1, len(ctx.InputParams))
	}

	return ctx.InputParams[index], "input", nil
}

// processHotDataVariable processes hot-data variables
func (ctx *VariableContext) processHotDataVariable(step PrepareStep) (string, string, error) {
	if step.HData == nil {
		return "", "", fmt.Errorf("hdata configuration missing for hot-data variable")
	}

	switch step.HData.Type {
	case "dbquery":
		return ctx.processDBQueryVariable(step.HData)
	case "volume":
		return ctx.processVolumeVariable(step.HData)
	default:
		return "", "", fmt.Errorf("unsupported hot-data type: %s", step.HData.Type)
	}
}

// processDBQueryVariable executes database query and returns the result
func (ctx *VariableContext) processDBQueryVariable(hdata *HotDataSource) (string, string, error) {
	// Substitute variables in query
	query, err := ctx.SubstituteVariables(hdata.Query)
	if err != nil {
		return "", "", fmt.Errorf("failed to substitute variables in query: %v", err)
	}

	// Get database connection
	db, err := ctx.DBManager.GetConnection(hdata.DBMS)
	if err != nil {
		return "", "", fmt.Errorf("failed to get database connection: %v", err)
	}

	// Execute query
	var result string
	err = db.QueryRow(query).Scan(&result)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", fmt.Errorf("query returned no results: %s", query)
		}
		return "", "", fmt.Errorf("query execution failed: %v", err)
	}

	return result, "hot-data:dbquery", nil
}

// processVolumeVariable processes volume operations and returns the result
func (ctx *VariableContext) processVolumeVariable(hdata *HotDataSource) (string, string, error) {
	// TODO: Implement volume variable processing
	// This would involve:
	// 1. Substitute variables in glob pattern
	// 2. Find matching files in the volume
	// 3. Apply sorting if specified
	// 4. Apply regex replacement if specified
	// 5. Return the result

	return "", "", fmt.Errorf("volume variable processing not yet implemented")
}

// SubstituteVariables substitutes all ${variable} patterns in a string
func (ctx *VariableContext) SubstituteVariables(text string) (string, error) {
	// Regular expression to match ${variable} patterns
	re := regexp.MustCompile(`\$\{([^}]+)\}`)

	result := re.ReplaceAllStringFunc(text, func(match string) string {
		// Extract variable name
		varName := match[2 : len(match)-1] // Remove ${ and }

		// Look up variable value
		if value, exists := ctx.Variables[varName]; exists {
			return value
		}

		// Variable not found - return original match for now
		// In production, this might be an error
		return match
	})

	return result, nil
}

// SubstituteExportDefinition substitutes variables in an export definition
func (ctx *VariableContext) SubstituteExportDefinition(exportDef *ExportDefinition) error {
	// Substitute variables in the data map
	for key, value := range exportDef.Data {
		switch v := value.(type) {
		case string:
			substituted, err := ctx.SubstituteVariables(v)
			if err != nil {
				return fmt.Errorf("failed to substitute variables in %s: %v", key, err)
			}
			exportDef.Data[key] = substituted
		case []interface{}:
			// Handle arrays (like tablelist)
			for i, item := range v {
				if str, ok := item.(string); ok {
					substituted, err := ctx.SubstituteVariables(str)
					if err != nil {
						return fmt.Errorf("failed to substitute variables in %s[%d]: %v", key, i, err)
					}
					v[i] = substituted
				}
			}
		}
	}

	return nil
}

// PackageTemplateProcessor handles template processing operations
type PackageTemplateProcessor struct {
	config          *config.Config
	databaseManager *db.DatabaseManager
}

// NewPackageTemplateProcessor creates a new package template processor
func NewPackageTemplateProcessor(cfg *config.Config, dbManager *db.DatabaseManager) *PackageTemplateProcessor {
	return &PackageTemplateProcessor{
		config:          cfg,
		databaseManager: dbManager,
	}
}

// ProcessTemplate processes a package template with parameters and generates a package
func (ptp *PackageTemplateProcessor) ProcessTemplate(ctx context.Context, template *PackageTemplate, params []string, userEmail, templateName string, timestamp int64) (string, *tmplstorage.PackageMetadata, error) {
	// Create variable context
	vctx := &VariableContext{
		InputParams:  params,
		Replacements: make([]Replacement, 0),
		Variables:    make(map[string]string),
	}

	// Process input parameters
	for i, param := range params {
		varName := fmt.Sprintf("${INPUT_%d}", i+1)
		vctx.Variables[varName] = param
		vctx.Replacements = append(vctx.Replacements, Replacement{
			Variable: varName,
			Value:    param,
			Source:   "input",
		})
	}

	// Process prepare steps to get variables
	for _, step := range template.Prepare {
		if err := vctx.ExecutePrepareStep(step); err != nil {
			return "", nil, fmt.Errorf("failed to process variable %s: %v", step.SetVar, err)
		}
	}

	// Create temporary directory for processing
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("pkg-template-%s-%d", templateName, timestamp))
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Process exports (database exports)
	for exportName, export := range template.Exports {
		if err := ptp.processExport(ctx, export, vctx, tempDir); err != nil {
			return "", nil, fmt.Errorf("failed to process export %s: %v", exportName, err)
		}
	}

	// Create METADATA subfolder
	metadataDir := filepath.Join(tempDir, "METADATA")
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		return "", nil, fmt.Errorf("failed to create metadata directory: %v", err)
	}

	// Convert Replacement types for metadata
	metadataReplacements := make([]tmplstorage.Replacement, len(vctx.Replacements))
	for i, r := range vctx.Replacements {
		metadataReplacements[i] = tmplstorage.Replacement{
			Variable: r.Variable,
			Value:    r.Value,
			Source:   r.Source,
		}
	}

	// Generate metadata
	metadata := &tmplstorage.PackageMetadata{
		Input:       params,
		Replacement: metadataReplacements,
		Timestamps: tmplstorage.Timestamps{
			PackageCreation: time.Unix(timestamp, 0),
			FileMTimes:      make(map[string]time.Time),
		},
		Template: templateName,
	}

	// Save metadata to JSON file
	metadataFile := filepath.Join(metadataDir, "package_metadata.json")
	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal metadata: %v", err)
	}

	if err := os.WriteFile(metadataFile, metadataJSON, 0644); err != nil {
		return "", nil, fmt.Errorf("failed to write metadata file: %v", err)
	}

	// Create package archive in a temporary location
	packagePath := filepath.Join("/tmp", fmt.Sprintf("pkg-%s-%d.tar.gz", templateName, timestamp))
	if err := ptp.createTarGz(tempDir, packagePath); err != nil {
		return "", nil, fmt.Errorf("failed to create package archive: %v", err)
	}

	return packagePath, metadata, nil
}

// processExport processes a database export (simplified implementation)
func (ptp *PackageTemplateProcessor) processExport(ctx context.Context, export ExportDefinition, vctx *VariableContext, targetDir string) error {
	// This is a simplified placeholder - integrate with existing packdownload module
	// For now, just create placeholder files
	exportFile := filepath.Join(targetDir, "export.sql")
	return os.WriteFile(exportFile, []byte("-- Export placeholder\n"), 0644)
}

// createTarGz creates a tar.gz archive from a directory
func (ptp *PackageTemplateProcessor) createTarGz(sourceDir, targetPath string) error {
	// Ensure target directory exists
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %v", err)
	}

	// Create target file
	tarFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create tar file: %v", err)
	}
	defer tarFile.Close()

	// Create gzip writer
	gzipWriter := gzip.NewWriter(tarFile)
	defer gzipWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Walk through source directory
	return filepath.Walk(sourceDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(sourceDir, filePath)
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// Write file content (if it's a regular file)
		if info.Mode().IsRegular() {
			file, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(tarWriter, file)
			return err
		}

		return nil
	})
}
