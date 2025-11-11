// Package server provides HTTP server for Memory Scramble game
package server

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"lab3/internal/board"
	"lab3/internal/commands"
)

// Server represents the HTTP server for Memory Scramble
type Server struct {
	board *board.Board
	mux   *http.ServeMux
}

// New creates a new server instance with the given board
func New(b *board.Board) *Server {
	s := &Server{
		board: b,
		mux:   http.NewServeMux(),
	}

	// Set the board in commands package
	commands.SetBoard(b)

	// Register handlers
	s.registerHandlers()

	return s
}

// Start starts the HTTP server on the given address
func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) registerHandlers() {
	// Serve the web interface
	s.mux.HandleFunc("/", s.handleIndex)

	// Game API endpoints
	s.mux.HandleFunc("/look/", s.handleLook)
	s.mux.HandleFunc("/flip/", s.handleFlip)
	s.mux.HandleFunc("/replace/", s.handleReplace)
	s.mux.HandleFunc("/watch/", s.handleWatch)
	s.mux.HandleFunc("/restart", s.handleRestart)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Serve the HTML file
	http.ServeFile(w, r, "index.html")
}

func (s *Server) handleLook(w http.ResponseWriter, r *http.Request) {
	// Extract player ID from path: /look/{playerID}
	path := strings.TrimPrefix(r.URL.Path, "/look/")
	playerID := path

	if playerID == "" {
		http.Error(w, "Missing player ID", http.StatusBadRequest)
		return
	}

	result, err := commands.Look(playerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*") // Enable CORS
	fmt.Fprint(w, result)
}

func (s *Server) handleFlip(w http.ResponseWriter, r *http.Request) {
	// Extract player ID and position from path: /flip/{playerID}/{row,col}
	path := strings.TrimPrefix(r.URL.Path, "/flip/")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) != 2 {
		http.Error(w, "Invalid flip path format", http.StatusBadRequest)
		return
	}

	playerID := parts[0]
	position := parts[1]

	if playerID == "" || position == "" {
		http.Error(w, "Missing player ID or position", http.StatusBadRequest)
		return
	}

	result, err := commands.Flip(playerID, position)
	if err != nil {
		log.Printf("Flip error for player %s at %s: %v", playerID, position, err)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*") // Enable CORS
	fmt.Fprint(w, result)
}

func (s *Server) handleReplace(w http.ResponseWriter, r *http.Request) {
	// Extract parameters from path: /replace/{playerID}/{fromCard}/{toCard}
	path := strings.TrimPrefix(r.URL.Path, "/replace/")
	parts := strings.SplitN(path, "/", 3)

	if len(parts) != 3 {
		http.Error(w, "Invalid replace path format", http.StatusBadRequest)
		return
	}

	playerID := parts[0]
	fromCard := parts[1]
	toCard := parts[2]

	if playerID == "" || fromCard == "" || toCard == "" {
		http.Error(w, "Missing parameters", http.StatusBadRequest)
		return
	}

	result, err := commands.Replace(playerID, fromCard, toCard)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*") // Enable CORS
	fmt.Fprint(w, result)
}

func (s *Server) handleWatch(w http.ResponseWriter, r *http.Request) {
	// Extract player ID from path: /watch/{playerID}
	path := strings.TrimPrefix(r.URL.Path, "/watch/")
	playerID := path

	if playerID == "" {
		http.Error(w, "Missing player ID", http.StatusBadRequest)
		return
	}

	result, err := commands.Watch(playerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*") // Enable CORS
	fmt.Fprint(w, result)
}

func (s *Server) handleRestart(w http.ResponseWriter, r *http.Request) {
	result, err := commands.Restart()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*") // Enable CORS
	fmt.Fprint(w, result)
}
