package http

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/andyp1xe1/network_programming_labs/packages/lab4/common"
	"github.com/andyp1xe1/network_programming_labs/packages/lab4/leader"
	"github.com/andyp1xe1/network_programming_labs/packages/lab4/store"
)

// HTTPServer wraps a store.Store and exposes it via HTTP
type HTTPServer struct {
	store store.Store
	mux   *http.ServeMux
}

// NewHTTPServer creates a new HTTP server for the given store
func NewHTTPServer(st store.Store) *HTTPServer {
	s := &HTTPServer{
		store: st,
		mux:   http.NewServeMux(),
	}
	s.setupRoutes()
	return s
}

// ServeHTTP implements http.Handler
func (s *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// ExposeHTTP starts the HTTP server on the given address
func ExposeHTTP(st store.Store, address string) error {
	server := NewHTTPServer(st)
	log.Printf("Starting HTTP server on %s", address)
	return http.ListenAndServe(address, server)
}

// setupRoutes configures all the HTTP routes
func (s *HTTPServer) setupRoutes() {
	s.mux.HandleFunc("/set", s.handleSet)
	s.mux.HandleFunc("/get", s.handleGet)
	s.mux.HandleFunc("/delete", s.handleDelete)
	s.mux.HandleFunc("/exists", s.handleExists)
	s.mux.HandleFunc("/status", s.handleStatus)
	s.mux.HandleFunc("/", s.handleHelp)

	// Add admin quorum endpoint if it is a leader
	if _, ok := s.store.(*leader.LeaderStore); ok {
		s.mux.HandleFunc("/admin/quorum", s.handleQuorumAdmin)
		log.Println("HTTP server: Registered admin quorum endpoint (leader mode)")
	}
}

func (s *HTTPServer) handleSet(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received %s request for %s", r.Method, r.URL.Path)
	if r.Method != http.MethodPost {
		writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := common.CtxWithID(req.ID)

	log.Printf("Set request: key=%s, value=%s, id=%s", req.Key, req.Value, req.ID)
	err := s.store.Set(ctx, req.Key, req.Value)
	if err != nil {
		log.Printf("Set error: %v", err)
		writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeSuccessResponse(w, Response{Success: true})
}

func (s *HTTPServer) handleGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	value, err := s.store.Get(context.Background(), req.Key)
	if err != nil {
		writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeSuccessResponse(w, Response{
		Success: true,
		Value:   value,
	})
}

func (s *HTTPServer) handleDelete(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received %s request for %s", r.Method, r.URL.Path)
	if r.Method != http.MethodPost {
		writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := common.CtxWithID(req.ID)

	log.Printf("Delete request: key=%s, id=%s", req.Key, req.ID)
	err := s.store.Delete(ctx, req.Key)
	if err != nil {
		log.Printf("Delete error: %v", err)
		writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeSuccessResponse(w, Response{Success: true})
}

func (s *HTTPServer) handleExists(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExistsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	exists, err := s.store.Exists(context.Background(), req.Key)
	if err != nil {
		writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]any{
		"success": true,
		"exists":  exists,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode exists response: %v", err)
	}
}

func (s *HTTPServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeSuccessResponse(w, Response{
		Success: true,
		Data: map[string]any{
			"status": "healthy",
			"type":   "kv-store",
		},
	})
}

func (s *HTTPServer) handleHelp(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	endpoints := map[string]string{
		"POST /set":    "Set a key-value pair: {\"key\": \"foo\", \"value\": \"bar\", \"id\": \"self-id\"}",
		"POST /get":    "Get a value by key: {\"key\": \"foo\"}",
		"POST /delete": "Delete a key: {\"key\": \"foo\", \"id\": \"self-id\"}",
		"POST /exists": "Check if key exists: {\"key\": \"foo\"}",
		"GET /status":  "Get server status",
	}

	// Add admin endpoint documentation if available
	if _, ok := s.store.(*leader.LeaderStore); ok {
		endpoints["GET /admin/quorum"] = "Get current quorum configuration"
		endpoints["POST /admin/quorum"] = "Update quorum: {\"quorum\": 2}"
	}

	help := map[string]any{
		"endpoints":   endpoints,
		"description": "HTTP interface for KV store",
	}

	writeSuccessResponse(w, Response{
		Success: true,
		Data:    help,
	})
}

func writeSuccessResponse(w http.ResponseWriter, resp Response) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

// handleQuorumAdmin handles GET and POST requests to /admin/quorum
func (s *HTTPServer) handleQuorumAdmin(w http.ResponseWriter, r *http.Request) {
	// Verify that the store implements QuorumManager
	qm, ok := s.store.(QuorumManager)
	if !ok {
		writeErrorResponse(w, "Quorum administration not available on this node", http.StatusNotImplemented)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetQuorumStatus(w, r, qm)
	case http.MethodPost:
		s.handleSetQuorum(w, r, qm)
	default:
		writeErrorResponse(w, "Method not allowed. Use GET to check status or POST to update quorum.", http.StatusMethodNotAllowed)
	}
}

// handleGetQuorumStatus returns current quorum configuration
func (s *HTTPServer) handleGetQuorumStatus(w http.ResponseWriter, _ *http.Request, qm QuorumManager) {
	log.Printf("Received GET request for quorum status")

	status := QuorumStatusResponse{
		CurrentQuorum:   qm.GetCommitThreshold(),
		MaxFollowers:    qm.GetFollowerCount(),
		ActiveFollowers: qm.GetFollowerCount(),
	}

	writeSuccessResponse(w, Response{
		Success: true,
		Data:    status,
	})
}

// handleSetQuorum updates the quorum configuration
func (s *HTTPServer) handleSetQuorum(w http.ResponseWriter, r *http.Request, qm QuorumManager) {
	log.Printf("Received POST request to update quorum")

	var req SetQuorumRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, "Invalid JSON in request body", http.StatusBadRequest)
		return
	}

	log.Printf("Attempting to set quorum to %d", req.Quorum)

	if err := qm.SetCommitThreshold(req.Quorum); err != nil {
		log.Printf("Failed to set quorum: %v", err)
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return the updated status
	status := QuorumStatusResponse{
		CurrentQuorum:   qm.GetCommitThreshold(),
		MaxFollowers:    qm.GetFollowerCount(),
		ActiveFollowers: qm.GetFollowerCount(),
	}

	log.Printf("Successfully updated quorum to %d", req.Quorum)
	writeSuccessResponse(w, Response{
		Success: true,
		Data:    status,
	})
}

func writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	resp := Response{
		Success: false,
		Error:   message,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Failed to encode error response: %v", err)
	}
}
