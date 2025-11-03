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

// SetupFileRoutes sets up all file and volume-related routes on the provided router
func SetupFileRoutes(router *mux.Router, cfg *config.Config) {
	// Volume routes
	router.HandleFunc("/volumes", func(w http.ResponseWriter, r *http.Request) {
		var volumes []string
		for volumeName := range cfg.Volumes {
			volumes = append(volumes, volumeName)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(volumes)
	}).Methods("GET")

	router.HandleFunc("/volumes/{volume}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		volume := vars["volume"]

		if _, exists := cfg.Volumes[volume]; !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}).Methods("HEAD")

	// File routes
	router.HandleFunc("/files/{volume}/list", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		volume := vars["volume"]

		volumeConfig, exists := cfg.Volumes[volume]
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
	}).Methods("GET")

	router.HandleFunc("/files/{volume}/download", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		volume := vars["volume"]

		volumeConfig, exists := cfg.Volumes[volume]
		if !exists {
			http.Error(w, "Volume not found", http.StatusNotFound)
			return
		}

		err := DownloadFile(w, r, volumeConfig.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}).Methods("GET")

	router.HandleFunc("/files/{volume}/{filepath:.*}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		volume := vars["volume"]
		filepath := vars["filepath"]

		volumeConfig, exists := cfg.Volumes[volume]
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
	}).Methods("HEAD")
}
