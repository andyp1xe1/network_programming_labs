// Package board implements the Memory Scramble game board ADT.
//
// Provides thread-safe operations for multi-player card matching games
// following MIT 6.102 Problem Set 4 specification.
//
// See docs/API_DOCUMENTATION.md for complete ADT specifications including
// representation invariants and safety from rep exposure arguments.
package board

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
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

// Board represents the Memory Scramble game board.
//
// Provides thread-safe concurrent operations for multiple players.
// Zero value is not usable; create boards using ParseFromFile.
//
// See docs/API_DOCUMENTATION.md for complete ADT specifications.
type Board struct {
	cards      [][]*Card
	rows       int
	cols       int
	players    map[string]*PlayerState
	mutex      sync.RWMutex
	version    int64                    // Board version for change detection
	watchers   map[string][]chan string // Player ID -> list of watch channels
	watchMutex sync.RWMutex             // Separate mutex for watchers
	filename   string                   // Original board file for restart functionality
}

// ParseFromFile creates a new board by parsing the given file.
//
// File format: "ROWxCOL" on first line, followed by ROW*COL lines of card content.
// Returns an error if file format is invalid.
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
		filename: filename,
	}

	board.checkRep()
	return board, nil
}

// Look returns the current board state from the specified player's perspective.
func (b *Board) Look(playerID string) string {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	// Log the look action
	// log.Printf("[GAME_LOG] %s | LOOK | Player: %s", time.Now().Format("15:04:05.000"), playerID)

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
				content := card.Content
				if content == "" {
					content = "_EMPTY_"
				}
				result.WriteString(fmt.Sprintf("my %s\n", content))
			} else {
				content := card.Content
				if content == "" {
					content = "_EMPTY_"
				}
				result.WriteString(fmt.Sprintf("up %s\n", content))
			}
		}
	}

	return result.String()
}

// WaitForCard waits for a card to become available for the specified player.
// Returns true if card becomes available, false if timeout.
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

// Flip attempts to flip a card at the given position for the specified player.
// Returns an error if the flip violates game rules.
func (b *Board) Flip(playerID string, row, col int) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	pos := Position{row, col}

	// Log the flip attempt with current board state
	log.Printf("[GAME_LOG] %s | FLIP_START | Player: %s | Position: (%d,%d) | Board_State: %s",
		time.Now().Format("15:04:05.000"), playerID, row, col, b.getBoardStateForLog())

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

	controlledCount := len(player.ControlledPos)

	// Rule 3: If player has previous cards, process them first before any new move
	if len(player.PreviousCards) > 0 {
		b.processPreviousCards(player)
		// Clear controlled positions since we processed the cards
		player.ControlledPos = player.ControlledPos[:0]
		controlledCount = 0 // Update controlled count after processing
	}

	// Rule 1-A: No card at position (after processing previous cards if any)
	if card == nil {
		return errors.New("no card at position")
	}

	switch controlledCount {
	case 0:
		// Trying to flip first card
		err := b.flipFirstCard(player, pos, card)
		log.Printf("[GAME_LOG] %s | FLIP_FIRST_CARD | Player: %s | Position: (%d,%d) | Result: %v | Board_State: %s",
			time.Now().Format("15:04:05.000"), player.ID, pos.Row, pos.Col, err, b.getBoardStateForLog())
		return err
	case 1:
		// Trying to flip second card
		err := b.flipSecondCard(player, pos, card, playerID)
		log.Printf("[GAME_LOG] %s | FLIP_SECOND_CARD | Player: %s | Position: (%d,%d) | Result: %v | Board_State: %s",
			time.Now().Format("15:04:05.000"), player.ID, pos.Row, pos.Col, err, b.getBoardStateForLog())
		return err
	case 2:
		// Player controls 2 cards - this should not happen after processing previous cards
		err := errors.New("player already controls two cards")
		log.Printf("[GAME_LOG] %s | FLIP_ERROR | Player: %s | Error: %s",
			time.Now().Format("15:04:05.000"), player.ID, err.Error())
		return err
	}

	err := errors.New("invalid player state")
	log.Printf("[GAME_LOG] %s | FLIP_ERROR | Player: %s | Error: %s",
		time.Now().Format("15:04:05.000"), player.ID, err.Error())
	return err
}

// Helper methods

func (b *Board) flipFirstCard(player *PlayerState, pos Position, card *Card) error {
	// Check if player is already waiting for this specific card and now controls it
	if player.Waiting && player.WaitingPos != nil && *player.WaitingPos == pos {
		// Check if we already gained control through notifyWaitingPlayers
		if b.isControlledBy(pos, player.ID) {
			// We already have control! Clear waiting state
			player.Waiting = false
			player.WaitingPos = nil
			return nil
		}
		// Still waiting, check if it's now available
		controller := b.getController(pos)
		if card.FaceUp && controller == nil {
			// Card is now available! Give control and clear waiting state
			player.Waiting = false
			player.WaitingPos = nil
			player.ControlledPos = append(player.ControlledPos, pos)
			return nil
		}
	}

	// Rule 1-B: Card is face down
	if !card.FaceUp {
		card.FaceUp = true
		player.ControlledPos = append(player.ControlledPos, pos)
		b.incrementVersion()
		log.Printf("[GAME_LOG] %s | RULE_1B | Player: %s | Card: %s | Action: Flipped face up and gained control",
			time.Now().Format("15:04:05.000"), player.ID, card.Content)
		return nil
	}

	// Rule 1-C: Card is face up but not controlled
	controller := b.getController(pos)
	if controller == nil {
		player.ControlledPos = append(player.ControlledPos, pos)
		log.Printf("[GAME_LOG] %s | RULE_1C | Player: %s | Card: %s | Action: Gained control of uncontrolled card",
			time.Now().Format("15:04:05.000"), player.ID, card.Content)
		return nil
	}

	// Rule 1-D: Card is controlled by another player - set waiting state
	player.Waiting = true
	player.WaitingPos = &pos
	log.Printf("[GAME_LOG] %s | RULE_1D | Player: %s | Card: %s | Controller: %s | Action: Waiting for card",
		time.Now().Format("15:04:05.000"), player.ID, card.Content, controller.ID)
	return errors.New("waiting for card to become available")
}

func (b *Board) flipSecondCard(player *PlayerState, pos Position, card *Card, playerID string) error {
	firstPos := player.ControlledPos[0]
	firstCard := b.cards[firstPos.Row][firstPos.Col]

	// Check if first card still exists (could have been removed by another player)
	if firstCard == nil {
		// First card has been removed, clear player's controlled positions
		player.ControlledPos = player.ControlledPos[:0]
		return errors.New("first card no longer exists")
	}

	// Rule 2-A: No card at position
	if card == nil {
		// Relinquish control of first card (Rule 2-A)
		player.ControlledPos = player.ControlledPos[:0]
		// Store first card for Rule 3-B processing on next move
		player.PreviousCards = []Position{firstPos}
		player.CardsMatched = false
		// First card remains face up but uncontrolled
		b.notifyWaitingPlayers() // Notify waiting players
		log.Printf("[GAME_LOG] %s | RULE_2A | Player: %s | FirstCard: %s | Action: No card at second position, stored first card for Rule 3-B",
			time.Now().Format("15:04:05.000"), player.ID, firstCard.Content)
		return errors.New("no card at position")
	}

	// Rule 2-B: Card is controlled by a player
	controller := b.getController(pos)
	if card.FaceUp && controller != nil {
		// Relinquish control of first card (Rule 2-B)
		player.ControlledPos = player.ControlledPos[:0]
		// Store first card for Rule 3-B processing on next move
		player.PreviousCards = []Position{firstPos}
		player.CardsMatched = false
		log.Printf("[GAME_LOG] %s | RULE_2B | Player: %s | FirstCard: %s | SecondCard: %s | Controller: %s | Action: Second card controlled by another player, stored first card for Rule 3-B",
			time.Now().Format("15:04:05.000"), player.ID, firstCard.Content, card.Content, controller.ID)
		// First card remains face up but uncontrolled
		b.notifyWaitingPlayers() // Notify waiting players
		return errors.New("card controlled by another player")
	}

	// Rule 2-C: Turn face up if needed
	if !card.FaceUp {
		card.FaceUp = true
		b.incrementVersion()
		log.Printf("[GAME_LOG] %s | RULE_2C | Player: %s | Card: %s | Action: Flipped second card face up",
			time.Now().Format("15:04:05.000"), player.ID, card.Content)
	}

	// Rule 2-D & 2-E: Check if cards match
	if firstCard.Content == card.Content {
		// Rule 2-D: Match! Keep control of both cards temporarily
		player.ControlledPos = append(player.ControlledPos, pos)
		// Store for Rule 3-A processing on next move
		player.PreviousCards = []Position{firstPos, pos}
		player.CardsMatched = true
		// Note: Player keeps control until they make next move (for display purposes)
		log.Printf("[GAME_LOG] %s | RULE_2D | Player: %s | Cards: %s,%s | Action: MATCH! Keeping control, will remove on next move",
			time.Now().Format("15:04:05.000"), player.ID, firstCard.Content, card.Content)
	} else {
		// Rule 2-E: No match, relinquish control but cards stay face up
		player.PreviousCards = []Position{firstPos, pos}
		player.CardsMatched = false
		player.ControlledPos = player.ControlledPos[:0] // Clear controlled positions immediately
		// Cards remain face up but uncontrolled (Rule 2-E)
		b.notifyWaitingPlayers() // Notify waiting players
		log.Printf("[GAME_LOG] %s | RULE_2E | Player: %s | Cards: %s,%s | Action: NO MATCH, relinquished control but cards stay face up",
			time.Now().Format("15:04:05.000"), player.ID, firstCard.Content, card.Content)
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
		var removedCards []string
		for _, pos := range player.PreviousCards {
			if b.isValidPosition(pos) {
				card := b.cards[pos.Row][pos.Col]
				if card != nil {
					removedCards = append(removedCards, card.Content)
				}
				b.cards[pos.Row][pos.Col] = nil
			}
		}
		log.Printf("[GAME_LOG] %s | RULE_3A | Player: %s | Cards: %v | Action: REMOVED matching cards from board",
			time.Now().Format("15:04:05.000"), player.ID, removedCards)
	} else {
		// Rule 3-B: Turn non-matching cards face down if still uncontrolled
		var flippedCards []string
		for _, pos := range player.PreviousCards {
			if b.isValidPosition(pos) {
				card := b.cards[pos.Row][pos.Col]
				if card != nil && card.FaceUp && b.getController(pos) == nil {
					flippedCards = append(flippedCards, card.Content)
					card.FaceUp = false
				}
			}
		}
		log.Printf("[GAME_LOG] %s | RULE_3B | Player: %s | Cards: %v | Action: Flipped non-matching cards face down",
			time.Now().Format("15:04:05.000"), player.ID, flippedCards)
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
			if card != nil && (!card.FaceUp || b.getController(*player.WaitingPos) == nil) {
				// Automatically grant control to the waiting player
				player.ControlledPos = append(player.ControlledPos, *player.WaitingPos)
				player.Waiting = false
				player.WaitingPos = nil

				// If card was face down, turn it face up
				if !card.FaceUp {
					card.FaceUp = true
					b.incrementVersion()
				}

				// Send notification that the card is now controlled
				select {
				case player.WaitChannel <- true:
				default:
				}
			}
		}
	}
}

// isValidPosition checks if the given position is within board boundaries.
func (b *Board) isValidPosition(pos Position) bool {
	return pos.Row >= 0 && pos.Row < b.rows && pos.Col >= 0 && pos.Col < b.cols
}

// isControlledBy checks if the given position is controlled by the specified player.
func (b *Board) isControlledBy(pos Position, playerID string) bool {
	player := b.players[playerID]
	if player == nil {
		return false
	}

	return slices.Contains(player.ControlledPos, pos)
}

// getController returns the player who controls the given position, or nil if uncontrolled.
func (b *Board) getController(pos Position) *PlayerState {
	for _, player := range b.players {
		if b.isControlledBy(pos, player.ID) {
			return player
		}
	}
	return nil
}

// checkRep verifies the representation invariant.
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

// Restart resets the board to its initial state by reloading from the original file.
// This is equivalent to restarting the entire program.
func (b *Board) Restart() error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Close all existing watchers before reloading
	b.watchMutex.Lock()
	for playerID := range b.watchers {
		for _, ch := range b.watchers[playerID] {
			close(ch)
		}
	}
	b.watchMutex.Unlock()

	// Reload the board from the original file
	newBoard, err := ParseFromFile(b.filename)
	if err != nil {
		return fmt.Errorf("failed to restart board: %w", err)
	}

	// Replace all board state with fresh data from file
	b.cards = newBoard.cards
	b.rows = newBoard.rows
	b.cols = newBoard.cols
	b.players = make(map[string]*PlayerState)
	b.version = 1
	b.watchers = make(map[string][]chan string)
	// Keep the same filename for future restarts

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

// String returns a human-readable representation of the board state.
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

// getBoardStateForLog returns a compact representation of the board state for logging
func (b *Board) getBoardStateForLog() string {
	var result strings.Builder

	// Add board dimensions
	result.WriteString(fmt.Sprintf("(%dx%d)", b.rows, b.cols))

	// Add card states in compact format
	result.WriteString(" Cards:")
	for r := 0; r < b.rows; r++ {
		for c := 0; c < b.cols; c++ {
			card := b.cards[r][c]
			if card == nil {
				result.WriteString(" _")
			} else if card.FaceUp {
				// Find who controls this card
				controller := ""
				for playerID, player := range b.players {
					for _, pos := range player.ControlledPos {
						if pos.Row == r && pos.Col == c {
							controller = playerID
							break
						}
					}
					if controller != "" {
						break
					}
				}
				if controller != "" {
					result.WriteString(fmt.Sprintf(" %s[%s]", card.Content, controller))
				} else {
					result.WriteString(fmt.Sprintf(" %s[uncontrolled]", card.Content))
				}
			} else {
				result.WriteString(" ?")
			}
		}
	}

	// Add player states
	result.WriteString(" Players:")
	for playerID, player := range b.players {
		result.WriteString(fmt.Sprintf(" %s(ctrl:%d,prev:%d,wait:%v)",
			playerID, len(player.ControlledPos), len(player.PreviousCards), player.Waiting))
	}

	return result.String()
}
