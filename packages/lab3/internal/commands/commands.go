// Package commands provides the API layer for Memory Scramble game operations
package commands

import (
	"errors"
	"strconv"
	"strings"

	"lab3/internal/board"
)

var gameBoard *board.Board

// SetBoard sets the global game board instance
func SetBoard(b *board.Board) {
	gameBoard = b
}

// Look returns the current state of the board from the specified player's perspective
// Returns board state in the format: ROWxCOL\n followed by card states
func Look(playerID string) (string, error) {
	if gameBoard == nil {
		return "", errors.New("board not initialized")
	}
	return gameBoard.Look(playerID), nil
}

// Flip attempts to flip a card at the given position for the specified player
// position should be in format "row,col"
// Returns updated board state on success, error on failure
func Flip(playerID, position string) (string, error) {
	if gameBoard == nil {
		return "", errors.New("board not initialized")
	}

	// Parse position
	parts := strings.Split(position, ",")
	if len(parts) != 2 {
		return "", errors.New("invalid position format, expected 'row,col'")
	}

	row, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", errors.New("invalid row number")
	}

	col, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", errors.New("invalid column number")
	}

	// Attempt flip
	err = gameBoard.Flip(playerID, row, col)
	if err != nil {
		return "", err
	}

	// Return updated board state
	return gameBoard.Look(playerID), nil
}

// Replace applies a transformation to replace all instances of fromCard with toCard
// This is the map operation from the MIT specification
func Replace(playerID, fromCard, toCard string) (string, error) {
	if gameBoard == nil {
		return "", errors.New("board not initialized")
	}

	err := gameBoard.ReplaceCard(fromCard, toCard)
	if err != nil {
		return "", err
	}

	return gameBoard.Look(playerID), nil
}

// Watch waits for the next change to the board and returns the updated state
// This implements the long-polling functionality
func Watch(playerID string) (string, error) {
	if gameBoard == nil {
		return "", errors.New("board not initialized")
	}

	return gameBoard.Watch(playerID)
}

// Restart resets the board to initial state
func Restart() (string, error) {
	if gameBoard == nil {
		return "", errors.New("board not initialized")
	}

	err := gameBoard.Restart()
	if err != nil {
		return "", err
	}

	return "restarted", nil
}
