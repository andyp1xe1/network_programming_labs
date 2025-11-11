package board_test

import (
	"testing"
)

func TestBugReproduction(t *testing.T) {
	// Create a 4x4 board with specific card layout to reproduce the bug
	// E at (2,0), E at (2,1), H at (3,2)
	content := "4x4\nA\nB\nC\nD\nE\nF\nG\nH\nE\nE\nI\nJ\nK\nL\nH\nM\n"

	b := createTestBoardForRules(t, content)

	t.Log("=== Reproducing Bug Scenario ===")

	// Step 1: Orange flips E at (2,1)
	t.Log("1. Orange flips E at (2,1)")
	err := b.Flip("orange_bean_485", 2, 1)
	if err != nil {
		t.Logf("Error: %v", err)
	}

	// Step 2: Blue flips H at (3,2)
	t.Log("2. Blue flips H at (3,2)")
	err = b.Flip("blue_bean_670", 3, 2)
	if err != nil {
		t.Logf("Error: %v", err)
	}

	// Step 3: Blue flips E at (2,0) - should trigger Rule 2-E
	t.Log("3. Blue flips E at (2,0) - should be Rule 2-E (no match)")
	err = b.Flip("blue_bean_670", 2, 0)
	if err != nil {
		t.Logf("Error: %v", err)
	}

	// Step 4: Blue tries to flip E at (2,1) - should trigger Rule 3-B first
	t.Log("4. Blue tries to flip E at (2,1) - should trigger Rule 3-B and then fail")
	err = b.Flip("blue_bean_670", 2, 1)
	if err != nil {
		t.Logf("Expected error: %v", err)
	}
}
