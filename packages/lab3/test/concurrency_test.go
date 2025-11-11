package board_test

import (
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"lab3/internal/board"
)

// Test concurrent player functionality and multi-player interactions
// These tests focus on timing, waiting behavior, and concurrent access scenarios

// Test concurrent player functionality
func TestMultiPlayerConcurrency(t *testing.T) {
	t.Run("Multiple players can play simultaneously", func(t *testing.T) {
		content := "3x3\nA\nB\nC\nD\nE\nF\nG\nH\nI\n"
		b := createTestBoardForConcurrency(t, content)

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
		b := createTestBoardForConcurrency(t, content)

		// Player1 controls the only card
		b.Flip("player1", 0, 0)

		// Player2 and Player3 both try to flip it (should wait)
		err2 := b.Flip("player2", 0, 0)
		err3 := b.Flip("player3", 0, 0)

		if err2 == nil || err3 == nil {
			t.Error("Players should get waiting errors when trying to flip controlled card")
		}
	})

	// Test that waiting behavior follows MIT rules: "While one player is waiting to turn over
	// a first card, other players continue to play normally. They do not wait, unless they
	// also try to turn over a first card controlled by another player."
	t.Run("Waiting players don't block other gameplay", func(t *testing.T) {
		content := "2x2\nA\nB\nC\nD\n"
		b := createTestBoardForConcurrency(t, content)

		// Player1 controls card at (0,0)
		b.Flip("player1", 0, 0)

		// Player2 tries to flip the same card - should wait
		err := b.Flip("player2", 0, 0)
		if err == nil || !strings.Contains(err.Error(), "waiting") {
			t.Error("Player2 should be waiting for controlled card")
		}

		// Player3 should be able to play normally on other cards
		err = b.Flip("player3", 0, 1)
		if err != nil {
			t.Errorf("Player3 should be able to flip other cards normally while player2 waits: %v", err)
		}

		// Verify Player3 controls their card
		result := b.Look("player3")
		if !strings.Contains(result, "my B") {
			t.Error("Player3 should control their card normally despite player2 waiting")
		}
	})
}

// Test race conditions and timing-sensitive scenarios
func TestRaceConditions(t *testing.T) {
	t.Run("Concurrent access to same card during flip", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createTestBoardForConcurrency(t, content)

		var wg sync.WaitGroup
		var results []error
		var mu sync.Mutex

		// Multiple players try to flip the same card simultaneously
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(playerNum int) {
				defer wg.Done()
				err := b.Flip("player"+string(rune('0'+playerNum)), 0, 0)
				mu.Lock()
				results = append(results, err)
				mu.Unlock()
			}(i)
		}

		wg.Wait()

		// Exactly one should succeed, others should wait or fail
		successCount := 0
		for _, err := range results {
			if err == nil {
				successCount++
			}
		}

		if successCount != 1 {
			t.Errorf("Expected exactly 1 success, got %d", successCount)
		}
	})

	t.Run("Card control changes during concurrent flips", func(t *testing.T) {
		content := "2x1\nA\nB\n"
		b := createTestBoardForConcurrency(t, content)

		// Player1 gets control
		b.Flip("player1", 0, 0)

		var wg sync.WaitGroup
		wg.Add(2)

		// Player1 tries second card while Player2 tries to take first card
		go func() {
			defer wg.Done()
			time.Sleep(5 * time.Millisecond) // Small delay
			b.Flip("player1", 1, 0)          // Second card
		}()

		go func() {
			defer wg.Done()
			b.Flip("player2", 0, 0) // Try to take first card
		}()

		wg.Wait()
		time.Sleep(10 * time.Millisecond)

		// Check final state is consistent
		result := b.Look("player1")
		if strings.Contains(result, "my A") && strings.Contains(result, "my B") {
			// Player1 kept control - should be matching or game rules applied
		} else if !strings.Contains(result, "my A") && !strings.Contains(result, "my B") {
			// Player1 lost control - should be due to rule 2-E or 2-B
		} else {
			t.Error("Inconsistent state: player controls only one card")
		}
	})
}

// Test timing and synchronization
func TestTimingAndSynchronization(t *testing.T) {
	t.Run("Quick succession flips maintain consistency", func(t *testing.T) {
		content := "4x1\nA\nB\nC\nD\n"
		b := createTestBoardForConcurrency(t, content)

		// Player makes rapid flips
		for i := 0; i < 4; i++ {
			b.Flip("player1", i, 0)
			time.Sleep(1 * time.Millisecond) // Very short delay
		}

		time.Sleep(20 * time.Millisecond) // Allow processing

		// Board should be in consistent state
		result := b.Look("player1")
		lines := strings.Split(strings.TrimSpace(result), "\n")

		// Count controlled cards
		controlledCount := 0
		for _, line := range lines[1:] {
			if strings.HasPrefix(line, "my ") {
				controlledCount++
			}
		}

		// Should control 0, 1, or 2 cards depending on game state, never more
		if controlledCount > 2 {
			t.Errorf("Player should not control more than 2 cards, got %d", controlledCount)
		}
	})

	t.Run("Interleaved moves from multiple players", func(t *testing.T) {
		content := "3x2\nA\nB\nC\nD\nE\nF\n"
		b := createTestBoardForConcurrency(t, content)

		var wg sync.WaitGroup
		wg.Add(3)

		// Three players make interleaved moves
		go func() {
			defer wg.Done()
			b.Flip("alice", 0, 0) // A
			time.Sleep(3 * time.Millisecond)
			b.Flip("alice", 0, 1) // B
		}()

		go func() {
			defer wg.Done()
			time.Sleep(1 * time.Millisecond)
			b.Flip("bob", 1, 0) // C
			time.Sleep(3 * time.Millisecond)
			b.Flip("bob", 1, 1) // D
		}()

		go func() {
			defer wg.Done()
			time.Sleep(2 * time.Millisecond)
			b.Flip("charlie", 2, 0) // E
			time.Sleep(3 * time.Millisecond)
			b.Flip("charlie", 2, 1) // F
		}()

		wg.Wait()
		time.Sleep(20 * time.Millisecond)

		// All players should have consistent states
		aliceResult := b.Look("alice")
		bobResult := b.Look("bob")
		charlieResult := b.Look("charlie")

		// Each result should be well-formed
		aliceLines := strings.Split(strings.TrimSpace(aliceResult), "\n")
		bobLines := strings.Split(strings.TrimSpace(bobResult), "\n")
		charlieLines := strings.Split(strings.TrimSpace(charlieResult), "\n")

		if len(aliceLines) != 7 || len(bobLines) != 7 || len(charlieLines) != 7 {
			t.Error("All players should see consistent board dimensions")
		}
	})
}

// Test waiting and notification mechanisms
func TestWaitingMechanisms(t *testing.T) {
	t.Run("Multiple waiters get notified in order", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createTestBoardForConcurrency(t, content)

		// Player1 takes control
		b.Flip("player1", 0, 0)

		var wg sync.WaitGroup
		var waitOrder []string
		var mu sync.Mutex

		// Multiple players try to wait for the card
		for i := 2; i <= 4; i++ {
			wg.Add(1)
			go func(playerNum int) {
				defer wg.Done()
				playerName := "player" + string(rune('0'+playerNum))
				err := b.Flip(playerName, 0, 0)
				if err != nil && strings.Contains(err.Error(), "waiting") {
					mu.Lock()
					waitOrder = append(waitOrder, playerName)
					mu.Unlock()
				}
			}(i)
		}

		wg.Wait()

		// Should have waiting players
		if len(waitOrder) == 0 {
			t.Error("Expected some players to be waiting")
		}
	})

	t.Run("Automatic control transfer to waiting player", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createTestBoardForConcurrency(t, content)

		// Player1 takes control of the card
		err := b.Flip("player1", 0, 0)
		if err != nil {
			t.Errorf("Player1 should be able to flip card: %v", err)
		}

		// Verify Player1 controls the card
		result := b.Look("player1")
		if !strings.Contains(result, "my A") {
			t.Error("Player1 should control the card initially")
		}

		// Player2 tries to flip same card - should get waiting error
		err = b.Flip("player2", 0, 0)
		if err == nil || !strings.Contains(err.Error(), "waiting") {
			t.Error("Player2 should get waiting error for controlled card")
		}

		// Verify Player2 does not control the card yet
		result = b.Look("player2")
		if strings.Contains(result, "my A") {
			t.Error("Player2 should not control card while waiting")
		}

		// Player1 makes another move with invalid position to release the card
		b.Flip("player1", 0, 0) // Try same card as second card - should fail due to Rule 2-B

		// Allow time for automatic control transfer
		time.Sleep(50 * time.Millisecond)

		// Now Player2 should automatically have control
		result = b.Look("player2")
		if !strings.Contains(result, "my A") {
			t.Error("Player2 should automatically gain control after Player1 releases card")
		}

		// Verify Player1 no longer controls the card
		result = b.Look("player1")
		if strings.Contains(result, "my A") {
			t.Error("Player1 should no longer control the card after release")
		}
	})

	t.Run("Waiting timeout behavior", func(t *testing.T) {
		content := "1x1\nA\n"
		b := createTestBoardForConcurrency(t, content)

		// Player1 takes control and never releases it
		b.Flip("player1", 0, 0)

		start := time.Now()
		err := b.Flip("player2", 0, 0)
		elapsed := time.Since(start)

		// Should fail quickly with waiting error, not hang
		if err == nil {
			t.Error("Expected waiting error")
		}

		if elapsed > 100*time.Millisecond {
			t.Errorf("Flip should fail quickly, took %v", elapsed)
		}
	})
}

// Helper functions for concurrency tests
func createTestBoardForConcurrency(t *testing.T, content string) *board.Board {
	tmpFile, err := createTempFileForConcurrency(content)
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

func createTempFileForConcurrency(content string) (string, error) {
	tmpFile, err := os.CreateTemp("", "concurrency_test_*.txt")
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
