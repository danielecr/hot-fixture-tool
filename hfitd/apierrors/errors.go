/*
 * Hot Fixture Tool Daemon - Error Handling
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

// Package apierrors provides structured error handling for hfitd API responses
package apierrors

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// ErrorCategory represents different categories of errors
type ErrorCategory string

const (
	// Infrastructure errors
	CategoryConnection ErrorCategory = "CONNECTION"
	CategoryDatabase   ErrorCategory = "DATABASE"
	CategoryVolume     ErrorCategory = "VOLUME"
	CategoryAuth       ErrorCategory = "AUTHENTICATION"

	// Resource errors
	CategoryNotFound   ErrorCategory = "NOT_FOUND"
	CategoryExists     ErrorCategory = "ALREADY_EXISTS"
	CategoryPermission ErrorCategory = "PERMISSION_DENIED"

	// Request errors
	CategoryValidation ErrorCategory = "VALIDATION_ERROR"
	CategoryFormat     ErrorCategory = "FORMAT_ERROR"
	CategoryBadRequest ErrorCategory = "BAD_REQUEST"

	// Server errors
	CategoryInternal    ErrorCategory = "INTERNAL_ERROR"
	CategoryTimeout     ErrorCategory = "TIMEOUT"
	CategoryUnavailable ErrorCategory = "SERVICE_UNAVAILABLE"
)

// ErrorCode represents specific error codes for different scenarios
type ErrorCode string

const (
	// Database errors
	CodeDBConnectionRefused ErrorCode = "DB_CONNECTION_REFUSED"
	CodeDBProviderNotFound  ErrorCode = "DB_PROVIDER_NOT_FOUND"
	CodeDBNotFound          ErrorCode = "DATABASE_NOT_FOUND"
	CodeTableNotFound       ErrorCode = "TABLE_NOT_FOUND"
	CodeDBPermissionDenied  ErrorCode = "DB_PERMISSION_DENIED"

	// Volume/File errors
	CodeVolumeNotFound       ErrorCode = "VOLUME_NOT_FOUND"
	CodeVolumeNotConfigured  ErrorCode = "VOLUME_NOT_CONFIGURED"
	CodeFileNotFound         ErrorCode = "FILE_NOT_FOUND"
	CodeFilePermissionDenied ErrorCode = "FILE_PERMISSION_DENIED"

	// Template errors
	CodeTemplateNotFound   ErrorCode = "TEMPLATE_NOT_FOUND"
	CodeTemplateExists     ErrorCode = "TEMPLATE_ALREADY_EXISTS"
	CodeTemplateInvalid    ErrorCode = "TEMPLATE_INVALID"
	CodeTemplateProcessing ErrorCode = "TEMPLATE_PROCESSING_ERROR"

	// Authentication errors
	CodeAuthTokenInvalid     ErrorCode = "AUTH_TOKEN_INVALID"
	CodeAuthTokenExpired     ErrorCode = "AUTH_TOKEN_EXPIRED"
	CodeAuthUserNotFound     ErrorCode = "AUTH_USER_NOT_FOUND"
	CodeAuthSignatureInvalid ErrorCode = "AUTH_SIGNATURE_INVALID"

	// Request errors
	CodeRequestInvalid   ErrorCode = "REQUEST_INVALID"
	CodeParameterMissing ErrorCode = "PARAMETER_MISSING"
	CodeParameterInvalid ErrorCode = "PARAMETER_INVALID"
	CodeFormatInvalid    ErrorCode = "FORMAT_INVALID"
)

// APIError represents a structured API error response
type APIError struct {
	Category    ErrorCategory `json:"category"`
	Code        ErrorCode     `json:"code"`
	Message     string        `json:"message"`
	Details     string        `json:"details,omitempty"`
	Resource    string        `json:"resource,omitempty"`
	Suggestions []string      `json:"suggestions,omitempty"`
	Timestamp   string        `json:"timestamp"`
}

// HTTPStatus returns the appropriate HTTP status code for the error
func (e *APIError) HTTPStatus() int {
	switch e.Category {
	case CategoryNotFound:
		return http.StatusNotFound
	case CategoryExists:
		return http.StatusConflict
	case CategoryAuth:
		return http.StatusUnauthorized
	case CategoryPermission:
		return http.StatusForbidden
	case CategoryValidation, CategoryFormat, CategoryBadRequest:
		return http.StatusBadRequest
	case CategoryTimeout:
		return http.StatusRequestTimeout
	case CategoryUnavailable:
		return http.StatusServiceUnavailable
	case CategoryConnection, CategoryDatabase, CategoryVolume, CategoryInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// WriteErrorResponse writes a structured error response to the HTTP response writer
func WriteErrorResponse(w http.ResponseWriter, apiErr *APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.HTTPStatus())

	// Add timestamp if not set
	if apiErr.Timestamp == "" {
		apiErr.Timestamp = fmt.Sprintf("%d", getCurrentTimestamp())
	}

	json.NewEncoder(w).Encode(apiErr)
}

// Database-specific error constructors
func NewDBConnectionError(dbmsName string, originalErr error) *APIError {
	// Log technical details server-side for debugging
	log.Printf("[ERROR] Database connection failed for %s: %v", dbmsName, originalErr)

	suggestions := []string{
		"Contact your administrator to verify database connectivity",
		"Check if the database service is operational",
		"Verify network connectivity to the database server",
	}

	return &APIError{
		Category:    CategoryConnection,
		Code:        CodeDBConnectionRefused,
		Message:     fmt.Sprintf("Unable to connect to database provider '%s'", dbmsName),
		Details:     "", // Never expose technical details to clients
		Resource:    dbmsName,
		Suggestions: suggestions,
	}
}

func NewDBProviderNotFoundError(dbmsName string) *APIError {
	return &APIError{
		Category: CategoryNotFound,
		Code:     CodeDBProviderNotFound,
		Message:  fmt.Sprintf("Database provider '%s' is not configured", dbmsName),
		Resource: dbmsName,
		Suggestions: []string{
			"Check available providers with GET /db/dbmss",
			"Contact your administrator to configure the database provider",
			"Verify the provider name is spelled correctly",
		},
	}
}

func NewDatabaseNotFoundError(dbmsName, dbName string) *APIError {
	return &APIError{
		Category: CategoryNotFound,
		Code:     CodeDBNotFound,
		Message:  fmt.Sprintf("Database '%s' not found in provider '%s'", dbName, dbmsName),
		Resource: fmt.Sprintf("%s/%s", dbmsName, dbName),
		Suggestions: []string{
			fmt.Sprintf("List available databases with GET /db/%s/dbs", dbmsName),
			"Verify the database exists and is accessible",
			"Check database permissions",
		},
	}
}

func NewTableNotFoundError(dbmsName, dbName, tableName string) *APIError {
	return &APIError{
		Category: CategoryNotFound,
		Code:     CodeTableNotFound,
		Message:  fmt.Sprintf("Table '%s' not found in database '%s/%s'", tableName, dbmsName, dbName),
		Resource: fmt.Sprintf("%s/%s/%s", dbmsName, dbName, tableName),
		Suggestions: []string{
			fmt.Sprintf("List available tables with GET /db/%s/%s/tables", dbmsName, dbName),
			"Verify the table name is spelled correctly",
			"Check table permissions",
		},
	}
}

// Volume/File-specific error constructors
func NewVolumeNotFoundError(volumeName string) *APIError {
	return &APIError{
		Category: CategoryNotFound,
		Code:     CodeVolumeNotFound,
		Message:  fmt.Sprintf("Volume '%s' is not configured", volumeName),
		Resource: volumeName,
		Suggestions: []string{
			"Check available volumes with GET /volumes",
			"Contact your administrator to configure the volume",
			"Verify the volume name is spelled correctly",
		},
	}
}

func NewFileNotFoundError(volumeName, filePath string) *APIError {
	return &APIError{
		Category: CategoryNotFound,
		Code:     CodeFileNotFound,
		Message:  fmt.Sprintf("File '%s' not found in volume '%s'", filePath, volumeName),
		Resource: fmt.Sprintf("%s:%s", volumeName, filePath),
		Suggestions: []string{
			fmt.Sprintf("List available files with GET /files/%s/list", volumeName),
			"Verify the file path is correct",
			"Check if the file has been moved or deleted",
		},
	}
}

// Template-specific error constructors
func NewTemplateNotFoundError(templateName string) *APIError {
	return &APIError{
		Category: CategoryNotFound,
		Code:     CodeTemplateNotFound,
		Message:  fmt.Sprintf("Package template '%s' not found", templateName),
		Resource: templateName,
		Suggestions: []string{
			"List available templates with GET /packtmpl",
			"Verify the template name is spelled correctly",
			"Create the template first with POST /packtmpl",
		},
	}
}

func NewTemplateExistsError(templateName string) *APIError {
	return &APIError{
		Category: CategoryExists,
		Code:     CodeTemplateExists,
		Message:  fmt.Sprintf("Package template '%s' already exists", templateName),
		Resource: templateName,
		Suggestions: []string{
			"Use PUT /packtmpl to update the existing template",
			"Use PATCH /packtmpl for partial updates",
			"Choose a different template name",
		},
	}
}

func NewTemplateInvalidError(templateName string, validationErr error) *APIError {
	// Log technical details server-side for debugging
	log.Printf("[ERROR] Template validation failed for %s: %v", templateName, validationErr)

	return &APIError{
		Category: CategoryValidation,
		Code:     CodeTemplateInvalid,
		Message:  fmt.Sprintf("Package template '%s' is invalid", templateName),
		Details:  "", // Never expose technical details to clients
		Resource: templateName,
		Suggestions: []string{
			"Check YAML syntax and structure",
			"Contact your administrator for template validation assistance",
			"Refer to template documentation for required fields",
		},
	}
}

func NewTemplateProcessingError(templateName string, processingErr error) *APIError {
	// Log technical details server-side for debugging
	log.Printf("[ERROR] Template processing failed for %s: %v", templateName, processingErr)

	suggestions := []string{
		"Contact your administrator for assistance with template processing",
		"Verify that all required resources are properly configured",
		"Check that the template references valid system resources",
	}

	return &APIError{
		Category:    CategoryInternal,
		Code:        CodeTemplateProcessing,
		Message:     fmt.Sprintf("Failed to process package template '%s'", templateName),
		Details:     "", // Never expose technical details to clients
		Resource:    templateName,
		Suggestions: suggestions,
	}
}

// Authentication-specific error constructors
func NewAuthTokenInvalidError() *APIError {
	return &APIError{
		Category: CategoryAuth,
		Code:     CodeAuthTokenInvalid,
		Message:  "Authentication token is invalid or malformed",
		Suggestions: []string{
			"Obtain a new token with POST /auth/authenticate",
			"Check that the Authorization header is properly formatted",
			"Ensure the token is not corrupted",
		},
	}
}

func NewAuthTokenExpiredError() *APIError {
	return &APIError{
		Category: CategoryAuth,
		Code:     CodeAuthTokenExpired,
		Message:  "Authentication token has expired",
		Suggestions: []string{
			"Obtain a new token with POST /auth/authenticate",
			"Check token expiration time in JWT claims",
		},
	}
}

func NewAuthUserNotFoundError(username string) *APIError {
	return &APIError{
		Category: CategoryAuth,
		Code:     CodeAuthUserNotFound,
		Message:  fmt.Sprintf("User '%s' not found or not authorized", username),
		Resource: username,
		Suggestions: []string{
			"Contact your administrator to register your account",
			"Verify your username is correct",
			"Ensure your public key is properly registered",
		},
	}
}

func NewAuthSignatureInvalidError() *APIError {
	return &APIError{
		Category: CategoryAuth,
		Code:     CodeAuthSignatureInvalid,
		Message:  "Authentication signature verification failed",
		Suggestions: []string{
			"Ensure your private key matches the registered public key",
			"Check that the challenge was signed correctly",
			"Verify key format compatibility (RSA, ECDSA, Ed25519)",
		},
	}
}

// Request validation error constructors
func NewParameterMissingError(paramName string) *APIError {
	return &APIError{
		Category: CategoryValidation,
		Code:     CodeParameterMissing,
		Message:  fmt.Sprintf("Required parameter '%s' is missing", paramName),
		Resource: paramName,
		Suggestions: []string{
			fmt.Sprintf("Include the '%s' parameter in your request", paramName),
			"Check API documentation for required parameters",
		},
	}
}

func NewParameterInvalidError(paramName, expectedFormat string) *APIError {
	return &APIError{
		Category: CategoryValidation,
		Code:     CodeParameterInvalid,
		Message:  fmt.Sprintf("Parameter '%s' has invalid format", paramName),
		Details:  fmt.Sprintf("Expected format: %s", expectedFormat),
		Resource: paramName,
		Suggestions: []string{
			fmt.Sprintf("Ensure '%s' parameter follows the expected format: %s", paramName, expectedFormat),
			"Check API documentation for parameter specifications",
		},
	}
}

func NewFormatInvalidError(expectedFormat, actualContent string) *APIError {
	return &APIError{
		Category: CategoryFormat,
		Code:     CodeFormatInvalid,
		Message:  fmt.Sprintf("Invalid content format, expected %s", expectedFormat),
		Details:  fmt.Sprintf("Received content: %s", truncateString(actualContent, 100)),
		Suggestions: []string{
			fmt.Sprintf("Ensure request content is valid %s", expectedFormat),
			"Check content-type header",
			"Validate syntax and structure",
		},
	}
}

// Utility functions
func getCurrentTimestamp() int64 {
	return 1699027200 // Mock timestamp, in real implementation use time.Now().Unix()
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Helper function to detect error types from Go errors
func AnalyzeError(err error) (ErrorCategory, ErrorCode) {
	errMsg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errMsg, "connection refused"):
		return CategoryConnection, CodeDBConnectionRefused
	case strings.Contains(errMsg, "not found"):
		return CategoryNotFound, CodeDBProviderNotFound
	case strings.Contains(errMsg, "invalid signature"):
		return CategoryAuth, CodeAuthSignatureInvalid
	case strings.Contains(errMsg, "unauthorized"):
		return CategoryAuth, CodeAuthTokenInvalid
	case strings.Contains(errMsg, "unmarshal"):
		return CategoryFormat, CodeFormatInvalid
	default:
		return CategoryInternal, "UNKNOWN_ERROR"
	}
}
