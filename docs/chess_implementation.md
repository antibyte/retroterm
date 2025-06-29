# TinyOS Chess Implementation

## Overview
The TinyOS chess system provides a complete chess game that can be played through the terminal interface. It includes:

- Complete chess engine with move validation
- 3 difficulty levels (Easy, Medium, Hard)
- Graphics support for chess pieces and board
- Text-based fallback for compatibility

## Architecture

### Backend Components
- `pkg/chess/engine.go` - Core chess game logic and AI
- `pkg/chess/ui.go` - User interface and graphics management
- `pkg/tinyos/commands.go` - Chess command integration
- `pkg/shared/message.go` - MessageTypeBitmap for graphics

### Frontend Components
- `js/retroconsole.js` - Enhanced with bitmap message support
- Sprite system for chess pieces (32x32 PNG)
- Bitmap system for chess board (288x288 PNG)

### Graphics Assets
Located in `chess_gfx/`:
- `Board.png` - Chess board background (288x288)
- `White*.png` - White chess pieces (32x32 each)
- `Black*.png` - Black chess pieces (32x32 each)

## Usage

### Starting a Chess Game
```
chess [difficulty] [color]
```

Examples:
- `chess` - Start with default settings (medium difficulty, play as white)
- `chess easy white` - Easy difficulty, play as white
- `chess hard black` - Hard difficulty, play as black

### Chess Commands
- `move e2 e4` - Make a move from e2 to e4
- `e2e4` or `e2 e4` - Alternative move notation
- `board` - Show current board
- `help` - Show help
- `quit` - Exit chess game

### Difficulty Levels
1. **Easy** - Random computer moves
2. **Medium** - Simple position evaluation with 2-ply search
3. **Hard** - Deeper evaluation with 3-ply search

## Technical Details

### Message Types
- **MessageTypeBitmap (25)** - New message type for PNG bitmap transfer
  - `bitmapData`: Base64-encoded PNG data
  - `bitmapX`, `bitmapY`: Position coordinates
  - `bitmapScale`: Scaling factor (1.0 = original size)
  - `bitmapRotate`: Rotation in degrees
  - `bitmapId`: Unique identifier

### Graphics Quantization
All graphics are automatically quantized to 16 brightness levels in the frontend to maintain the retro terminal aesthetic.

### Session Management
Chess games are stored per session and persist until explicitly quit. The system handles:
- Session-specific game state
- Input redirection during active games
- Cleanup on session end

### Move Validation
Complete chess rules implementation including:
- Piece-specific movement rules
- Path obstruction checking
- Capture validation
- Turn management
- Game over detection

## Integration with TinyOS

The chess system integrates seamlessly with TinyOS:
- Uses existing logging system
- Follows TinyOS command patterns
- Integrates with session management
- Uses established message types and WebSocket communication

## Future Enhancements

Potential improvements:
- Castling support
- En passant captures
- Pawn promotion
- Check and checkmate detection
- Move history display
- Game save/load functionality
