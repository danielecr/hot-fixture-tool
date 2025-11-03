/*
 * Hot Fixture Tool CLI - Error Handling
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

// Package errors provides user-friendly error handling for hfit CLI commands
package errors

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	// Connection errors
	ErrorTypeConnection ErrorType = "connection"
	ErrorTypeAuth       ErrorType = "authentication"
	ErrorTypeTimeout    ErrorType = "timeout"

	// Resource errors
	ErrorTypeNotFound   ErrorType = "not_found"
	ErrorTypeExists     ErrorType = "already_exists"
	ErrorTypePermission ErrorType = "permission"

	// Configuration errors
	ErrorTypeConfig     ErrorType = "configuration"
	ErrorTypeFormat     ErrorType = "format"
	ErrorTypeValidation ErrorType = "validation"

	// Server errors
	ErrorTypeServer   ErrorType = "server"
	ErrorTypeInternal ErrorType = "internal"
)

// Command represents the CLI command that generated the error
type Command string

const (
	CommandDBMSS       Command = "dbmss"
	CommandDBS         Command = "dbs"
	CommandTables      Command = "tables"
	CommandRows        Command = "rows"
	CommandFiles       Command = "files"
	CommandDownload    Command = "download"
	CommandPkgTmpl     Command = "pkg-tmpl"
	CommandPkgDownload Command = "pkg-download"
	CommandLogin       Command = "login"
	CommandConfig      Command = "config"
)

// ErrorContext contains information about the error context
type ErrorContext struct {
	Command   Command
	Resource  string // e.g., DBMS name, volume name, template name
	Operation string // e.g., "list", "create", "delete"
	Details   string // original error message
}

// UserFriendlyError converts technical errors into user-friendly messages
func UserFriendlyError(ctx ErrorContext, originalErr error) error {
	if originalErr == nil {
		return nil
	}

	errMsg := strings.ToLower(originalErr.Error())

	// Detect error type based on common patterns
	errorType := detectErrorType(errMsg)

	// Generate user-friendly message
	friendlyMsg := generateFriendlyMessage(ctx, errorType, errMsg)

	return fmt.Errorf("%s", friendlyMsg)
}

// detectErrorType analyzes the error message to determine the error type
func detectErrorType(errMsg string) ErrorType {
	switch {
	// Connection errors
	case strings.Contains(errMsg, "connection refused"):
		return ErrorTypeConnection
	case strings.Contains(errMsg, "dial tcp"):
		return ErrorTypeConnection
	case strings.Contains(errMsg, "timeout"):
		return ErrorTypeTimeout
	case strings.Contains(errMsg, "no such host"):
		return ErrorTypeConnection

	// Authentication errors
	case strings.Contains(errMsg, "unauthorized"):
		return ErrorTypeAuth
	case strings.Contains(errMsg, "invalid signature"):
		return ErrorTypeAuth
	case strings.Contains(errMsg, "token"):
		return ErrorTypeAuth

	// Not found errors
	case strings.Contains(errMsg, "not found"):
		return ErrorTypeNotFound
	case strings.Contains(errMsg, "404"):
		return ErrorTypeNotFound
	case strings.Contains(errMsg, "does not exist"):
		return ErrorTypeNotFound

	// Format/validation errors
	case strings.Contains(errMsg, "invalid format"):
		return ErrorTypeFormat
	case strings.Contains(errMsg, "parse"):
		return ErrorTypeFormat
	case strings.Contains(errMsg, "unmarshal"):
		return ErrorTypeFormat

	// Server errors
	case strings.Contains(errMsg, "500"):
		return ErrorTypeServer
	case strings.Contains(errMsg, "internal server"):
		return ErrorTypeInternal

	default:
		return ErrorTypeServer
	}
}

// generateFriendlyMessage creates user-friendly error messages
func generateFriendlyMessage(ctx ErrorContext, errorType ErrorType, originalErr string) string {
	switch ctx.Command {
	case CommandDBMSS:
		return handleDBMSSErrors(errorType, originalErr)
	case CommandDBS:
		return handleDBSErrors(ctx, errorType, originalErr)
	case CommandTables:
		return handleTablesErrors(ctx, errorType, originalErr)
	case CommandRows:
		return handleRowsErrors(ctx, errorType, originalErr)
	case CommandFiles:
		return handleFilesErrors(ctx, errorType, originalErr)
	case CommandDownload:
		return handleDownloadErrors(ctx, errorType, originalErr)
	case CommandPkgTmpl:
		return handlePkgTmplErrors(ctx, errorType, originalErr)
	case CommandPkgDownload:
		return handlePkgDownloadErrors(ctx, errorType, originalErr)
	case CommandLogin:
		return handleLoginErrors(errorType, originalErr)
	case CommandConfig:
		return handleConfigErrors(errorType, originalErr)
	default:
		return fmt.Sprintf("An unexpected error occurred: %s", originalErr)
	}
}

// DBMS-related error handlers
func handleDBMSSErrors(errorType ErrorType, originalErr string) string {
	switch errorType {
	case ErrorTypeConnection:
		return "Unable to connect to the hfitd server. Please check:\n" +
			"  • Server is running (./hfitd)\n" +
			"  • Server URL is correct in ~/.hfit/config\n" +
			"  • Network connectivity"
	case ErrorTypeAuth:
		return "Authentication failed. Please run 'hfit login' to authenticate."
	default:
		return "Failed to list database providers. Please contact your administrator."
	}
}

func handleDBSErrors(ctx ErrorContext, errorType ErrorType, originalErr string) string {
	switch errorType {
	case ErrorTypeConnection:
		return fmt.Sprintf("Cannot connect to database provider '%s':\n"+
			"  • Database server may not be running\n"+
			"  • Check connection settings for '%s'\n"+
			"  • Verify network access to the database", ctx.Resource, ctx.Resource)
	case ErrorTypeNotFound:
		return fmt.Sprintf("Database provider '%s' not found. Use 'hfit dbmss' to see available providers.", ctx.Resource)
	case ErrorTypeAuth:
		return "Authentication failed. Please run 'hfit login' to authenticate."
	default:
		return fmt.Sprintf("Failed to list databases for '%s'. Please check the database configuration.", ctx.Resource)
	}
}

func handleTablesErrors(ctx ErrorContext, errorType ErrorType, originalErr string) string {
	switch errorType {
	case ErrorTypeConnection:
		return "Database connection failed. Please verify the database server is accessible."
	case ErrorTypeNotFound:
		return fmt.Sprintf("Database or provider not found. Please verify '%s' exists.", ctx.Resource)
	case ErrorTypeAuth:
		return "Authentication failed. Please run 'hfit login' to authenticate."
	default:
		return "Failed to list tables. Please check your database permissions."
	}
}

func handleRowsErrors(ctx ErrorContext, errorType ErrorType, originalErr string) string {
	switch errorType {
	case ErrorTypeConnection:
		return "Database connection failed. Please verify the database server is accessible."
	case ErrorTypeNotFound:
		return "Table not found. Please verify the table name and database."
	case ErrorTypeAuth:
		return "Authentication failed. Please run 'hfit login' to authenticate."
	default:
		return "Failed to retrieve table data. Please check your query and permissions."
	}
}

// File-related error handlers
func handleFilesErrors(ctx ErrorContext, errorType ErrorType, originalErr string) string {
	switch errorType {
	case ErrorTypeNotFound:
		return fmt.Sprintf("Volume '%s' not found. Contact your administrator to configure volumes.", ctx.Resource)
	case ErrorTypeAuth:
		return "Authentication failed. Please run 'hfit login' to authenticate."
	default:
		return fmt.Sprintf("Failed to access files in volume '%s'. Please check the volume configuration.", ctx.Resource)
	}
}

func handleDownloadErrors(ctx ErrorContext, errorType ErrorType, originalErr string) string {
	switch errorType {
	case ErrorTypeNotFound:
		if strings.Contains(originalErr, "volume") {
			return fmt.Sprintf("Volume not found. Please check the volume name in '%s'.\n"+
				"Use format: volume:/path/to/file", ctx.Resource)
		}
		return fmt.Sprintf("File not found: '%s'\n"+
			"Please verify the file path exists in the specified volume.", ctx.Resource)
	case ErrorTypeFormat:
		return "Invalid file path format. Please use: volume:/path/to/file\n" +
			"Example: hfit download datavol1:/logs/app.log"
	case ErrorTypeAuth:
		return "Authentication failed. Please run 'hfit login' to authenticate."
	default:
		return fmt.Sprintf("Failed to download file '%s'. Please check the file path and permissions.", ctx.Resource)
	}
}

// Template-related error handlers
func handlePkgTmplErrors(ctx ErrorContext, errorType ErrorType, originalErr string) string {
	switch errorType {
	case ErrorTypeNotFound:
		if ctx.Operation == "show" || ctx.Operation == "delete" {
			return fmt.Sprintf("Template '%s' not found. Use 'hfit pkg-tmpl list' to see available templates.", ctx.Resource)
		}
		return "Template file not found. Please check the file path."
	case ErrorTypeExists:
		return fmt.Sprintf("Template '%s' already exists. Use 'hfit pkg-tmpl update' to modify it.", ctx.Resource)
	case ErrorTypeFormat:
		return "Invalid template format. Please check your YAML syntax and ensure 'templateName' field is present."
	case ErrorTypeAuth:
		return "Authentication failed. Please run 'hfit login' to authenticate."
	default:
		return fmt.Sprintf("Template operation failed. Please check your template file and try again.")
	}
}

func handlePkgDownloadErrors(ctx ErrorContext, errorType ErrorType, originalErr string) string {
	switch errorType {
	case ErrorTypeNotFound:
		return fmt.Sprintf("Template '%s' not found. Use 'hfit pkg-tmpl list' to see available templates.", ctx.Resource)
	case ErrorTypeConfig:
		return "Package generation failed due to missing configuration (database providers, volumes, etc.).\n" +
			"Please contact your administrator to configure the required resources."
	case ErrorTypeAuth:
		return "Authentication failed. Please run 'hfit login' to authenticate."
	default:
		return fmt.Sprintf("Failed to generate package from template '%s'. Please check the template configuration.", ctx.Resource)
	}
}

// Authentication-related error handlers
func handleLoginErrors(errorType ErrorType, originalErr string) string {
	switch errorType {
	case ErrorTypeConnection:
		return "Cannot connect to authentication server. Please check:\n" +
			"  • Server is running\n" +
			"  • Server URL is correct in ~/.hfit/config"
	case ErrorTypeAuth:
		if strings.Contains(originalErr, "passphrase") {
			return "Invalid passphrase. Please check your private key passphrase."
		}
		if strings.Contains(originalErr, "private key") {
			return "Private key error. Please check:\n" +
				"  • Key file exists and is readable\n" +
				"  • Key format is supported (RSA, ECDSA, Ed25519)\n" +
				"  • Key matches the public key registered with the server"
		}
		return "Authentication failed. Please check your credentials and try again."
	case ErrorTypeFormat:
		return "Invalid key format. Please ensure your private key is in PEM or OpenSSH format."
	default:
		return "Login failed. Please check your configuration and credentials."
	}
}

func handleConfigErrors(errorType ErrorType, originalErr string) string {
	switch errorType {
	case ErrorTypeFormat:
		return "Invalid configuration format. Please check your parameters."
	case ErrorTypePermission:
		return "Cannot write configuration file. Please check file permissions for ~/.hfit/"
	default:
		return "Configuration failed. Please check your parameters and file permissions."
	}
}

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
