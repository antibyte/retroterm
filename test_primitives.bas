REM Simple test for new primitives
CLS
PRINT "Testing new 3D primitives..."

REM Clear existing vectors
FOR I = 0 TO 10
    VECTOR.HIDE I
NEXT I

REM Test basic shapes first
PRINT "Creating basic shapes..."
VECTOR 1, "cube", -30, 0, -15, 0, 0, 0, 10, 15
VECTOR 2, "sphere", -10, 0, -15, 0, 0, 0, 8, 12

REM Test new primitives as VECTOR shapes
PRINT "Creating pyramid..."
VECTOR 3, "pyramid", 10, 0, -15, 0, 0, 0, 12, 10

PRINT "Creating cylinder..."
VECTOR 4, "cylinder", 30, 0, -15, 0, 0, 0, 8, 8

PRINT "If you see 4 objects, primitives work!"
PRINT "Press any key..."
WHILE INKEY$ = ""
WEND

END