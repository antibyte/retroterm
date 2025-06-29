```markdown
# TINYBASIC INTERPRETER MANUAL

**Version 1.0**
**(C) 1986 SKYNET INC**

Welcome to the world of TinyBASIC! This manual will guide you through the features and commands of the TinyBASIC interpreter, your window into programming the Retro-Terminal system.

## INTRODUCTION

TinyBASIC is a classic, line-based BASIC interpreter designed to run within the Retro-Terminal's virtual operating system, TinyOS. It simulates the programming experience of the early 1980s. TinyBASIC is intended to be comprehensive and user-friendly, supporting a wide range of commands for programming, file handling, and even multimedia!

The Retro-Terminal system simulates a monochrome green screen, providing a traditional 80-column by 24-row display. While it doesn't support colors in the text mode, it offers 16 levels of brightness for graphics commands.

## GETTING STARTED

To enter the TinyBASIC interpreter from the TinyOS command prompt, simply type:

`basic`

Your terminal display will clear, and you will see the TinyBASIC prompt. To exit TinyBASIC and return to TinyOS, type:

`exit`

TinyBASIC operates in two modes:

1.  **Direct Mode:** Enter commands without a line number. The command is executed immediately after you press Enter.
2.  **Program Mode:** Enter lines prefixed with a number (1-65535). These lines are stored as part of your program. Type `RUN` to execute the stored program.

## BASIC FUNDAMENTALS

### Line Numbers

Program lines are identified by line numbers. Lines are executed in numerical order when you `RUN` a program. You can enter lines in any order, and TinyBASIC will store them correctly. To delete a line, type its number followed by nothing.

### Variables

TinyBASIC supports two types of variables:

*   **Numeric Variables:** Hold floating-point numbers. Names must start with a letter followed by zero or more letters or digits (e.g., `A`, `X1`, `COUNTER`).
*   **String Variables:** Hold text strings. Names are the same as numeric variables but *must* end with a dollar sign (`$`) (e.g., `A$`, `NAME$`, `MESSAGE$`).

### Arrays

Variables can be arrays, allowing you to store multiple values under a single name, accessed by an index in parentheses. Both numeric and string arrays are supported (e.g., `A(5)`, `W$(I)`). Array indices must be numeric expressions evaluating to a non-negative integer.

### Expressions

Expressions combine variables, constants (numbers or string literals), operators, and functions to produce a value.
*   **Numeric Expressions:** Use standard arithmetic operators (`+`, `-`, `*`, `/`, `^` for power) and comparison operators (`=`, `<>`, `<`, `>`, `<=`, `>=`).
*   **String Expressions:** Can be concatenated using the `+` operator.
*   **String Literals:** Text enclosed in double quotes (`"`). Use `""` within a string to represent a single quote.

### Error Handling

TinyBASIC provides structured error messages to help you debug your programs. Errors are categorized (e.g., `SYNTAX ERROR`, `EXECUTION ERROR`, `IO ERROR`) and often include the line number where the error occurred.

## COMMAND REFERENCE

Below is a list of common TinyBASIC commands:

### Program Management

*   **NEW**
    *   Syntax: `NEW`
    *   Clears the current program and all variables from memory, preparing for a new program.

*   **LIST**
    *   Syntax: `LIST [startLine][-endLine]`
    *   Displays the lines of the current program. Optionally specify a range of line numbers to list.
    *   Example: `LIST` (lists entire program), `LIST 100-200`, `LIST 50` (lists line 50), `LIST 10-` (lists from line 10 to the end).

*   **RUN**
    *   Syntax: `RUN`
    *   Starts execution of the program currently in memory, beginning with the lowest line number. Program output is sent asynchronously. Execution can be stopped with `__BREAK__` (typically Ctrl+C in the terminal).

*   **END**
    *   Syntax: `END`
    *   Terminates program execution immediately when encountered.

*   **REM**
    *   Syntax: `REM comment`
    *   Allows you to add comments to your program code. Everything after `REM` on the line is ignored by the interpreter.

### Input/Output (Console)

*   **PRINT**
    *   Syntax: `PRINT [expr][,|;]...`
    *   Displays the value of expressions (numeric or string) on the terminal.
    *   Items separated by a comma (`,`) are printed in columns.
    *   Items separated by a semicolon (`;`) are printed immediately after each other.
    *   A `PRINT` statement normally ends by moving to the next line. Ending the statement with a comma or semicolon prevents this.
    *   Example: `PRINT "HELLO"; " "; "WORLD!"`, `PRINT A, B`

*   **INPUT**
    *   Syntax: `INPUT [prompt_string;] var1 [, var2...]`
    *   Pauses program execution and waits for the user to enter data from the terminal.
    *   Optionally displays a `prompt_string`. A question mark `?` is displayed by default if no prompt is given.
    *   The entered value(s) are assigned to the specified variable(s). If multiple variables are listed, values should be separated by commas in the input.
    *   Example: `INPUT NAME$`, `INPUT "Enter Age: "; AGE`

*   **CLS**
    *   Syntax: `CLS`
    *   Clears the terminal screen.

*   **WAIT**
    *   Syntax: `WAIT milliseconds`
    *   Pauses program execution for the specified number of milliseconds.
    *   Example: `WAIT 1000` (pauses for 1 second).

### File Commands (Virtual File System)

TinyBASIC programs interact with the Retro-Terminal's Virtual File System (VFS). Files are typically stored within your user's home directory.

*   **LOAD**
    *   Syntax: `LOAD "filename"`
    *   Clears the current program and state, then loads a program from the VFS.
    *   Automatically adds the `.bas` extension if the filename doesn't have one.
    *   Example: `LOAD "MYPROG"`, `LOAD "GAME.BAS"`

*   **SAVE**
    *   Syntax: `SAVE "filename"`
    *   Saves the program currently in memory to the VFS.
    *   Automatically adds the `.bas` extension if the filename doesn't have one.
    *   Example: `SAVE "NEWPROG"`, `SAVE "BACKUP.BAS"`

*   **DIR**
    *   Syntax: `DIR`
    *   Lists the `.bas` program files available in your current directory within the VFS.

*   **OPEN**
    *   Syntax: `OPEN "filename" FOR INPUT|OUTPUT AS #handle`
    *   Opens a file for subsequent reading (`INPUT`) or writing (`OUTPUT`) operations.
    *   Assigns a unique numerical handle (starting from `#1`) to the opened file. This handle is used by other file I/O commands.
    *   Example: `OPEN "DATA.TXT" FOR INPUT AS #1`, `OPEN "OUTPUT.LOG" FOR OUTPUT AS #2`

*   **CLOSE**
    *   Syntax: `CLOSE #handle`
    *   Closes an opened file identified by its handle. Any buffered output for files opened `FOR OUTPUT` will be written when the file is closed.

*   **INPUT #**
    *   Syntax: `INPUT #handle, var1 [, var2...]`
    *   Reads comma-separated values from the file opened with `#handle` into the specified variables. Handles both numeric and string variables. Skips empty or whitespace-only lines.
    *   Example: `INPUT #1, A, B, C$`, `INPUT #2, VALUE`

*   **LINE INPUT #**
    *   Syntax: `LINE INPUT #handle, stringVar$`
    *   Reads an entire line (including spaces and commas) from the file opened with `#handle` into the specified string variable. Does *not* skip empty lines.
    *   Example: `LINE INPUT #1, FULLLINE$`

*   **PRINT #**
    *   Syntax: `PRINT #handle, [expr][,|;]...`
    *   Writes the value of expressions to the file opened with `#handle`. Formatting works similar to console `PRINT` regarding commas and semicolons.
    *   Example: `PRINT #2, "Logging value: "; X`

### Flow Control

*   **GOTO**
    *   Syntax: `GOTO lineNumber`
    *   Transfers program execution unconditionally to the specified line number.
    *   Example: `GOTO 100`

*   **GOSUB**
    *   Syntax: `GOSUB lineNumber`
    *   Transfers program execution to a subroutine starting at `lineNumber`. The current line number is pushed onto a stack so that execution can return later.
    *   Example: `GOSUB 500`

*   **RETURN**
    *   Syntax: `RETURN`
    *   Transfers execution back to the statement immediately following the most recent `GOSUB` call.

*   **IF / THEN**
    *   Syntax: `IF condition THEN statement`
    *   Evaluates a `condition`. If the condition is true, the `statement` following `THEN` is executed.
    *   Numeric conditions are true if the value is non-zero. String conditions are true if the string is non-empty.
    *   The `statement` after `THEN` can be any valid BASIC command.
    *   Example: `IF A > 10 THEN PRINT "A IS LARGE"`, `IF B$ = "YES" THEN GOTO 200`

*   **FOR / NEXT**
    *   Syntax: `FOR var = start TO end [STEP value]`
    *   `NEXT [var]`
    *   Creates a loop that executes a block of code repeatedly. The loop variable `var` is set to `start`, and incremented by `STEP` (default is 1). The loop continues until `var` passes `end`.
    *   The loop body consists of the statements between the `FOR` line and the matching `NEXT` line.
    *   The `NEXT` command marks the end of the loop body. If `var` is omitted, the inner-most active loop's variable is assumed.
    *   Example: `FOR I = 1 TO 10 : PRINT I : NEXT I`, `FOR J = 100 TO 0 STEP -10 : PRINT J : NEXT`

### Data Management

*   **DATA**
    *   Syntax: `DATA item1, item2, ...`
    *   Stores a list of data items (numbers or strings) within your program. These items can be read sequentially using the `READ` command. `DATA` statements themselves do not execute at runtime.
    *   String data items containing commas or leading/trailing spaces should be enclosed in quotes. Double quotes within a quoted string are represented by two double quotes (`""`).
    *   Example: `DATA 10, 20, "HELLO", 3.14`

*   **READ**
    *   Syntax: `READ var1, var2, ...`
    *   Reads the next available data item(s) from the `DATA` statements in the program and assigns them to the specified variables. The data pointer is reset by `RESTORE` or `RUN`. Handles type conversion automatically (numeric data into numeric vars, string data into string vars).
    *   Example: `READ A, B, C$`

*   **RESTORE**
    *   Syntax: `RESTORE`
    *   Resets the data pointer back to the beginning of the first `DATA` statement in the program, allowing `READ` statements to start over.

### Multimedia

*   **BEEP**
    *   Syntax: `BEEP`
    *   Sends a signal to the terminal to produce a short beep sound.

*   **SOUND**
    *   Syntax: `SOUND frequency, duration`
    *   Plays a tone with the specified `frequency` (in Hz) for the given `duration` (in milliseconds).
    *   Example: `SOUND 440, 500` (Plays an A note for half a second)

*   **SAY / SPEAK**
    *   Syntax: `SAY "text"` or `SPEAK "text"`
    *   Both commands are identical and send the specified text to the frontend for speech synthesis.
    *   Optional: Add `, WAIT` after the text expression to pause program execution until the frontend reports that speech is finished. This prevents the program from continuing while the computer is still talking.
    *   Example: `SAY "Hello, World!"`, `SPEAK MESSAGE$, WAIT`

### Graphics (Partially Implemented)

TinyBASIC includes commands for drawing graphics using the terminal's character grid and brightness levels. The graphics commands are processed by the backend.

*   **INK**
    *   Syntax: `INK color`
    *   Sets the current drawing brightness level (0-15) for subsequent graphics commands.

*   **PLOT**
    *   Syntax: `PLOT x, y`
    *   Draws a single point at coordinates (x, y) using the current INK color. Coordinates are rounded to the nearest integer.

*   **DRAW**
    *   Syntax: `DRAW x1, y1, x2, y2`
    *   Draws a line from point (x1, y1) to (x2, y2) using the current INK color. Coordinates are rounded to the nearest integer.

*   **CIRCLE**
    *   Syntax: `CIRCLE x, y, radius`
    *   Draws a circle centered at (x, y) with the specified radius. Coordinates are rounded to the nearest integer. (Note: Source indicates "Diverse Grafikbefehle (noch nicht vollständig implementiert)", and `gfx_commands.go` exists, but detailed implementation isn't fully visible across all sources provided. Syntax comes from helpTexts and commandUsageHints).

*   **RECT**
    *   Syntax: `RECT x, y, width, height`
    *   Draws a rectangle with its top-left corner at (x, y) and the specified width and height. Coordinates are rounded to the nearest integer.

*   **FILL**
    *   Syntax: `FILL x, y`
    *   Starts a flood fill operation from the point (x, y) using the current INK color. Coordinates are rounded to the nearest integer.

*   **POLY**
    *   Syntax: `POLY x1, y1, x2, y2, ...`
    *   Draws a polygon or polyline connecting the specified points using the current INK color. Requires an even number of coordinate values (at least 4 for one line segment). Coordinates are rounded to the nearest integer.

### Utility

*   **RANDOMIZE**
    *   Syntax: `RANDOMIZE` or `RANDOMIZE number`
    *   Initializes the random number generator. If no number is provided, the current time is used as the seed. If a number is provided, that number is used as the seed. Useful for getting different sequences of "random" numbers from the `RND` function.

## BUILT-IN FUNCTIONS

Functions return a value and are used within expressions.

*   `RND(numeric_expr)`: Returns a pseudo-random number. The argument's value might influence the sequence or range (details vary by BASIC implementation, but typically `RND(1)` gives a new value).
*   `EOF(handle)`: Returns true (-1) if the end of the file specified by `handle` has been reached during reading, false (0) otherwise.
*   Other functions mentioned (details not fully available in provided sources but listed as "known functions"): `ABS`, `ATN`, `COS`, `EXP`, `INT`, `LOG`, `SGN`, `SIN`, `SQR`, `TAN`, `CHR$`, `LEFT$`, `MID$`, `RIGHT$`, `STR$`, `LEN`, `ASC`, `VAL`.

## FILESYSTEM & EXAMPLES

TinyBASIC interacts with the Virtual File System provided by TinyOS. When you `LOAD` or `SAVE` a program, it's stored persistently (for logged-in users) or temporarily (for guest users).

Example BASIC programs are available and are copied to your home directory when you first start TinyBASIC (or log in). You can `LOAD` and `RUN` these examples:
*   `hello.bas`
*   `graphics.bas`
*   `sound_demo.bas` (Includes examples of `SOUND` and `SAY`/`SPEAK`)

Use the `DIR` command to see available `.bas` files.

## ADVANCED TOPICS

*   **Multiple Statements per Line:** You can place multiple BASIC statements on a single line, separated by a colon (`:`).
*   **Synchronization:** The `SAY`/`SPEAK` command with the `, WAIT` option and the `INPUT` command will cause the program to pause execution until the frontend responds, ensuring smooth interaction.
*   **Debugging:** Use `PRINT` statements to display variable values. Error messages include line numbers to help locate issues.

We hope you enjoy programming with TinyBASIC!

```