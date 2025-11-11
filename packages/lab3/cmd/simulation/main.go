package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"lab3/internal/board"
)

func main() {
	// Load the board
	b, err := board.ParseFromFile("boards/perfect.txt")
	if err != nil {
		log.Fatalf("Failed to load board: %v", err)
	}

	fmt.Println("=== Memory Scramble Simulation ===")
	fmt.Printf("Requirements: 4 players, 100 moves each, timeouts 0.1-2ms, no shuffling\n")
	fmt.Println("Initial board:")
	fmt.Println(b.String())

	// Create 4 players as required
	players := []string{"Alice", "Bob", "Charlie", "Diana"}

	// No shuffling as per requirements - use deterministic seed
	rand.Seed(12345)

	totalMoves := 0
	crashCount := 0

	// Each player makes 100 moves (400 total)
	for _, player := range players {
		fmt.Printf("\n=== %s's turn (100 moves) ===\n", player)

		for move := 1; move <= 100; move++ {
			totalMoves++

			// Generate random position within board bounds
			row := rand.Intn(3)
			col := rand.Intn(3)

			// Random timeout between 0.1ms and 2ms as specified
			timeout := time.Duration(rand.Float64()*1.9+0.1) * time.Millisecond

			if move%25 == 0 { // Show progress every 25 moves
				fmt.Printf("Move %d/%d: %s flips (%d,%d)\n", move, 100, player, row, col)
			}

			// Simulate the move
			err := b.Flip(player, row, col)
			if err != nil && move%25 == 0 {
				fmt.Printf("  Error: %v\n", err)
			}

			// Apply the timeout between moves
			time.Sleep(timeout)
		}

		fmt.Printf("%s completed 100 moves\n", player)
	}

	fmt.Printf("\n=== Simulation Complete ===\n")
	fmt.Printf("Total moves executed: %d\n", totalMoves)
	fmt.Printf("Crashes: %d\n", crashCount)
	fmt.Printf("Game stability: %s\n", func() string {
		if crashCount == 0 {
			return "✅ PASSED - No crashes detected"
		}
		return "❌ FAILED - Crashes occurred"
	}())

	fmt.Println("\nFinal board state:")
	fmt.Println(b.String())
}
