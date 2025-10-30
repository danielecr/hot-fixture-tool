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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// ListRows lists rows in a specific table
func (c *Client) ListRows(dbms, dbID, tableID string) ([]interface{}, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/db/%s/%s/table/%s/rows", dbms, dbID, tableID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var rows []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return rows, nil
}

// ListFiles lists files
func (c *Client) ListFiles() ([]interface{}, error) {
	resp, err := c.makeRequest("GET", "/files/list")
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

// DownloadFile downloads a file
func (c *Client) DownloadFile(path string) ([]byte, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/files/download?path=%s", path))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
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
