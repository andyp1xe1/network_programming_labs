package board_test

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"lab3/internal/board"
)

// Test real-world gameplay scenarios and complex interactions
// These tests simulate actual game situations and comprehensive user stories

// Test MIT Example scenario from the website
func TestMITExampleScenario(t *testing.T) {
	// MIT Website Example: Alice, Bob, Charlie scenario with 3x3 grid of hearts
	// This test follows the exact example from the MIT website documentation
	t.Run("Alice/Bob/Charlie heart scenario", func(t *testing.T) {
		// Setup: 3x3 grid with hearts as shown in MIT example
		// Row 0: ❤️ 💛 ❤️  (positions (0,0), (0,1), (0,2))
		// Row 1: 💚 💛 💚  (positions (1,0), (1,1), (1,2))
		// Row 2: 💜 💜 💜  (positions (2,0), (2,1), (2,2))
		content := "3x3\n❤️\n💛\n❤️\n💚\n💛\n💚\n💜\n💜\n💜\n"
		b := createTestBoardForScenarios(t, content)

		// Step 1: Alice turns over the top left card (0,0 - ❤️) - Rule 1-B
		err := b.Flip("alice", 0, 0)
		if err != nil {
			t.Errorf("Alice flip (0,0) failed: %v", err)
		}

		// Alice should control the red heart
		result := b.Look("alice")
		if !strings.Contains(result, "my ❤️") {
			t.Error("Alice should control the red heart at (0,0)")
		}

		// Step 2: Bob and Charlie both try to flip the same card - Rule 1-D (waiting)
		err = b.Flip("bob", 0, 0)
		if err == nil || !strings.Contains(err.Error(), "waiting") {
			t.Error("Bob should be waiting for the card Alice controls")
		}

		err = b.Flip("charlie", 0, 0)
		if err == nil || !strings.Contains(err.Error(), "waiting") {
			t.Error("Charlie should be waiting for the card Alice controls")
		}

		// Step 3: Alice turns over bottom right card (2,2 - 💜) - Rule 2-C, then Rule 2-E
		err = b.Flip("alice", 2, 2)
		if err != nil {
			t.Errorf("Alice flip (2,2) failed: %v", err)
		}

		// Give time for Rule 2-E processing (non-matching cards, relinquish control)
		time.Sleep(10 * time.Millisecond)

		// Alice should no longer control any cards, but they remain face up
		result = b.Look("alice")
		if strings.Contains(result, "my ❤️") || strings.Contains(result, "my 💜") {
			t.Error("Alice should have relinquished control after non-matching cards")
		}
		if !strings.Contains(result, "up ❤️") || !strings.Contains(result, "up 💜") {
			t.Error("Cards should remain face up after mismatch (Rule 2-E)")
		}

		// Step 4: Either Bob or Charlie should now automatically have control - one of the waiters gets it
		// Check that either Bob or Charlie now controls the red heart (automatic control transfer)
		bobResult := b.Look("bob")
		charlieResult := b.Look("charlie")

		bobHasControl := strings.Contains(bobResult, "my ❤️")
		charlieHasControl := strings.Contains(charlieResult, "my ❤️")

		if !bobHasControl && !charlieHasControl {
			t.Error("Either Bob or Charlie should automatically have control of the red heart after Alice released it")
		}

		if bobHasControl && charlieHasControl {
			t.Error("Only one player should control the red heart, not both")
		}

		// Step 5: Alice starts a new move by flipping center card (1,1 - 💛) - Rule 3-B applies
		err = b.Flip("alice", 1, 1)
		if err != nil {
			t.Errorf("Alice flip (1,1) failed: %v", err)
		}

		// Give time for Rule 3-B processing
		time.Sleep(10 * time.Millisecond)

		// The purple heart should now be face down (Rule 3-B), but red heart stays up (controlled by whoever got it)
		result = b.Look("alice")
		lines := strings.Split(strings.TrimSpace(result), "\n")

		// Position (0,0) should still show "up ❤️" (controlled by either Bob or Charlie)
		if !strings.Contains(lines[1], "up ❤️") {
			t.Error("Red heart should still be face up (controlled by waiting player who got it)")
		}

		// Position (2,2) should now be "down" (Rule 3-B - was face up but uncontrolled)
		if lines[9] != "down" {
			t.Errorf("Purple heart should be face down after Rule 3-B, got: %s", lines[9])
		}

		// Alice should control the yellow heart at (1,1)
		if lines[5] != "my 💛" {
			t.Errorf("Alice should control yellow heart, got: %s", lines[5])
		}
	})
}

// Test complex multi-player interactions
func TestComplexGameplayScenarios(t *testing.T) {
	t.Run("Complex multi-player card interaction scenario", func(t *testing.T) {
		// Setup: 4 cards in a 2x2 grid for the scenario
		content := "2x2\nA\nB\nA\nB\n"
		b := createTestBoardForScenarios(t, content)

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
}

// Test edge case scenarios
func TestEdgeCaseScenarios(t *testing.T) {
	t.Run("Empty board after all cards removed", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createTestBoardForScenarios(t, content)

		// This is a bit artificial, but tests the "none" state
		b.ReplaceCard("A", "") // Remove the card content (simulating removal)

		result := b.Look("player1")
		// The card is still there but empty - this is a limitation of our test approach
		// In real gameplay, cards are removed via matching pairs
		if !strings.Contains(result, "1x1") {
			t.Error("Board should still report correct dimensions")
		}
	})

	t.Run("Rapid successive moves by single player", func(t *testing.T) {
		content := "3x2\nA\nB\nA\nB\nC\nC\n"
		b := createTestBoardForScenarios(t, content)

		// Player makes rapid moves
		b.Flip("speedPlayer", 0, 0) // A
		b.Flip("speedPlayer", 0, 1) // B (mismatch)

		time.Sleep(5 * time.Millisecond)

		b.Flip("speedPlayer", 1, 0) // A (new first card)
		b.Flip("speedPlayer", 1, 1) // B (mismatch again)

		time.Sleep(5 * time.Millisecond)

		b.Flip("speedPlayer", 2, 0) // C
		b.Flip("speedPlayer", 2, 1) // C (match!)

		// Verify final state
		result := b.Look("speedPlayer")
		if !strings.Contains(result, "my C") {
			t.Error("Player should control matching C cards")
		}
	})

	t.Run("Players competing for same matching pair", func(t *testing.T) {
		content := "2x2\nA\nB\nA\nB\n"
		b := createTestBoardForScenarios(t, content)

		// Player1 flips first A
		b.Flip("player1", 0, 0)

		// Player2 flips second A
		b.Flip("player2", 1, 0)

		// Player1 tries to match with second A but it's controlled by Player2
		err := b.Flip("player1", 1, 0)
		if err == nil {
			t.Error("Player1 should not be able to flip Player2's controlled card")
		}

		// Player1 should have lost control due to Rule 2-B
		result := b.Look("player1")
		if strings.Contains(result, "my A") {
			t.Error("Player1 should have lost control after Rule 2-B failure")
		}

		// Player2 should still control their card
		result = b.Look("player2")
		if !strings.Contains(result, "my A") {
			t.Error("Player2 should still control their A card")
		}
	})
}

// Test complete game scenarios
func TestCompleteGameScenarios(t *testing.T) {
	t.Run("Full game with matches and removals", func(t *testing.T) {
		content := "2x2\nA\nA\nB\nB\n"
		b := createTestBoardForScenarios(t, content)

		// Player1 finds first matching pair
		b.Flip("player1", 0, 0) // A
		b.Flip("player1", 0, 1) // A (match!)

		// Player1 makes next move to remove matched cards
		b.Flip("player1", 1, 0) // B (triggers Rule 3-A)

		time.Sleep(10 * time.Millisecond)

		// First two cards should be removed
		result := b.Look("player1")
		lines := strings.Split(strings.TrimSpace(result), "\n")
		if lines[1] != "none" || lines[2] != "none" {
			t.Error("Matched A cards should be removed from board")
		}

		// Player1 should control the B
		if lines[3] != "my B" {
			t.Error("Player1 should control the B card")
		}

		// Player1 completes second match
		b.Flip("player1", 1, 1) // B (match!)

		// Make next move to clear second pair
		b.Flip("player1", 0, 0) // Should fail - no card there

		time.Sleep(10 * time.Millisecond)

		// All cards should be gone
		result = b.Look("player1")
		lines = strings.Split(strings.TrimSpace(result), "\n")
		for i := 1; i < len(lines); i++ {
			if lines[i] != "none" {
				t.Errorf("All cards should be removed, but position %d has: %s", i-1, lines[i])
			}
		}
	})
}

// Helper functions for scenario tests
func createTestBoardForScenarios(t *testing.T, content string) *board.Board {
	tmpFile, err := createTempFileForScenarios(content)
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

func createTempFileForScenarios(content string) (string, error) {
	tmpFile, err := os.CreateTemp("", "scenarios_test_*.txt")
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
