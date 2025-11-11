# Memory Scramble Design and API Documentation

## Module Architecture

This implementation follows the required three-layer architecture for the Memory Scramble game server:

```
┌─────────────────────────────────────────────────┐
│                  Server Layer                   │
│             (internal/server)                   │
│  HTTP routing, CORS, request/response handling  │
└─────────────────┬───────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────┐
│                Commands Layer                   │
│              (internal/commands)                │
│    API interface, parameter validation,         │
│         coordinate conversion                   │
└─────────────────┬───────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────┐
│                 Board Layer                     │
│               (internal/board)                  │
│     Core game ADT, thread safety, rule         │
│              enforcement                        │
└─────────────────────────────────────────────────┘
```

### Commands Module Structure Compliance

The **commands** module serves as the required intermediate layer between the HTTP server and the board ADT, providing:

1. **API Interface Abstraction**: All board operations are exposed through simple function calls
2. **Parameter Validation**: Position parsing and format validation 
3. **Error Translation**: Board-level errors converted to appropriate HTTP responses
4. **State Management**: Global board instance management with `SetBoard()` 

**Required Commands Module Functions:**
- `Look(playerID string) (string, error)` - Board state queries
- `Flip(playerID, position string) (string, error)` - Card flip operations  
- `Replace(playerID, fromCard, toCard string) (string, error)` - Card replacement
- `Watch(playerID string) (string, error)` - Long-polling board changes
- `Restart() (string, error)` - Game reset functionality

## ADT Specifications

### Board ADT (internal/board)

#### Abstraction Function
```
AF(cards, rows, cols, players, mutex, version, watchers, watchMutex, filename) = 
  A Memory Scramble game board representing a rows×cols grid where:
  - cards[r][c] represents the card at position (r,c), or nil if empty
  - players maps player IDs to their current game state and controlled positions  
  - mutex provides thread-safe access to all mutable board state
  - version tracks board changes for efficient watcher notifications
  - watchers maps player IDs to channels for real-time update delivery
  - watchMutex protects concurrent watcher registration/removal
  - filename stores the original board file path for restart functionality
```

#### Representation Invariant
```
RI(cards, rows, cols, players, mutex, version, watchers, watchMutex, filename) = 
  cards != null ∧ len(cards) = rows ∧
  (∀i ∈ [0, rows): len(cards[i]) = cols) ∧
  filename != "" ∧
  (∀p ∈ players: 
    len(p.ControlledPos) ≤ 2 ∧
    (∀pos ∈ p.ControlledPos: 
      0 ≤ pos.Row < rows ∧ 0 ≤ pos.Col < cols ∧
      cards[pos.Row][pos.Col] != nil ∧
      cards[pos.Row][pos.Col].FaceUp = true)) ∧
  (∀pos: |{p ∈ players | pos ∈ p.ControlledPos}| ≤ 1)
```

**Note**: The above invariants are actively checked by `checkRep()`. Additional logical 
invariants exist but are not programmatically verified:
- `version > 0` and increments on changes  
- Watcher channels are non-nil when created
- Player IDs are non-empty strings
- PreviousCards contains valid positions from completed moves

#### Safety from Rep Exposure
The Board ADT prevents representation exposure through the following mechanisms:

1. **Private Fields**: All representation fields (`cards`, `players`, `mutex`, `filename`, etc.) are unexported
2. **Defensive Copying**: 
   - `Look()` method constructs new strings rather than exposing card array
   - Position structs are passed by value, not reference  
   - Card contents are copied to response strings, never referenced directly
3. **Controlled Access**:
   - No public access to `cards` array - all access through validated methods
   - `players` map never exposed - only individual player states accessed through controlled operations
   - Internal helper methods (`getController`, `isControlledBy`) prevent direct access to player control lists
4. **Thread Safety**:  
   - All public methods acquire appropriate mutex locks before accessing representation
   - Concurrent modifications prevented by proper locking discipline
   - Read operations use `RLock()` for safe concurrent access
5. **Immutable Interfaces**: Public methods return strings and errors only, never references to internal state

### PlayerState ADT

#### Specification
```go
type PlayerState struct {
    ID            string     // Unique player identifier  
    ControlledPos []Position // Cards currently controlled (0-2 cards max)
    PreviousCards []Position // Cards from last move awaiting Rule 3 processing
    CardsMatched  bool       // True if PreviousCards were matching (Rule 3-A vs 3-B)
    Waiting       bool       // True if player is waiting for card availability
    WaitingPos    *Position  // Position player is waiting for (if Waiting=true)
    WaitChannel   chan bool  // Notification channel for wait completion
}
```

#### Representation Invariant
```
RI(PlayerState) = 
  ID != "" ∧
  len(ControlledPos) ≤ 2 ∧
  len(PreviousCards) ≤ 2 ∧
  (Waiting = true ⟹ WaitingPos != nil) ∧
  WaitChannel != nil
```
Note: WaitingPos may be non-nil when Waiting = false (not enforced by checkRep)

### Card ADT  

#### Specification
```go
type Card struct {
    Content string // Card content/label
    FaceUp  bool   // True if card is face-up, false if face-down
}
```

#### Representation Invariant
```
RI(Card) = true
```
(No constraints on Content - empty strings are allowed via ReplaceCard)

## Method Specifications

### Board Methods

#### ParseFromFile
```go
func ParseFromFile(filename string) (*Board, error)
```
**Requires:** `filename` refers to a readable file with valid board format  
**Effects:** Creates new Board instance from file contents and stores filename for restart functionality  
**Returns:** New Board with cards initialized face-down and filename field set, or error if file invalid  
**Format:** First line "ROWxCOL", followed by ROW×COL lines of card content

#### Look  
```go 
func (b *Board) Look(playerID string) string
```
**Requires:** `playerID != ""`  
**Effects:** If player doesn't exist, creates new PlayerState for playerID  
**Returns:** Board state string in format "ROWSxCOLS\n" + card states  
**Thread Safety:** Uses RLock for safe concurrent read access

#### Flip
```go
func (b *Board) Flip(playerID string, row, col int) error  
```
**Requires:** `playerID != ""`, `0 ≤ row < b.rows`, `0 ≤ col < b.cols`  
**Effects:** 
- If no card at position: Rule 1-A violation
- If first card: applies Rules 1-B, 1-C, 1-D based on card/control state
- If second card: applies Rules 2-A through 2-E, sets up Rule 3 processing
- If player has PreviousCards: processes them via Rule 3-A/3-B first
**Modifies:** Card face-up state, player ControlledPos, PreviousCards, board version  
**Returns:** Error if rule violation, nil if successful  
**Thread Safety:** Uses exclusive Lock for atomic state modification

#### WaitForCard
```go
func (b *Board) WaitForCard(playerID string, row, col int) bool
```
**Requires:** `playerID != ""`, valid position coordinates  
**Effects:** Blocks if card not immediately available, sets player waiting state  
**Returns:** True if card becomes available, false on timeout  
**Thread Safety:** Uses Lock for state modification, channel for blocking

#### ReplaceCard  
```go
func (b *Board) ReplaceCard(fromCard, toCard string) error
```
**Requires:** `fromCard != ""`, `toCard != ""`  
**Effects:** Replaces all card contents matching `fromCard` with `toCard`  
**Modifies:** Card content, board version (if any changes made)  
**Returns:** Always nil (no error conditions)  
**Thread Safety:** Uses exclusive Lock for atomic multi-card update

#### Restart
```go  
func (b *Board) Restart() error
```
**Requires:** `b.filename` must represent a valid, readable file path  
**Effects:** Completely resets board to initial state by reloading from original file - equivalent to program restart  
**Modifies:** 
- Closes all existing watchers and creates fresh watchers map
- Reloads entire card array from `b.filename` (restores removed cards)
- Clears all player states completely (creates fresh players map)
- Resets board version to 1
- Preserves original filename for future restarts
**Returns:** Error if file cannot be read, nil if successful  
**Thread Safety:** Uses both mutex locks for atomic complete state replacement

#### Watch
```go
func (b *Board) Watch(playerID string) (string, error)  
```
**Requires:** `playerID != ""`  
**Effects:** Registers watcher channel, waits for board changes  
**Returns:** Board state when change occurs or immediate state if timeout  
**Thread Safety:** Uses watchMutex for channel management, coordinates with version updates

### Commands Module Methods

#### SetBoard
```go
func SetBoard(b *board.Board)
```
**Requires:** `b != nil`  
**Effects:** Sets global board instance for API operations  
**Modifies:** Global `gameBoard` variable  

#### Look
```go  
func Look(playerID string) (string, error)
```
**Requires:** `playerID != ""`, `gameBoard != nil`  
**Effects:** Delegates to `gameBoard.Look(playerID)`  
**Returns:** Board state string or "board not initialized" error

#### Flip
```go
func Flip(playerID, position string) (string, error)  
```
**Requires:** `playerID != ""`, `position` in "row,col" format, `gameBoard != nil`  
**Effects:** Parses position coordinates, delegates to `gameBoard.Flip()`  
**Returns:** Updated board state or error (parsing error or game rule violation)

#### Replace
```go
func Replace(playerID, fromCard, toCard string) (string, error)
```
**Requires:** `playerID != ""`, `fromCard != ""`, `toCard != ""`, `gameBoard != nil`  
**Effects:** Delegates to `gameBoard.ReplaceCard()`, returns updated state  
**Returns:** Board state after replacements or initialization error

#### Watch  
```go
func Watch(playerID string) (string, error)
```
**Requires:** `playerID != ""`, `gameBoard != nil`  
**Effects:** Delegates to `gameBoard.Watch()` for long-polling  
**Returns:** Board state when change occurs or initialization error

#### Restart
```go
func Restart() (string, error)  
```
**Requires:** `gameBoard != nil`  
**Effects:** Delegates to `gameBoard.Restart()`  
**Returns:** "restarted" confirmation or initialization error

### Server Module Methods

The server module provides HTTP endpoint handlers that extract parameters from URLs and delegate to the commands module. All handlers:

**Common Preconditions:** Valid HTTP request with proper URL structure  
**Common Effects:** Parse URL parameters, call corresponding commands function  
**Common Returns:** HTTP response with board state or error message  
**CORS:** All responses include `Access-Control-Allow-Origin: *` header

#### Error Handling
- **400 Bad Request**: Invalid URL format, missing parameters
- **409 Conflict**: Game rule violations (from board operations)  
- **500 Internal Server Error**: Initialization errors, file serving errors

## Game Rule Implementation

### Rule 1: First Card Flip
- **1-A**: `if card == nil` → return "no card at position" error
- **1-B**: `if !card.FaceUp` → set FaceUp=true, add to ControlledPos  
- **1-C**: `if card.FaceUp && getController(pos) == nil` → add to ControlledPos
- **1-D**: `if card.FaceUp && getController(pos) != nil` → set Waiting=true, return error

### Rule 2: Second Card Flip  
- **2-A**: `if card == nil` → clear ControlledPos, return error
- **2-B**: `if card.FaceUp && getController(pos) != nil` → clear ControlledPos, return error  
- **2-C**: `if !card.FaceUp` → set FaceUp=true
- **2-D**: `if firstCard.Content == card.Content` → add to ControlledPos, set PreviousCards, CardsMatched=true
- **2-E**: `if firstCard.Content != card.Content` → clear ControlledPos, set PreviousCards, CardsMatched=false

### Rule 3: Next Move Processing
- **3-A**: `if CardsMatched` → set cards to nil (remove from board)
- **3-B**: `if !CardsMatched` → set FaceUp=false (if uncontrolled)

## Thread Safety Guarantees

1. **Board Operations**: All board state modifications are atomic via mutex locks
2. **Player State**: Individual player states protected by board-level locking  
3. **Watcher Management**: Separate mutex prevents watcher corruption during notifications
4. **Version Control**: Atomic increment ensures consistent change detection
5. **Channel Operations**: Non-blocking sends prevent deadlock in notification system

## HTTP API Specification

### Endpoint Format
All game endpoints follow RESTful patterns with path parameters:
- `/look/{playerID}` - GET board state
- `/flip/{playerID}/{row},{col}` - POST card flip  
- `/replace/{playerID}/{fromCard}/{toCard}` - POST card replacement
- `/watch/{playerID}` - GET with long-polling
- `/restart` - POST game reset
- `/` - GET web interface

### Response Format
All successful game operations return board state as:
```
ROWSxCOLS
card_state_line_1  
card_state_line_2
...
card_state_line_N
```
Where N = ROWS×COLS and card states are: `none`, `down`, `up {content}`, `my {content}`
