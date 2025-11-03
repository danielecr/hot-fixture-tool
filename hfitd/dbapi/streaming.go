/*
 * Hot Fixture Tool Daemon - Database API Streaming
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

package dbapi

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

/*
* streamTableRows streams database table rows as NDJSON format with optional filtering.
* Uses O(1) memory by processing one row at a time.
 */
func StreamTableRows(w http.ResponseWriter, conn *sql.DB, dbms string, dbid string, tableid string, filterpart string) error {
	// Build the base query
	baseQuery := fmt.Sprintf("SELECT * FROM %s", tableid)

	// Add filter part if provided and validated
	var finalQuery string
	if filterpart != "" {
		validatedFilter, err := ValidateFilterPart(filterpart)
		if err != nil {
			return fmt.Errorf("invalid filterpart: %v", err)
		}
		finalQuery = fmt.Sprintf("%s %s", baseQuery, validatedFilter)
	} else {
		// Default limit to prevent accidental massive queries
		finalQuery = fmt.Sprintf("%s LIMIT 1000", baseQuery)
	}

	rows, err := conn.Query(finalQuery)
	if err != nil {
		return fmt.Errorf("query execution failed: %v", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %v", err)
	}

	// Set response headers for NDJSON streaming
	w.Header().Set("Content-Type", "application/x-json-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Create a flusher for real-time streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}

	encoder := json.NewEncoder(w)

	// Stream each row as a separate JSON object
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			// Log error but continue with next row
			fmt.Fprintf(w, `{"error":"row scan failed: %v"}`+"\n", err)
			flusher.Flush()
			continue
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}

		// Encode and stream this row
		if err := encoder.Encode(row); err != nil {
			return fmt.Errorf("failed to encode row: %v", err)
		}

		// Flush immediately for real-time streaming
		flusher.Flush()
	}

	// Check for any errors from iterating over rows
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error during row iteration: %v", err)
	}

	return nil
}
