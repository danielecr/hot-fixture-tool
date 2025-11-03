/*
 * Hot Fixture Tool Daemon - File API Streaming
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
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

/*
* streamFiles streams files in a volume with optional filtering using NDJSON format.
* This provides performance similar to Unix find command for large directories.
 */
func StreamFiles(w http.ResponseWriter, volumePath string, query url.Values) error {
	// Ensure the response is flushed immediately for streaming
	flusher, canFlush := w.(http.Flusher)

	// Validate volume path exists and is accessible
	if _, err := os.Stat(volumePath); os.IsNotExist(err) {
		return fmt.Errorf("volume path does not exist: %s", volumePath)
	}

	// Use filepath.Walk which is optimized like Unix find
	return filepath.Walk(volumePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Log error but continue processing other files (like Unix find does)
			// Don't return error unless it's a critical filesystem issue
			if os.IsPermission(err) {
				// Permission denied - skip this path and continue
				return nil
			}
			return nil
		}

		// Skip directories for now, only return files
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(volumePath, path)
		if err != nil {
			return nil // Skip this file and continue
		}

		fileInfo := map[string]interface{}{
			"name":    info.Name(),
			"path":    relPath,
			"size":    info.Size(),
			"modtime": info.ModTime().Unix(),
			"isdir":   info.IsDir(),
		}

		// Apply filters from query parameters
		if PassesFilters(fileInfo, query) {
			// Write each file as a separate JSON line (NDJSON)
			jsonData, err := json.Marshal(fileInfo)
			if err != nil {
				return nil // Skip this file and continue
			}

			// Write the JSON object followed by newline
			if _, err := w.Write(jsonData); err != nil {
				return err // Stop on write error (client disconnected)
			}
			if _, err := w.Write([]byte("\n")); err != nil {
				return err // Stop on write error
			}

			// Flush after every file for real-time streaming
			if canFlush {
				flusher.Flush()
			}
		}

		return nil
	})
}

/*
* findBestFile finds the best matching file based on filters and sorting criteria in O(n) time.
* This streams through files, applies filters, and keeps track of the best candidate.
 */
func FindBestFile(volumePath string, query url.Values) (map[string]interface{}, error) {
	filters := query["filter[]"]
	var sortCriteria string
	var sortOrder string = "asc" // default ascending

	// Extract sort criteria from filters
	for _, filter := range filters {
		parts := strings.SplitN(filter, ":", 2)
		if len(parts) == 2 && parts[0] == "sort" {
			sortSpec := parts[1]
			// Handle sort direction (e.g., "mtime:desc", "size:asc", "name")
			if strings.Contains(sortSpec, ":") {
				sortParts := strings.SplitN(sortSpec, ":", 2)
				sortCriteria = sortParts[0]
				sortOrder = sortParts[1]
			} else {
				sortCriteria = sortSpec
			}
			break
		}
	}

	var bestFile map[string]interface{}

	// Stream through files and find the best one in a single pass
	err := filepath.Walk(volumePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip errors and continue (like Unix find)
			if os.IsPermission(err) {
				return nil
			}
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(volumePath, path)
		if err != nil {
			return nil // Skip this file and continue
		}

		fileInfo := map[string]interface{}{
			"name":    info.Name(),
			"path":    relPath,
			"size":    info.Size(),
			"modtime": info.ModTime().Unix(),
			"isdir":   info.IsDir(),
		}

		// Apply filters - skip if doesn't match
		if !PassesFilters(fileInfo, query) {
			return nil
		}

		// If no best file yet, this becomes the best
		if bestFile == nil {
			bestFile = fileInfo
			return nil
		}

		// Compare with current best based on sort criteria
		if IsBetterFile(fileInfo, bestFile, sortCriteria, sortOrder) {
			bestFile = fileInfo
		}

		return nil
	})

	return bestFile, err
}
