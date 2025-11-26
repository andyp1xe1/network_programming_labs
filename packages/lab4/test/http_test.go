package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	kvhttp "github.com/andyp1xe1/network_programming_labs/packages/lab4/http"
	"github.com/andyp1xe1/network_programming_labs/packages/lab4/leader"
	"github.com/andyp1xe1/network_programming_labs/packages/lab4/store"
)

func TestHTTPEndpoints(t *testing.T) {
	// Create a leader store with no followers for simple testing
	baseStore := store.NewMapStore()
	config := leader.LeaderConfig{
		CommitThreshold: 1, // No followers, so just need local confirmation
		MaxDelay:        time.Millisecond * 10,
		MinDelay:        time.Millisecond * 1,
	}
	leaderStore := leader.NewLeaderStore(baseStore, config, []leader.KVClient{}, "test-leader")

	// Create HTTP handler
	handler := kvhttp.NewHTTPServer(leaderStore)

	// Test SET operation
	setBody := map[string]string{"key": "test", "value": "hello", "id": "test-client"}
	jsonBody, _ := json.Marshal(setBody)

	req := httptest.NewRequest("POST", "/set", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Test GET operation
	getBody := map[string]string{"key": "test"}
	jsonBody, _ = json.Marshal(getBody)
	req = httptest.NewRequest("POST", "/get", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Use the proper Response type
	type Response struct {
		Success bool   `json:"success"`
		Value   string `json:"value,omitempty"`
		Error   string `json:"error,omitempty"`
	}

	var response Response
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v. Response body: %s", err, w.Body.String())
	}

	if !response.Success {
		t.Fatalf("Response indicates failure: %s", response.Error)
	}

	if response.Value != "hello" {
		t.Fatalf("Expected 'hello', got '%s'", response.Value)
	}

	t.Log("HTTP endpoints test passed!")
}

func TestHTTPConcurrentOperations(t *testing.T) {
	// Create leader with no followers for simplicity
	baseStore := store.NewMapStore()
	config := leader.LeaderConfig{
		CommitThreshold: 1,
		MaxDelay:        time.Millisecond * 5,
		MinDelay:        time.Millisecond * 1,
	}
	leaderStore := leader.NewLeaderStore(baseStore, config, []leader.KVClient{}, "test-leader")

	handler := kvhttp.NewHTTPServer(leaderStore)

	// Test concurrent SET operations
	numOperations := 50
	done := make(chan bool, numOperations)

	for i := range numOperations {
		go func(id int) {
			setBody := map[string]string{
				"key":   fmt.Sprintf("key_%d", id),
				"value": fmt.Sprintf("value_%d", id),
				"id":    "test-client",
			}
			jsonBody, _ := json.Marshal(setBody)

			req := httptest.NewRequest("POST", "/set", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			done <- w.Code == http.StatusOK
		}(i)
	}

	// Wait for all operations to complete
	successCount := 0
	for range numOperations {
		if <-done {
			successCount++
		}
	}

	t.Logf("Concurrent operations: %d/%d succeeded", successCount, numOperations)

	if successCount != numOperations {
		t.Fatalf("Expected all %d operations to succeed, got %d", numOperations, successCount)
	}

	t.Log("Concurrent HTTP operations test passed!")
}
