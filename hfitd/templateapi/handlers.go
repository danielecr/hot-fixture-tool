/*
 * Hot Fixture Tool Daemon - Template API Handlers
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

package templateapi

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"hfitd/apierrors"
	"hfitd/config"
	"hfitd/db"
	"hfitd/pkgtmpl"
	redisclient "hfitd/redis"
	"hfitd/tmplstorage"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
)

// TemplateHandlers contains the dependencies needed for template operations
type TemplateHandlers struct {
	config          *config.Config
	databaseManager *db.DatabaseManager
	redisClient     *redisclient.Client
	templateStorage *tmplstorage.TemplateStorage
}

// NewTemplateHandlers creates a new TemplateHandlers instance
func NewTemplateHandlers(cfg *config.Config, databaseManager *db.DatabaseManager, redisClient *redisclient.Client) *TemplateHandlers {
	return &TemplateHandlers{
		config:          cfg,
		databaseManager: databaseManager,
		redisClient:     redisClient,
		templateStorage: tmplstorage.NewTemplateStorage(redisClient),
	}
}

// GetTemplates godoc
//
//	@Summary		List package templates
//	@Description	Get list of all package templates for the authenticated user
//	@Tags			templates
//	@Accept			json
//	@Produce		json
//	@Success		200		{array}		object	"List of template objects"
//	@Failure		401		{object}	map[string]string	"Unauthorized"
//	@Failure		500		{object}	map[string]string	"Internal server error"
//	@Security		BearerAuth
//	@Router			/packtmpl [get]
func (h *TemplateHandlers) GetTemplates(w http.ResponseWriter, r *http.Request) {
	userEmail := r.Header.Get("X-User")
	if userEmail == "" {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	err := HandleListTemplates(w, r, userEmail, h.templateStorage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetTemplate godoc
//
//	@Summary		Get package template
//	@Description	Get a specific package template by name
//	@Tags			templates
//	@Accept			json
//	@Produce		json
//	@Param			templatename	path		string	true	"Template name"	example(webapp-starter)
//	@Success		200				{object}	object	"Template object"
//	@Failure		401				{object}	map[string]string	"Unauthorized"
//	@Failure		404				{object}	map[string]string	"Template not found"
//	@Failure		500				{object}	map[string]string	"Internal server error"
//	@Security		BearerAuth
//	@Router			/packtmpl/{templatename} [get]
func (h *TemplateHandlers) GetTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	templateName := vars["templatename"]
	userEmail := r.Header.Get("X-User")
	if userEmail == "" {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	err := HandleGetTemplate(w, r, userEmail, templateName, h.templateStorage)
	if err != nil {
		// Check if it's a "not found" error
		if strings.Contains(strings.ToLower(err.Error()), "key not found") {
			apiErr := apierrors.NewTemplateNotFoundError(templateName)
			apierrors.WriteErrorResponse(w, apiErr)
		} else {
			// Generic template processing error
			apiErr := apierrors.NewTemplateProcessingError(templateName, err)
			apierrors.WriteErrorResponse(w, apiErr)
		}
		return
	}
}

// CreateTemplate godoc
//
//	@Summary		Create package template
//	@Description	Create a new package template (templateName from YAML content)
//	@Tags			templates
//	@Accept			application/x-yaml
//	@Produce		json
//	@Param			template		body		string	true	"Template YAML content"
//	@Success		201				{object}	map[string]string	"Template created successfully"
//	@Failure		400				{object}	map[string]string	"Bad request"
//	@Failure		401				{object}	map[string]string	"Unauthorized"
//	@Failure		409				{object}	map[string]string	"Template already exists"
//	@Failure		500				{object}	map[string]string	"Internal server error"
//	@Security		BearerAuth
//	@Router			/packtmpl [post]
func (h *TemplateHandlers) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	userEmail := r.Header.Get("X-User")
	if userEmail == "" {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Read YAML content from request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse YAML to extract templateName
	var templateContent map[string]interface{}
	if err := yaml.Unmarshal(body, &templateContent); err != nil {
		http.Error(w, "Invalid YAML content", http.StatusBadRequest)
		return
	}

	templateName, ok := templateContent["templateName"].(string)
	if !ok || templateName == "" {
		http.Error(w, "templateName field required in YAML", http.StatusBadRequest)
		return
	}

	// Validate hfitVersion
	if version, ok := templateContent["hfitVersion"]; !ok {
		http.Error(w, "hfitVersion field required", http.StatusBadRequest)
		return
	} else if versionInt, ok := version.(int); !ok || versionInt != 1 {
		http.Error(w, "hfitVersion must be 1", http.StatusBadRequest)
		return
	}

	// Check if template already exists
	_, err = h.templateStorage.GetTemplate(r.Context(), userEmail, templateName)
	if err == nil {
		http.Error(w, "Template already exists", http.StatusConflict)
		return
	}

	// Store template as YAML string
	templateYAML := string(body)
	err = h.templateStorage.StoreTemplate(r.Context(), userEmail, templateName, templateYAML)
	if err != nil {
		http.Error(w, "Failed to create template", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Template created successfully",
		"name":    templateName,
	})
}

// UpdateTemplate godoc
//
//	@Summary		Update package template
//	@Description	Update an existing package template (templateName from YAML content)
//	@Tags			templates
//	@Accept			application/x-yaml
//	@Produce		json
//	@Param			template		body		string	true	"Template YAML content"
//	@Success		200				{object}	map[string]string	"Template updated successfully"
//	@Failure		400				{object}	map[string]string	"Bad request"
//	@Failure		401				{object}	map[string]string	"Unauthorized"
//	@Failure		404				{object}	map[string]string	"Template not found"
//	@Failure		500				{object}	map[string]string	"Internal server error"
//	@Security		BearerAuth
//	@Router			/packtmpl [put]
func (h *TemplateHandlers) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	userEmail := r.Header.Get("X-User")
	if userEmail == "" {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Read YAML content from request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse YAML to extract templateName
	var templateContent map[string]interface{}
	if err := yaml.Unmarshal(body, &templateContent); err != nil {
		http.Error(w, "Invalid YAML content", http.StatusBadRequest)
		return
	}

	templateName, ok := templateContent["templateName"].(string)
	if !ok || templateName == "" {
		http.Error(w, "templateName field required in YAML", http.StatusBadRequest)
		return
	}

	// Validate hfitVersion
	if version, ok := templateContent["hfitVersion"]; !ok {
		http.Error(w, "hfitVersion field required", http.StatusBadRequest)
		return
	} else if versionInt, ok := version.(int); !ok || versionInt != 1 {
		http.Error(w, "hfitVersion must be 1", http.StatusBadRequest)
		return
	}

	// Check if template exists
	_, err = h.templateStorage.GetTemplate(r.Context(), userEmail, templateName)
	if err != nil {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	// Update template
	templateYAML := string(body)
	err = h.templateStorage.StoreTemplate(r.Context(), userEmail, templateName, templateYAML)
	if err != nil {
		http.Error(w, "Failed to update template", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Template updated successfully",
		"name":    templateName,
	})
}

// PatchTemplate godoc
//
//	@Summary		Partially update package template with diff output
//	@Description	Partially update an existing package template and show diff between old and new versions
//	@Tags			templates
//	@Accept			application/x-yaml
//	@Produce		text/plain
//	@Param			template		body		string	true	"Partial template YAML content"
//	@Success		200				{string}	string	"Unified diff output showing changes"
//	@Failure		400				{object}	map[string]string	"Bad request"
//	@Failure		401				{object}	map[string]string	"Unauthorized"
//	@Failure		404				{object}	map[string]string	"Template not found"
//	@Failure		500				{object}	map[string]string	"Internal server error"
//	@Security		BearerAuth
//	@Router			/packtmpl [patch]
func (h *TemplateHandlers) PatchTemplate(w http.ResponseWriter, r *http.Request) {
	userEmail := r.Header.Get("X-User")
	if userEmail == "" {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Read YAML content from request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse partial YAML to extract templateName
	var patchContent map[string]interface{}
	if err := yaml.Unmarshal(body, &patchContent); err != nil {
		http.Error(w, "Invalid YAML content", http.StatusBadRequest)
		return
	}

	templateName, ok := patchContent["templateName"].(string)
	if !ok || templateName == "" {
		http.Error(w, "templateName field required in YAML", http.StatusBadRequest)
		return
	}

	// Get existing template
	existingYAML, err := h.templateStorage.GetTemplate(r.Context(), userEmail, templateName)
	if err != nil {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	// Parse existing template
	var existingContent map[string]interface{}
	if err := yaml.Unmarshal([]byte(existingYAML), &existingContent); err != nil {
		http.Error(w, "Failed to parse existing template", http.StatusInternalServerError)
		return
	}

	// Store original for diff
	originalYAML := existingYAML

	// Merge patch into existing template
	for key, value := range patchContent {
		existingContent[key] = value
	}

	// Convert back to YAML
	mergedYAML, err := yaml.Marshal(existingContent)
	if err != nil {
		http.Error(w, "Failed to marshal merged template", http.StatusInternalServerError)
		return
	}

	// Store updated template
	err = h.templateStorage.StoreTemplate(r.Context(), userEmail, templateName, string(mergedYAML))
	if err != nil {
		http.Error(w, "Failed to update template", http.StatusInternalServerError)
		return
	}

	// Generate unified diff output
	diff := generateUnifiedDiff(originalYAML, string(mergedYAML), templateName)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(diff))
}

// DeleteTemplate godoc
//
//	@Summary		Delete package template
//	@Description	Delete a specific package template
//	@Tags			templates
//	@Accept			json
//	@Produce		json
//	@Param			templatename	path		string	true	"Template name"	example(webapp-starter)
//	@Success		200				{object}	map[string]string	"Success message"
//	@Failure		401				{object}	map[string]string	"Unauthorized"
//	@Failure		404				{object}	map[string]string	"Template not found"
//	@Failure		500				{object}	map[string]string	"Internal server error"
//	@Security		BearerAuth
//	@Router			/packtmpl/{templatename} [delete]
func (h *TemplateHandlers) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	templateName := vars["templatename"]
	userEmail := r.Header.Get("X-User")
	if userEmail == "" {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	err := HandleDeleteTemplate(w, r, userEmail, templateName, h.templateStorage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// BulkUploadTemplates godoc
//
//	@Summary		Bulk upload package templates
//	@Description	Upload multiple package templates in a single request
//	@Tags			templates
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file	formData	file	true	"Templates archive file"
//	@Success		200		{object}	map[string]string	"Success message"
//	@Failure		400		{object}	map[string]string	"Bad request"
//	@Failure		401		{object}	map[string]string	"Unauthorized"
//	@Failure		500		{object}	map[string]string	"Internal server error"
//	@Security		BearerAuth
//	@Router			/packtmplpackupld [post]
func (h *TemplateHandlers) BulkUploadTemplates(w http.ResponseWriter, r *http.Request) {
	userEmail := r.Header.Get("X-User")
	if userEmail == "" {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	err := HandleBulkUploadTemplates(w, r, userEmail, h.templateStorage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// DownloadTemplatePackage godoc
//
//	@Summary		Download package template
//	@Description	Download a rendered package template with parameters
//	@Tags			templates
//	@Accept			json
//	@Produce		application/gzip
//	@Param			templatename	path		string	true	"Template name"	example(webapp-starter)
//	@Param			parameters		body		object	true	"Template parameters"
//	@Success		200				{file}		binary	"Template package file"
//	@Failure		400				{object}	map[string]string	"Bad request"
//	@Failure		401				{object}	map[string]string	"Unauthorized"
//	@Failure		404				{object}	map[string]string	"Template not found"
//	@Failure		500				{object}	map[string]string	"Internal server error"
//	@Security		BearerAuth
//	@Router			/packdownload/{templatename} [post]
func (h *TemplateHandlers) DownloadTemplatePackage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	templateName := vars["templatename"]
	userEmail := r.Header.Get("X-User")
	if userEmail == "" {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	err := HandleTemplatePackageDownload(w, r, userEmail, templateName, h.config, h.databaseManager, h.redisClient, h.templateStorage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// SetupTemplateRoutes sets up all template-related routes on the provided router
func SetupTemplateRoutes(router *mux.Router, cfg *config.Config, databaseManager *db.DatabaseManager, redisClient *redisclient.Client) {
	handlers := NewTemplateHandlers(cfg, databaseManager, redisClient)

	// Template routes
	router.HandleFunc("/packtmpl", handlers.GetTemplates).Methods("GET")
	router.HandleFunc("/packtmpl", handlers.CreateTemplate).Methods("POST") // Create (templateName from YAML)
	router.HandleFunc("/packtmpl", handlers.UpdateTemplate).Methods("PUT")  // Update (templateName from YAML)
	router.HandleFunc("/packtmpl", handlers.PatchTemplate).Methods("PATCH") // Partial update (templateName from YAML)
	router.HandleFunc("/packtmpl/{templatename}", handlers.GetTemplate).Methods("GET")
	router.HandleFunc("/packtmpl/{templatename}", handlers.DeleteTemplate).Methods("DELETE")
	router.HandleFunc("/packtmplpackupld", handlers.BulkUploadTemplates).Methods("POST")
	router.HandleFunc("/packdownload/{templatename}", handlers.DownloadTemplatePackage).Methods("POST")
}

/*
* HandleListTemplates lists all templates for a user
 */
func HandleListTemplates(w http.ResponseWriter, r *http.Request, userEmail string, templateStorage *tmplstorage.TemplateStorage) error {
	templates, err := templateStorage.ListTemplates(r.Context(), userEmail)
	if err != nil {
		return fmt.Errorf("failed to list templates: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(templates)
}

// HandleGetTemplate retrieves a specific package template
func HandleGetTemplate(w http.ResponseWriter, r *http.Request, userEmail, templateName string, templateStorage *tmplstorage.TemplateStorage) error {
	yamlContent, err := templateStorage.GetTemplate(r.Context(), userEmail, templateName)
	if err != nil {
		return fmt.Errorf("failed to get template: %v", err)
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Write([]byte(yamlContent))
	return nil
}

// HandleSetTemplate creates or updates a package template
func HandleSetTemplate(w http.ResponseWriter, r *http.Request, userEmail, templateName string, templateStorage *tmplstorage.TemplateStorage) error {
	// Read YAML content from request body
	yamlContent, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %v", err)
	}

	// Validate YAML by attempting to parse it
	var template pkgtmpl.PackageTemplate
	if err := yaml.Unmarshal(yamlContent, &template); err != nil {
		return fmt.Errorf("invalid YAML content: %v", err)
	}

	// Store the template
	if err := templateStorage.StoreTemplate(r.Context(), userEmail, templateName, string(yamlContent)); err != nil {
		return fmt.Errorf("failed to store template: %v", err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Template stored successfully"))
	return nil
}

// HandleDeleteTemplate deletes a package template
func HandleDeleteTemplate(w http.ResponseWriter, r *http.Request, userEmail, templateName string, templateStorage *tmplstorage.TemplateStorage) error {
	// Check if template exists
	exists, err := templateStorage.TemplateExists(r.Context(), userEmail, templateName)
	if err != nil {
		return fmt.Errorf("failed to check template existence: %v", err)
	}

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Template not found"))
		return nil
	}

	// Delete the template
	if err := templateStorage.DeleteTemplate(r.Context(), userEmail, templateName); err != nil {
		return fmt.Errorf("failed to delete template: %v", err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Template deleted successfully"))
	return nil
}

// HandleBulkUploadTemplates handles bulk upload of package templates from tar.gz
func HandleBulkUploadTemplates(w http.ResponseWriter, r *http.Request, userEmail string, templateStorage *tmplstorage.TemplateStorage) error {
	// Parse multipart form
	err := r.ParseMultipartForm(32 << 20) // 32MB max memory
	if err != nil {
		return fmt.Errorf("failed to parse multipart form: %v", err)
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		return fmt.Errorf("failed to get uploaded file: %v", err)
	}
	defer file.Close()

	// Create gzip reader
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzipReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzipReader)

	var report []map[string]interface{}

	// Process each file in the archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %v", err)
		}

		// Skip directories and non-yaml files
		if header.Typeflag != tar.TypeReg || !strings.HasSuffix(header.Name, ".yaml") {
			continue
		}

		// Extract template name from filename
		templateName := strings.TrimSuffix(filepath.Base(header.Name), ".yaml")

		// Read file content
		content, err := io.ReadAll(tarReader)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %v", header.Name, err)
		}

		// Validate YAML
		var template pkgtmpl.PackageTemplate
		if err := yaml.Unmarshal(content, &template); err != nil {
			report = append(report, map[string]interface{}{
				"template": templateName,
				"status":   "error",
				"message":  fmt.Sprintf("Invalid YAML: %v", err),
			})
			continue
		}

		// Check if template exists
		exists, err := templateStorage.TemplateExists(r.Context(), userEmail, templateName)
		if err != nil {
			report = append(report, map[string]interface{}{
				"template": templateName,
				"status":   "error",
				"message":  fmt.Sprintf("Failed to check existence: %v", err),
			})
			continue
		}

		// Store the template
		if err := templateStorage.StoreTemplate(r.Context(), userEmail, templateName, string(content)); err != nil {
			report = append(report, map[string]interface{}{
				"template": templateName,
				"status":   "error",
				"message":  fmt.Sprintf("Failed to store: %v", err),
			})
			continue
		}

		action := "created"
		if exists {
			action = "updated"
		}

		report = append(report, map[string]interface{}{
			"template": templateName,
			"status":   "success",
			"action":   action,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(map[string]interface{}{
		"templates_processed": len(report),
		"report":              report,
	})
}

// HandleTemplatePackageDownload processes template-based package downloads with parameters
func HandleTemplatePackageDownload(w http.ResponseWriter, r *http.Request, userEmail, templateName string, cfg *config.Config, databaseManager *db.DatabaseManager, redisClient *redisclient.Client, templateStorage *tmplstorage.TemplateStorage) error {
	// Parse JSON parameters from request body
	var params []string
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		return fmt.Errorf("failed to parse parameters: %v", err)
	}

	// Get the template
	yamlContent, err := templateStorage.GetTemplate(r.Context(), userEmail, templateName)
	if err != nil {
		return fmt.Errorf("template not found: %v", err)
	}

	// Parse the template
	var template pkgtmpl.PackageTemplate
	if err := yaml.Unmarshal([]byte(yamlContent), &template); err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	// Create package template processor
	processor := pkgtmpl.NewPackageTemplateProcessor(cfg, databaseManager)

	// Process the template with parameters
	timestamp := time.Now().Unix()
	packagePath, metadata, err := processor.ProcessTemplate(r.Context(), &template, params, userEmail, templateName, timestamp)
	if err != nil {
		return fmt.Errorf("failed to process template: %v", err)
	}

	// Log the download
	if err := templateStorage.LogDownload(r.Context(), userEmail, templateName, timestamp); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: failed to log download: %v\n", err)
	}

	// Store download metadata
	if err := templateStorage.StoreDownloadMetadata(r.Context(), userEmail, templateName, timestamp, *metadata); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: failed to store download metadata: %v\n", err)
	}

	// Open the generated package file
	packageFile, err := os.Open(packagePath)
	if err != nil {
		return fmt.Errorf("failed to open package file: %v", err)
	}
	defer packageFile.Close()

	// Get file info for Content-Length header
	fileInfo, err := packageFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}

	// Set response headers
	filename := fmt.Sprintf("pkg-%s-%d.tar.gz", templateName, timestamp)
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// Stream the file to the client
	_, err = io.Copy(w, packageFile)
	return err
}

// generateUnifiedDiff creates a unified diff output similar to 'diff -u'
func generateUnifiedDiff(original, updated, templateName string) string {
	now := time.Now().Format("2006-01-02 15:04:05.000000000 -0700")

	// Split into lines for comparison
	originalLines := strings.Split(original, "\n")
	updatedLines := strings.Split(updated, "\n")

	var result strings.Builder

	// Header
	result.WriteString(fmt.Sprintf("--- %s.original\t%s\n", templateName, now))
	result.WriteString(fmt.Sprintf("+++ %s.updated\t%s\n", templateName, now))

	// Simple line-by-line comparison (basic implementation)
	maxLines := len(originalLines)
	if len(updatedLines) > maxLines {
		maxLines = len(updatedLines)
	}

	// Find differences
	var chunks []string
	lineNum := 1

	for i := 0; i < maxLines; i++ {
		originalLine := ""
		updatedLine := ""

		if i < len(originalLines) {
			originalLine = originalLines[i]
		}
		if i < len(updatedLines) {
			updatedLine = updatedLines[i]
		}

		if originalLine != updatedLine {
			if originalLine != "" {
				chunks = append(chunks, fmt.Sprintf("-%s", originalLine))
			}
			if updatedLine != "" {
				chunks = append(chunks, fmt.Sprintf("+%s", updatedLine))
			}
		}
		lineNum++
	}

	if len(chunks) > 0 {
		result.WriteString(fmt.Sprintf("@@ -1,%d +1,%d @@\n", len(originalLines), len(updatedLines)))
		for _, chunk := range chunks {
			result.WriteString(chunk + "\n")
		}
	} else {
		result.WriteString("No changes detected\n")
	}

	return result.String()
}
