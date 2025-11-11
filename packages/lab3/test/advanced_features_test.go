package board_test

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"lab3/internal/board"
)

// Test board replacement functionality (Map operation)
func TestReplaceCardFeature(t *testing.T) {
	t.Run("Replace all instances of a card", func(t *testing.T) {
		content := "2x2\nA\nB\nA\nC\n"
		b := createAdvancedTestBoard(t, content)

		err := b.ReplaceCard("A", "X")
		if err != nil {
			t.Errorf("ReplaceCard failed: %v", err)
		}

		result := b.Look("player1")

		// All A's should be replaced with X's
		if strings.Contains(result, " A") {
			t.Error("Should not contain any A cards after replacement")
		}
		if !strings.Contains(result, "down") {
			t.Error("Cards should still be face down after replacement")
		}
	})

	t.Run("Replace preserves card state", func(t *testing.T) {
		content := "2x1\nA\nB\n"
		b := createAdvancedTestBoard(t, content)

		// Flip first card
		b.Flip("player1", 0, 0)

		// Replace A with X
		b.ReplaceCard("A", "X")

		result := b.Look("player1")
		lines := strings.Split(strings.TrimSpace(result), "\n")

		// Card should still be controlled by player, but content changed
		if lines[1] != "my X" {
			t.Errorf("Expected 'my X' after replacement, got '%s'", lines[1])
		}
	})

	t.Run("Replace card that doesn't exist", func(t *testing.T) {
		content := "2x1\nA\nB\n"
		b := createAdvancedTestBoard(t, content)

		// Try to replace a card that doesn't exist
		err := b.ReplaceCard("Z", "X")
		if err != nil {
			t.Errorf("ReplaceCard should succeed even if card doesn't exist: %v", err)
		}

		// Board should remain unchanged
		result := b.Look("player1")
		if strings.Contains(result, "X") {
			t.Error("Board should not contain replacement card when original doesn't exist")
		}
	})

	t.Run("Replace with empty string (remove card content)", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createAdvancedTestBoard(t, content)

		// Flip the card first to see it
		b.Flip("player1", 0, 0)

		// Replace A with empty string
		err := b.ReplaceCard("A", "")
		if err != nil {
			t.Errorf("ReplaceCard with empty string failed: %v", err)
		}

		result := b.Look("player1")
		lines := strings.Split(strings.TrimSpace(result), "\n")

		// Card should now show empty content but still be controlled
		if !strings.HasPrefix(lines[1], "my ") {
			t.Error("Card should still be controlled by player after content replacement")
		}
	})

	t.Run("Replace multiple different cards", func(t *testing.T) {
		content := "2x2\nA\nB\nC\nD\n"
		b := createAdvancedTestBoard(t, content)

		// Replace multiple cards
		b.ReplaceCard("A", "X")
		b.ReplaceCard("C", "Y")

		// Flip cards to see the changes
		b.Flip("player1", 0, 0)
		b.Flip("player1", 1, 0)

		result := b.Look("player1")

		// Should see replaced content
		if !strings.Contains(result, "X") {
			t.Error("Should see replaced A -> X")
		}
		if strings.Contains(result, "A") {
			t.Error("Should not see original A after replacement")
		}
	})

	t.Run("Replace affects matching logic", func(t *testing.T) {
		content := "2x1\nA\nB\n"
		b := createAdvancedTestBoard(t, content)

		// Replace B with A to create a matching pair
		b.ReplaceCard("B", "A")

		// Now flip both cards - they should match
		b.Flip("player1", 0, 0)
		b.Flip("player1", 1, 0)

		result := b.Look("player1")
		lines := strings.Split(strings.TrimSpace(result), "\n")

		// Both cards should be controlled (matching pair)
		if lines[1] != "my A" || lines[2] != "my A" {
			t.Error("Replaced cards should create matching pair")
		}
	})

	t.Run("Replace cards in complex game state", func(t *testing.T) {
		content := "3x1\nA\nB\nA\n"
		b := createAdvancedTestBoard(t, content)

		// Player1 controls first card
		b.Flip("player1", 0, 0)

		// Player2 controls second card
		b.Flip("player2", 1, 0)

		// Replace A with C (affects controlled and uncontrolled cards)
		b.ReplaceCard("A", "C")

		result1 := b.Look("player1")
		result2 := b.Look("player2")

		// Player1's controlled A should become controlled C
		if !strings.Contains(result1, "my C") {
			t.Error("Player1's controlled card should show replaced content")
		}

		// Player2 should see player1's card as "up C"
		if !strings.Contains(result2, "up C") {
			t.Error("Player2 should see replaced content of other player's card")
		}

		// Third card (face down A) should also be replaced
		// Player2 flips it as second card - B and C don't match, so Rule 2-E applies
		b.Flip("player2", 2, 0)
		result2 = b.Look("player2")
		// Player2 should NOT control the C (Rule 2-E: relinquish control of non-matching cards)
		// But the card should show the replaced content when face up
		if !strings.Contains(result2, "up C") {
			t.Error("Face-up replaced card should show new content, but player should not control it due to Rule 2-E")
		}
	})
}

// Test watch functionality
func TestWatchFeature(t *testing.T) {
	t.Run("Watch returns current state immediately", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createAdvancedTestBoard(t, content)

		// Watch should return current state
		result, err := b.Watch("player1")
		if err != nil {
			t.Errorf("Watch failed: %v", err)
		}

		if !strings.Contains(result, "1x1") {
			t.Error("Watch should return board state")
		}
		if !strings.Contains(result, "down") {
			t.Error("Watch should show initial face-down card state")
		}
	})

	t.Run("Watch reflects current player perspective", func(t *testing.T) {
		content := "2x1\nA\nB\n"
		b := createAdvancedTestBoard(t, content)

		// Player1 flips a card
		b.Flip("player1", 0, 0)

		// Watch from player1 perspective
		result1, err := b.Watch("player1")
		if err != nil {
			t.Errorf("Watch failed for player1: %v", err)
		}

		// Watch from player2 perspective
		result2, err := b.Watch("player2")
		if err != nil {
			t.Errorf("Watch failed for player2: %v", err)
		}

		// Player1 should see "my A", player2 should see "up A"
		if !strings.Contains(result1, "my A") {
			t.Error("Player1 watch should show controlled card")
		}
		if !strings.Contains(result2, "up A") {
			t.Error("Player2 watch should show other player's card as face-up")
		}
	})

	t.Run("Watch during game state changes", func(t *testing.T) {
		content := "2x1\nA\nA\n"
		b := createAdvancedTestBoard(t, content)

		// Initial watch
		result, err := b.Watch("player1")
		if err != nil {
			t.Errorf("Initial watch failed: %v", err)
		}
		if !strings.Contains(result, "down") {
			t.Error("Initial watch should show face-down cards")
		}

		// Make a matching pair
		b.Flip("player1", 0, 0)
		b.Flip("player1", 1, 0)

		// Watch after matching
		result, err = b.Watch("player1")
		if err != nil {
			t.Errorf("Watch after matching failed: %v", err)
		}
		if !strings.Contains(result, "my A") {
			t.Error("Watch should show controlled matching cards")
		}

		// Start new move (should remove matched cards)
		b.Flip("player1", 0, 0)

		// Give time for Rule 3-A processing
		time.Sleep(10 * time.Millisecond)

		// Watch after removal
		result, err = b.Watch("player1")
		if err != nil {
			t.Errorf("Watch after removal failed: %v", err)
		}
		if !strings.Contains(result, "none") {
			t.Error("Watch should show removed cards as 'none'")
		}
	})

	t.Run("Watch with different player names", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createAdvancedTestBoard(t, content)

		// Test various player name formats
		playerNames := []string{
			"alice",
			"player_123",
			"Player-1",
			"🎮player",
			"", // Empty player name should still work
		}

		for _, name := range playerNames {
			result, err := b.Watch(name)
			if err != nil {
				t.Errorf("Watch failed for player '%s': %v", name, err)
			}
			if !strings.Contains(result, "1x1") {
				t.Errorf("Watch for player '%s' should return board state", name)
			}
		}
	})

	t.Run("Watch shows correct multi-player state", func(t *testing.T) {
		content := "3x1\nA\nB\nC\n"
		b := createAdvancedTestBoard(t, content)

		// Multiple players control different cards
		b.Flip("alice", 0, 0)   // Alice controls A
		b.Flip("bob", 1, 0)     // Bob controls B
		b.Flip("charlie", 2, 0) // Charlie controls C

		// Each player's watch should show their perspective
		resultAlice, _ := b.Watch("alice")
		resultBob, _ := b.Watch("bob")
		resultCharlie, _ := b.Watch("charlie")

		// Alice should see her card as controlled, others as face-up
		if !strings.Contains(resultAlice, "my A") {
			t.Error("Alice should see her controlled card")
		}
		if !strings.Contains(resultAlice, "up B") || !strings.Contains(resultAlice, "up C") {
			t.Error("Alice should see other players' cards as face-up")
		}

		// Bob should see his card as controlled
		if !strings.Contains(resultBob, "my B") {
			t.Error("Bob should see his controlled card")
		}

		// Charlie should see his card as controlled
		if !strings.Contains(resultCharlie, "my C") {
			t.Error("Charlie should see his controlled card")
		}
	})
}

func TestAdvancedFeatureInteractions(t *testing.T) {
	t.Run("Replace card then watch", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createAdvancedTestBoard(t, content)

		// Flip card to see it
		b.Flip("player1", 0, 0)

		// Replace the card
		b.ReplaceCard("A", "🃏")

		// Watch should show the replaced card
		result, err := b.Watch("player1")
		if err != nil {
			t.Errorf("Watch after replace failed: %v", err)
		}

		if !strings.Contains(result, "🃏") {
			t.Error("Watch should show replaced card content")
		}
	})

	t.Run("Watch during replacement of controlled cards", func(t *testing.T) {
		content := "2x1\nA\nA\n"
		b := createAdvancedTestBoard(t, content)

		// Create matching pair
		b.Flip("player1", 0, 0)
		b.Flip("player1", 1, 0)

		// Replace one of the matching cards
		b.ReplaceCard("A", "B")

		// Watch should show the replacement
		result, err := b.Watch("player1")
		if err != nil {
			t.Errorf("Watch after replace failed: %v", err)
		}

		// Cards should no longer match, but player still controls them
		if !strings.Contains(result, "my B") {
			t.Error("Watch should show replaced controlled cards")
		}
	})

	t.Run("Multiple replaces and watch", func(t *testing.T) {
		content := "2x2\nA\nB\nC\nD\n"
		b := createAdvancedTestBoard(t, content)

		// Multiple sequential replacements
		b.ReplaceCard("A", "X")
		b.ReplaceCard("B", "Y")
		b.ReplaceCard("X", "Z") // Replace the replacement

		// Watch should reflect all changes
		result, err := b.Watch("player1")
		if err != nil {
			t.Errorf("Watch after multiple replaces failed: %v", err)
		}

		// Original A should now be Z, B should be Y
		if strings.Contains(result, "A") || strings.Contains(result, "B") {
			t.Error("Original cards should be completely replaced")
		}

		// Flip cards to verify replacements
		b.Flip("player1", 0, 0)
		b.Flip("player1", 0, 1)

		result = b.Look("player1")
		if !strings.Contains(result, "Z") || !strings.Contains(result, "Y") {
			t.Error("Flipped cards should show final replacement values")
		}
	})
}

// Helper function to create a board for advanced features testing
func createAdvancedTestBoard(t *testing.T, content string) *board.Board {
	tmpFile, err := createAdvancedTempFile(content)
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

// Helper function to create a temporary file for advanced features testing
func createAdvancedTempFile(content string) (string, error) {
	tmpFile, err := os.CreateTemp("", "advanced_test_*.txt")
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
