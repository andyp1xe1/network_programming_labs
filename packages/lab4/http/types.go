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

// QuorumManager interface for dynamic quorum configurationtype
type QuorumManager interface {
	SetCommitThreshold(threshold int) error
	GetCommitThreshold() int
	GetFollowerCount() int
}

// SetQuorumRequest for updating quorum configuration
type SetQuorumRequest struct {
	Quorum int `json:"quorum"`
}

// QuorumStatusResponse for quorum status information
type QuorumStatusResponse struct {
	CurrentQuorum   int `json:"current_quorum"`
	MaxFollowers    int `json:"max_followers"`
	ActiveFollowers int `json:"active_followers"`
}
