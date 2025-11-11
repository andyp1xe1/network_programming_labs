package board_test

import (
	"io"
	"os"
	"strings"
	"testing"

	"lab3/internal/board"
	"lab3/internal/commands"
)

// Test Commands API layer
func TestCommandsAPILayer(t *testing.T) {
	t.Run("Commands Look function", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createCommandsTestBoard(t, content)
		commands.SetBoard(b)

		result, err := commands.Look("player1")
		if err != nil {
			t.Errorf("Commands.Look failed: %v", err)
		}

		if !strings.Contains(result, "1x1") {
			t.Error("Commands.Look should return board state")
		}
	})

	t.Run("Commands Flip function", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createCommandsTestBoard(t, content)
		commands.SetBoard(b)

		result, err := commands.Flip("player1", "0,0")
		if err != nil {
			t.Errorf("Commands.Flip failed: %v", err)
		}

		if !strings.Contains(result, "my A") {
			t.Error("Commands.Flip should return updated board state showing controlled card")
		}
	})

	t.Run("Commands Flip with invalid position format", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createCommandsTestBoard(t, content)
		commands.SetBoard(b)

		_, err := commands.Flip("player1", "invalid")
		if err == nil {
			t.Error("Expected error for invalid position format")
		}
	})

	t.Run("Commands Flip with out of bounds position", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createCommandsTestBoard(t, content)
		commands.SetBoard(b)

		_, err := commands.Flip("player1", "5,5")
		if err == nil {
			t.Error("Expected error for out of bounds position")
		}
	})

	t.Run("Commands Replace function", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createCommandsTestBoard(t, content)
		commands.SetBoard(b)

		result, err := commands.Replace("player1", "A", "X")
		if err != nil {
			t.Errorf("Commands.Replace failed: %v", err)
		}

		// Since card is face down, we can't directly see the replacement
		// But the function should succeed
		if !strings.Contains(result, "1x1") {
			t.Error("Commands.Replace should return board state")
		}
	})

	t.Run("Commands Replace with face-up card", func(t *testing.T) {
		content := "2x1\nA\nB\n"
		b := createCommandsTestBoard(t, content)
		commands.SetBoard(b)

		// Flip a card first
		commands.Flip("player1", "0,0")

		// Replace the A with X
		result, err := commands.Replace("player1", "A", "X")
		if err != nil {
			t.Errorf("Commands.Replace failed: %v", err)
		}

		// The card should now show X instead of A
		if !strings.Contains(result, "my X") {
			t.Error("Commands.Replace should show replaced card content")
		}
	})

	t.Run("Commands Watch function", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createCommandsTestBoard(t, content)
		commands.SetBoard(b)

		result, err := commands.Watch("player1")
		if err != nil {
			t.Errorf("Commands.Watch failed: %v", err)
		}

		if !strings.Contains(result, "1x1") {
			t.Error("Commands.Watch should return board state")
		}
	})

	t.Run("Commands Restart function", func(t *testing.T) {
		content := "1x1\nA\n"

		// Create temporary file but don't delete it until after the test
		tmpFile, err := createCommandsTempFile(content)
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile)

		// Parse board from file
		b, err := board.ParseFromFile(tmpFile)
		if err != nil {
			t.Fatalf("ParseFromFile failed: %v", err)
		}

		commands.SetBoard(b)

		result, err := commands.Restart()
		if err != nil {
			t.Errorf("Commands.Restart failed: %v", err)
		}

		if result != "restarted" {
			t.Errorf("Expected 'restarted', got '%s'", result)
		}
	})

	t.Run("Commands without board initialization", func(t *testing.T) {
		commands.SetBoard(nil)

		_, err := commands.Look("player1")
		if err == nil {
			t.Error("Expected error when board not initialized")
		}
	})
}

func TestCommandsErrorHandling(t *testing.T) {
	t.Run("Commands Flip with malformed coordinates", func(t *testing.T) {
		content := "2x2\nA\nB\nC\nD\n"
		b := createCommandsTestBoard(t, content)
		commands.SetBoard(b)

		testCases := []string{
			"1",     // Missing comma
			"1,",    // Missing second coordinate
			",1",    // Missing first coordinate
			"a,b",   // Non-numeric coordinates
			"1,2,3", // Too many coordinates
			"",      // Empty string
		}

		for _, pos := range testCases {
			_, err := commands.Flip("player1", pos)
			if err == nil {
				t.Errorf("Expected error for malformed position '%s'", pos)
			}
		}
	})

	t.Run("Commands Replace with empty parameters", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createCommandsTestBoard(t, content)
		commands.SetBoard(b)

		// Test empty old card
		_, err := commands.Replace("player1", "", "X")
		if err != nil {
			t.Error("Replace with empty old card should succeed (no-op)")
		}

		// Test empty new card (should work - removes the card content)
		_, err = commands.Replace("player1", "A", "")
		if err != nil {
			t.Error("Replace with empty new card should succeed")
		}
	})

	t.Run("Commands operations without board", func(t *testing.T) {
		commands.SetBoard(nil)

		testCases := []func() error{
			func() error { _, err := commands.Look("player1"); return err },
			func() error { _, err := commands.Flip("player1", "0,0"); return err },
			func() error { _, err := commands.Replace("player1", "A", "B"); return err },
			func() error { _, err := commands.Watch("player1"); return err },
			func() error { _, err := commands.Restart(); return err },
		}

		for i, testFunc := range testCases {
			err := testFunc()
			if err == nil {
				t.Errorf("Test case %d: Expected error when board not initialized", i)
			}
		}
	})
}

func TestCommandsIntegration(t *testing.T) {
	t.Run("Full game sequence through Commands API", func(t *testing.T) {
		content := "2x1\nA\nA\n"

		// Create temporary file
		tmpFile, err := createCommandsTempFile(content)
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile)

		// Initialize board
		b, err := board.ParseFromFile(tmpFile)
		if err != nil {
			t.Fatalf("ParseFromFile failed: %v", err)
		}
		commands.SetBoard(b)

		// Step 1: Look at initial board
		result, err := commands.Look("player1")
		if err != nil {
			t.Errorf("Initial Look failed: %v", err)
		}
		if !strings.Contains(result, "2x1") || !strings.Contains(result, "down") {
			t.Error("Initial board should show dimensions and face-down cards")
		}

		// Step 2: Flip first card
		result, err = commands.Flip("player1", "0,0")
		if err != nil {
			t.Errorf("First flip failed: %v", err)
		}
		if !strings.Contains(result, "my A") {
			t.Error("First flip should show controlled card")
		}

		// Step 3: Flip second matching card
		result, err = commands.Flip("player1", "1,0")
		if err != nil {
			t.Errorf("Second flip failed: %v", err)
		}
		if !strings.Contains(result, "my A") {
			t.Error("Matching cards should be controlled by player")
		}

		// Step 4: Start new move (should trigger Rule 3-A - remove matched cards)
		result, err = commands.Flip("player1", "0,0")
		if err != nil {
			t.Errorf("New move flip failed: %v", err)
		}

		// Cards should be removed
		if !strings.Contains(result, "none") {
			t.Error("Matched cards should be removed after new move")
		}

		// Step 5: Restart the game
		result, err = commands.Restart()
		if err != nil {
			t.Errorf("Restart failed: %v", err)
		}
		if result != "restarted" {
			t.Error("Restart should return 'restarted'")
		}

		// Step 6: Verify board is reset
		result, err = commands.Look("player1")
		if err != nil {
			t.Errorf("Look after restart failed: %v", err)
		}
		if !strings.Contains(result, "down") {
			t.Error("Board should be reset with face-down cards")
		}
	})

	t.Run("Multi-player Commands API interaction", func(t *testing.T) {
		content := "2x2\nA\nB\nC\nD\n"
		b := createCommandsTestBoard(t, content)
		commands.SetBoard(b)

		// Player1 flips a card
		result1, err := commands.Flip("player1", "0,0")
		if err != nil {
			t.Errorf("Player1 flip failed: %v", err)
		}

		// Player2 flips a different card
		result2, err := commands.Flip("player2", "1,1")
		if err != nil {
			t.Errorf("Player2 flip failed: %v", err)
		}

		// Both players should see each other's cards
		if !strings.Contains(result1, "my A") {
			t.Error("Player1 should control their card")
		}
		if !strings.Contains(result2, "my D") {
			t.Error("Player2 should control their card")
		}

		// Player1 tries to flip Player2's card - should get waiting error
		_, err = commands.Flip("player1", "1,1")
		if err == nil {
			t.Error("Player1 should get error trying to flip Player2's controlled card")
		}
	})
}

// Helper function to create a board for Commands testing
func createCommandsTestBoard(t *testing.T, content string) *board.Board {
	tmpFile, err := createCommandsTempFile(content)
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

// Helper function to create a temporary file for Commands testing
func createCommandsTempFile(content string) (string, error) {
	tmpFile, err := os.CreateTemp("", "commands_test_*.txt")
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
