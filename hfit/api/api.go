// Package api provides API client functionality for the hfit CLI tool
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// ListDatabases lists all databases
func (c *Client) ListDatabases() ([]interface{}, error) {
	resp, err := c.makeRequest("GET", "/db/dbs")
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
func (c *Client) ListTables(dbID string) ([]interface{}, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/db/%s/tables", dbID))
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
func (c *Client) ListRows(dbID, tableID string) ([]interface{}, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/db/%s/table/%s/rows", dbID, tableID))
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