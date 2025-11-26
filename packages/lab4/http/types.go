// Package http provides HTTP client and server interfaces for the KV store.
package http

// SetRequest JSON request structures
type SetRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	ID    string `json:"id,omitempty"`
}

type GetRequest struct {
	Key string `json:"key"`
}

type DeleteRequest struct {
	Key string `json:"key"`
	ID  string `json:"id,omitempty"`
}

type ExistsRequest struct {
	Key string `json:"key"`
}

// Response JSON response structure
type Response struct {
	Success bool   `json:"success"`
	Value   string `json:"value,omitempty"`
	Exists  bool   `json:"exists,omitempty"`
	Error   string `json:"error,omitempty"`
	Data    any    `json:"data,omitempty"`
}
