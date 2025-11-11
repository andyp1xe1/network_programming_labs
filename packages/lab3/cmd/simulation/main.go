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
	fmt.Println("Initial board:")
	fmt.Println(b.String())

	// Create multiple players
	players := []string{"Alice", "Bob", "Charlie", "Diana"}

	// Simulate random moves
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < 20; i++ {
		player := players[rand.Intn(len(players))]
		row := rand.Intn(3)
		col := rand.Intn(3)

		fmt.Printf("\n--- Move %d: %s flips (%d,%d) ---\n", i+1, player, row, col)

		err := b.Flip(player, row, col)
		if err != nil {
			fmt.Printf("❌ %s: %v\n", player, err)
		} else {
			fmt.Printf("✅ %s successfully flipped (%d,%d)\n", player, row, col)
		}

		// Show board state from player's perspective
		state := b.Look(player)
		fmt.Printf("Board state for %s:\n%s\n", player, state)

		// Small delay between moves
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("\n=== Final Board State ===")
	fmt.Println(b.String())
}
