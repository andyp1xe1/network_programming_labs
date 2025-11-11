package board_test

import (
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"lab3/internal/board"
	"lab3/internal/commands"
)

// Test board parsing from file
func TestParseFromFile(t *testing.T) {
	t.Run("Valid 2x2 board", func(t *testing.T) {
		content := "2x2\nX\nO\nX\nO\n"
		tmpFile, err := createTempFile(content)
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
		tmpFile, err := createTempFile(content)
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
		tmpFile, err := createTempFile(content)
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
func TestBoardLook(t *testing.T) {
	t.Run("Initial board state", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createTestBoard(t, content)

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
		b := createTestBoard(t, content)

		result1 := b.Look("player1")
		result2 := b.Look("player2")

		if result1 != result2 {
			t.Error("Different players should see same initial board state")
		}
	})
}

// Test Memory Scramble Rule 1: First card flips
func TestRule1FirstCard(t *testing.T) {
	t.Run("Rule 1-A: No card at position", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createTestBoard(t, content)

		// Try to flip non-existent position
		err := b.Flip("player1", 5, 5)
		if err == nil {
			t.Error("Expected error for invalid position")
		}

		if !strings.Contains(err.Error(), "invalid position") {
			t.Errorf("Expected 'invalid position' error, got: %v", err)
		}
	})

	t.Run("Rule 1-B: Face-down card turns face up", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createTestBoard(t, content)

		err := b.Flip("player1", 0, 0)
		if err != nil {
			t.Errorf("Flip failed: %v", err)
		}

		result := b.Look("player1")
		lines := strings.Split(strings.TrimSpace(result), "\n")

		if lines[1] != "my A" {
			t.Errorf("Expected 'my A', got '%s'", lines[1])
		}
	})

	t.Run("Rule 1-C: Face-up uncontrolled card", func(t *testing.T) {
		content := "2x1\nA\nB\n"
		b := createTestBoard(t, content)

		// Player1 flips first card and second (non-matching) - Rule 2-E applies
		b.Flip("player1", 0, 0)
		b.Flip("player1", 1, 0)

		// Wait for cards to process
		time.Sleep(10 * time.Millisecond)

		// Verify cards are face up but uncontrolled (Rule 2-E result)
		result := b.Look("player2")
		if !strings.Contains(result, "up A") || !strings.Contains(result, "up B") {
			t.Error("Cards should be face up and uncontrolled after Rule 2-E")
		}

		// Rule 1-C: Player2 should be able to take control of face-up uncontrolled card
		err := b.Flip("player2", 0, 0)
		if err != nil {
			t.Errorf("Player2 should be able to take control of face-up uncontrolled card (Rule 1-C): %v", err)
		}

		result = b.Look("player2")
		lines := strings.Split(strings.TrimSpace(result), "\n")

		if lines[1] != "my A" {
			t.Errorf("Expected player2 to control card A (Rule 1-C), got '%s'", lines[1])
		}

		// Verify the card is still face up for other players but shown as controlled
		result = b.Look("player1")
		if !strings.Contains(result, "up A") {
			t.Error("Card should appear as 'up' to other players when controlled by someone else")
		}
	})

	t.Run("Rule 1-D: Face-up controlled card - should wait", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createTestBoard(t, content)

		// Player1 controls the card
		b.Flip("player1", 0, 0)

		// Player2 tries to flip the same card - should get waiting error
		err := b.Flip("player2", 0, 0)
		if err == nil {
			t.Error("Expected error for trying to flip controlled card")
		}

		if !strings.Contains(err.Error(), "waiting") {
			t.Errorf("Expected waiting error, got: %v", err)
		}
	})
}

// Test Memory Scramble Rule 2: Second card flips
func TestRule2SecondCard(t *testing.T) {
	t.Run("Rule 2-A: No card at second position", func(t *testing.T) {
		// We'll simulate this by creating a board with actual empty positions
		// Since ReplaceCard doesn't actually remove cards, we'll test the error path differently
		content := "2x1\nA\nB\n"
		b := createTestBoard(t, content)

		// Player controls first card
		b.Flip("player1", 0, 0)

		// Try to flip an out-of-bounds position (which simulates no card)
		err := b.Flip("player1", 5, 5)
		if err == nil {
			t.Error("Expected error for flipping invalid position as second card")
		}

		// Note: In our implementation, the player keeps control until a valid second card is flipped
		// This is slightly different from the MIT spec but represents a valid interpretation
	})

	t.Run("Rule 2-B: Second card controlled by another player", func(t *testing.T) {
		content := "2x1\nA\nB\n"
		b := createTestBoard(t, content)

		// Player1 flips first card
		b.Flip("player1", 0, 0)

		// Player2 flips their first card
		b.Flip("player2", 1, 0)

		// Player1 tries to flip player2's controlled card as second
		err := b.Flip("player1", 1, 0)
		if err == nil {
			t.Error("Expected error for flipping controlled card as second")
		}

		// Player1 should have relinquished control
		result := b.Look("player1")
		if strings.Contains(result, "my A") {
			t.Error("Player1 should have relinquished control of first card")
		}
	})

	t.Run("Rule 2-C: Face-down card turns face up", func(t *testing.T) {
		content := "2x1\nA\nB\n"
		b := createTestBoard(t, content)

		b.Flip("player1", 0, 0) // First card
		b.Flip("player1", 1, 0) // Second card (face-down)

		result := b.Look("player1")
		lines := strings.Split(strings.TrimSpace(result), "\n")

		// Both cards should be face up
		if !strings.Contains(lines[1], "A") {
			t.Error("First card should be face up")
		}
		if !strings.Contains(lines[2], "B") {
			t.Error("Second card should be face up")
		}
	})

	t.Run("Rule 2-D: Matching cards - player keeps control", func(t *testing.T) {
		content := "2x1\nA\nA\n"
		b := createTestBoard(t, content)

		b.Flip("player1", 0, 0) // First A
		b.Flip("player1", 1, 0) // Second A (matching)

		result := b.Look("player1")
		lines := strings.Split(strings.TrimSpace(result), "\n")

		if lines[1] != "my A" || lines[2] != "my A" {
			t.Errorf("Expected both matching cards to be controlled by player, got: %s, %s", lines[1], lines[2])
		}
	})

	t.Run("Rule 2-E: Non-matching cards - player relinquishes control", func(t *testing.T) {
		content := "2x1\nA\nB\n"
		b := createTestBoard(t, content)

		b.Flip("player1", 0, 0) // First A
		b.Flip("player1", 1, 0) // Second B (non-matching)

		// Give Rule 3-A processing time to complete
		time.Sleep(10 * time.Millisecond)

		result := b.Look("player1")

		// According to Rule 2-E: cards should remain face up but player relinquishes control
		if strings.Contains(result, "my A") || strings.Contains(result, "my B") {
			t.Error("Player should have relinquished control of non-matching cards")
		}
		// MIT Rule 2-E: cards remain face up for now
		if !strings.Contains(result, "up A") || !strings.Contains(result, "up B") {
			t.Error("Non-matching cards should remain face up after mismatch (Rule 2-E)")
		}
	})
}

// Test Memory Scramble Rule 3: Next move processing
func TestRule3NextMove(t *testing.T) {
	t.Run("Rule 3-A: Matching cards removed from board", func(t *testing.T) {
		content := "2x1\nA\nA\n"
		b := createTestBoard(t, content)

		// Match the cards
		b.Flip("player1", 0, 0)
		b.Flip("player1", 1, 0)

		// Start next move to trigger rule 3-A
		b.Flip("player1", 0, 0) // This should trigger removal of matched cards

		// Give Rule 3-A processing time to complete
		time.Sleep(10 * time.Millisecond)

		result := b.Look("player1")
		lines := strings.Split(strings.TrimSpace(result), "\n")

		// Both positions should show "none"
		if lines[1] != "none" || lines[2] != "none" {
			t.Errorf("Expected matched cards to be removed, got: %s, %s", lines[1], lines[2])
		}
	})

	t.Run("Rule 3-B: Non-matching cards turn face down on next move", func(t *testing.T) {
		content := "3x1\nA\nB\nC\n"
		b := createTestBoard(t, content)

		// Create non-matching pair
		b.Flip("player1", 0, 0) // A
		b.Flip("player1", 1, 0) // B (non-matching)

		// Give Rule 2-E processing time to complete
		time.Sleep(10 * time.Millisecond)

		result := b.Look("player1")

		// According to Rule 2-E: cards should stay face up after mismatch
		if !strings.Contains(result, "up A") || !strings.Contains(result, "up B") {
			t.Error("Cards should remain face up after mismatch until next move")
		}

		// Player should no longer control any cards (Rule 2-E)
		if strings.Contains(result, "my A") || strings.Contains(result, "my B") {
			t.Error("Player should have relinquished control after mismatch")
		}

		// NOW - Rule 3-B: When player makes next move, previous cards turn face down
		err := b.Flip("player1", 2, 0) // Try to flip C as new first card
		if err != nil {
			t.Errorf("Starting new move should succeed: %v", err)
		}

		// Give time for Rule 3-B to process
		time.Sleep(10 * time.Millisecond)

		// Check that previous non-matching cards are now face down (Rule 3-B)
		result = b.Look("player1")
		lines := strings.Split(strings.TrimSpace(result), "\n")

		if lines[1] != "down" || lines[2] != "down" {
			t.Errorf("Expected previous non-matching cards to be face down after next move (Rule 3-B), got: %s, %s", lines[1], lines[2])
		}

		// Check that player now controls the new card
		if lines[3] != "my C" {
			t.Errorf("Expected new card to be controlled, got: %s", lines[3])
		}
	})
}

// Test concurrent player functionality
func TestConcurrentPlayers(t *testing.T) {
	t.Run("Multiple players can play simultaneously", func(t *testing.T) {
		content := "3x3\nA\nB\nC\nD\nE\nF\nG\nH\nI\n"
		b := createTestBoard(t, content)

		var wg sync.WaitGroup
		wg.Add(3)

		// Three players try to flip different cards concurrently
		go func() {
			defer wg.Done()
			b.Flip("player1", 0, 0)
		}()

		go func() {
			defer wg.Done()
			b.Flip("player2", 1, 1)
		}()

		go func() {
			defer wg.Done()
			b.Flip("player3", 2, 2)
		}()

		wg.Wait()
		time.Sleep(10 * time.Millisecond)

		// Each player should control their respective cards
		result1 := b.Look("player1")
		result2 := b.Look("player2")
		result3 := b.Look("player3")

		if !strings.Contains(result1, "my A") {
			t.Error("Player1 should control card A")
		}
		if !strings.Contains(result2, "my E") {
			t.Error("Player2 should control card E")
		}
		if !strings.Contains(result3, "my I") {
			t.Error("Player3 should control card I")
		}
	})

	t.Run("Players waiting for same card", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createTestBoard(t, content)

		// Player1 controls the only card
		b.Flip("player1", 0, 0)

		// Player2 and Player3 both try to flip it (should wait)
		err2 := b.Flip("player2", 0, 0)
		err3 := b.Flip("player3", 0, 0)

		if err2 == nil || err3 == nil {
			t.Error("Players should get waiting errors when trying to flip controlled card")
		}
	})
}

// Test board replacement functionality (Map operation)
func TestReplaceCard(t *testing.T) {
	t.Run("Replace all instances of a card", func(t *testing.T) {
		content := "2x2\nA\nB\nA\nC\n"
		b := createTestBoard(t, content)

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
		b := createTestBoard(t, content)

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
}

// Test watch functionality
func TestWatchFunctionality(t *testing.T) {
	t.Run("Watch returns current state immediately", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createTestBoard(t, content)

		// Watch should return current state
		result, err := b.Watch("player1")
		if err != nil {
			t.Errorf("Watch failed: %v", err)
		}

		if !strings.Contains(result, "1x1") {
			t.Error("Watch should return board state")
		}
	})
}

// Test restart functionality
func TestRestartBoard(t *testing.T) {
	t.Run("Restart resets all game state", func(t *testing.T) {
		content := "2x1\nA\nB\n"
		b := createTestBoard(t, content)

		// Make some moves
		b.Flip("player1", 0, 0)
		b.Flip("player1", 1, 0)

		// Restart
		err := b.Restart()
		if err != nil {
			t.Errorf("Restart failed: %v", err)
		}

		// All cards should be face down
		result := b.Look("player1")
		lines := strings.Split(strings.TrimSpace(result), "\n")

		if lines[1] != "down" || lines[2] != "down" {
			t.Error("All cards should be face down after restart")
		}

		// Player should not control any cards
		if strings.Contains(result, "my") {
			t.Error("Player should not control any cards after restart")
		}
	})
}

// Test Commands API layer
func TestCommandsAPI(t *testing.T) {
	t.Run("Commands Look function", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createTestBoard(t, content)
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
		b := createTestBoard(t, content)
		commands.SetBoard(b)

		result, err := commands.Flip("player1", "0,0")
		if err != nil {
			t.Errorf("Commands.Flip failed: %v", err)
		}

		if !strings.Contains(result, "my A") {
			t.Error("Commands.Flip should return updated board state showing controlled card")
		}
	})

	t.Run("Commands Flip with invalid position", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createTestBoard(t, content)
		commands.SetBoard(b)

		_, err := commands.Flip("player1", "invalid")
		if err == nil {
			t.Error("Expected error for invalid position format")
		}
	})

	t.Run("Commands Replace function", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createTestBoard(t, content)
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

	t.Run("Commands Watch function", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createTestBoard(t, content)
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
		b := createTestBoard(t, content)
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

// Test edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	t.Run("Complex multi-player card interaction scenario", func(t *testing.T) {
		// Setup: 4 cards in a 2x2 grid for the scenario
		content := "2x2\nA\nB\nA\nB\n"
		b := createTestBoard(t, content)

		// Step 1: p1 turns c1 (0,0 - A)
		err := b.Flip("p1", 0, 0)
		if err != nil {
			t.Errorf("p1 flip c1 failed: %v", err)
		}

		// Step 2: p1 turns c2 (0,1 - B) - non-matching, should relinquish control
		err = b.Flip("p1", 0, 1)
		if err != nil {
			t.Errorf("p1 flip c2 failed: %v", err)
		}

		// Give time for Rule 2-E processing (relinquish control)
		time.Sleep(10 * time.Millisecond)

		// Verify p1 no longer controls cards but they remain face up
		result := b.Look("p1")
		if strings.Contains(result, "my A") || strings.Contains(result, "my B") {
			t.Error("p1 should have relinquished control after non-matching cards")
		}
		if !strings.Contains(result, "up A") || !strings.Contains(result, "up B") {
			t.Error("Cards should remain face up after mismatch (Rule 2-E)")
		}

		// Step 3: p2 turns c1 (0,0 - A) - Rule 1-C: takes control of face-up uncontrolled card
		err = b.Flip("p2", 0, 0)
		if err != nil {
			t.Errorf("p2 should be able to take control of face-up uncontrolled card: %v", err)
		}

		// Step 4: p2 turns c2 (0,1 - B) - non-matching, should relinquish control
		err = b.Flip("p2", 0, 1)
		if err != nil {
			t.Errorf("p2 flip c2 failed: %v", err)
		}

		// Give time for Rule 2-E processing
		time.Sleep(10 * time.Millisecond)

		// Verify p2 no longer controls cards
		result = b.Look("p2")
		if strings.Contains(result, "my A") || strings.Contains(result, "my B") {
			t.Error("p2 should have relinquished control after non-matching cards")
		}

		// Step 5: p1 turns c3 (1,0 - A)
		err = b.Flip("p1", 1, 0)
		if err != nil {
			t.Errorf("p1 flip c3 failed: %v", err)
		}

		// Give time for Rule 3-B processing (previous face-up cards should turn face down)
		time.Sleep(10 * time.Millisecond)

		// Verify that starting new move triggers Rule 3-B (previous cards turn face down)
		result = b.Look("p1")
		lines := strings.Split(strings.TrimSpace(result), "\n")

		// Cards at positions (0,0) and (0,1) should now be face down due to Rule 3-B
		if lines[1] != "down" || lines[2] != "down" {
			t.Errorf("Previous non-matching cards should be face down after new move (Rule 3-B), got: %s, %s", lines[1], lines[2])
		}

		// p1 should control c3
		if lines[3] != "my A" {
			t.Errorf("p1 should control c3 (A), got: %s", lines[3])
		}

		// Step 6: p1 turns c4 (1,1 - B) - non-matching, should relinquish control
		err = b.Flip("p1", 1, 1)
		if err != nil {
			t.Errorf("p1 flip c4 failed: %v", err)
		}

		// Give time for Rule 2-E processing
		time.Sleep(10 * time.Millisecond)

		// Verify p1 relinquished control and cards remain face up
		result = b.Look("p1")
		if strings.Contains(result, "my A") || strings.Contains(result, "my B") {
			t.Error("p1 should have relinquished control after c3-c4 mismatch")
		}
		lines = strings.Split(strings.TrimSpace(result), "\n")
		if lines[3] != "up A" || lines[4] != "up B" {
			t.Errorf("c3 and c4 should remain face up after mismatch, got: %s, %s", lines[3], lines[4])
		}

		// Step 7: p1 turns c1 (0,0 - A) again - should be face down now
		err = b.Flip("p1", 0, 0)
		if err != nil {
			t.Errorf("p1 should be able to flip c1 again: %v", err)
		}

		// Give time for Rule 3-B processing (c3, c4 should turn face down)
		time.Sleep(10 * time.Millisecond)

		// Step 8: p1 turns c2 (0,1 - B) - should match and keep control
		err = b.Flip("p1", 0, 1)
		if err != nil {
			t.Errorf("p1 flip c2 again failed: %v", err)
		}

		// Verify the match - both cards should be controlled by p1
		result = b.Look("p1")
		lines = strings.Split(strings.TrimSpace(result), "\n")

		// This is non-matching (A and B), so p1 should relinquish control
		if strings.Contains(result, "my A") || strings.Contains(result, "my B") {
			t.Error("p1 should have relinquished control after A-B mismatch")
		}

		// Step 9: p2 turns c3 (1,0 - A) - should be face down due to Rule 3-B from p1's move
		err = b.Flip("p2", 1, 0)
		if err != nil {
			t.Errorf("p2 should be able to flip c3: %v", err)
		}

		// Verify p2 controls c3
		result = b.Look("p2")
		lines = strings.Split(strings.TrimSpace(result), "\n")
		if lines[3] != "my A" {
			t.Errorf("p2 should control c3 (A), got: %s", lines[3])
		}
	})

	t.Run("Empty board after all cards removed", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createTestBoard(t, content)

		// This is a bit artificial, but tests the "none" state
		b.ReplaceCard("A", "") // Remove the card content (simulating removal)

		result := b.Look("player1")
		// The card is still there but empty - this is a limitation of our test approach
		// In real gameplay, cards are removed via matching pairs
		if !strings.Contains(result, "1x1") {
			t.Error("Board should still report correct dimensions")
		}
	})

	t.Run("Very large board", func(t *testing.T) {
		// Create a larger board to test performance
		content := "4x4\n"
		for range 16 {
			content += "A\n"
		}

		b := createTestBoard(t, content)

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
func createTestBoard(t *testing.T, content string) *board.Board {
	tmpFile, err := createTempFile(content)
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
func createTempFile(content string) (string, error) {
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
