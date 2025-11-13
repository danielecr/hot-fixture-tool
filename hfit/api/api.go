/*
 * Hot Fixture Tool CLI - API Client
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

// Package api provides API client functionality for the hfit CLI tool
package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// Client represents an API client for hfitd
type Client struct {
	BaseURL string
	Token   string
}

// NewClient creates a new API client
func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
	}
}

// makeRequest makes an authenticated HTTP request
func (c *Client) makeRequest(method, endpoint string) (*http.Response, error) {
	req, err := http.NewRequest(method, c.BaseURL+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	return resp, nil
}

// ListDBMSProviders lists all available DBMS providers
func (c *Client) ListDBMSProviders() ([]interface{}, error) {
	resp, err := c.makeRequest("GET", "/db/dbmss")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var providers []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&providers); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return providers, nil
}

// ListDatabases lists all databases for a specific DBMS provider
func (c *Client) ListDatabases(dbms string) ([]interface{}, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/db/%s/dbs", dbms))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var databases []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&databases); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return databases, nil
}

// ListTables lists tables in a specific database
func (c *Client) ListTables(dbms, dbID string) ([]interface{}, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/db/%s/%s/tables", dbms, dbID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tables []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tables); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return tables, nil
}

// ListFiles lists files in a volume
func (c *Client) ListFiles(volume string) ([]interface{}, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/files/%s/list", volume))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var files []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return files, nil
}

// StreamFiles streams files from a volume using NDJSON format for high performance
func (c *Client) StreamFiles(volume string) error {
	req, err := http.NewRequest("GET", c.BaseURL+fmt.Sprintf("/files/%s/list", volume), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/x-json-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Read streaming NDJSON response line by line
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			// Output raw NDJSON line
			fmt.Println(line)
		}
	}

	return scanner.Err()
}

// StreamRows streams table rows using NDJSON format with optional filterpart parameter
func (c *Client) StreamRows(dbms, dbID, tableID, filterpart string) error {
	// Build the endpoint URL with optional filterpart parameter
	endpoint := fmt.Sprintf("/db/%s/%s/table/%s/rows", dbms, dbID, tableID)
	if filterpart != "" {
		endpoint += fmt.Sprintf("?filterpart=%s", url.QueryEscape(filterpart))
	}

	req, err := http.NewRequest("GET", c.BaseURL+endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/x-json-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Read streaming NDJSON response line by line
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			// Output raw NDJSON line
			fmt.Println(line)
		}
	}

	return scanner.Err()
}

// StreamFilesWithFilters streams files from a volume with advanced filtering using NDJSON format
func (c *Client) StreamFilesWithFilters(volume string, filters []string) error {
	// Build the endpoint URL with filter parameters
	endpoint := fmt.Sprintf("/files/%s/list", volume)
	if len(filters) > 0 {
		endpoint += "?"
		for i, filter := range filters {
			if i > 0 {
				endpoint += "&"
			}
			endpoint += fmt.Sprintf("filter[]=%s", filter)
		}
	}

	req, err := http.NewRequest("GET", c.BaseURL+endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/x-json-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Read streaming NDJSON response line by line
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			// Output raw NDJSON line
			fmt.Println(line)
		}
	}

	return scanner.Err()
}

// StreamDownloadFile downloads a file using volume:/path format and streams it to the provided writer
func (c *Client) StreamDownloadFile(volumePath string, writer io.Writer) error {
	// Parse volume:/path format
	parts := strings.SplitN(volumePath, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid path format, expected 'volume:/path/to/file', got '%s'", volumePath)
	}

	volume := parts[0]
	filePath := parts[1]

	// Use the correct API endpoint: /files/{volume}/download?path={path}
	endpoint := fmt.Sprintf("/files/%s/download", volume)
	if filePath != "" {
		endpoint += fmt.Sprintf("?path=%s", url.QueryEscape(filePath))
	}

	resp, err := c.makeRequest("GET", endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Stream the file content directly to the writer
	_, err = io.Copy(writer, resp.Body)
	return err
}

// makeHeadRequest makes an authenticated HEAD HTTP request to check resource existence
func (c *Client) makeHeadRequest(endpoint string) (bool, error) {
	req, err := http.NewRequest("HEAD", c.BaseURL+endpoint, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// CheckDBMSExists checks if a DBMS provider exists
func (c *Client) CheckDBMSExists(dbms string) (bool, error) {
	return c.makeHeadRequest(fmt.Sprintf("/db/%s", dbms))
}

// CheckDatabaseExists checks if a database exists for a DBMS provider
func (c *Client) CheckDatabaseExists(dbms, dbid string) (bool, error) {
	return c.makeHeadRequest(fmt.Sprintf("/db/%s/%s", dbms, dbid))
}

// CheckTableExists checks if a table exists in a database
func (c *Client) CheckTableExists(dbms, dbid, tableid string) (bool, error) {
	return c.makeHeadRequest(fmt.Sprintf("/db/%s/%s/table/%s", dbms, dbid, tableid))
}

// CheckVolumeExists checks if a volume exists
func (c *Client) CheckVolumeExists(volume string) (bool, error) {
	return c.makeHeadRequest(fmt.Sprintf("/volumes/%s", volume))
}

// CheckFileExists checks if a file exists in a volume
func (c *Client) CheckFileExists(volume, filepath string) (bool, error) {
	return c.makeHeadRequest(fmt.Sprintf("/files/%s/%s", volume, filepath))
}

// DownloadPackage downloads a package using the packdownload API
func (c *Client) DownloadPackage(packname string, yamlData []byte) (io.ReadCloser, error) {
	req, err := http.NewRequest("POST", c.BaseURL+fmt.Sprintf("/packdownload/%s", packname), strings.NewReader(string(yamlData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/x-yaml")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return resp.Body, nil
}

// ListTemplates lists all package templates for the authenticated user
func (c *Client) ListTemplates() ([]string, error) {
	resp, err := c.makeRequest("GET", "/packtmpl")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var templates []string
	if err := json.NewDecoder(resp.Body).Decode(&templates); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return templates, nil
}

// GetTemplate retrieves a specific template's YAML content
func (c *Client) GetTemplate(templateName string) (string, error) {
	resp, err := c.makeRequest("GET", "/packtmpl/"+templateName)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(content), nil
}

// CreateTemplate creates a new template from YAML content
func (c *Client) CreateTemplate(yamlContent []byte) error {
	return c.sendTemplateRequest("POST", "/packtmpl", yamlContent)
}

// UpdateTemplate updates an existing template with YAML content
func (c *Client) UpdateTemplate(yamlContent []byte) error {
	return c.sendTemplateRequest("PUT", "/packtmpl", yamlContent)
}

// PatchTemplate partially updates a template and returns diff output
func (c *Client) PatchTemplate(yamlContent []byte) (string, error) {
	req, err := http.NewRequest("PATCH", c.BaseURL+"/packtmpl", strings.NewReader(string(yamlContent)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/x-yaml")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	diffOutput, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(diffOutput), nil
}

// DeleteTemplate deletes a specific template
func (c *Client) DeleteTemplate(templateName string) error {
	resp, err := c.makeRequest("DELETE", "/packtmpl/"+templateName)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GenerateAndDownloadPackage generates a package from a template with parameters
func (c *Client) GenerateAndDownloadPackage(templateName string, params []string) error {
	// Send parameters as a JSON array directly
	payloadBytes, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal request payload: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/packdownload/"+templateName, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// The response should be a downloadable package
	// For now, just save it to a file based on the template name
	packageFile := fmt.Sprintf("%s_package.tar.gz", templateName)
	file, err := os.Create(packageFile)
	if err != nil {
		return fmt.Errorf("failed to create package file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write package file: %w", err)
	}

	fmt.Printf("Package saved to: %s\n", packageFile)
	return nil
}

// sendTemplateRequest is a helper function for sending template requests
func (c *Client) sendTemplateRequest(method, endpoint string, yamlContent []byte) error {
	req, err := http.NewRequest(method, c.BaseURL+endpoint, strings.NewReader(string(yamlContent)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/x-yaml")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
