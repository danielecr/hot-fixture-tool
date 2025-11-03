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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// CreateTemplate godoc
//
//	@Summary		Create/Update package template
//	@Description	Create or update a package template
//	@Tags			templates
//	@Accept			json
//	@Produce		json
//	@Param			templatename	path		string	true	"Template name"	example(webapp-starter)
//	@Param			template		body		object	true	"Template data"
//	@Success		200				{object}	map[string]string	"Success message"
//	@Failure		400				{object}	map[string]string	"Bad request"
//	@Failure		401				{object}	map[string]string	"Unauthorized"
//	@Failure		500				{object}	map[string]string	"Internal server error"
//	@Security		BearerAuth
//	@Router			/packtmpl/{templatename} [post]
func (h *TemplateHandlers) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	templateName := vars["templatename"]
	userEmail := r.Header.Get("X-User")
	if userEmail == "" {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	err := HandleSetTemplate(w, r, userEmail, templateName, h.templateStorage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
	router.HandleFunc("/packtmpl/{templatename}", handlers.GetTemplate).Methods("GET")
	router.HandleFunc("/packtmpl/{templatename}", handlers.CreateTemplate).Methods("POST")
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
