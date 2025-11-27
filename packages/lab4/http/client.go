package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/andyp1xe1/network_programming_labs/packages/lab4/common"
)

// HTTPClient implements the store.Store interface over HTTP
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
	clientID   string
}

// NewHTTPClient creates a new HTTP client for the KV store
func NewHTTPClient(baseURL, clientID string) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		clientID: clientID,
	}
}

// Set implements store.Store.Set over HTTP
func (c *HTTPClient) Set(ctx context.Context, key, value string) error {
	expectedVersion, _ := common.ExpectedVersionFromCtx(ctx)

	req := SetRequest{
		Key:             key,
		Value:           value,
		ExpectedVersion: expectedVersion,
		ID:              c.clientID,
	}

	var resp Response
	if err := c.makeRequest(ctx, "POST", "/set", req, &resp); err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("set failed: %s", resp.Error)
	}

	return nil
}

// Get implements store.Store.Get over HTTP
func (c *HTTPClient) Get(ctx context.Context, key string) (string, error) {
	req := GetRequest{Key: key}

	var resp Response
	if err := c.makeRequest(ctx, "POST", "/get", req, &resp); err != nil {
		return "", err
	}

	if !resp.Success {
		return "", fmt.Errorf("get failed: %s", resp.Error)
	}

	return resp.Value, nil
}

// Delete implements store.Store.Delete over HTTP
func (c *HTTPClient) Delete(ctx context.Context, key string) error {
	expectedVersion, _ := common.ExpectedVersionFromCtx(ctx)
	req := DeleteRequest{
		Key:             key,
		ExpectedVersion: expectedVersion,
		ID:              c.clientID,
	}

	var resp Response
	if err := c.makeRequest(ctx, "POST", "/delete", req, &resp); err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("delete failed: %s", resp.Error)
	}

	return nil
}

// Exists implements store.Store.Exists over HTTP
func (c *HTTPClient) Exists(ctx context.Context, key string) (bool, error) {
	req := ExistsRequest{Key: key}

	var resp Response
	err := c.makeRequest(ctx, "POST", "/exists", req, &resp)
	if err != nil {
		return false, err
	}

	if !resp.Success {
		return false, fmt.Errorf("exists check failed: %s", resp.Error)
	}

	return resp.Exists, nil
}

// makeRequest is a helper method to make HTTP requests with JSON
func (c *HTTPClient) makeRequest(ctx context.Context, method, endpoint string, reqBody, respBody any) error {
	// Marshal request body
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := c.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Make the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Decode response
	if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}
