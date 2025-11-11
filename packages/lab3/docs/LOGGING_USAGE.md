# Game Logging System Usage

## Overview

The MIT Memory Scramble game now includes comprehensive logging to help debug game rule violations and track player actions. All game events are logged with timestamps and detailed context.

## Log Format

All game logs follow this format:
```
[GAME_LOG] HH:MM:SS.mmm | EVENT_TYPE | Context | Details
[HTTP_LOG] IP | EVENT_TYPE | Context | Details
```

## Game Log Types

### Player Actions
- **LOOK**: When a player requests board state
- **FLIP_START**: Beginning of a flip operation with current board state
- **FLIP_FIRST_CARD**: Result of flipping the first card
- **FLIP_SECOND_CARD**: Result of flipping the second card

### Rule Processing
- **RULE_1B**: Card flipped from face-down to face-up
- **RULE_1C**: Player gained control of uncontrolled face-up card
- **RULE_1D**: Player waiting for card controlled by another player
- **RULE_2A**: No card at second position (triggers Rule 3-B)
- **RULE_2B**: Second card controlled by another player (triggers Rule 3-B)
- **RULE_2C**: Second card flipped face-up
- **RULE_2D**: Cards match - player keeps control (triggers Rule 3-A)
- **RULE_2E**: Cards don't match - player relinquishes control (triggers Rule 3-B)
- **RULE_3A**: Matching cards removed from board
- **RULE_3B**: Non-matching cards flipped face-down

### Board State Format
Board state is logged as: `(rows x cols) Cards: [card states] Players: [player states]`

**Card States:**
- `_` = Empty space
- `?` = Face-down card
- `A[player1]` = Face-up card 'A' controlled by 'player1'
- `A[uncontrolled]` = Face-up card 'A' not controlled by anyone

**Player States:**
- `player1(ctrl:2,prev:2,wait:false)` = player1 controls 2 cards, has 2 previous cards, not waiting

## How to Use

### 1. Run Game with Logging
```bash
go run main.go 2>&1 | tee game.log
```

### 2. Filter Specific Events
```bash
# See only rule violations
grep "RULE_2B\|RULE_3B" game.log

# See all player actions
grep "FLIP\|LOOK" game.log

# See board state changes
grep "Board_State" game.log
```

### 3. Debug Scenarios
When you encounter unexpected behavior:

1. **Save the full log**: Copy all log output from when you start playing
2. **Include your actions**: Note what buttons you clicked and when
3. **Describe the issue**: What you expected vs. what happened

### Example Log Output
```
[GAME_LOG] 13:55:45.813 | FLIP_START | Player: alice | Position: (0,0) | Board_State: (2x2) Cards: ? ? ? ? Players:
[GAME_LOG] 13:55:45.813 | RULE_1B | Player: alice | Card: heart | Action: Flipped face up and gained control
[GAME_LOG] 13:55:45.813 | FLIP_FIRST_CARD | Player: alice | Position: (0,0) | Result: <nil> | Board_State: (2x2) Cards: heart[alice] ? ? ? Players: alice(ctrl:1,prev:0,wait:false)
```

This shows:
- Alice started flipping at position (0,0)
- Rule 1-B applied (face-down card turned face-up)
- Alice gained control of the heart card
- Final board state shows alice controlling 1 card with no previous cards or waiting state

## Debugging Edge Cases

The logging is especially useful for debugging:
- **Rule 2-B failures**: When trying to flip cards controlled by other players
- **Rule 3-B processing**: When cards should flip face-down after mismatches
- **Waiting mechanisms**: When players wait for cards and when they gain control
- **Concurrent access**: Multiple players accessing the same cards

Send the complete log output along with your description of the issue for faster debugging!