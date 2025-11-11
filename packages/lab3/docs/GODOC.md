# Memory Scramble - Go Package Documentation

Generated on: Tuesday, November 11, 2025

## Package: main

**Import Path:** `lab3`

Package main implements the Memory Scramble game server.

Memory Scramble is a multi-player card matching game server that implements the MIT 6.102 Problem Set 4 specification. The server provides:

- Thread-safe concurrent gameplay for multiple players
- HTTP API endpoints for all game operations (look, flip, replace, watch, restart)  
- Web-based user interface with real-time updates
- Comprehensive game rule enforcement (Rules 1-A through 3-B)
- Board file configuration support

### Usage

```go
memory-scramble -port=8080 -board=boards/perfect.txt
```

The server starts an HTTP server on the specified port and serves both the game API and the web interface. Players can connect using the web interface or make direct HTTP requests to the API endpoints.

### Architecture Overview

The application is structured in three main internal packages:
- `lab3/internal/board`: Core game board ADT with thread safety
- `lab3/internal/commands`: HTTP command interface layer  
- `lab3/internal/server`: HTTP server implementation with CORS

---

## Package: board

**Import Path:** `lab3/internal/board`

Package board implements the Memory Scramble game board ADT.

### Types

#### type Board

```go
type Board struct {
    // Has unexported fields.
}
```

Board represents the Memory Scramble game board.

**Abstraction Function:**
```
AF(cards, players, mutex) = A Memory Scramble game board where:
- cards[r][c] represents the card at position (r,c), or nil if empty
- players maps player IDs to their current game state  
- mutex protects concurrent access to the board state
```

**Representation Invariant:**
- cards is a rectangular 2D array (all rows have same length)
- `cards[r][c]` is nil iff there is no card at position (r,c)
- For each player in players:
  - All positions in ControlledPos are valid board positions
  - All positions in ControlledPos have non-nil cards that are face-up
  - All positions in PreviousCards are valid board positions from completed moves
  - CardsMatched is true iff PreviousCards contains matching card contents
  - If Waiting is true, WaitingPos is a valid position with a face-up card controlled by another player
  - No card is controlled by more than one player
  - A player controls at most 2 cards at any time
  - PreviousCards contains at most 2 positions from the most recent completed move

**Safety from Rep Exposure:**
- cards array is never returned directly; only copies of card contents are returned
- players map is never exposed; only individual player states are accessed  
- All public methods use mutex to ensure thread safety
- Position structs are passed by value

### Functions

#### ParseFromFile
```go
func ParseFromFile(filename string) (*Board, error)
```
**Specification:**
- **Requires:** filename refers to a readable file with valid board format
- **Effects:** Creates new Board instance from file contents with all cards face-down
- **Returns:** New Board with initialized game state, or error if file invalid  
- **Format:** First line "ROWxCOL", followed by ROW×COL lines of card content

#### Flip  
```go
func (b *Board) Flip(playerID string, row, col int) error
```
**Specification:**
- **Requires:** playerID != "", 0 ≤ row < b.rows, 0 ≤ col < b.cols
- **Effects:** 
  - If no card at position: Rule 1-A violation
  - If first card: applies Rules 1-B, 1-C, 1-D based on card/control state  
  - If second card: applies Rules 2-A through 2-E, sets up Rule 3 processing
  - If player has PreviousCards: processes them via Rule 3-A/3-B first
- **Modifies:** Card face-up state, player ControlledPos, PreviousCards, board version
- **Returns:** Error if rule violation, nil if successful
- **Thread Safety:** Uses exclusive Lock for atomic state modification

#### Look
```go
func (b *Board) Look(playerID string) string  
```
**Specification:**
- **Requires:** playerID != ""
- **Effects:** If player doesn't exist, creates new PlayerState for playerID
- **Returns:** Board state string in format "ROWSxCOLS\n" + card states
- **Thread Safety:** Uses RLock for safe concurrent read access

#### ReplaceCard
```go
func (b *Board) ReplaceCard(fromCard, toCard string) error
```
**Specification:**
- **Requires:** fromCard != "", toCard != ""
- **Effects:** Replaces all card contents matching fromCard with toCard
- **Modifies:** Card content, board version (if any changes made)
- **Returns:** Always nil (no error conditions)
- **Thread Safety:** Uses exclusive Lock for atomic multi-card update

#### Restart
```go
func (b *Board) Restart() error
```
**Specification:**
- **Requires:** None
- **Effects:** Resets all cards to face-down, clears all player states and watchers
- **Modifies:** All card FaceUp states, all PlayerState fields, watchers map, version
- **Returns:** Always nil (no error conditions)
- **Thread Safety:** Uses both mutex locks for complete state reset

#### String
```go
func (b *Board) String() string
```
**Specification:**
- **Requires:** None
- **Effects:** Creates human-readable representation of current board state
- **Returns:** Multi-line string showing card positions and face-up status
- **Thread Safety:** Uses RLock for safe concurrent access

#### WaitForCard
```go
func (b *Board) WaitForCard(playerID string, row, col int) bool
```
**Specification:**
- **Requires:** playerID != "", valid position coordinates  
- **Effects:** Blocks if card not immediately available, sets player waiting state
- **Returns:** True if card becomes available, false on timeout
- **Thread Safety:** Uses Lock for state modification, channel for blocking

#### Watch
```go
func (b *Board) Watch(playerID string) (string, error)
```
**Specification:**
- **Requires:** playerID != ""
- **Effects:** Registers watcher channel, waits for board changes
- **Returns:** Board state when change occurs or immediate state if timeout
- **Thread Safety:** Uses watchMutex for channel management, coordinates with version updates

### Type Definitions

#### type Card

```go
type Card struct {
    Content string
    FaceUp  bool
}
```
Card represents a card with its content.

#### type CardStatus

```go
type CardStatus string
```
CardStatus represents the state of a card from a player's perspective.

**Constants:**
```go
const (
    None CardStatus = "none" // Empty space
    Down CardStatus = "down" // Face-down card  
    Up   CardStatus = "up"   // Face-up card controlled by another player
    My   CardStatus = "my"   // Face-up card controlled by this player
)
```

#### type PlayerState

```go
type PlayerState struct {
    ID            string
    ControlledPos []Position // Positions of cards controlled by this player
    PreviousCards []Position // Previous cards that need processing on next move (Rule 3)
    CardsMatched  bool       // Whether previous cards matched (for Rule 3-A vs 3-B)
    Waiting       bool       // Whether player is waiting for a card
    WaitingPos    *Position  // Position player is waiting for (if any)  
    WaitChannel   chan bool  // Channel for notifying when wait is over
}
```
PlayerState tracks a player's current game state including cards from previous moves that need processing according to Rules 3-A and 3-B.

#### type Position

```go
type Position struct {
    Row, Col int
}
```
Position represents a coordinate on the board.

---

## Package: commands

**Import Path:** `lab3/internal/commands`

Package commands provides the API layer for Memory Scramble game operations.

### Functions

#### SetBoard
```go
func SetBoard(b *board.Board)
```
**Specification:**
- **Requires:** b != nil
- **Effects:** Sets global board instance for API operations
- **Modifies:** Global gameBoard variable

#### Flip
```go
func Flip(playerID, position string) (string, error)
```
**Specification:**
- **Requires:** playerID != "", position in "row,col" format, gameBoard != nil
- **Effects:** Parses position coordinates, delegates to gameBoard.Flip()
- **Returns:** Updated board state or error (parsing error or game rule violation)
- **Position Format:** "row,col" where row,col are non-negative integers

#### Look
```go
func Look(playerID string) (string, error)
```
**Specification:**
- **Requires:** playerID != "", gameBoard != nil
- **Effects:** Delegates to gameBoard.Look(playerID)
- **Returns:** Board state string or "board not initialized" error
- **Format:** "ROWSxCOLS\n" followed by card state lines

#### Replace
```go
func Replace(playerID, fromCard, toCard string) (string, error)
```
**Specification:**
- **Requires:** playerID != "", fromCard != "", toCard != "", gameBoard != nil
- **Effects:** Delegates to gameBoard.ReplaceCard(), returns updated state
- **Returns:** Board state after replacements or initialization error
- **MIT Spec:** Implements the required "map" operation for card transformation

#### Restart
```go
func Restart() (string, error)
```
**Specification:**
- **Requires:** gameBoard != nil
- **Effects:** Delegates to gameBoard.Restart()
- **Returns:** "restarted" confirmation or initialization error

#### Watch
```go
func Watch(playerID string) (string, error)  
```
**Specification:**
- **Requires:** playerID != "", gameBoard != nil
- **Effects:** Delegates to gameBoard.Watch() for long-polling
- **Returns:** Board state when change occurs or initialization error
- **Blocking:** May block until board state changes or timeout occurs

---

## Package: server

**Import Path:** `lab3/internal/server`

Package server provides HTTP server for Memory Scramble game.

### Types

#### type Server

```go
type Server struct {
    // Has unexported fields.
}
```
Server represents the HTTP server for Memory Scramble.

### Functions

#### New
```go
func New(b *board.Board) *Server
```
**Specification:**
- **Requires:** b != nil
- **Effects:** Creates Server instance with HTTP route handlers, sets board in commands module
- **Modifies:** commands.gameBoard via SetBoard()  
- **Returns:** Configured Server ready to accept HTTP requests
- **Endpoints:** Registers handlers for /, /look/, /flip/, /replace/, /watch/, /restart

#### Start  
```go  
func (s *Server) Start(addr string) error
```
**Specification:**
- **Requires:** addr is valid network address (e.g., ":8080")
- **Effects:** Starts HTTP server listening on addr, blocks until server stops
- **Returns:** Error if server fails to start or during operation
- **Blocking:** Method blocks indefinitely serving HTTP requests

### HTTP Handler Specifications

All HTTP handlers follow common patterns:
- **CORS Headers:** All responses include Access-Control-Allow-Origin: *
- **Content-Type:** text/plain for API responses, text/html for web interface
- **Error Codes:** 
  - 400 Bad Request: Invalid URL format, missing parameters
  - 409 Conflict: Game rule violations from board operations
  - 500 Internal Server Error: Initialization errors, file serving errors

#### Route Patterns
- `GET /` → serves index.html web interface
- `GET /look/{playerID}` → returns board state for playerID
- `GET /flip/{playerID}/{row,col}` → flips card at position for playerID
- `GET /replace/{playerID}/{fromCard}/{toCard}` → replaces card content
- `GET /watch/{playerID}` → long-polling board state changes  
- `GET /restart` → resets game to initial state

---

## Testing

The package includes comprehensive test coverage in `/test/board_test.go` with 43 test cases covering:

- Board parsing and initialization
- Card flipping with MIT Memory Scramble rule enforcement (Rules 1-A through 3-B)
- Complex multi-player interaction scenarios
- Player state management including previous move processing
- Concurrent access safety and thread synchronization
- Board replacement operations (map functionality)
- Game restart functionality
- Commands API layer integration
- Edge case handling and error conditions

Run tests with:
```bash
go test ./test/...
```

## Build and Run

Build the application:
```bash
go build -o memory-scramble .
```

Run with custom configuration:
```bash
./memory-scramble -port=8080 -board=boards/perfect.txt
```
