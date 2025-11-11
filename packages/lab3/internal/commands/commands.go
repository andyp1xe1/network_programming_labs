// Package commands provides the API layer for Memory Scramble game operations.
//
// Acts as the interface between HTTP server and board ADT, handling
// parameter validation and coordinate conversion.
package commands

import (
	"errors"
	"strconv"
	"strings"

	"lab3/internal/board"
)

var gameBoard *board.Board

// SetBoard sets the global game board instance.
func SetBoard(b *board.Board) {
	gameBoard = b
}

// Look returns the current board state from the specified player's perspective.
func Look(playerID string) (string, error) {
	if gameBoard == nil {
		return "", errors.New("board not initialized")
	}
	return gameBoard.Look(playerID), nil
}

// Flip attempts to flip a card at the given position for the specified player.
// Position should be in format "row,col".
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

// Replace replaces all instances of fromCard with toCard.
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

// Watch waits for the next board change and returns the updated state.
func Watch(playerID string) (string, error) {
	if gameBoard == nil {
		return "", errors.New("board not initialized")
	}

	return gameBoard.Watch(playerID)
}

// Restart resets the board to its initial state.
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
