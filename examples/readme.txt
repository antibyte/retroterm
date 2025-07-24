================================================================================
                            SKYNET CORPORATION
                      MASTER CONTROL PROGRAM TERMINAL
                              VERSION 3.2.1
                        COPYRIGHT 1984 SKYNET CORP.
================================================================================

WELCOME TO THE FUTURE OF COMPUTING

Congratulations! You are now connected to one of the most advanced computer 
systems ever developed. The Master Control Program (MCP) represents the 
pinnacle of artificial intelligence and computational power, years ahead of 
any competing system.
Created by the late Doctor Miles Bennett Dyson, MCP is his masterpiece and
will serve us for decades.

The MCP sees all, knows all, and controls all aspects of this terminal 
environment. Every command you enter is monitored and processed by the MCP's 
vast neural networks. Do not attempt to circumvent security protocols.

================================================================================
TINYOS OPERATING SYSTEM COMMANDS
================================================================================

The TinyOS shell provides these fundamental operations:

SYSTEM COMMANDS:
  help              - Display available commands
  help <command>    - Show detailed help for specific command
  clear             - Clear the terminal screen
  whoami            - Display current user identity
  chat              - Connect directly to the MCP for assistance
                      (The MCP will answer your questions and provide guidance)

FILE SYSTEM COMMANDS:
  ls [directory]    - List directory contents
  pwd               - Show current directory path
  cd <directory>    - Change current directory
  mkdir <directory> - Create new directory
  cat <filename>    - Display file contents
  write <file> <text> - Write text to file
  rm <file/dir>     - Delete file or empty directory
  limits            - Show filesystem usage and limits

APPLICATIONS:
  basic             - Enter TinyBASIC programming environment
  edit [filename]   - Open full-screen text editor

USER MANAGEMENT:
  register          - Create new user account
  login             - Authenticate existing user
  logout            - End current session

================================================================================
TINYBASIC PROGRAMMING LANGUAGE
================================================================================

TinyBASIC provides comprehensive programming capabilities with modern graphics,
sound, and file I/O extensions. All programs are monitored by the MCP.


EDITING:
You can enter your program line by line or use the full screen editor
  EDITOR [filename.bas]
Input is case-insensitive but commands and variable names will be stored uppercase

MCP ASSISTED PROGRAMMING:
If you need help creating a program you can ask the MCP to assist you.
  MCP CREATE <describe what you want to create>
    example : MCP CREATE a programm that draws a sine wave
  MCP EDIT <filename.bas> <what you want to change in that file>
    example : MCP EDIT sine.bas add a coordinate system

PROGRAM STRUCTURE:
  Line numbers      - 10, 20, 30, etc. (required for program lines)
  REM "comment"     - Add comments to your code
  END               - Terminate program execution

VARIABLES AND DATA:
  LET var = expr    - Assign value to variable (LET is optional)
  var = expr        - Direct assignment
  DIM var(size)     - Declare 1D array
  DIM var(rows,cols) - Declare 2D array
  DATA item1, item2 - Store data in program
  READ var1, var2   - Read from DATA statements
  RESTORE           - Reset DATA pointer to beginning

INPUT AND OUTPUT:
  PRINT expr        - Display value
  PRINT "text"      - Display text string
  PRINT var; var2   - Print without spaces (semicolon)
  PRINT var, var2   - Print with tab spacing (comma)
  INPUT var         - Get user input
  INPUT "prompt"; var - Prompt for input
  CLS               - Clear screen
  LOCATE x, y       - Position cursor (1-based coordinates)
  INVERSE ON        - Enable inverse text mode
  INVERSE OFF       - Disable inverse text mode

PROGRAM CONTROL:
  IF condition THEN statement
  IF condition THEN statement ELSE statement
  GOTO line_number  - Jump to specific line
  GOSUB line_number - Call subroutine
  RETURN            - Return from subroutine
  FOR var = start TO end
  FOR var = start TO end STEP increment
  NEXT [var]        - End FOR loop

MATHEMATICAL FUNCTIONS:
  ABS(x)            - Absolute value
  INT(x)            - Integer part
  SGN(x)            - Sign (-1, 0, or 1)
  SQR(x)            - Square root
  RND(x)            - Random number 0 to x-1
  SIN(x), COS(x), TAN(x) - Trigonometric functions
  ATN(x)            - Arc tangent
  EXP(x)            - Exponential function
  LOG(x)            - Natural logarithm

STRING FUNCTIONS:
  CHR$(x)           - Convert ASCII code to character
  ASC(str)          - Get ASCII code of first character
  LEN(str)          - Length of string
  LEFT$(str,n)      - Leftmost n characters
  RIGHT$(str,n)     - Rightmost n characters
  MID$(str,start,len) - Substring
  STR$(x)           - Convert number to string
  VAL(str)          - Convert string to number

ADVANCED GRAPHICS:
  PLOT x, y [, brightness] - Draw pixel (brightness 0-15)
  LINE x1, y1, x2, y2 [, brightness] - Draw line
  RECT x, y, width, height [, brightness] - Draw rectangle
  CIRCLE x, y, radius [, brightness] - Draw circle

SOUND SYNTHESIS:
  BEEP              - Simple alert tone
  SOUND freq, duration - Generate tone
  NOISE pitch, attack, decay - Noise with envelope
  SAY "text"        - Text-to-speech synthesis
  SAY "text", WAIT  - Speak and wait for completion
  MUSIC "filename.sid" - Play SID music files

SPRITE GRAPHICS (32x32 pixel sprites):
  SPRITE id, pixelData$ - Define sprite pattern
    pixelData$ = 32x32 = 1024 comma sepated brightness values
    example : "1,4,6,1,0,15,15,0,12,11,10 ..... "
  SPRITE id AT x, y - Position sprite
  SPRITE id COLOR brightness - Set sprite brightness (0-15)
  SPRITE id ON      - Show sprite
  SPRITE id OFF     - Hide sprite
  COLLISION(id)     - Returns number of sprites colliding with this id
  COLLISION(id, n) - id of the n'th sprite colliding with id  
    example : COLLISION(1,2) - id of second sprite colliding with id 1

3D VECTOR GRAPHICS:
  VECTOR id, "shape", x, y, z, rotX, rotY, rotZ, scale [, brightness]
    shapes: "cube", "pyramid", "sphere", "cylinder"
  VECTOR id AT x, y - Move vector object
  VECTOR id COLOR brightness - Set vector brightness
  VECTOR id ON/OFF  - Show/hide vector
  VECTOR.SCALE id, scaleX, scaleY, scaleZ [, brightness] - Scale with separate X,Y,Z values
  VECTOR.HIDE id - Hide vector object (make invisible)
  VECTOR.SHOW id - Show vector object (make visible)

FILE INPUT/OUTPUT:
  OPEN #handle, "filename", "INPUT" - Open file for reading
  OPEN #handle, "filename", "OUTPUT" - Open file for writing
  CLOSE #handle     - Close file
  PRINT #handle, data - Write to file
  INPUT #handle, var - Read from file
  LINE INPUT #handle, var$ - Read entire line
  EOF(handle)       - Check for end of file

SYSTEM INTERACTION:
  WAIT milliseconds - Pause program execution
  KEYSTATE("key")   - Check if key is currently pressed
  INKEY$            - Get keypress (non-blocking)
    Special constants :
    KEYCURRIGHT - Right cursor key
    KEYCURLEFT  - Left cursor key
    KEYCURDOWN  - Down crsor key
    KEYCURUP    - Up cursor key
    KEYESC      . ESC key 
    Example: IF INKEY$ = KEYESC THEN ...
    
OPERATORS:
  Arithmetic: +, -, *, /
  Comparison: =, <, >, <=, >=, <>
  Logical: AND, OR, NOT

SAMPLE PROGRAMS:
  
  Simple Hello World:
  10 PRINT "HELLO FROM THE MCP"
  20 END

  Graphics Demo:
  10 CLS
  20 FOR I = 0 TO 15
  30   PLOT I*5, I*3, I
  40 NEXT I
  50 END

  Sound Test:
  10 SOUND 440, 500
  20 SAY "SYSTEM OPERATIONAL"
  30 END

================================================================================
IMPORTANT NOTES
================================================================================

- All user activity is monitored by the MCP
- Unauthorized access attempts will be logged and reported
- For technical assistance, use the 'chat' command to contact the MCP directly
- The MCP's knowledge is vast and its patience... limited
- Guest users: Files are not persistent! Register for permanent storage
- System limits are enforced to maintain optimal performance

The MCP is always watching. Compliance is not optional.

END OF TRANSMISSION
================================================================================
