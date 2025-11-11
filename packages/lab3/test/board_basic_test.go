package board_test

import (
	"io"
	"os"
	"strings"
	"testing"

	"lab3/internal/board"
)

// Test basic board functionality - parsing, looking, and core operations
// These tests cover the fundamental board operations that don't involve gameplay rules

// Test board parsing from file
func TestBasicParseFromFile(t *testing.T) {
	t.Run("Valid 2x2 board", func(t *testing.T) {
		content := "2x2\nX\nO\nX\nO\n"
		tmpFile, err := createBasicTempFile(content)
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile)

		b, err := board.ParseFromFile(tmpFile)
		if err != nil {
			t.Fatalf("ParseFromFile failed: %v", err)
		}

		result := b.Look("test_player")
		lines := strings.Split(strings.TrimSpace(result), "\n")

		if lines[0] != "2x2" {
			t.Errorf("Expected dimensions '2x2', got '%s'", lines[0])
		}

		if len(lines) != 5 {
			t.Errorf("Expected 5 lines, got %d", len(lines))
		}
	})

	t.Run("Invalid format - wrong card count", func(t *testing.T) {
		content := "2x2\nA\nB\n" // Missing 2 cards
		tmpFile, err := createBasicTempFile(content)
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile)

		_, err = board.ParseFromFile(tmpFile)
		if err == nil {
			t.Error("Expected error for wrong card count")
		}
	})

	t.Run("Invalid format - bad dimensions", func(t *testing.T) {
		content := "axb\nA\n"
		tmpFile, err := createBasicTempFile(content)
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile)

		_, err = board.ParseFromFile(tmpFile)
		if err == nil {
			t.Error("Expected error for invalid dimensions")
		}
	})
}

// Test basic board looking functionality
func TestBasicBoardLook(t *testing.T) {
	t.Run("Initial board state", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createBasicTestBoard(t, content)

		result := b.Look("player1")
		lines := strings.Split(strings.TrimSpace(result), "\n")

		if lines[0] != "1x1" {
			t.Errorf("Expected dimensions '1x1', got '%s'", lines[0])
		}

		if lines[1] != "down" {
			t.Errorf("Expected 'down', got '%s'", lines[1])
		}
	})

	t.Run("Multiple players see same initial state", func(t *testing.T) {
		content := "2x1\nA\nB\n"
		b := createBasicTestBoard(t, content)

		result1 := b.Look("player1")
		result2 := b.Look("player2")

		if result1 != result2 {
			t.Error("Different players should see same initial board state")
		}
	})
}

// Test restart functionality
func TestBasicRestartBoard(t *testing.T) {
	t.Run("Restart resets all game state", func(t *testing.T) {
		content := "2x1\nA\nB\n"

		// Create temporary file but don't delete it until after the test
		tmpFile, err := createBasicTempFile(content)
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile)

		// Parse board from file
		b, err := board.ParseFromFile(tmpFile)
		if err != nil {
			t.Fatalf("ParseFromFile failed: %v", err)
		}

		// Make some moves that modify the board
		b.Flip("player1", 0, 0)
		b.Flip("player1", 1, 0)

		// Verify the board is modified
		result := b.Look("player1")
		if !strings.Contains(result, "A") || !strings.Contains(result, "B") {
			t.Error("Board should have cards face up before restart")
		}

		// Restart - this should reload the entire board from the original file
		err = b.Restart()
		if err != nil {
			t.Errorf("Restart failed: %v", err)
		}

		// All cards should be face down (back to initial state)
		result = b.Look("player1")
		lines := strings.Split(strings.TrimSpace(result), "\n")

		if lines[1] != "down" || lines[2] != "down" {
			t.Error("All cards should be face down after restart")
		}

		// Player should not control any cards
		if strings.Contains(result, "my") {
			t.Error("Player should not control any cards after restart")
		}

		// Test that the board is truly reset - we can make the same moves again
		b.Flip("player1", 0, 0)
		result = b.Look("player1")
		if !strings.Contains(result, "A") {
			t.Error("After restart, should be able to flip cards normally again")
		}
	})
}

// Test performance with larger boards
func TestLargeBoard(t *testing.T) {
	t.Run("Very large board", func(t *testing.T) {
		// Create a larger board to test performance
		content := "4x4\n"
		for range 16 {
			content += "A\n"
		}

		b := createBasicTestBoard(t, content)

		result := b.Look("player1")
		if !strings.Contains(result, "4x4") {
			t.Error("Should handle larger boards")
		}

		// Should have 17 lines (1 dimension + 16 cards)
		lines := strings.Split(strings.TrimSpace(result), "\n")
		if len(lines) != 17 {
			t.Errorf("Expected 17 lines for 4x4 board, got %d", len(lines))
		}
	})
}

// Helper function to create a board for testing
func createBasicTestBoard(t *testing.T, content string) *board.Board {
	tmpFile, err := createBasicTempFile(content)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	b, err := board.ParseFromFile(tmpFile)
	if err != nil {
		t.Fatalf("ParseFromFile failed: %v", err)
	}

	return b
}

// Helper function to create a temporary file with given content
func createBasicTempFile(content string) (string, error) {
	tmpFile, err := os.CreateTemp("", "board_test_*.txt")
	if err != nil {
		return "", err
	}

	_, err = io.WriteString(tmpFile, content)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", err
	}

	err = tmpFile.Close()
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}
