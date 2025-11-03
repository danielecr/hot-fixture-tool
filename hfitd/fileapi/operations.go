/*
 * Hot Fixture Tool Daemon - File API Operations
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

package fileapi

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

/*
* downloadFile handles file download requests.
 */
func DownloadFile(w http.ResponseWriter, r *http.Request, volumePath string) error {
	query := r.URL.Query()
	filePath := query.Get("path")
	folder := query.Get("folder")

	if filePath != "" {
		// Direct file download
		fullPath := filepath.Join(volumePath, filePath)

		// Security check - ensure path is within volume
		if !strings.HasPrefix(fullPath, volumePath) {
			return fmt.Errorf("invalid file path")
		}

		file, err := os.Open(fullPath)
		if err != nil {
			return err
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			return err
		}

		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(filePath)))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))

		_, err = io.Copy(w, file)
		return err
	} else if folder != "" {
		// Find the best matching file in folder with filters and sorting in O(n) time
		bestFile, err := FindBestFile(filepath.Join(volumePath, folder), r.URL.Query())
		if err != nil {
			return err
		}

		if bestFile == nil {
			return fmt.Errorf("no matching files found")
		}

		// Download the best matching file
		return DownloadFile(w, &http.Request{
			URL: &url.URL{
				RawQuery: fmt.Sprintf("path=%s", bestFile["path"]),
			},
		}, volumePath)
	}

	return fmt.Errorf("either path or folder parameter is required")
}

/*
* checkFileExists checks if a file exists in a volume.
 */
func CheckFileExists(volumePath string, filePath string) (bool, error) {
	fullPath := filepath.Join(volumePath, filePath)

	// Security check - ensure path is within volume
	if !strings.HasPrefix(fullPath, volumePath) {
		return false, fmt.Errorf("invalid file path")
	}

	_, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}
