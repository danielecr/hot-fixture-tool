/*
 * Hot Fixture Tool Daemon - File API Filtering
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

package fileapi

import (
	"net/url"
	"strconv"
	"strings"
	"time"
)

/*
* passesFilters checks if a file passes the filter criteria.
* Supports name wildcards, time filtering, and sorting hints.
 */
func PassesFilters(fileInfo map[string]interface{}, query url.Values) bool {
	filters := query["filter[]"]

	for _, filter := range filters {
		parts := strings.SplitN(filter, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		switch key {
		case "name":
			// Enhanced wildcard matching with glob-like patterns
			fileName := fileInfo["name"].(string)
			if !MatchesPattern(fileName, value) {
				return false
			}
		case "mtime":
			// Time-based filtering (days ago)
			if days, err := strconv.Atoi(value); err == nil {
				fileTime := time.Unix(fileInfo["modtime"].(int64), 0)
				if days < 0 {
					// Negative days means "modified before X days ago"
					cutoff := time.Now().AddDate(0, 0, days)
					if fileTime.After(cutoff) {
						return false
					}
				} else {
					// Positive days means "modified after X days ago"
					cutoff := time.Now().AddDate(0, 0, -days)
					if fileTime.Before(cutoff) {
						return false
					}
				}
			}
		case "size":
			// Size filtering (bytes)
			fileSize := fileInfo["size"].(int64)
			if strings.HasPrefix(value, ">") {
				if size, err := strconv.ParseInt(value[1:], 10, 64); err == nil {
					if fileSize <= size {
						return false
					}
				}
			} else if strings.HasPrefix(value, "<") {
				if size, err := strconv.ParseInt(value[1:], 10, 64); err == nil {
					if fileSize >= size {
						return false
					}
				}
			} else {
				if size, err := strconv.ParseInt(value, 10, 64); err == nil {
					if fileSize != size {
						return false
					}
				}
			}
		case "sort":
			// Sort is handled at a higher level, just pass through
			continue
		}
	}

	return true
}

/*
* matchesPattern performs glob-like pattern matching for file names.
 */
func MatchesPattern(fileName, pattern string) bool {
	// Simple glob pattern matching
	// * matches any sequence of characters
	// ? matches any single character

	if pattern == "*" {
		return true
	}

	// Convert glob pattern to regex-like matching
	if strings.Contains(pattern, "*") {
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			// Simple case: prefix*suffix
			prefix, suffix := parts[0], parts[1]
			return strings.HasPrefix(fileName, prefix) && strings.HasSuffix(fileName, suffix)
		}
		// For more complex patterns, fall back to simple contains check
		cleanPattern := strings.ReplaceAll(pattern, "*", "")
		return strings.Contains(fileName, cleanPattern)
	}

	if strings.Contains(pattern, "?") {
		// For now, just treat ? as any character - could be enhanced with regex
		cleanPattern := strings.ReplaceAll(pattern, "?", "")
		return strings.Contains(fileName, cleanPattern)
	}

	// Exact match or contains
	return strings.Contains(fileName, pattern)
}

/*
* isBetterFile compares two files based on sorting criteria to determine which is "better".
* Returns true if newFile is better than currentBest.
 */
func IsBetterFile(newFile, currentBest map[string]interface{}, sortCriteria, sortOrder string) bool {
	switch sortCriteria {
	case "mtime", "modtime":
		newTime := newFile["modtime"].(int64)
		bestTime := currentBest["modtime"].(int64)
		if sortOrder == "desc" {
			return newTime > bestTime // newer is better
		}
		return newTime < bestTime // older is better

	case "size":
		newSize := newFile["size"].(int64)
		bestSize := currentBest["size"].(int64)
		if sortOrder == "desc" {
			return newSize > bestSize // larger is better
		}
		return newSize < bestSize // smaller is better

	case "name":
		newName := newFile["name"].(string)
		bestName := currentBest["name"].(string)
		if sortOrder == "desc" {
			return newName > bestName // Z-A order
		}
		return newName < bestName // A-Z order

	case "path":
		newPath := newFile["path"].(string)
		bestPath := currentBest["path"].(string)
		if sortOrder == "desc" {
			return newPath > bestPath
		}
		return newPath < bestPath

	default:
		// No sorting criteria, keep the first one found
		return false
	}
}
