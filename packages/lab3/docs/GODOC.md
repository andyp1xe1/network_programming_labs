# Lab 3 Go Package Documentation\n
Generated on: Tue Nov 11 03:42:04 PM EET 2025\n
## Package: main\n
Package main implements the Memory Scramble game server.

Memory Scramble is a multi-player card matching game server that implements the
MIT 6.102 Problem Set 4 specification. The server provides:

  - Thread-safe concurrent gameplay for multiple players
  - HTTP API endpoints for all game operations (look, flip, replace, watch,
    restart)
  - Web-based user interface with real-time updates
  - Comprehensive game rule enforcement (Rules 1-A through 3-B)
  - Board file configuration support

Usage:

    memory-scramble -port=8080 -board=boards/perfect.txt

The server starts an HTTP server on the specified port and serves both the game
API and the web interface. Players can connect using the web interface or make
direct HTTP requests to the API endpoints.

For detailed API documentation, see the internal packages:
  - lab3/internal/board: Core game board ADT with thread safety
  - lab3/internal/commands: HTTP command interface layer
  - lab3/internal/server: HTTP server implementation with CORS
\n## Package: board\n
package board // import "lab3/internal/board"

Package board implements the Memory Scramble game board ADT

TYPES

type Board struct {
	// Has unexported fields.
}
    Board represents the Memory Scramble game board

    Abstraction Function:

        AF(cards, players, mutex) = A Memory Scramble game board where:
        - cards[r][c] represents the card at position (r,c), or nil if empty
        - players maps player IDs to their current game state
        - mutex protects concurrent access to the board state

    Representation Invariant:
      - cards is a rectangular 2D array (all rows have same length)
      - cards[r][c] is nil iff there is no card at position (r,c)
      - For each player in players:
      - All positions in ControlledPos are valid board positions
      - All positions in ControlledPos have non-nil cards that are face-up
      - If Waiting is true, WaitingPos is a valid position with a face-up card
        controlled by another player
      - No card is controlled by more than one player
      - A player controls at most 2 cards at any time

    Safety from Rep Exposure:
      - cards array is never returned directly; only copies of card contents are
        returned
      - players map is never exposed; only individual player states are accessed
      - All public methods use mutex to ensure thread safety
      - Position structs are passed by value

func ParseFromFile(filename string) (*Board, error)
    ParseFromFile creates a new board by parsing the given file The file format
    is: ROWxCOLUMN\n followed by ROW*COLUMN lines with card content

func (b *Board) Flip(playerID string, row, col int) error
    Flip attempts to flip a card at the given position for the specified player
    Returns an error if the flip is invalid according to the game rules

func (b *Board) Look(playerID string) string
    Look returns the current state of the board from the specified player's
    perspective

func (b *Board) ReplaceCard(fromCard, toCard string) error
    ReplaceCard replaces all instances of fromCard content with toCard content

func (b *Board) Restart() error
    Restart resets the board to its initial state

func (b *Board) String() string

func (b *Board) WaitForCard(playerID string, row, col int) bool
    WaitForCard waits for a card to become available for the specified player
    Returns true if card becomes available, false if timeout

func (b *Board) Watch(playerID string) (string, error)
    Watch waits for the next change to the board for the given player

type Card struct {
	Content string
	FaceUp  bool
}
    Card represents a card with its content

type CardStatus string
    CardStatus represents the state of a card from a player's perspective

const (
	None CardStatus = "none" // Empty space
	Down CardStatus = "down" // Face-down card
	Up   CardStatus = "up"   // Face-up card controlled by another player
	My   CardStatus = "my"   // Face-up card controlled by this player
)
type PlayerState struct {
	ID            string
	ControlledPos []Position // Positions of cards controlled by this player
	Waiting       bool       // Whether player is waiting for a card
	WaitingPos    *Position  // Position player is waiting for (if any)
	WaitChannel   chan bool  // Channel for notifying when wait is over
}
    PlayerState tracks a player's current game state

type Position struct {
	Row, Col int
}
    Position represents a coordinate on the board

\n## Package: commands\n
package commands // import "lab3/internal/commands"

Package commands provides the API layer for Memory Scramble game operations

FUNCTIONS

func Flip(playerID, position string) (string, error)
    Flip attempts to flip a card at the given position for the specified player
    position should be in format "row,col" Returns updated board state on
    success, error on failure

func Look(playerID string) (string, error)
    Look returns the current state of the board from the specified player's
    perspective Returns board state in the format: ROWxCOL\n followed by card
    states

func Replace(playerID, fromCard, toCard string) (string, error)
    Replace applies a transformation to replace all instances of fromCard with
    toCard This is the map operation from the MIT specification

func Restart() (string, error)
    Restart resets the board to initial state

func SetBoard(b *board.Board)
    SetBoard sets the global game board instance

func Watch(playerID string) (string, error)
    Watch waits for the next change to the board and returns the updated state
    This implements the long-polling functionality

\n## Package: server\n
package server // import "lab3/internal/server"

Package server provides HTTP server for Memory Scramble game

TYPES

type Server struct {
	// Has unexported fields.
}
    Server represents the HTTP server for Memory Scramble

func New(b *board.Board) *Server
    New creates a new server instance with the given board

func (s *Server) Start(addr string) error
    Start starts the HTTP server on the given address

