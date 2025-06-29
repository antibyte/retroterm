## TinyOS Chess Demo

This demo shows how to play chess in TinyOS.

### Starting the Game
```
> chess
Welcome to TinyOS Chess!

Commands:
  move <from> <to> - Make a move (e.g., move e2 e4)
  board - Show current board
  help - Show help
  quit - Exit chess game

[Chess board with pieces displayed graphically]
```

### Making Moves
```
> move e2 e4
Move: e2 -> e4
[Updated board display]
Computer is thinking...
Computer moves: e7 -> e5
[Updated board display]

> move g1 f3
Move: g1 -> f3
[Updated board display]
Computer is thinking...
Computer moves: b8 -> c6
[Updated board display]
```

### Getting Help
```
> help
Chess Commands:
  move <from> <to> - Make a move (e.g., move e2 e4)
  board - Show current board
  help - Show this help
  quit - Exit chess game

Notation: Use standard chess notation (a1-h8)
```

### Viewing the Board
```
> board
[Chess board display with current position]
```

### Exiting the Game
```
> quit
Chess game ended. Back to TinyOS.
>
```

### Different Game Modes
```
> chess easy black
[Start as black pieces with easy computer opponent]

> chess hard white  
[Start as white pieces with hard computer opponent]
```

The chess system provides a complete chess experience within the retro terminal environment, with both graphical and text-based display options.
