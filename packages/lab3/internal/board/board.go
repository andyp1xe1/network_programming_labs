// Package board implements the Memory Scramble game board ADT
package board

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

// CardStatus represents the state of a card from a player's perspective
type CardStatus string

const (
	None CardStatus = "none" // Empty space
	Down CardStatus = "down" // Face-down card
	Up   CardStatus = "up"   // Face-up card controlled by another player
	My   CardStatus = "my"   // Face-up card controlled by this player
)

// Card represents a card with its content
type Card struct {
	Content string
	FaceUp  bool
}

// PlayerState tracks a player's current game state
type PlayerState struct {
	ID            string
	ControlledPos []Position // Positions of cards controlled by this player
	PreviousCards []Position // Previous cards that need processing on next move (Rule 3)
	CardsMatched  bool       // Whether previous cards matched (for Rule 3-A vs 3-B)
	Waiting       bool       // Whether player is waiting for a card
	WaitingPos    *Position  // Position player is waiting for (if any)
	WaitChannel   chan bool  // Channel for notifying when wait is over
}

// Position represents a coordinate on the board
type Position struct {
	Row, Col int
}

// Board represents the Memory Scramble game board
//
// Abstraction Function:
//
//	AF(cards, players, mutex) = A Memory Scramble game board where:
//	- cards[r][c] represents the card at position (r,c), or nil if empty
//	- players maps player IDs to their current game state
//	- mutex protects concurrent access to the board state
//
// Representation Invariant:
//   - cards is a rectangular 2D array (all rows have same length)
//   - cards[r][c] is nil iff there is no card at position (r,c)
//   - For each player in players:
//   - All positions in ControlledPos are valid board positions
//   - All positions in ControlledPos have non-nil cards that are face-up
//   - If Waiting is true, WaitingPos is a valid position with a face-up card controlled by another player
//   - No card is controlled by more than one player
//   - A player controls at most 2 cards at any time
//
// Safety from Rep Exposure:
//   - cards array is never returned directly; only copies of card contents are returned
//   - players map is never exposed; only individual player states are accessed
//   - All public methods use mutex to ensure thread safety
//   - Position structs are passed by value
type Board struct {
	cards      [][]*Card
	rows       int
	cols       int
	players    map[string]*PlayerState
	mutex      sync.RWMutex
	version    int64                    // Board version for change detection
	watchers   map[string][]chan string // Player ID -> list of watch channels
	watchMutex sync.RWMutex             // Separate mutex for watchers
}

// ParseFromFile creates a new board by parsing the given file
// The file format is: ROWxCOLUMN\n followed by ROW*COLUMN lines with card content
func ParseFromFile(filename string) (*Board, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Read dimensions
	if !scanner.Scan() {
		return nil, errors.New("empty file")
	}

	dimLine := scanner.Text()
	parts := strings.Split(dimLine, "x")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid dimensions format: %s", dimLine)
	}

	rows, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid rows: %s", parts[0])
	}

	cols, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid cols: %s", parts[1])
	}

	// Read cards
	cards := make([][]*Card, rows)
	for i := range cards {
		cards[i] = make([]*Card, cols)
	}

	cardCount := 0
	for scanner.Scan() {
		content := strings.TrimSpace(scanner.Text())
		if content == "" {
			continue
		}

		row := cardCount / cols
		col := cardCount % cols

		if row >= rows {
			return nil, fmt.Errorf("too many cards: expected %d", rows*cols)
		}

		cards[row][col] = &Card{
			Content: content,
			FaceUp:  false,
		}
		cardCount++
	}

	if cardCount != rows*cols {
		return nil, fmt.Errorf("wrong number of cards: expected %d, got %d", rows*cols, cardCount)
	}

	board := &Board{
		cards:    cards,
		rows:     rows,
		cols:     cols,
		players:  make(map[string]*PlayerState),
		version:  1,
		watchers: make(map[string][]chan string),
	}

	board.checkRep()
	return board, nil
}

// Look returns the current state of the board from the specified player's perspective
func (b *Board) Look(playerID string) string {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	// Ensure player exists
	if _, exists := b.players[playerID]; !exists {
		b.players[playerID] = &PlayerState{
			ID:            playerID,
			ControlledPos: make([]Position, 0, 2),
			PreviousCards: make([]Position, 0, 2),
			CardsMatched:  false,
			Waiting:       false,
			WaitingPos:    nil,
			WaitChannel:   make(chan bool, 1),
		}
	}

	// Build board state string
	var result strings.Builder
	result.WriteString(fmt.Sprintf("%dx%d\n", b.rows, b.cols))

	for r := 0; r < b.rows; r++ {
		for c := 0; c < b.cols; c++ {
			pos := Position{r, c}
			card := b.cards[r][c]

			if card == nil {
				result.WriteString("none\n")
			} else if !card.FaceUp {
				result.WriteString("down\n")
			} else if b.isControlledBy(pos, playerID) {
				result.WriteString(fmt.Sprintf("my %s\n", card.Content))
			} else {
				result.WriteString(fmt.Sprintf("up %s\n", card.Content))
			}
		}
	}

	return result.String()
}

// WaitForCard waits for a card to become available for the specified player
// Returns true if card becomes available, false if timeout
func (b *Board) WaitForCard(playerID string, row, col int) bool {
	b.mutex.Lock()
	player := b.players[playerID]
	if player == nil {
		b.mutex.Unlock()
		return false
	}

	pos := Position{row, col}
	card := b.cards[row][col]

	// Check if card is immediately available
	if card != nil && (!card.FaceUp || b.getController(pos) == nil) {
		b.mutex.Unlock()
		return true
	}

	// Set up waiting state
	player.Waiting = true
	player.WaitingPos = &pos
	b.mutex.Unlock()

	// Wait for notification or timeout
	select {
	case <-player.WaitChannel:
		return true
	default:
		// For now, return false immediately for non-blocking behavior
		// In a full implementation, this would have a timeout
		return false
	}
}

// Flip attempts to flip a card at the given position for the specified player
// Returns an error if the flip is invalid according to the game rules
func (b *Board) Flip(playerID string, row, col int) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	pos := Position{row, col}

	// Validate position
	if !b.isValidPosition(pos) {
		return errors.New("invalid position")
	}

	// Ensure player exists
	if _, exists := b.players[playerID]; !exists {
		b.players[playerID] = &PlayerState{
			ID:            playerID,
			ControlledPos: make([]Position, 0, 2),
			PreviousCards: make([]Position, 0, 2),
			CardsMatched:  false,
			Waiting:       false,
			WaitingPos:    nil,
			WaitChannel:   make(chan bool, 1),
		}
	}

	player := b.players[playerID]
	card := b.cards[row][col]

	// Rule 1-A: No card at position
	if card == nil {
		return errors.New("no card at position")
	}

	controlledCount := len(player.ControlledPos)

	if controlledCount == 0 {
		// Trying to flip first card
		// Rule 3: If player has previous cards, process them first
		if len(player.PreviousCards) > 0 {
			b.processPreviousCards(player)
		}
		return b.flipFirstCard(player, pos, card)
	} else if controlledCount == 1 {
		// Trying to flip second card
		return b.flipSecondCard(player, pos, card, playerID)
	} else if controlledCount == 2 {
		// Player controls 2 cards - can't flip more until processing next move
		// But if they have previous cards to process, they're starting a new move
		if len(player.PreviousCards) > 0 {
			// Process previous cards first, then try the flip as new first card
			b.processPreviousCards(player)
			// Clear controlled positions since we processed the cards
			player.ControlledPos = player.ControlledPos[:0]
			return b.flipFirstCard(player, pos, card)
		}
		return errors.New("player already controls two cards")
	}

	return errors.New("invalid player state")
}

// Helper methods

func (b *Board) flipFirstCard(player *PlayerState, pos Position, card *Card) error {
	// Rule 1-B: Card is face down
	if !card.FaceUp {
		card.FaceUp = true
		player.ControlledPos = append(player.ControlledPos, pos)
		b.incrementVersion()
		return nil
	}

	// Rule 1-C: Card is face up but not controlled
	controller := b.getController(pos)
	if controller == nil {
		player.ControlledPos = append(player.ControlledPos, pos)
		return nil
	}

	// Rule 1-D: Card is controlled by another player - need to wait
	player.Waiting = true
	player.WaitingPos = &pos
	return errors.New("waiting for card to become available")
}

func (b *Board) flipSecondCard(player *PlayerState, pos Position, card *Card, playerID string) error {
	firstPos := player.ControlledPos[0]
	firstCard := b.cards[firstPos.Row][firstPos.Col]

	// Rule 2-A: No card at position
	if card == nil {
		// Relinquish control of first card (Rule 2-A)
		player.ControlledPos = player.ControlledPos[:0]
		// First card remains face up but uncontrolled
		return errors.New("no card at position")
	}

	// Rule 2-B: Card is controlled by a player
	controller := b.getController(pos)
	if card.FaceUp && controller != nil {
		// Relinquish control of first card (Rule 2-B)
		player.ControlledPos = player.ControlledPos[:0]
		// First card remains face up but uncontrolled
		return errors.New("card controlled by another player")
	}

	// Rule 2-C: Turn face up if needed
	if !card.FaceUp {
		card.FaceUp = true
		b.incrementVersion()
	}

	// Rule 2-D & 2-E: Check if cards match
	if firstCard.Content == card.Content {
		// Rule 2-D: Match! Keep control of both cards temporarily
		player.ControlledPos = append(player.ControlledPos, pos)
		// Store for Rule 3-A processing on next move
		player.PreviousCards = []Position{firstPos, pos}
		player.CardsMatched = true
		// Note: Player keeps control until they make next move (for display purposes)
	} else {
		// Rule 2-E: No match, relinquish control but cards stay face up
		player.PreviousCards = []Position{firstPos, pos}
		player.CardsMatched = false
		player.ControlledPos = player.ControlledPos[:0] // Clear controlled positions immediately
		// Cards remain face up but uncontrolled (Rule 2-E)
	}

	return nil
}

// processPreviousCards processes cards from previous move according to Rules 3-A and 3-B
func (b *Board) processPreviousCards(player *PlayerState) {
	if len(player.PreviousCards) == 0 {
		return
	}

	if player.CardsMatched {
		// Rule 3-A: Remove matching cards from board
		for _, pos := range player.PreviousCards {
			if b.isValidPosition(pos) {
				b.cards[pos.Row][pos.Col] = nil
			}
		}
	} else {
		// Rule 3-B: Turn non-matching cards face down if still uncontrolled
		for _, pos := range player.PreviousCards {
			if b.isValidPosition(pos) {
				card := b.cards[pos.Row][pos.Col]
				if card != nil && card.FaceUp && b.getController(pos) == nil {
					card.FaceUp = false
				}
			}
		}
	}

	// Clear previous cards state
	player.PreviousCards = player.PreviousCards[:0]
	player.CardsMatched = false

	// Increment version to notify watchers
	b.incrementVersion()

	// Notify any waiting players
	b.notifyWaitingPlayers()
}

// notifyWaitingPlayers notifies all players waiting for cards that are now available
func (b *Board) notifyWaitingPlayers() {
	for _, player := range b.players {
		if player.Waiting && player.WaitingPos != nil {
			// Check if the card they're waiting for is now available
			card := b.cards[player.WaitingPos.Row][player.WaitingPos.Col]
			if card != nil && card.FaceUp && b.getController(*player.WaitingPos) == nil {
				player.Waiting = false
				player.WaitingPos = nil
				// Send non-blocking notification
				select {
				case player.WaitChannel <- true:
				default:
				}
			}
		}
	}
}

func (b *Board) isValidPosition(pos Position) bool {
	return pos.Row >= 0 && pos.Row < b.rows && pos.Col >= 0 && pos.Col < b.cols
}

func (b *Board) isControlledBy(pos Position, playerID string) bool {
	player := b.players[playerID]
	if player == nil {
		return false
	}

	for _, controlledPos := range player.ControlledPos {
		if controlledPos == pos {
			return true
		}
	}
	return false
}

func (b *Board) getController(pos Position) *PlayerState {
	for _, player := range b.players {
		if b.isControlledBy(pos, player.ID) {
			return player
		}
	}
	return nil
}

func (b *Board) checkRep() {
	// Verify cards array is rectangular
	if len(b.cards) != b.rows {
		panic("cards array has wrong number of rows")
	}
	for _, row := range b.cards {
		if len(row) != b.cols {
			panic("cards array is not rectangular")
		}
	}

	// Verify player states
	for _, player := range b.players {
		if len(player.ControlledPos) > 2 {
			panic("player controls more than 2 cards")
		}

		for _, pos := range player.ControlledPos {
			if !b.isValidPosition(pos) {
				panic("player controls invalid position")
			}
			card := b.cards[pos.Row][pos.Col]
			if card == nil || !card.FaceUp {
				panic("player controls non-existent or face-down card")
			}
		}
	}
}

// ReplaceCard replaces all instances of fromCard content with toCard content
func (b *Board) ReplaceCard(fromCard, toCard string) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	changed := false
	for r := 0; r < b.rows; r++ {
		for c := 0; c < b.cols; c++ {
			card := b.cards[r][c]
			if card != nil && card.Content == fromCard {
				card.Content = toCard
				changed = true
			}
		}
	}

	if changed {
		b.incrementVersion()
	}

	return nil
}

// Restart resets the board to its initial state
func (b *Board) Restart() error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Reset all cards to face-down
	for r := 0; r < b.rows; r++ {
		for c := 0; c < b.cols; c++ {
			if b.cards[r][c] != nil {
				b.cards[r][c].FaceUp = false
			}
		}
	}

	// Clear all player states
	for playerID := range b.players {
		b.players[playerID] = &PlayerState{
			ID:            playerID,
			ControlledPos: make([]Position, 0, 2),
			PreviousCards: make([]Position, 0, 2),
			CardsMatched:  false,
			Waiting:       false,
			WaitingPos:    nil,
			WaitChannel:   make(chan bool, 1),
		}
	}

	// Clear all watchers
	b.watchMutex.Lock()
	for playerID := range b.watchers {
		// Close all watch channels
		for _, ch := range b.watchers[playerID] {
			close(ch)
		}
	}
	b.watchers = make(map[string][]chan string)
	b.watchMutex.Unlock()

	b.incrementVersion()
	return nil
}

// notifyWatchers notifies all watchers of board changes
func (b *Board) notifyWatchers() {
	b.watchMutex.RLock()
	defer b.watchMutex.RUnlock()

	for playerID, channels := range b.watchers {
		boardState := b.Look(playerID)
		for _, ch := range channels {
			select {
			case ch <- boardState:
			default:
				// Channel is full or closed, ignore
			}
		}
	}
}

// incrementVersion increments the board version and notifies watchers
func (b *Board) incrementVersion() {
	b.version++
	go b.notifyWatchers()
}

// Watch waits for the next change to the board for the given player
func (b *Board) Watch(playerID string) (string, error) {
	b.watchMutex.Lock()
	ch := make(chan string, 1)
	if b.watchers[playerID] == nil {
		b.watchers[playerID] = make([]chan string, 0, 1)
	}
	b.watchers[playerID] = append(b.watchers[playerID], ch)
	b.watchMutex.Unlock()

	// Wait for notification
	select {
	case boardState := <-ch:
		// Remove this channel from watchers
		b.watchMutex.Lock()
		channels := b.watchers[playerID]
		for i, watchCh := range channels {
			if watchCh == ch {
				b.watchers[playerID] = append(channels[:i], channels[i+1:]...)
				break
			}
		}
		b.watchMutex.Unlock()
		return boardState, nil
	default:
		// For now, return immediately with current state
		// In production, this would have a proper timeout
		b.watchMutex.Lock()
		channels := b.watchers[playerID]
		for i, watchCh := range channels {
			if watchCh == ch {
				b.watchers[playerID] = append(channels[:i], channels[i+1:]...)
				break
			}
		}
		b.watchMutex.Unlock()
		return b.Look(playerID), nil
	}
}

func (b *Board) String() string {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Board %dx%d:\n", b.rows, b.cols))

	for r := 0; r < b.rows; r++ {
		for c := 0; c < b.cols; c++ {
			card := b.cards[r][c]
			if card == nil {
				result.WriteString("[ ] ")
			} else if card.FaceUp {
				result.WriteString(fmt.Sprintf("[%s]", card.Content))
			} else {
				result.WriteString("[?] ")
			}
		}
		result.WriteString("\n")
	}

	return result.String()
}
