/*
 * Hot Fixture Tool Daemon - File API Handlers
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

package fileapi

import (
	"encoding/json"
	"net/http"

	"hfitd/config"

	"github.com/gorilla/mux"
)

// FileHandlers contains the configuration needed for file operations
type FileHandlers struct {
	config *config.Config
}

// NewFileHandlers creates a new FileHandlers instance
func NewFileHandlers(cfg *config.Config) *FileHandlers {
	return &FileHandlers{
		config: cfg,
	}
}

// GetVolumes godoc
//
//	@Summary		List volumes
//	@Description	Get list of all configured volumes
//	@Tags			files
//	@Accept			json
//	@Produce		json
//	@Success		200		{array}		string	"List of volume names"
//	@Failure		401		{object}	map[string]string	"Unauthorized"
//	@Failure		500		{object}	map[string]string	"Internal server error"
//	@Security		BearerAuth
//	@Router			/volumes [get]
func (h *FileHandlers) GetVolumes(w http.ResponseWriter, r *http.Request) {
	var volumes []string
	for volumeName := range h.config.Volumes {
		volumes = append(volumes, volumeName)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(volumes)
}

// CheckVolumeExists godoc
//
//	@Summary		Check volume exists
//	@Description	Check if a specific volume is configured
//	@Tags			files
//	@Param			volume	path	string	true	"Volume name"	example(data)
//	@Success		200
//	@Failure		404
//	@Security		BearerAuth
//	@Router			/volumes/{volume} [head]
func (h *FileHandlers) CheckVolumeExists(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	volume := vars["volume"]

	if _, exists := h.config.Volumes[volume]; !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// ListFiles godoc
//
//	@Summary		List files in volume
//	@Description	Get list of all files in a specific volume with optional filtering
//	@Tags			files
//	@Accept			json
//	@Produce		application/x-json-stream
//	@Param			volume		path		string	true	"Volume name"	example(data)
//	@Success		200			{array}		object	"Stream of file information objects"
//	@Failure		404			{object}	map[string]string	"Volume not found"
//	@Failure		401			{object}	map[string]string	"Unauthorized"
//	@Failure		500			{object}	map[string]string	"Internal server error"
//	@Security		BearerAuth
//	@Router			/files/{volume}/list [get]
func (h *FileHandlers) ListFiles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	volume := vars["volume"]

	volumeConfig, exists := h.config.Volumes[volume]
	if !exists {
		http.Error(w, "Volume not found", http.StatusNotFound)
		return
	}

	// Set streaming headers for NDJSON
	w.Header().Set("Content-Type", "application/x-json-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Use streaming file listing for performance
	err := StreamFiles(w, volumeConfig.Path, r.URL.Query())
	if err != nil {
		// Error handling: if we haven't written anything yet, send HTTP error
		// If we've started streaming, we can only log the error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// DownloadFile godoc
//
//	@Summary		Download file
//	@Description	Download a specific file from a volume
//	@Tags			files
//	@Accept			json
//	@Produce		application/octet-stream
//	@Param			volume		path		string	true	"Volume name"	example(data)
//	@Param			filepath	query		string	true	"File path"		example(documents/report.pdf)
//	@Success		200			{file}		binary	"File content"
//	@Failure		404			{object}	map[string]string	"Volume or file not found"
//	@Failure		401			{object}	map[string]string	"Unauthorized"
//	@Failure		500			{object}	map[string]string	"Internal server error"
//	@Security		BearerAuth
//	@Router			/files/{volume}/download [get]
func (h *FileHandlers) DownloadFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	volume := vars["volume"]

	volumeConfig, exists := h.config.Volumes[volume]
	if !exists {
		http.Error(w, "Volume not found", http.StatusNotFound)
		return
	}

	err := DownloadFile(w, r, volumeConfig.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// CheckFileExists godoc
//
//	@Summary		Check file exists
//	@Description	Check if a specific file exists in a volume
//	@Tags			files
//	@Param			volume		path	string	true	"Volume name"	example(data)
//	@Param			filepath	path	string	true	"File path"		example(documents/report.pdf)
//	@Success		200
//	@Failure		404
//	@Security		BearerAuth
//	@Router			/files/{volume}/{filepath} [head]
func (h *FileHandlers) CheckFileExists(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	volume := vars["volume"]
	filepath := vars["filepath"]

	volumeConfig, exists := h.config.Volumes[volume]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	exists, err := CheckFileExists(volumeConfig.Path, filepath)
	if err != nil || !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// SetupFileRoutes sets up all file and volume-related routes on the provided router
func SetupFileRoutes(router *mux.Router, cfg *config.Config) {
	handlers := NewFileHandlers(cfg)

	// Volume routes
	router.HandleFunc("/volumes", handlers.GetVolumes).Methods("GET")
	router.HandleFunc("/volumes/{volume}", handlers.CheckVolumeExists).Methods("HEAD")

	// File routes
	router.HandleFunc("/files/{volume}/list", handlers.ListFiles).Methods("GET")
	router.HandleFunc("/files/{volume}/download", handlers.DownloadFile).Methods("GET")
	router.HandleFunc("/files/{volume}/{filepath:.*}", handlers.CheckFileExists).Methods("HEAD")
}
