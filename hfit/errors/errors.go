/*
 * Hot Fixture Tool CLI - Error Handling
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

// Package errors provides error handling for hfit CLI commands
package errors

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ServerErrorResponse represents the structured error response from hfitd server
type ServerErrorResponse struct {
	Category    string   `json:"category"`
	Code        string   `json:"code"`
	Message     string   `json:"message"`
	Details     string   `json:"details,omitempty"`
	Resource    string   `json:"resource,omitempty"`
	Suggestions []string `json:"suggestions,omitempty"`
	Timestamp   string   `json:"timestamp"`
}

// ParseServerError parses a server error response and returns a user-friendly message
func ParseServerError(errorBody string) string {
	// Try to parse as JSON first
	var serverErr ServerErrorResponse
	if err := json.Unmarshal([]byte(errorBody), &serverErr); err != nil {
		// If parsing fails, return the original error
		return errorBody
	}

	// Build a user-friendly error message
	var builder strings.Builder

	// Add the main message with a prefix based on category
	icon := getErrorIcon(serverErr.Category)
	builder.WriteString(fmt.Sprintf("%s %s\n", icon, serverErr.Message))

	// Add resource context if available
	if serverErr.Resource != "" {
		builder.WriteString(fmt.Sprintf("   Resource: %s\n", serverErr.Resource))
	}

	// Add suggestions if available
	if len(serverErr.Suggestions) > 0 {
		builder.WriteString("\nSuggestions:\n")
		for _, suggestion := range serverErr.Suggestions {
			builder.WriteString(fmt.Sprintf("   - %s\n", suggestion))
		}
	}

	// Add technical details if available (server should not send sensitive details)
	if serverErr.Details != "" {
		builder.WriteString(fmt.Sprintf("\nTechnical details: %s\n", serverErr.Details))
	}

	return strings.TrimSpace(builder.String())
}

// getErrorIcon returns an appropriate prefix for the error category
func getErrorIcon(category string) string {
	switch strings.ToUpper(category) {
	case "CONNECTION":
		return "[CONNECTION]"
	case "NOT_FOUND":
		return "[NOT_FOUND]"
	case "AUTHENTICATION", "AUTH":
		return "[AUTH]"
	case "PERMISSION_DENIED":
		return "[PERMISSION]"
	case "VALIDATION_ERROR", "FORMAT_ERROR", "BAD_REQUEST":
		return "[WARNING]"
	case "INTERNAL_ERROR", "DATABASE", "VOLUME":
		return "[ERROR]"
	case "TIMEOUT":
		return "[TIMEOUT]"
	case "SERVICE_UNAVAILABLE":
		return "[UNAVAILABLE]"
	default:
		return "[ERROR]"
	}
}

// FormatAPIError takes an API error response and formats it for user display
func FormatAPIError(statusCode int, errorBody string) string {
	// Check if it's a structured server error
	if strings.Contains(errorBody, `"category"`) && strings.Contains(errorBody, `"message"`) {
		return ParseServerError(errorBody)
	}

	// Fallback for non-structured errors
	switch statusCode {
	case 401:
		return "[AUTH] Authentication required. Please run 'hfit login' to authenticate."
	case 403:
		return "[PERMISSION] Permission denied. You don't have access to this resource."
	case 404:
		return "[NOT_FOUND] Resource not found. Please check the resource name and try again."
	case 429:
		return "[RATE_LIMIT] Rate limit exceeded. Please wait a moment and try again."
	case 500:
		return fmt.Sprintf("[ERROR] Server error occurred.\nTechnical details: %s", errorBody)
	case 502, 503:
		return "[UNAVAILABLE] Service temporarily unavailable. Please try again later."
	default:
		return fmt.Sprintf("[ERROR] Request failed with status %d.\nDetails: %s", statusCode, errorBody)
	}
}
